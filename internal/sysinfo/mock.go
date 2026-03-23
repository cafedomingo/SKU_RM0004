package sysinfo

import "time"

// MockCollector is a test double that returns preconfigured values.
type MockCollector struct {
	CPU       float64
	RAM       float64
	Disk      float64
	Temp      float64
	Host      string
	IPv4      string
	IPv6      string
	Freq      CPUFreq
	Net       NetBandwidth
	DIO       DiskIO
	Up        time.Duration
	Throttle  uint32
	DietPi    DietPiStatus
	APT       int
	LinkSpeed int
}

func (m *MockCollector) CPUPercent() float64       { return m.CPU }
func (m *MockCollector) RAMPercent() float64        { return m.RAM }
func (m *MockCollector) DiskPercent() float64       { return m.Disk }
func (m *MockCollector) Temperature() float64       { return m.Temp }
func (m *MockCollector) Hostname() string           { return m.Host }
func (m *MockCollector) IPv4Address() string        { return m.IPv4 }
func (m *MockCollector) IPv6Suffix() string         { return m.IPv6 }
func (m *MockCollector) CPUFreq() CPUFreq           { return m.Freq }
func (m *MockCollector) NetBandwidth() NetBandwidth { return m.Net }
func (m *MockCollector) DiskIO() DiskIO             { return m.DIO }
func (m *MockCollector) Uptime() time.Duration      { return m.Up }
func (m *MockCollector) ThrottleStatus() uint32     { return m.Throttle }
func (m *MockCollector) DietPiStatus() DietPiStatus { return m.DietPi }
func (m *MockCollector) APTUpdateCount() int        { return m.APT }
func (m *MockCollector) LinkSpeedMbps() int         { return m.LinkSpeed }
func (m *MockCollector) Refresh()                   {}
