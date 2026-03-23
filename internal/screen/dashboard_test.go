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
	(&dashboardScreen{}).Render(&fb, m, defaultCfg())

	// Hostname text should appear at top (y=0..15), white pixels (8x16 font)
	if !hasColorInRegion(&fb, 0, 0, 80, 16, theme.ColorFG) {
		t.Error("expected white hostname pixels in top row")
	}

	// IP address should appear at y=18..29 (6x12 font), light blue pixels
	if !hasColorInRegion(&fb, 0, 18, 120, 12, theme.ColorIP) {
		t.Error("expected light-blue IP pixels in second row")
	}

	// Separator line at y=30
	if !hasColorInRegion(&fb, 0, 30, st7735.Width, 1, theme.ColorSep) {
		t.Error("expected separator line at y=30")
	}

	// CPU bar area should have colored pixels (y=46, 6px tall)
	if !hasNonBGInRegion(&fb, 2, 46, 65, 6) {
		t.Error("expected CPU bar pixels")
	}

	// RAM bar area should have colored pixels (y=68, 6px tall)
	if !hasNonBGInRegion(&fb, 2, 68, 65, 6) {
		t.Error("expected RAM bar pixels")
	}
}

func TestDashboardThresholds(t *testing.T) {
	cfg := defaultCfg()

	// Test at exact boundaries where colors are deterministic
	t.Run("at_crit", func(t *testing.T) {
		var fb st7735.Framebuffer
		fb.Fill(theme.ColorBG)
		m := &sysinfo.MockCollector{
			Host: "pi", IP: "10.0.0.1",
			CPU: theme.CPUCrit, RAM: theme.RAMCrit, Disk: theme.DiskCrit, Temp: 45,
		}
		(&dashboardScreen{}).Render(&fb, m, cfg)
		if !hasColorInRegion(&fb, 2, 46, 74, 6, theme.ColorCrit) {
			t.Error("CPU bar at crit should be ColorCrit")
		}
	})

	// Intermediate values produce visible (non-background) bars with lerped colors
	t.Run("intermediate", func(t *testing.T) {
		var fb st7735.Framebuffer
		fb.Fill(theme.ColorBG)
		m := &sysinfo.MockCollector{
			Host: "pi", IP: "10.0.0.1",
			CPU: 30, RAM: 65, Disk: 50, Temp: 45,
		}
		(&dashboardScreen{}).Render(&fb, m, cfg)
		if !hasNonBGInRegion(&fb, 2, 46, 74, 6) {
			t.Error("CPU bar at 30% should have colored pixels")
		}
		if !hasNonBGInRegion(&fb, 2, 68, 74, 6) {
			t.Error("RAM bar at 65% should have colored pixels")
		}
		if !hasNonBGInRegion(&fb, 84, 68, 74, 6) {
			t.Error("Disk bar at 50% should have colored pixels")
		}
	})
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
	(&dashboardScreen{}).Render(&fb, m, defaultCfg())

	// Even with 0% values, bars should show 1% (some colored pixels)
	if !hasNonBGInRegion(&fb, 2, 46, 65, 6) {
		t.Error("CPU bar should show at least 1% when value is 0")
	}
	if !hasNonBGInRegion(&fb, 2, 68, 65, 6) {
		t.Error("RAM bar should show at least 1% when value is 0")
	}
	if !hasNonBGInRegion(&fb, 82, 68, 65, 6) {
		t.Error("Disk bar should show at least 1% when value is 0")
	}
}

func TestDashboardDietPiDiamond(t *testing.T) {
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	m := defaultMock()
	m.DietPi = sysinfo.DietPiUpdateAvail
	(&dashboardScreen{}).Render(&fb, m, defaultCfg())

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
	(&dashboardScreen{}).Render(&fb, m, defaultCfg())

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
	(&dashboardScreen{}).Render(&fb, m, defaultCfg())

	// APT badge "^3" should render on the IP row (y=18..29, 6x12 font) in warn color
	if !hasColorInRegion(&fb, 120, 18, 40, 12, theme.ColorWarn) {
		t.Error("expected APT badge (warn color) on IP row, right side")
	}
}

func TestDashboardAPTBadgeCrit(t *testing.T) {
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	m := defaultMock()
	m.APT = 15 // >= APTCrit (10)
	(&dashboardScreen{}).Render(&fb, m, defaultCfg())

	// APT badge should render in crit color
	if !hasColorInRegion(&fb, 120, 18, 40, 12, theme.ColorCrit) {
		t.Error("expected APT badge (crit color) when count >= 10")
	}
}

func TestDashboardAPTBadgeAbsent(t *testing.T) {
	var fb st7735.Framebuffer
	fb.Fill(theme.ColorBG)
	m := defaultMock()
	m.APT = 0
	(&dashboardScreen{}).Render(&fb, m, defaultCfg())

	// No warn/crit pixels in the badge area (right side of IP row)
	if hasColorInRegion(&fb, 140, 18, 20, 12, theme.ColorWarn) {
		t.Error("APT badge should not appear when count is 0")
	}
}
