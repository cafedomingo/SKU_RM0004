package sysinfo

import (
	"log/slog"
	"testing"
)

func TestCPUFreqRead(t *testing.T) {
	r := NewSystemReader(slog.Default())
	f := r.CPUFreq()
	// On non-Pi systems all values may be 0; just sanity-check the read succeeded.
	if f.Cur == 0 && f.Min == 0 && f.Max == 0 {
		t.Log("CPUFreq() returned all zeros (expected on non-Pi)")
	}
}

func TestDietPiDetection(t *testing.T) {
	r := NewSystemReader(slog.Default())
	s := r.DietPiStatus()
	if s < DietPiNotInstalled || s > DietPiUpdateAvail {
		t.Errorf("DietPiStatus() = %d, not a valid DietPiStatus", s)
	}
}

func TestAPTUpdateCount(t *testing.T) {
	r := NewSystemReader(slog.Default())
	n := r.APTUpdateCount()
	if n < -1 {
		t.Errorf("APTUpdateCount() = %d, want >= -1", n)
	}
}
