package sysinfo

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"unsafe"
)

func TestCPUFreqRead(t *testing.T) {
	r := NewSystemReader(slog.Default())
	f := r.CPUFreq()
	// Min <= Max when both are available (Cur can exceed Max with turbo boost).
	if f.Min > 0 && f.Max > 0 && f.Min > f.Max {
		t.Errorf("CPUFreq().Min (%d) > Max (%d)", f.Min, f.Max)
	}
}

func TestIoctlMailbox(t *testing.T) {
	// IOCTL_MBOX_PROPERTY's size field follows the pointer size of the build.
	want := uintptr(0xC0046400)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		want = 0xC0086400
	}
	if ioctlMailbox != want {
		t.Errorf("ioctlMailbox = %#x, want %#x", ioctlMailbox, want)
	}
}

type countHandler struct{ n int }

func (h *countHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *countHandler) Handle(context.Context, slog.Record) error {
	h.n++
	return nil
}
func (h *countHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *countHandler) WithGroup(string) slog.Handler      { return h }

func TestThrottleStatusLogsEachFailure(t *testing.T) {
	if _, err := os.Stat(vcioPath); err == nil {
		t.Skip(vcioPath + " exists")
	}
	h := &countHandler{}
	r := NewSystemReader(slog.New(h))
	for range 3 {
		if v, ok := r.ThrottleStatus(); v != 0 || ok {
			t.Fatalf("ThrottleStatus() = %#x, %v, want 0, false", v, ok)
		}
	}
	if h.n != 3 {
		t.Errorf("logged %d records, want 3", h.n)
	}
}

func TestParseThrottled(t *testing.T) {
	tests := []struct {
		name    string
		mbox    uint32
		tag     uint32
		want    uint32
		wantErr bool
	}{
		{"ok", mailboxSuccess, mailboxSuccess | 4, 0x50005, false},
		{"mailbox error", 0x80000001, mailboxSuccess | 4, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := [8]uint32{32, tt.mbox, tagGetThrottled, 4, tt.tag, 0x50005, 0, 0}
			got, err := parseThrottled(buf)
			if got != tt.want || (err != nil) != tt.wantErr {
				t.Errorf("parseThrottled() = %#x, %v, want %#x, err=%v", got, err, tt.want, tt.wantErr)
			}
		})
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
