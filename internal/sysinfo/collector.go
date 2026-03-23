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

type liveCollector struct {
	logger *slog.Logger

	// cached values updated by Refresh()
	cpu       float64
	ram       float64
	disk      float64
	temp      float64
	hostname  string
	ip        string
	ipv6      string
	freq      CPUFreq
	net       NetBandwidth
	diskIO    DiskIO
	uptime    time.Duration
	throttle  uint32
	dietpi    DietPiStatus
	apt       int
	linkSpeed int

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

func (c *liveCollector) CPUPercent() float64       { return c.cpu }
func (c *liveCollector) RAMPercent() float64        { return c.ram }
func (c *liveCollector) DiskPercent() float64       { return c.disk }
func (c *liveCollector) Temperature() float64       { return c.temp }
func (c *liveCollector) Hostname() string           { return c.hostname }
func (c *liveCollector) IPAddress() string          { return c.ip }
func (c *liveCollector) IPv6Suffix() string         { return c.ipv6 }
func (c *liveCollector) CPUFreq() CPUFreq           { return c.freq }
func (c *liveCollector) NetBandwidth() NetBandwidth { return c.net }
func (c *liveCollector) DiskIO() DiskIO             { return c.diskIO }
func (c *liveCollector) Uptime() time.Duration      { return c.uptime }
func (c *liveCollector) ThrottleStatus() uint32     { return c.throttle }
func (c *liveCollector) DietPiStatus() DietPiStatus { return c.dietpi }
func (c *liveCollector) APTUpdateCount() int        { return c.apt }
func (c *liveCollector) LinkSpeedMbps() int         { return c.linkSpeed }

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
	// Try reading directly from sysfs (works on Linux).
	if data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp"); err == nil {
		var milliC int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &milliC); err == nil {
			c.temp = float64(milliC) / 1000.0
			return
		}
	}
	// Fallback: gopsutil sensors (works on macOS via smc).
	// On unsupported platforms this returns 0.
	c.temp = 0
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
	c.ip, c.ipv6 = interfaceAddresses(iface)

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
// On Linux it parses /proc/net/route; on other platforms it falls back to
// finding the first non-loopback interface with an IPv4 address.
func defaultInterface() string {
	f, err := os.Open("/proc/net/route")
	if err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 2 && fields[1] == "00000000" {
				return fields[0]
			}
		}
	}

	// Fallback for non-Linux (e.g. macOS during development).
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				return iface.Name
			}
		}
	}
	return ""
}

// interfaceAddresses returns the IPv4 address and IPv6 suffix for the named interface.
func interfaceAddresses(name string) (ipv4, ipv6suffix string) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return "", "no IPv6"
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", "no IPv6"
	}

	ipv6suffix = "no IPv6"
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			ipv4 = ip4.String()
		} else if ip6 := ipnet.IP.To16(); ip6 != nil && !ip6.IsLinkLocalUnicast() {
			// Return last segment of the IPv6 address.
			full := ipnet.IP.String()
			parts := strings.Split(full, ":")
			if len(parts) > 0 {
				ipv6suffix = parts[len(parts)-1]
				if ipv6suffix == "" {
					ipv6suffix = "0"
				}
			}
		}
	}
	return ipv4, ipv6suffix
}

// readLinkSpeed reads the link speed in Mbps from sysfs.
func readLinkSpeed(iface string) int {
	data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/speed", iface))
	if err != nil {
		return 0
	}
	var speed int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &speed); err != nil {
		return 0
	}
	if speed < 0 {
		return 0
	}
	return speed
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

	f, err := os.Open("/proc/mounts")
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
	// macOS disk names
	case name == "disk0" || name == "disk1" || name == "disk2":
		return true
	}
	return false
}
