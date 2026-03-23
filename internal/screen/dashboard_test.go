package screen

import (
	"testing"

	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
	"github.com/cafedomingo/SKU_RM0004/internal/theme"
)

func defaultMock() *sysinfo.MockCollector {
	return &sysinfo.MockCollector{
		Host: "dietpi",
		IP:   "192.168.1.42",
		CPU:  47,
		RAM:  63,
		Disk: 42,
		Temp: 52,
	}
}

func defaultCfg() config.Config {
	return config.Config{TempUnit: "C"}
}

// hasColorInRegion returns true if any pixel in the rectangle (x,y,w,h)
// matches the given color.
func hasColorInRegion(fb *st7735.Framebuffer, x, y, w, h int, color uint16) bool {
	for row := y; row < y+h && row < st7735.Height; row++ {
		for col := x; col < x+w && col < st7735.Width; col++ {
			if fb.Pixels[row*st7735.Width+col] == color {
				return true
			}
		}
	}
	return false
}

// hasNonBGInRegion returns true if any pixel in the rectangle is not the
// background color.
func hasNonBGInRegion(fb *st7735.Framebuffer, x, y, w, h int) bool {
	for row := y; row < y+h && row < st7735.Height; row++ {
		for col := x; col < x+w && col < st7735.Width; col++ {
			if fb.Pixels[row*st7735.Width+col] != theme.ColorBG {
				return true
			}
		}
	}
	return false
}

func TestDashboardRenders(t *testing.T) {
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	m := defaultMock()
	RenderDashboard(&fb, m, defaultCfg())

	// Hostname text should appear at top (y=0..15), white pixels (8x16 font)
	if !hasColorInRegion(&fb, 0, 0, 80, 16, theme.ColorFG) {
		t.Error("expected white hostname pixels in top row")
	}

	// IP address should appear at y=18..25 (5x8 font), light blue pixels
	if !hasColorInRegion(&fb, 0, 18, 120, 8, theme.ColorIP) {
		t.Error("expected light-blue IP pixels in second row")
	}

	// Separator line at y=28
	if !hasColorInRegion(&fb, 0, 28, st7735.Width, 1, theme.ColorSep) {
		t.Error("expected separator line at y=28")
	}

	// CPU bar area should have colored pixels (y=40, 6px tall)
	if !hasNonBGInRegion(&fb, 2, 40, 65, 6) {
		t.Error("expected CPU bar pixels")
	}

	// RAM bar area should have colored pixels (y=58, 6px tall)
	if !hasNonBGInRegion(&fb, 2, 58, 65, 6) {
		t.Error("expected RAM bar pixels")
	}
}

func TestDashboardThresholds(t *testing.T) {
	cfg := defaultCfg()

	tests := []struct {
		name     string
		cpu      float64
		wantCPU  uint16
		ram      float64
		wantRAM  uint16
		disk     float64
		wantDisk uint16
	}{
		{"normal", 30, theme.ColorOK, 30, theme.ColorOK, 30, theme.ColorOK},
		{"warn", 65, theme.ColorWarn, 65, theme.ColorWarn, 75, theme.ColorWarn},
		{"crit", 90, theme.ColorCrit, 90, theme.ColorCrit, 95, theme.ColorCrit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fb st7735.Framebuffer
			fb.Fill(theme.ColorBG)
			m := &sysinfo.MockCollector{
				Host: "pi",
				IP:   "10.0.0.1",
				CPU:  tt.cpu,
				RAM:  tt.ram,
				Disk: tt.disk,
				Temp: 45,
			}
			RenderDashboard(&fb, m, cfg)

			// CPU bar region (y=40, 6px tall)
			if !hasColorInRegion(&fb, 2, 40, 65, 6, tt.wantCPU) {
				t.Errorf("CPU bar: expected color 0x%04X for cpu=%.0f%%", tt.wantCPU, tt.cpu)
			}
			// RAM bar region (y=58, 6px tall)
			if !hasColorInRegion(&fb, 2, 58, 65, 6, tt.wantRAM) {
				t.Errorf("RAM bar: expected color 0x%04X for ram=%.0f%%", tt.wantRAM, tt.ram)
			}
			// Disk bar region (y=58, 6px tall)
			if !hasColorInRegion(&fb, 82, 58, 65, 6, tt.wantDisk) {
				t.Errorf("Disk bar: expected color 0x%04X for disk=%.0f%%", tt.wantDisk, tt.disk)
			}
		})
	}
}

func TestDashboardDisplayFloor(t *testing.T) {
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	m := &sysinfo.MockCollector{
		Host: "pi",
		IP:   "10.0.0.1",
		CPU:  0,
		RAM:  0,
		Disk: 0,
		Temp: 30,
	}
	RenderDashboard(&fb, m, defaultCfg())

	// Even with 0% values, bars should show 1% (some colored pixels)
	if !hasNonBGInRegion(&fb, 2, 40, 65, 6) {
		t.Error("CPU bar should show at least 1% when value is 0")
	}
	if !hasNonBGInRegion(&fb, 2, 58, 65, 6) {
		t.Error("RAM bar should show at least 1% when value is 0")
	}
	if !hasNonBGInRegion(&fb, 82, 58, 65, 6) {
		t.Error("Disk bar should show at least 1% when value is 0")
	}
}

func TestDashboardDietPiDiamond(t *testing.T) {
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	m := defaultMock()
	m.DietPi = sysinfo.DietPiUpdateAvail
	RenderDashboard(&fb, m, defaultCfg())

	// Diamond character should render near top-right corner in alert color (8x16 font)
	if !hasColorInRegion(&fb, 148, 0, 12, 16, theme.ColorAlert) {
		t.Error("expected DietPi diamond (alert color) near top-right")
	}
}

func TestDashboardDietPiDiamondAbsent(t *testing.T) {
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	m := defaultMock()
	m.DietPi = sysinfo.DietPiUpToDate
	RenderDashboard(&fb, m, defaultCfg())

	// No alert-colored pixels in the diamond area
	if hasColorInRegion(&fb, 148, 0, 12, 16, theme.ColorAlert) {
		t.Error("DietPi diamond should not appear when up to date")
	}
}

func TestDashboardAPTBadge(t *testing.T) {
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	m := defaultMock()
	m.APT = 3
	RenderDashboard(&fb, m, defaultCfg())

	// APT badge "^3" should render on the IP row (y=18..25, 5x8 font) in warn color
	if !hasColorInRegion(&fb, 120, 18, 40, 8, theme.ColorWarn) {
		t.Error("expected APT badge (warn color) on IP row, right side")
	}
}

func TestDashboardAPTBadgeCrit(t *testing.T) {
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	m := defaultMock()
	m.APT = 15 // >= APTCrit (10)
	RenderDashboard(&fb, m, defaultCfg())

	// APT badge should render in crit color
	if !hasColorInRegion(&fb, 120, 18, 40, 8, theme.ColorCrit) {
		t.Error("expected APT badge (crit color) when count >= 10")
	}
}

func TestDashboardAPTBadgeAbsent(t *testing.T) {
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	m := defaultMock()
	m.APT = 0
	RenderDashboard(&fb, m, defaultCfg())

	// No warn/crit pixels in the badge area (right side of IP row)
	if hasColorInRegion(&fb, 140, 18, 20, 8, theme.ColorWarn) {
		t.Error("APT badge should not appear when count is 0")
	}
}
