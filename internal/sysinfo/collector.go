package sysinfo

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	psnet "github.com/shirou/gopsutil/v4/net"
)

const (
	thermalPath      = "/sys/class/thermal/thermal_zone0/temp"
	procRoutePath    = "/proc/net/route"
	defaultRouteDest = "00000000"
	procMountsPath   = "/proc/mounts"
	NoIPv6           = "no IPv6"
	linkSpeedPath    = "/sys/class/net/%s/speed"
)

type liveCollector struct {
	logger *slog.Logger

	// cached values updated by Refresh()
	cpu       float64
	ram       float64
	disk      float64
	temp      float64
	hostname  string
	uptime    time.Duration
	ipv4      string
	ipv6      string
	net       NetBandwidth
	linkSpeed int
	diskIO    DiskIO
	freq      CPUFreq
	throttle  uint32
	dietpi    DietPiStatus
	apt       int

	// state for delta calculations
	prevNetRx, prevNetTx               uint64
	prevDiskRead, prevDiskWrite        uint64
	prevDiskReadOps, prevDiskWriteOps  uint64
	lastRefresh                        time.Time
}

// NewCollector creates a live system metrics collector.
func NewCollector(logger *slog.Logger) Collector {
	c := &liveCollector{logger: logger}
	c.Refresh()
	return c
}

// CPU
func (c *liveCollector) CPUPercent() float64 { return c.cpu }

// RAM
func (c *liveCollector) RAMPercent() float64 { return c.ram }

// Disk
func (c *liveCollector) DiskPercent() float64 { return c.disk }

// Temperature
func (c *liveCollector) Temperature() float64 { return c.temp }

// Host and uptime
func (c *liveCollector) Hostname() string      { return c.hostname }
func (c *liveCollector) Uptime() time.Duration { return c.uptime }

// Network
func (c *liveCollector) IPv4Address() string        { return c.ipv4 }
func (c *liveCollector) IPv6Suffix() string         { return c.ipv6 }
func (c *liveCollector) NetBandwidth() NetBandwidth { return c.net }
func (c *liveCollector) LinkSpeedMbps() int         { return c.linkSpeed }

// Disk I/O
func (c *liveCollector) DiskIO() DiskIO { return c.diskIO }

// Pi-specific
func (c *liveCollector) CPUFreq() CPUFreq            { return c.freq }
func (c *liveCollector) ThrottleStatus() uint32       { return c.throttle }
func (c *liveCollector) DietPiStatus() DietPiStatus   { return c.dietpi }
func (c *liveCollector) APTUpdateCount() int           { return c.apt }

// Refresh collects all system metrics.
func (c *liveCollector) Refresh() {
	now := time.Now()
	elapsed := now.Sub(c.lastRefresh).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}

	c.refreshCPU()
	c.refreshRAM()
	c.refreshDisk()
	c.refreshTemp()
	c.refreshHostUptime()
	c.refreshNetwork(elapsed)
	c.refreshDiskIO(elapsed)
	c.refreshPi()

	c.lastRefresh = now
}

func (c *liveCollector) refreshCPU() {
	pcts, err := cpu.Percent(0, false)
	if err != nil {
		c.logger.Debug("cpu percent", "err", err)
		return
	}
	if len(pcts) > 0 {
		c.cpu = pcts[0]
	}
}

func (c *liveCollector) refreshRAM() {
	v, err := mem.VirtualMemory()
	if err != nil {
		c.logger.Debug("virtual memory", "err", err)
		return
	}
	c.ram = v.UsedPercent
}

func (c *liveCollector) refreshDisk() {
	c.disk = aggregateDiskUsage()
}

func (c *liveCollector) refreshTemp() {
	data, err := os.ReadFile(thermalPath)
	if err != nil {
		return
	}
	var milliC int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &milliC); err == nil {
		c.temp = float64(milliC) / 1000
	}
}

func (c *liveCollector) refreshHostUptime() {
	info, err := host.Info()
	if err != nil {
		c.logger.Debug("host info", "err", err)
		return
	}
	c.hostname = info.Hostname
	c.uptime = time.Duration(info.Uptime) * time.Second
}

func (c *liveCollector) refreshNetwork(elapsed float64) {
	iface := defaultInterface()
	if iface == "" {
		return
	}

	// IP addresses
	c.ipv4, c.ipv6 = interfaceAddresses(iface)

	// Link speed
	c.linkSpeed = readLinkSpeed(iface)

	// Bandwidth via gopsutil
	counters, err := psnet.IOCounters(true)
	if err != nil {
		c.logger.Debug("net io counters", "err", err)
		return
	}
	for _, s := range counters {
		if s.Name == iface {
			if c.prevNetRx > 0 || c.prevNetTx > 0 {
				c.net = NetBandwidth{
					RxBytesPerSec: uint64(float64(s.BytesRecv-c.prevNetRx) / elapsed),
					TxBytesPerSec: uint64(float64(s.BytesSent-c.prevNetTx) / elapsed),
				}
			}
			c.prevNetRx = s.BytesRecv
			c.prevNetTx = s.BytesSent
			break
		}
	}
}

func (c *liveCollector) refreshDiskIO(elapsed float64) {
	counters, err := disk.IOCounters()
	if err != nil {
		c.logger.Debug("disk io counters", "err", err)
		return
	}

	var totalRead, totalWrite uint64
	var totalReadOps, totalWriteOps uint64
	for name, s := range counters {
		if isWholeDisk(name) {
			totalRead += s.ReadBytes
			totalWrite += s.WriteBytes
			totalReadOps += s.ReadCount
			totalWriteOps += s.WriteCount
		}
	}

	if c.prevDiskRead > 0 || c.prevDiskWrite > 0 {
		c.diskIO = DiskIO{
			ReadBytesPerSec:  uint64(float64(totalRead-c.prevDiskRead) / elapsed),
			WriteBytesPerSec: uint64(float64(totalWrite-c.prevDiskWrite) / elapsed),
			ReadIOPS:         uint32(float64(totalReadOps-c.prevDiskReadOps) / elapsed),
			WriteIOPS:        uint32(float64(totalWriteOps-c.prevDiskWriteOps) / elapsed),
		}
	}
	c.prevDiskRead = totalRead
	c.prevDiskWrite = totalWrite
	c.prevDiskReadOps = totalReadOps
	c.prevDiskWriteOps = totalWriteOps
}

func (c *liveCollector) refreshPi() {
	c.freq = readCPUFreq()
	c.throttle = readThrottleStatus()
	c.dietpi = readDietPiStatus()
	c.apt = readAPTUpdateCount()
}

// defaultInterface finds the network interface used for the default route.
// defaultInterface finds the network interface used for the default route
// by parsing /proc/net/route.
func defaultInterface() string {
	f, err := os.Open(procRoutePath)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == defaultRouteDest {
			return fields[0]
		}
	}
	return ""
}

// interfaceAddresses returns the IPv4 address and IPv6 suffix for the named interface.
func interfaceAddresses(name string) (ipv4, ipv6suffix string) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return "", NoIPv6
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", NoIPv6
	}

	ipv6suffix = NoIPv6
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			ipv4 = ip4.String()
		} else if ip6 := ipnet.IP.To16(); ip6 != nil && !ip6.IsLinkLocalUnicast() {
			// Return host portion: all groups after the prefix length.
			ones, _ := ipnet.Mask.Size()
			firstGroup := ones / 16
			var groups []string
			for g := firstGroup; g < 8; g++ {
				val := uint16(ip6[g*2])<<8 | uint16(ip6[g*2+1])
				groups = append(groups, fmt.Sprintf("%x", val))
			}
			if len(groups) > 0 {
				ipv6suffix = "::" + strings.Join(groups, ":")
			}
		}
	}
	return ipv4, ipv6suffix
}

// readLinkSpeed reads the link speed in Mbps from sysfs.
func readLinkSpeed(iface string) int {
	data, err := os.ReadFile(fmt.Sprintf(linkSpeedPath, iface))
	if err != nil {
		return 0
	}
	var speed int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &speed); err != nil {
		return 0
	}
	return max(speed, 0)
}

// aggregateDiskUsage computes the weighted disk usage percentage across
// the root filesystem and any sda/nvme mount points.
func aggregateDiskUsage() float64 {
	mounts := diskMountPoints()
	if len(mounts) == 0 {
		mounts = []string{"/"}
	}

	var totalSize, totalUsed uint64
	seen := make(map[string]bool)
	for _, mp := range mounts {
		if seen[mp] {
			continue
		}
		seen[mp] = true

		var stat syscall.Statfs_t
		if err := syscall.Statfs(mp, &stat); err != nil {
			continue
		}
		size := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bavail * uint64(stat.Bsize)
		if size == 0 {
			continue
		}
		totalSize += size
		totalUsed += size - free
	}
	if totalSize == 0 {
		return 0
	}
	return float64(totalUsed) / float64(totalSize) * 100
}

// diskMountPoints returns mount points for root and sda/nvme devices.
func diskMountPoints() []string {
	result := []string{"/"}

	f, err := os.Open(procMountsPath)
	if err != nil {
		return result
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		dev := fields[0]
		mp := fields[1]
		if strings.Contains(dev, "/sda") || strings.Contains(dev, "/nvme") {
			result = append(result, mp)
		}
	}
	return result
}

// isWholeDisk returns true for whole-disk device names (not partitions).
func isWholeDisk(name string) bool {
	switch {
	case strings.HasPrefix(name, "sd") && len(name) == 3:
		return true // sda, sdb
	case strings.HasPrefix(name, "mmcblk") && !strings.Contains(name, "p"):
		return true // mmcblk0
	case strings.HasPrefix(name, "nvme") && strings.HasSuffix(name, "n1") && !strings.Contains(name, "p"):
		return true // nvme0n1
	}
	return false
}
