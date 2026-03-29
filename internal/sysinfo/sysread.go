package sysinfo

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	thermalPath      = "/sys/class/thermal/thermal_zone0/temp"
	procRoutePath    = "/proc/net/route"
	defaultRouteDest = "00000000"
	procMountsPath   = "/proc/mounts"
	linkSpeedPath    = "/sys/class/net/%s/speed"

	procStatPath      = "/proc/stat"
	procMeminfoPath   = "/proc/meminfo"
	procUptimePath    = "/proc/uptime"
	netStatPath       = "/sys/class/net/%s/statistics/%s"
	procDiskstatsPath = "/proc/diskstats"
	sectorSize        = 512

	cpuFreqPath      = "/sys/devices/system/cpu/cpu0/cpufreq/"
	cpuFreqCurPath   = cpuFreqPath + "scaling_cur_freq"
	cpuFreqMinPath   = cpuFreqPath + "cpuinfo_min_freq"
	cpuFreqMaxPath   = cpuFreqPath + "cpuinfo_max_freq"
	vcioPath         = "/dev/vcio"
	tagGetThrottled  = 0x00030046
	ioctlMailbox     = 0xC0046400
	mailboxSuccess   = 0x80000000
	dietpiRunPath    = "/run/dietpi"
	dietpiUpdatePath = dietpiRunPath + "/.update_available"
	dietpiAPTPath    = dietpiRunPath + "/.apt_updates"
)

type linuxReader struct {
	logger    *slog.Logger
	prevIdle  uint64
	prevTotal uint64
}

// NewSystemReader creates a SystemReader that reads from Linux system files.
func NewSystemReader(logger *slog.Logger) SystemReader {
	return &linuxReader{logger: logger}
}

func (r *linuxReader) CPUPercent() float64 {
	f, err := os.Open(procStatPath)
	if err != nil {
		r.logger.Debug("failed to open /proc/stat", "err", err)
		return 0
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0
	}

	var idle, total uint64
	for i, s := range fields[1:] {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			continue
		}
		total += v
		if i == 3 || i == 4 { // idle and iowait
			idle += v
		}
	}

	deltaIdle := idle - r.prevIdle
	deltaTotal := total - r.prevTotal
	r.prevIdle = idle
	r.prevTotal = total

	if deltaTotal == 0 {
		return 0
	}
	return (1 - float64(deltaIdle)/float64(deltaTotal)) * 100
}

func (r *linuxReader) RAMPercent() float64 {
	f, err := os.Open(procMeminfoPath)
	if err != nil {
		r.logger.Debug("failed to open /proc/meminfo", "err", err)
		return 0
	}
	defer func() { _ = f.Close() }()

	var total, available uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			total = parseKBLine(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			available = parseKBLine(line)
		}
		if total > 0 && available > 0 {
			break
		}
	}
	if total == 0 {
		return 0
	}
	return float64(total-available) / float64(total) * 100
}

func (r *linuxReader) Hostname() string {
	name, err := os.Hostname()
	if err != nil {
		r.logger.Debug("failed to read hostname", "err", err)
		return ""
	}
	return name
}

func (r *linuxReader) Uptime() time.Duration {
	data, err := os.ReadFile(procUptimePath)
	if err != nil {
		r.logger.Debug("failed to read /proc/uptime", "err", err)
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		r.logger.Debug("failed to parse uptime", "err", err)
		return 0
	}
	return time.Duration(secs * float64(time.Second))
}

func (r *linuxReader) NetIOCounters(iface string) (rx, tx uint64) {
	rx = r.readUint64File(fmt.Sprintf(netStatPath, iface, "rx_bytes"))
	tx = r.readUint64File(fmt.Sprintf(netStatPath, iface, "tx_bytes"))
	return rx, tx
}

func (r *linuxReader) DiskIOCounters() (read, write, readOps, writeOps uint64) {
	f, err := os.Open(procDiskstatsPath)
	if err != nil {
		r.logger.Debug("failed to open /proc/diskstats", "err", err)
		return 0, 0, 0, 0
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		name := fields[2]
		if !isWholeDisk(name) {
			continue
		}
		rOps, _ := strconv.ParseUint(fields[3], 10, 64)
		rSectors, _ := strconv.ParseUint(fields[5], 10, 64)
		wOps, _ := strconv.ParseUint(fields[7], 10, 64)
		wSectors, _ := strconv.ParseUint(fields[9], 10, 64)

		read += rSectors * sectorSize
		write += wSectors * sectorSize
		readOps += rOps
		writeOps += wOps
	}
	return read, write, readOps, writeOps
}

func (r *linuxReader) readUint64File(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		r.logger.Debug("failed to read file", "path", path, "err", err)
		return 0
	}
	v, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	return v
}

func parseKBLine(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, _ := strconv.ParseUint(fields[1], 10, 64)
	return v
}

// --- Direct system reads ---

func (r *linuxReader) Temperature() float64 {
	data, err := os.ReadFile(thermalPath)
	if err != nil {
		r.logger.Debug("failed to read temperature", "err", err)
		return 0
	}
	milliC, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		r.logger.Debug("failed to parse temperature", "err", err)
		return 0
	}
	return float64(milliC) / 1000
}

func (r *linuxReader) DiskUsage() float64 {
	return aggregateDiskUsage()
}

func (r *linuxReader) DefaultInterface() string {
	f, err := os.Open(procRoutePath)
	if err != nil {
		r.logger.Debug("failed to open /proc/net/route", "err", err)
		return ""
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == defaultRouteDest {
			return fields[0]
		}
	}
	return ""
}

func (r *linuxReader) InterfaceAddresses(name string) (ipv4, ipv6suffix string) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		r.logger.Debug("failed to get interface", "name", name, "err", err)
		return "", NoIPv6
	}
	addrs, err := iface.Addrs()
	if err != nil {
		r.logger.Debug("failed to get interface addresses", "name", name, "err", err)
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

func (r *linuxReader) LinkSpeed(iface string) int {
	data, err := os.ReadFile(fmt.Sprintf(linkSpeedPath, iface))
	if err != nil {
		r.logger.Debug("failed to read link speed", "iface", iface, "err", err)
		return 0
	}
	speed, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		r.logger.Debug("failed to parse link speed", "iface", iface, "err", err)
		return 0
	}
	return max(speed, 0)
}

// --- Pi-specific reads (moved from pi.go) ---

func (r *linuxReader) CPUFreq() CPUFreq {
	read := func(path string) uint16 {
		data, err := os.ReadFile(path)
		if err != nil {
			return 0
		}
		kHz, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return 0
		}
		return uint16(kHz / 1000)
	}

	return CPUFreq{
		Cur: read(cpuFreqCurPath),
		Min: read(cpuFreqMinPath),
		Max: read(cpuFreqMaxPath),
	}
}

func (r *linuxReader) ThrottleStatus() uint32 {
	f, err := os.OpenFile(vcioPath, os.O_RDWR, 0)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()

	buf := [8]uint32{
		32,              // buffer size
		0,               // request code
		tagGetThrottled, // GET_THROTTLED
		4,               // value buffer size
		0,               // request/response indicator
		0,               // value (filled by firmware)
		0,               // end tag
		0,
	}

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		ioctlMailbox,
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if errno != 0 {
		return 0
	}
	if buf[1] != mailboxSuccess {
		return 0
	}
	return buf[5]
}

func (r *linuxReader) DietPiStatus() DietPiStatus {
	if !isDietPi() {
		return DietPiNotInstalled
	}
	if _, err := os.Stat(dietpiUpdatePath); err == nil {
		return DietPiUpdateAvail
	}
	return DietPiUpToDate
}

func (r *linuxReader) APTUpdateCount() int {
	if !isDietPi() {
		return -1
	}
	data, err := os.ReadFile(dietpiAPTPath)
	if err != nil {
		return 0
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return count
}

// --- Private helpers ---

// isDietPi reports whether the system is running DietPi.
// Checked once and cached — can't change during program lifetime.
var isDietPi = sync.OnceValue(func() bool {
	_, err := os.Stat(dietpiRunPath)
	return err == nil
})

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
	defer func() { _ = f.Close() }()

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
