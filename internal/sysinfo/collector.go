package sysinfo

import (
	"log/slog"
	"time"
)

type liveCollector struct {
	logger *slog.Logger
	reader SystemReader

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
	prevNetRx, prevNetTx              uint64
	prevDiskRead, prevDiskWrite       uint64
	prevDiskReadOps, prevDiskWriteOps uint64
	lastRefresh                       time.Time
}

// NewCollector creates a live system metrics collector.
func NewCollector(logger *slog.Logger) Collector {
	return NewCollectorWithReader(NewSystemReader(logger), logger)
}

// NewCollectorWithReader creates a collector with a custom SystemReader,
// useful for testing the delta/rate logic.
func NewCollectorWithReader(r SystemReader, logger *slog.Logger) Collector {
	c := &liveCollector{reader: r, logger: logger}
	c.Refresh()
	return c
}

func (c *liveCollector) CPUPercent() float64        { return c.cpu }
func (c *liveCollector) RAMPercent() float64        { return c.ram }
func (c *liveCollector) DiskPercent() float64       { return c.disk }
func (c *liveCollector) Temperature() float64       { return c.temp }
func (c *liveCollector) Hostname() string           { return c.hostname }
func (c *liveCollector) Uptime() time.Duration      { return c.uptime }
func (c *liveCollector) IPv4Address() string        { return c.ipv4 }
func (c *liveCollector) IPv6Suffix() string         { return c.ipv6 }
func (c *liveCollector) NetBandwidth() NetBandwidth { return c.net }
func (c *liveCollector) LinkSpeedMbps() int         { return c.linkSpeed }
func (c *liveCollector) DiskIO() DiskIO             { return c.diskIO }
func (c *liveCollector) CPUFreq() CPUFreq           { return c.freq }
func (c *liveCollector) ThrottleStatus() uint32     { return c.throttle }
func (c *liveCollector) DietPiStatus() DietPiStatus { return c.dietpi }
func (c *liveCollector) APTUpdateCount() int        { return c.apt }

// Refresh collects all system metrics.
func (c *liveCollector) Refresh() {
	now := time.Now()
	elapsed := now.Sub(c.lastRefresh).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}

	c.cpu = c.reader.CPUPercent()
	c.ram = c.reader.RAMPercent()
	c.disk = c.reader.DiskUsage()
	c.temp = c.reader.Temperature()
	c.hostname = c.reader.Hostname()
	c.uptime = c.reader.Uptime()

	c.refreshNetwork(elapsed)
	c.refreshDiskIO(elapsed)

	c.freq = c.reader.CPUFreq()
	c.throttle = c.reader.ThrottleStatus()
	c.dietpi = c.reader.DietPiStatus()
	c.apt = c.reader.APTUpdateCount()

	c.lastRefresh = now
}

func (c *liveCollector) refreshNetwork(elapsed float64) {
	iface := c.reader.DefaultInterface()
	if iface == "" {
		c.ipv4 = "no network"
		c.ipv6 = NoIPv6
		return
	}

	c.ipv4, c.ipv6 = c.reader.InterfaceAddresses(iface)
	c.linkSpeed = c.reader.LinkSpeed(iface)

	rx, tx := c.reader.NetIOCounters(iface)
	if (c.prevNetRx > 0 || c.prevNetTx > 0) &&
		rx >= c.prevNetRx && tx >= c.prevNetTx {
		c.net = NetBandwidth{
			RxBytesPerSec: uint64(float64(rx-c.prevNetRx) / elapsed),
			TxBytesPerSec: uint64(float64(tx-c.prevNetTx) / elapsed),
		}
	}
	c.prevNetRx = rx
	c.prevNetTx = tx
}

func (c *liveCollector) refreshDiskIO(elapsed float64) {
	totalRead, totalWrite, totalReadOps, totalWriteOps := c.reader.DiskIOCounters()

	if (c.prevDiskRead > 0 || c.prevDiskWrite > 0) &&
		totalRead >= c.prevDiskRead && totalWrite >= c.prevDiskWrite {
		c.diskIO = DiskIO{
			ReadBytesPerSec:  uint64(float64(totalRead-c.prevDiskRead) / elapsed),
			WriteBytesPerSec: uint64(float64(totalWrite-c.prevDiskWrite) / elapsed),
			ReadIOPS:         uint64(float64(totalReadOps-c.prevDiskReadOps) / elapsed),
			WriteIOPS:        uint64(float64(totalWriteOps-c.prevDiskWriteOps) / elapsed),
		}
	}
	c.prevDiskRead = totalRead
	c.prevDiskWrite = totalWrite
	c.prevDiskReadOps = totalReadOps
	c.prevDiskWriteOps = totalWriteOps
}
