package screen

import (
	"fmt"

	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/font"
	"github.com/cafedomingo/SKU_RM0004/internal/format"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
	"github.com/cafedomingo/SKU_RM0004/internal/theme"
)

// RenderDashboard draws the main status screen onto the framebuffer.
//
// Layout (160x80, two-font approach):
//
//	y=0:   Hostname (8x16, white)          ◆ (8x16, top-right if DietPi update)
//	y=18:  IP address (5x8, light blue)    APT badge right-aligned (5x8)
//	y=28:  ─── separator line (1px) ───
//	y=30:  CPU:NNN% (5x8, left)            TMP:NNC (5x8, right)
//	y=40:  [CPU bar]                       [Temp bar] (6px tall)
//	y=48:  RAM:NNN% (5x8, left)            DSK:NNN% (5x8, right)
//	y=58:  [RAM bar]                       [Disk bar] (6px tall)
func RenderDashboard(fb *st7735.Framebuffer, c sysinfo.Collector, cfg config.Config) {
	big := font.Spleen8x16
	sm := font.Spleen5x8

	// Header: hostname (big font)
	fb.String(2, 0, c.Hostname(), big, theme.ColorFG)

	// IP address (small font)
	fb.String(2, 18, c.IPAddress(), sm, theme.ColorIP)

	// DietPi update diamond indicator (big font for the symbol)
	if c.DietPiStatus() == sysinfo.DietPiUpdateAvail {
		fb.Char(152, 0, '\u25C6', big, theme.ColorAlert)
	}

	// APT update badge (small font, right-aligned on IP row)
	badge := format.APTBadge(c.APTUpdateCount())
	if badge != "" {
		color := theme.ColorWarn
		if c.APTUpdateCount() >= theme.APTCrit {
			color = theme.ColorCrit
		}
		bx := st7735.Width - len(badge)*sm.Width - 2
		fb.String(bx, 18, badge, sm, color)
	}

	// Separator
	fb.Rect(0, 28, st7735.Width, 1, theme.ColorSep)

	// Clamp display values to minimum 1% so bars are always visible
	cpu := clampMin(c.CPUPercent(), 1)
	ram := clampMin(c.RAMPercent(), 1)
	disk := clampMin(c.DiskPercent(), 1)
	temp := c.Temperature()

	const (
		barW   = 65
		barH   = 6
		leftX  = 2
		rightX = 82
	)

	// CPU (left column)
	cpuColor := theme.ThresholdColor(c.CPUPercent(), theme.CPUWarn, theme.CPUCrit)
	fb.String(leftX, 30, fmt.Sprintf("CPU:%3d%%", int(cpu)), sm, cpuColor)
	fb.Bar(leftX, 40, barW, barH, int(cpu), cpuColor, theme.ColorSep)

	// Temperature (right column)
	tempColor := theme.TempRampColor(temp)
	tempStr := format.Temp(temp, cfg.TempUnit)
	fb.String(rightX, 30, "TMP:"+tempStr, sm, tempColor)
	tempPct := temp
	if tempPct > 100 {
		tempPct = 100
	}
	fb.Bar(rightX, 40, barW, barH, int(tempPct), tempColor, theme.ColorSep)

	// RAM (left column)
	ramColor := theme.ThresholdColor(c.RAMPercent(), theme.RAMWarn, theme.RAMCrit)
	fb.String(leftX, 48, fmt.Sprintf("RAM:%3d%%", int(ram)), sm, ramColor)
	fb.Bar(leftX, 58, barW, barH, int(ram), ramColor, theme.ColorSep)

	// Disk (right column)
	diskColor := theme.ThresholdColor(c.DiskPercent(), theme.DiskWarn, theme.DiskCrit)
	fb.String(rightX, 48, fmt.Sprintf("DSK:%3d%%", int(disk)), sm, diskColor)
	fb.Bar(rightX, 58, barW, barH, int(disk), diskColor, theme.ColorSep)
}

// clampMin returns v if v >= min, otherwise min.
func clampMin(v, min float64) float64 {
	if v < min {
		return min
	}
	return v
}
