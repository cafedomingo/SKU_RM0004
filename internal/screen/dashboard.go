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
//	y=18:  IP address (6x12, light blue)   APT badge right-aligned (6x12)
//	y=30:  ─── separator line (1px) ───
//	y=34:  CPU:NNN% (6x12, left)           TEMP:NNC (6x12, right)
//	y=46:  [CPU bar]                       [Temp bar] (6px tall)
//	y=56:  RAM:NNN% (6x12, left)           DSK:NNN% (6x12, right)
//	y=68:  [RAM bar]                       [Disk bar] (6px tall)
func RenderDashboard(fb *st7735.Framebuffer, c sysinfo.Collector, cfg config.Config) {
	big := font.Spleen8x16
	sm := font.Spleen6x12

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
	fb.Rect(0, 30, st7735.Width, 1, theme.ColorSep)

	// Clamp display values to minimum 1% so bars are always visible
	cpu := format.ClampMin(c.CPUPercent(), 1)
	ram := format.ClampMin(c.RAMPercent(), 1)
	disk := format.ClampMin(c.DiskPercent(), 1)
	temp := c.Temperature()

	const (
		barW   = 74
		barH   = 6
		leftX  = 2
		rightX = 84
	)

	// drawMetric renders "LABEL:" in white, value right-aligned in color, bar below
	drawMetric := func(x, y int, label, value string, pct int, color uint16) {
		fb.String(x, y, label, sm, theme.ColorFG)
		valX := x + barW - format.RuneLen(value)*sm.Width
		fb.String(valX, y, value, sm, color)
		fb.Bar(x, y+12, barW, barH, pct, color, theme.ColorSep)
	}

	// CPU (left column)
	cpuColor := theme.CPUColor(c.CPUPercent())
	drawMetric(leftX, 34, "CPU:", fmt.Sprintf("%3d%%", int(cpu)), int(cpu), cpuColor)

	// Temperature (right column)
	tempColor := theme.TempRampColor(temp)
	tempStr := format.Temp(temp, cfg.TempUnit)
	tempPct := int(temp)
	if tempPct > 100 {
		tempPct = 100
	}
	drawMetric(rightX, 34, "TEMP:", tempStr, tempPct, tempColor)

	// RAM (left column)
	ramColor := theme.RAMColor(c.RAMPercent())
	drawMetric(leftX, 56, "RAM:", fmt.Sprintf("%3d%%", int(ram)), int(ram), ramColor)

	// Disk (right column)
	diskColor := theme.DiskColor(c.DiskPercent())
	drawMetric(rightX, 56, "DISK:", fmt.Sprintf("%3d%%", int(disk)), int(disk), diskColor)
}
