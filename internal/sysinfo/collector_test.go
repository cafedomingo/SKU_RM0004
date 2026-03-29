//go:build linux

package sysinfo

import (
	"log/slog"
	"testing"
	"time"
)

func TestCPUPercentRange(t *testing.T) {
	c := NewCollector(slog.Default())
	v := c.CPUPercent()
	if v < 0 || v > 100 {
		t.Errorf("CPUPercent() = %f, want 0-100", v)
	}
}

func TestRAMPercentRange(t *testing.T) {
	c := NewCollector(slog.Default())
	v := c.RAMPercent()
	if v < 1 || v > 100 {
		t.Errorf("RAMPercent() = %f, want 1-100", v)
	}
}

func TestHostnameNonEmpty(t *testing.T) {
	c := NewCollector(slog.Default())
	if c.Hostname() == "" {
		t.Error("Hostname() returned empty string")
	}
}

func TestUptimePositive(t *testing.T) {
	c := NewCollector(slog.Default())
	if c.Uptime() <= 0 {
		t.Errorf("Uptime() = %v, want > 0", c.Uptime())
	}
}

// fakeReader is a test double for SystemReader that returns configured values.
type fakeReader struct {
	cpu       float64
	ram       float64
	temp      float64
	diskUsage float64
	host      string
	up        time.Duration
	iface     string
	ipv4      string
	ipv6      string
	linkSpeed int
	netRx     uint64
	netTx     uint64
	diskRead  uint64
	diskWrite uint64
	diskROps  uint64
	diskWOps  uint64
	freq      CPUFreq
	throttle  uint32
	dietpi    DietPiStatus
	apt       int
}

func (f *fakeReader) CPUPercent() float64                        { return f.cpu }
func (f *fakeReader) RAMPercent() float64                        { return f.ram }
func (f *fakeReader) Temperature() float64                       { return f.temp }
func (f *fakeReader) DiskUsage() float64                         { return f.diskUsage }
func (f *fakeReader) Hostname() string                           { return f.host }
func (f *fakeReader) Uptime() time.Duration                      { return f.up }
func (f *fakeReader) DefaultInterface() string                   { return f.iface }
func (f *fakeReader) InterfaceAddresses(string) (string, string) { return f.ipv4, f.ipv6 }
func (f *fakeReader) LinkSpeed(string) int                       { return f.linkSpeed }
func (f *fakeReader) NetIOCounters(string) (uint64, uint64)      { return f.netRx, f.netTx }
func (f *fakeReader) DiskIOCounters() (uint64, uint64, uint64, uint64) {
	return f.diskRead, f.diskWrite, f.diskROps, f.diskWOps
}
func (f *fakeReader) CPUFreq() CPUFreq           { return f.freq }
func (f *fakeReader) ThrottleStatus() uint32     { return f.throttle }
func (f *fakeReader) DietPiStatus() DietPiStatus { return f.dietpi }
func (f *fakeReader) APTUpdateCount() int        { return f.apt }

func TestNetBandwidthDelta(t *testing.T) {
	r := &fakeReader{
		iface: "eth0",
		ipv4:  "10.0.0.1",
		ipv6:  NoIPv6,
		netRx: 1000,
		netTx: 2000,
	}

	c := NewCollectorWithReader(r, slog.Default())

	// First refresh happened in constructor — no rates yet (no previous sample).
	bw := c.NetBandwidth()
	if bw.RxBytesPerSec != 0 || bw.TxBytesPerSec != 0 {
		t.Errorf("first refresh: got bandwidth %+v, want zeros", bw)
	}

	// Simulate exactly 1 second elapsed with 500 bytes rx, 1000 bytes tx.
	r.netRx = 1500
	r.netTx = 3000
	lc := c.(*liveCollector)
	lc.lastRefresh = time.Now().Add(-1 * time.Second)
	c.Refresh()

	bw = c.NetBandwidth()
	if bw.RxBytesPerSec < 499 || bw.RxBytesPerSec > 500 {
		t.Errorf("RxBytesPerSec = %d, want ~500", bw.RxBytesPerSec)
	}
	if bw.TxBytesPerSec < 999 || bw.TxBytesPerSec > 1000 {
		t.Errorf("TxBytesPerSec = %d, want ~1000", bw.TxBytesPerSec)
	}
}

func TestDiskIODelta(t *testing.T) {
	r := &fakeReader{
		diskRead:  10000,
		diskWrite: 20000,
		diskROps:  100,
		diskWOps:  200,
	}

	c := NewCollectorWithReader(r, slog.Default())

	// First refresh — no rates.
	dio := c.DiskIO()
	if dio.ReadBytesPerSec != 0 || dio.WriteBytesPerSec != 0 {
		t.Errorf("first refresh: got disk IO %+v, want zeros", dio)
	}

	// Simulate exactly 1 second elapsed with known deltas.
	r.diskRead = 15000
	r.diskWrite = 25000
	r.diskROps = 150
	r.diskWOps = 250
	lc := c.(*liveCollector)
	lc.lastRefresh = time.Now().Add(-1 * time.Second)
	c.Refresh()

	dio = c.DiskIO()
	if dio.ReadBytesPerSec < 4999 || dio.ReadBytesPerSec > 5000 {
		t.Errorf("ReadBytesPerSec = %d, want ~5000", dio.ReadBytesPerSec)
	}
	if dio.WriteBytesPerSec < 4999 || dio.WriteBytesPerSec > 5000 {
		t.Errorf("WriteBytesPerSec = %d, want ~5000", dio.WriteBytesPerSec)
	}
	if dio.ReadIOPS < 49 || dio.ReadIOPS > 50 {
		t.Errorf("ReadIOPS = %d, want ~50", dio.ReadIOPS)
	}
	if dio.WriteIOPS < 49 || dio.WriteIOPS > 50 {
		t.Errorf("WriteIOPS = %d, want ~50", dio.WriteIOPS)
	}
}

func TestNoNetworkInterface(t *testing.T) {
	r := &fakeReader{iface: ""}

	c := NewCollectorWithReader(r, slog.Default())

	if c.IPv4Address() != "no network" {
		t.Errorf("IPv4Address() = %q, want %q", c.IPv4Address(), "no network")
	}
	if c.IPv6Suffix() != NoIPv6 {
		t.Errorf("IPv6Suffix() = %q, want %q", c.IPv6Suffix(), NoIPv6)
	}
}

func TestSimpleReaderPassthrough(t *testing.T) {
	r := &fakeReader{
		cpu:       42.5,
		ram:       67.3,
		temp:      55.0,
		diskUsage: 80.1,
		host:      "testhost",
		up:        3 * time.Hour,
		freq:      CPUFreq{Cur: 1800, Min: 600, Max: 2400},
		throttle:  0x50005,
		dietpi:    DietPiUpToDate,
		apt:       3,
	}

	c := NewCollectorWithReader(r, slog.Default())

	if c.CPUPercent() != 42.5 {
		t.Errorf("CPUPercent() = %f, want 42.5", c.CPUPercent())
	}
	if c.RAMPercent() != 67.3 {
		t.Errorf("RAMPercent() = %f, want 67.3", c.RAMPercent())
	}
	if c.Temperature() != 55.0 {
		t.Errorf("Temperature() = %f, want 55.0", c.Temperature())
	}
	if c.DiskPercent() != 80.1 {
		t.Errorf("DiskPercent() = %f, want 80.1", c.DiskPercent())
	}
	if c.Hostname() != "testhost" {
		t.Errorf("Hostname() = %q, want %q", c.Hostname(), "testhost")
	}
	if c.Uptime() != 3*time.Hour {
		t.Errorf("Uptime() = %v, want 3h", c.Uptime())
	}
	if c.CPUFreq() != (CPUFreq{Cur: 1800, Min: 600, Max: 2400}) {
		t.Errorf("CPUFreq() = %+v, unexpected", c.CPUFreq())
	}
	if c.ThrottleStatus() != 0x50005 {
		t.Errorf("ThrottleStatus() = %d, want %d", c.ThrottleStatus(), 0x50005)
	}
	if c.DietPiStatus() != DietPiUpToDate {
		t.Errorf("DietPiStatus() = %d, want DietPiUpToDate", c.DietPiStatus())
	}
	if c.APTUpdateCount() != 3 {
		t.Errorf("APTUpdateCount() = %d, want 3", c.APTUpdateCount())
	}
}
