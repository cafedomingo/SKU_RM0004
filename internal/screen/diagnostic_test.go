package screen

import (
	"strings"
	"testing"
	"time"

	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
	"github.com/cafedomingo/SKU_RM0004/internal/theme"
)

func diagMock() *sysinfo.MockCollector {
	return &sysinfo.MockCollector{
		Host:  "testhost",
		IP:    "192.168.1.1",
		IPv6:  "::1",
		CPU:   47,
		RAM:   63,
		Disk:  42,
		Temp:  52,
		Freq:  sysinfo.CPUFreq{Cur: 1800},
		Up:    2*time.Hour + 30*time.Minute,
		Net:   sysinfo.NetBandwidth{RxBytesPerSec: 1024, TxBytesPerSec: 512},
		DIO:   sysinfo.DiskIO{ReadBytesPerSec: 1024 * 1024, WriteBytesPerSec: 512 * 1024, ReadIOPS: 150, WriteIOPS: 80},
		APT:   3,
		DietPi: sysinfo.DietPiUpToDate,
	}
}

func diagCfg() config.Config {
	return config.Config{TempUnit: "C"}
}

// pixelsInRow returns the set of unique colors found in the row at y offset [y, y+8) (5x8 font).
func pixelsInRow(fb *st7735.Framebuffer, y int) map[uint16]bool {
	colors := make(map[uint16]bool)
	for row := y; row < y+8 && row < st7735.Height; row++ {
		for col := 0; col < st7735.Width; col++ {
			c := fb.Pixels[row*st7735.Width+col]
			if c != theme.ColorBG {
				colors[c] = true
			}
		}
	}
	return colors
}

// TestDiagnosticPageCount verifies 15 rows produce exactly 2 pages.
func TestDiagnosticPageCount(t *testing.T) {
	rows := collectDiagData(diagMock())
	if len(rows) != 15 {
		t.Fatalf("expected 15 rows, got %d", len(rows))
	}
	numPages := (len(rows) + diagRowsPerPage - 1) / diagRowsPerPage
	if numPages != 2 {
		t.Errorf("expected 2 pages, got %d", numPages)
	}
}

// TestDiagnosticPage0Content renders page 0 and verifies the hostname row
// appears at y=0 with white pixels.
func TestDiagnosticPage0Content(t *testing.T) {
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	state := &DiagState{}
	RenderDiagnostic(&fb, diagMock(), diagCfg(), state)

	// Row 0 is hostname header — should have white (ColorFG) pixels at y=0
	if !hasColorInRegion(&fb, 0, 0, st7735.Width, 8, theme.ColorFG) {
		t.Error("expected white hostname pixels at y=0 on page 0")
	}
}

// TestDiagnosticPage1Content renders pages 0 then 1 and verifies the
// temperature row (page 1, row 0 = y=0) shows both C and F.
func TestDiagnosticPage1Content(t *testing.T) {
	m := diagMock()
	rows := collectDiagData(m)

	// Row 5 is the temp row; with diagRowsPerPage=10 it's on page 0
	// Row 10 is the first row on page 1 (tx row)
	tempRow := rows[5]
	if !strings.Contains(tempRow.value, "C") {
		t.Errorf("temp row value %q missing C", tempRow.value)
	}
	if !strings.Contains(tempRow.value, "F") {
		t.Errorf("temp row value %q missing F", tempRow.value)
	}

	// Verify page 1 renders non-empty content
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	state := &DiagState{}
	// Render page 0 (advances to page 1)
	RenderDiagnostic(&fb, m, diagCfg(), state)
	// Render page 1 (advances to page 0)
	fb.Fill(theme.ColorBG)
	RenderDiagnostic(&fb, m, diagCfg(), state)

	// Page 1 starts at row 10 (tx row); should have non-BG pixels
	if !hasNonBGInRegion(&fb, 0, 0, st7735.Width, 8) {
		t.Error("expected non-background pixels at y=0 on page 1")
	}
}

// TestDiagnosticPageWraps verifies that after 2 renders the page wraps back to 0.
func TestDiagnosticPageWraps(t *testing.T) {
	m := diagMock()
	state := &DiagState{}

	for i := 0; i < 2; i++ {
		var fb st7735.Framebuffer
		fb.Fill(theme.ColorBG)
		RenderDiagnostic(&fb, m, diagCfg(), state)
	}

	if state.Page != 0 {
		t.Errorf("expected page 0 after 2 renders, got %d", state.Page)
	}
}

// TestDiagnosticTempBothUnits verifies that the temp row always includes
// both C and F regardless of cfg.TempUnit.
func TestDiagnosticTempBothUnits(t *testing.T) {
	m := diagMock()
	m.Temp = 52

	for _, unit := range []string{"C", "F"} {
		cfg := config.Config{TempUnit: unit}
		rows := collectDiagData(m)
		tempRow := rows[5]
		_ = cfg // cfg is not used by collectDiagData, intentionally

		if !strings.Contains(tempRow.value, "C") {
			t.Errorf("unit=%s: temp row %q missing C", unit, tempRow.value)
		}
		if !strings.Contains(tempRow.value, "F") {
			t.Errorf("unit=%s: temp row %q missing F", unit, tempRow.value)
		}
	}
}

// TestDiagnosticThrottleStates verifies the three throttle states produce correct colors.
func TestDiagnosticThrottleStates(t *testing.T) {
	tests := []struct {
		name      string
		throttle  uint32
		wantValue string
		wantColor uint16
	}{
		{"active", 0x00000001, "ACTIVE", theme.ColorCrit},
		{"past", 0x00010000, "past", theme.ColorWarn},
		{"ok", 0x00000000, "OK", theme.ColorOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := diagMock()
			m.Throttle = tt.throttle
			rows := collectDiagData(m)

			// Row 7 is the throttle row
			row := rows[7]
			if row.value != tt.wantValue {
				t.Errorf("throttle=%#x: got value %q, want %q", tt.throttle, row.value, tt.wantValue)
			}
			if row.color != tt.wantColor {
				t.Errorf("throttle=%#x: got color 0x%04X, want 0x%04X", tt.throttle, row.color, tt.wantColor)
			}
		})
	}
}
