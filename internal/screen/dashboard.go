package screen

import (
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
type dashboardScreen struct {
	disp        st7735.Display
	collector   sysinfo.Collector
	front, back st7735.Framebuffer
}

func (d *dashboardScreen) Buffer() *st7735.Framebuffer { return &d.back }

func (d *dashboardScreen) Update(cfg config.Config) {
	d.back.Fill(theme.ColorBG)
	d.render(&d.back, cfg)
}

func (d *dashboardScreen) Draw() {
	drawChanged(d.disp, &d.front, &d.back)
}

func (d *dashboardScreen) render(fb *st7735.Framebuffer, cfg config.Config) {
	headerFont := font.Spleen8x16
	metricFont := font.Spleen6x12

	// Header: hostname (big font)
	fb.String(2, 0, d.collector.Hostname(), headerFont, theme.ColorFG)

	// IP address (small font)
	fb.String(2, 18, d.collector.IPv4Address(), metricFont, theme.ColorIdentity)

	// DietPi update diamond indicator (big font for the symbol)
	if d.collector.DietPiStatus() == sysinfo.DietPiUpdateAvail {
		fb.Char(st7735.Width-headerFont.Width-2, 0, '\u25C6', headerFont, theme.ColorAlert)
	}

	// APT update badge (small font, right-aligned on IP row)
	badge := format.APTBadge(d.collector.APTUpdateCount())
	if badge != "" {
		color := theme.ColorWarn
		if d.collector.APTUpdateCount() >= theme.APTCrit {
			color = theme.ColorCrit
		}
		badgeX := st7735.Width - format.StringWidth(badge, metricFont) - 2
		fb.String(badgeX, 18, badge, metricFont, color)
	}

	// Separator
	fb.Rect(0, 30, st7735.Width, 1, theme.ColorSep)

	// Clamp display values to minimum 1% so bars are always visible
	cpu := max(d.collector.CPUPercent(), 1)
	ram := max(d.collector.RAMPercent(), 1)
	disk := max(d.collector.DiskPercent(), 1)
	temp := d.collector.Temperature()

	const (
		leftX  = 2
		rightX = 84
		row1Y  = 34
		row2Y  = 56
	)

	tempStr := format.Temp(temp, cfg.TempUnit == config.TempFahrenheit)

	drawMetric(fb, metricFont, leftX, row1Y, "CPU:", format.Pct(cpu), cpu, theme.CPUColor(cpu))
	drawMetric(fb, metricFont, rightX, row1Y, "TEMP:", tempStr, min(temp, 100), theme.TempColor(temp))
	drawMetric(fb, metricFont, leftX, row2Y, "RAM:", format.Pct(ram), ram, theme.RAMColor(ram))
	drawMetric(fb, metricFont, rightX, row2Y, "DISK:", format.Pct(disk), disk, theme.DiskColor(disk))
}

// drawMetric renders a labeled metric: "LABEL:" in white, value right-aligned in color, bar below.
func drawMetric(fb *st7735.Framebuffer, f *font.Font, x, y int, label, value string, pct float64, color uint16) {
	const (
		barW = 74
		barH = 6
	)
	fb.String(x, y, label, f, theme.ColorFG)
	valX := x + barW - format.StringWidth(value, f)
	fb.String(valX, y, value, f, color)
	fb.Bar(x, y+f.Height, barW, barH, int(pct), color, theme.ColorSep)
}
