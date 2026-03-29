package sysinfo

import (
	"log/slog"
	"testing"
)

func TestCPUFreqRead(t *testing.T) {
	r := NewSystemReader(slog.Default())
	f := r.CPUFreq()
	// Min <= Max when both are available (Cur can exceed Max with turbo boost).
	if f.Min > 0 && f.Max > 0 && f.Min > f.Max {
		t.Errorf("CPUFreq().Min (%d) > Max (%d)", f.Min, f.Max)
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

func TestIsWholeDisk(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		// SCSI/SATA disks
		{"sda", true},
		{"sdb", true},
		{"sda1", false},
		{"sdb2", false},
		{"sdaa", false},

		// MMC (SD cards)
		{"mmcblk0", true},
		{"mmcblk1", true},
		{"mmcblk0p1", false},
		{"mmcblk0p2", false},

		// NVMe
		{"nvme0n1", true},
		{"nvme1n1", true},
		{"nvme0n1p1", false},
		{"nvme0n1p2", false},

		// Not disks
		{"loop0", false},
		{"dm-0", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWholeDisk(tt.name); got != tt.want {
				t.Errorf("isWholeDisk(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
