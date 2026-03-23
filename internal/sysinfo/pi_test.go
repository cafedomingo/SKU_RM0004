package sysinfo

import "testing"

func TestCPUFreqRead(t *testing.T) {
	f := readCPUFreq()
	// On non-Pi systems all values may be 0; just check they're non-negative.
	if f.Cur < 0 || f.Min < 0 || f.Max < 0 {
		t.Errorf("readCPUFreq() returned negative values: %+v", f)
	}
}

func TestDietPiDetection(t *testing.T) {
	s := readDietPiStatus()
	if s < DietPiNotInstalled || s > DietPiUpdateAvail {
		t.Errorf("readDietPiStatus() = %d, not a valid DietPiStatus", s)
	}
}

func TestAPTUpdateCount(t *testing.T) {
	n := readAPTUpdateCount()
	if n < -1 {
		t.Errorf("readAPTUpdateCount() = %d, want >= -1", n)
	}
}
