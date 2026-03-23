package sysinfo

import "time"

// CPUFreq holds current, minimum, and maximum CPU frequency in MHz.
type CPUFreq struct {
	Cur, Min, Max uint16
}

// NetBandwidth holds per-second network throughput.
type NetBandwidth struct {
	RxBytesPerSec, TxBytesPerSec uint64
}

// DiskIO holds per-second disk throughput and IOPS.
type DiskIO struct {
	ReadBytesPerSec, WriteBytesPerSec uint64
	ReadIOPS, WriteIOPS               uint32
}

// DietPiStatus indicates whether DietPi is installed and up to date.
type DietPiStatus int

const (
	DietPiNotInstalled DietPiStatus = iota
	DietPiUpToDate
	DietPiUpdateAvail
)

// Throttle status bitmasks (from VideoCore firmware).
const (
	ThrottleCurrentMask = 0x0000000F // bits 0-3: currently throttled
	ThrottlePastMask    = 0x000F0000 // bits 16-19: throttled since boot
)

// Collector provides system metrics for display on the LCD.
type Collector interface {
	CPUPercent() float64
	RAMPercent() float64
	DiskPercent() float64
	Temperature() float64
	Hostname() string
	IPv4Address() string
	IPv6Suffix() string
	CPUFreq() CPUFreq
	NetBandwidth() NetBandwidth
	DiskIO() DiskIO
	Uptime() time.Duration
	ThrottleStatus() uint32
	DietPiStatus() DietPiStatus
	APTUpdateCount() int
	LinkSpeedMbps() int
	Refresh()
}
