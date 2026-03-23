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
// Layout (160x80, Spleen 8x16 font):
//
//	y=0:  Hostname (white)              ◆ (top-right, if DietPi update)
//	y=16: IP address (light blue)       APT badge right-aligned (if updates)
//	y=33: ─── separator line (1px) ───
//	y=35: CPU:NNN% (left)               TMP:NN°C (right)
//	y=52: [CPU bar]                     [Temp bar]
//	y=56: RAM:NNN% (left)               DSK:NNN% (right)
//	y=73: [RAM bar]                     [Disk bar]
func RenderDashboard(fb *st7735.Framebuffer, c sysinfo.Collector, cfg config.Config) {
	f := font.Spleen8x16

	// Header: hostname and IP
	fb.String(2, 0, c.Hostname(), f, theme.ColorFG)
	fb.String(2, 16, c.IPAddress(), f, theme.ColorIP)

	// DietPi update diamond indicator
	if c.DietPiStatus() == sysinfo.DietPiUpdateAvail {
		fb.Char(152, 0, '\u25C6', f, theme.ColorAlert)
	}

	// APT update badge (right-aligned on IP row)
	badge := format.APTBadge(c.APTUpdateCount())
	if badge != "" {
		color := theme.ColorWarn
		if c.APTUpdateCount() >= theme.APTCrit {
			color = theme.ColorCrit
		}
		bx := st7735.Width - len(badge)*f.Width - 2
		fb.String(bx, 16, badge, f, color)
	}

	// Separator
	fb.Rect(0, 33, st7735.Width, 1, theme.ColorSep)

	// Clamp display values to minimum 1% so bars are always visible
	cpu := clampMin(c.CPUPercent(), 1)
	ram := clampMin(c.RAMPercent(), 1)
	disk := clampMin(c.DiskPercent(), 1)
	temp := c.Temperature()

	const (
		barW   = 65
		barH   = 4
		leftX  = 2
		rightX = 82
	)

	// CPU (left column)
	cpuColor := theme.ThresholdColor(c.CPUPercent(), theme.CPUWarn, theme.CPUCrit)
	fb.String(leftX, 35, fmt.Sprintf("CPU:%3d%%", int(cpu)), f, cpuColor)
	fb.Bar(leftX, 52, barW, barH, int(cpu), cpuColor, theme.ColorSep)

	// Temperature (right column)
	tempColor := theme.TempRampColor(temp)
	tempStr := format.Temp(temp, cfg.TempUnit)
	fb.String(rightX, 35, "TMP:"+tempStr, f, tempColor)
	tempPct := temp
	if tempPct > 100 {
		tempPct = 100
	}
	fb.Bar(rightX, 52, barW, barH, int(tempPct), tempColor, theme.ColorSep)

	// RAM (left column)
	ramColor := theme.ThresholdColor(c.RAMPercent(), theme.RAMWarn, theme.RAMCrit)
	fb.String(leftX, 56, fmt.Sprintf("RAM:%3d%%", int(ram)), f, ramColor)
	fb.Bar(leftX, 73, barW, barH, int(ram), ramColor, theme.ColorSep)

	// Disk (right column)
	diskColor := theme.ThresholdColor(c.DiskPercent(), theme.DiskWarn, theme.DiskCrit)
	fb.String(rightX, 56, fmt.Sprintf("DSK:%3d%%", int(disk)), f, diskColor)
	fb.Bar(rightX, 73, barW, barH, int(disk), diskColor, theme.ColorSep)
}

// clampMin returns v if v >= min, otherwise min.
func clampMin(v, min float64) float64 {
	if v < min {
		return min
	}
	return v
}
