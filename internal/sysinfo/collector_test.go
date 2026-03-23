package sysinfo

import (
	"log/slog"
	"testing"
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
