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
		IPv4: "192.168.1.42",
		CPU:  theme.CPUWarn,
		RAM:  theme.RAMCrit,
		Disk: theme.DiskWarn,
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

func TestDashboardRenders(t *testing.T) {
	m := defaultMock()
	d := &dashboardScreen{collector: m}
	// filled by Update
	d.Update(defaultCfg())

	// Hostname text should appear at top (y=0..15), white pixels (8x16 font)
	if !hasColorInRegion(d.Buffer(), 0, 0, 80, 16, theme.ColorFG) {
		t.Error("expected white hostname pixels in top row")
	}

	// IP address should appear at y=18..29 (6x12 font), light blue pixels
	if !hasColorInRegion(d.Buffer(), 0, 18, 120, 12, theme.ColorIdentity) {
		t.Error("expected light-blue IP pixels in second row")
	}

	// Separator line at y=30
	if !hasColorInRegion(d.Buffer(), 0, 30, st7735.Width, 1, theme.ColorSep) {
		t.Error("expected separator line at y=30")
	}

	// CPU bar at CPUWarn should be exactly ColorWarn
	if !hasColorInRegion(d.Buffer(), 2, 46, 65, 6, theme.ColorWarn) {
		t.Error("expected CPU bar in warn color at CPUWarn threshold")
	}

	// RAM bar at RAMCrit should be exactly ColorCrit
	if !hasColorInRegion(d.Buffer(), 2, 68, 65, 6, theme.ColorCrit) {
		t.Error("expected RAM bar in crit color at RAMCrit threshold")
	}
}

func TestDashboardThresholds(t *testing.T) {
	cfg := defaultCfg()

	// Test at exact boundaries where colors are deterministic
	t.Run("at_crit", func(t *testing.T) {
		m := &sysinfo.MockCollector{
			Host: "pi", IPv4: "10.0.0.1",
			CPU: theme.CPUCrit, RAM: theme.RAMCrit, Disk: theme.DiskCrit, Temp: 45,
		}
		d := &dashboardScreen{collector: m}
		// filled by Update
		d.Update(cfg)
		if !hasColorInRegion(d.Buffer(), 2, 46, 74, 6, theme.ColorCrit) {
			t.Error("CPU bar at crit should be ColorCrit")
		}
	})

	// At warn thresholds, colors should be exactly ColorWarn
	t.Run("at_warn", func(t *testing.T) {
		m := &sysinfo.MockCollector{
			Host: "pi", IPv4: "10.0.0.1",
			CPU: theme.CPUWarn, RAM: theme.RAMWarn, Disk: theme.DiskWarn, Temp: 45,
		}
		d := &dashboardScreen{collector: m}
		d.Update(cfg)
		if !hasColorInRegion(d.Buffer(), 2, 46, 74, 6, theme.ColorWarn) {
			t.Error("CPU bar at CPUWarn should be ColorWarn")
		}
		if !hasColorInRegion(d.Buffer(), 2, 68, 74, 6, theme.ColorWarn) {
			t.Error("RAM bar at RAMWarn should be ColorWarn")
		}
		if !hasColorInRegion(d.Buffer(), 84, 68, 74, 6, theme.ColorWarn) {
			t.Error("Disk bar at DiskWarn should be ColorWarn")
		}
	})
}

func TestDashboardDisplayFloor(t *testing.T) {
	m := &sysinfo.MockCollector{
		Host: "pi",
		IPv4: "10.0.0.1",
		CPU:  0,
		RAM:  0,
		Disk: 0,
		Temp: 30,
	}
	d := &dashboardScreen{collector: m}
	// filled by Update
	d.Update(defaultCfg())

	// Even with 0% values, bars should show 1% (clamped) in the expected color
	floorColor := theme.CPUColor(1)
	if !hasColorInRegion(d.Buffer(), 2, 46, 65, 6, floorColor) {
		t.Error("CPU bar should render at 1% when value is 0")
	}
	if !hasColorInRegion(d.Buffer(), 2, 68, 65, 6, floorColor) {
		t.Error("RAM bar should render at 1% when value is 0")
	}
	diskFloorColor := theme.DiskColor(1)
	if !hasColorInRegion(d.Buffer(), 82, 68, 65, 6, diskFloorColor) {
		t.Error("Disk bar should render at 1% when value is 0")
	}
}

func TestDashboardDietPiDiamond(t *testing.T) {
	m := defaultMock()
	m.DietPi = sysinfo.DietPiUpdateAvail
	d := &dashboardScreen{collector: m}
	// filled by Update
	d.Update(defaultCfg())

	// Diamond character should render near top-right corner in alert color (8x16 font)
	if !hasColorInRegion(d.Buffer(), 148, 0, 12, 16, theme.ColorAlert) {
		t.Error("expected DietPi diamond (alert color) near top-right")
	}
}

func TestDashboardDietPiDiamondAbsent(t *testing.T) {
	m := defaultMock()
	m.DietPi = sysinfo.DietPiUpToDate
	d := &dashboardScreen{collector: m}
	// filled by Update
	d.Update(defaultCfg())

	// No alert-colored pixels in the diamond area
	if hasColorInRegion(d.Buffer(), 148, 0, 12, 16, theme.ColorAlert) {
		t.Error("DietPi diamond should not appear when up to date")
	}
}

func TestDashboardAPTBadge(t *testing.T) {
	m := defaultMock()
	m.APT = 1 // exactly at APTWarn threshold
	d := &dashboardScreen{collector: m}
	d.Update(defaultCfg())

	if !hasColorInRegion(d.Buffer(), 120, 18, 40, 12, theme.ColorWarn) {
		t.Error("expected APT badge in warn color at threshold")
	}
}

func TestDashboardAPTBadgeCrit(t *testing.T) {
	m := defaultMock()
	m.APT = 10 // exactly at APTCrit threshold
	d := &dashboardScreen{collector: m}
	d.Update(defaultCfg())

	if !hasColorInRegion(d.Buffer(), 120, 18, 40, 12, theme.ColorCrit) {
		t.Error("expected APT badge in crit color at threshold")
	}
}

func TestDashboardAPTBadgeAbsent(t *testing.T) {
	m := defaultMock()
	m.APT = 0
	d := &dashboardScreen{collector: m}
	// filled by Update
	d.Update(defaultCfg())

	// No warn/crit pixels in the badge area (right side of IP row)
	if hasColorInRegion(d.Buffer(), 140, 18, 20, 12, theme.ColorWarn) {
		t.Error("APT badge should not appear when count is 0")
	}
}
