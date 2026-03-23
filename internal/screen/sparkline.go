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

// SparklineHistory is the number of historical samples stored per metric.
const SparklineHistory = 13

// SparklineState holds the rolling history and ticker phase for the sparkline screen.
type SparklineState struct {
	CPUHistory  [SparklineHistory]float64
	RAMHistory  [SparklineHistory]float64
	TickerPhase int
}

// RenderSparkline draws the sparkline history screen onto the framebuffer.
//
// Layout (160x80, Spleen 8x16 font):
//
//	y=0:   Ticker (hostname -> IPv4 -> IPv6, cycling)     | badges right-aligned
//	y=16:  CPU freq + throttle indicator                  | Temp + Disk% right
//	y=33:  --- separator (1px) ---
//	y=35:  [sparkline graph area -- CPU bars left, RAM bars right]
//	       Graph height: 24px (y=35 to y=58)
//	y=60:  CPU N% (left)                                  RAM N% (right)
//	y=76:  down-arrow rx up-arrow tx (left)               R/W disk (right)
func RenderSparkline(fb *st7735.Framebuffer, c sysinfo.Collector, cfg config.Config, state *SparklineState) {
	f := font.Spleen8x16

	// Shift history left, add new values
	copy(state.CPUHistory[0:], state.CPUHistory[1:])
	state.CPUHistory[SparklineHistory-1] = c.CPUPercent()
	copy(state.RAMHistory[0:], state.RAMHistory[1:])
	state.RAMHistory[SparklineHistory-1] = c.RAMPercent()

	// Row 1: Ticker
	drawTicker(fb, f, c, state)

	// Row 2: Freq + temp/disk
	drawFreqRow(fb, f, c, cfg)

	// Separator
	fb.Rect(0, 33, st7735.Width, 1, theme.ColorSep)

	// Sparkline graphs
	drawSparklineGraph(fb, 0, state.CPUHistory[:], theme.CPUWarn, theme.CPUCrit)
	drawSparklineGraph(fb, 82, state.RAMHistory[:], theme.RAMWarn, theme.RAMCrit)

	// CPU/RAM summary
	drawCPURAMValues(fb, f, c)

	// I/O row
	drawIORow(fb, f, c)
}

// drawTicker renders the cycling ticker row at y=0.
// Phase 0 = hostname (white), phase 1 = IPv4 (IP color), phase 2 = IPv6 (IP color).
// If IPv6 is empty, phase 2 is skipped.
func drawTicker(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector, state *SparklineState) {
	var text string
	var color uint16

	switch state.TickerPhase {
	case 0:
		text = c.Hostname()
		color = theme.ColorFG
	case 1:
		text = c.IPAddress()
		color = theme.ColorIP
	case 2:
		text = c.IPv6Suffix()
		color = theme.ColorIP
	}

	fb.String(2, 0, text, f, color)

	// Right side: DietPi diamond + APT badge
	if c.DietPiStatus() == sysinfo.DietPiUpdateAvail {
		fb.Char(152, 0, '\u25C6', f, theme.ColorAlert)
	}

	badge := format.APTBadge(c.APTUpdateCount())
	if badge != "" {
		badgeColor := theme.ColorWarn
		if c.APTUpdateCount() >= theme.APTCrit {
			badgeColor = theme.ColorCrit
		}
		bx := st7735.Width - len(badge)*f.Width - 2
		// Don't overlap the diamond
		if c.DietPiStatus() == sysinfo.DietPiUpdateAvail {
			bx = 152 - len(badge)*f.Width - 2
		}
		fb.String(bx, 0, badge, f, badgeColor)
	}

	// Advance ticker phase
	maxPhase := 2
	if c.IPv6Suffix() == "" {
		maxPhase = 1
	}
	state.TickerPhase = (state.TickerPhase + 1) % (maxPhase + 1)
}

// drawFreqRow renders CPU freq + throttle on the left, temp + disk% on the right at y=16.
func drawFreqRow(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector, cfg config.Config) {
	const (
		leftX  = 2
		rightX = 82
	)

	// Left: CPU frequency
	freq := c.CPUFreq()
	freqStr := format.Freq(freq.Cur)

	// Throttle indicator
	throttle := c.ThrottleStatus()
	const (
		throttleCurrent = uint32(0x0000000F)
		throttlePast    = uint32(0x000F0000)
	)

	var throttleStr string
	var throttleColor uint16
	switch {
	case throttle&throttleCurrent != 0:
		throttleStr = "!"
		throttleColor = theme.ColorCrit
	case throttle&throttlePast != 0:
		throttleStr = "~"
		throttleColor = theme.ColorWarn
	}

	fb.String(leftX, 16, freqStr, f, theme.ColorFG)
	if throttleStr != "" {
		tx := leftX + len([]rune(freqStr))*f.Width
		fb.String(tx, 16, throttleStr, f, throttleColor)
	}

	// Right: Temp + Disk%
	tempStr := format.Temp(c.Temperature(), cfg.TempUnit)
	tempColor := theme.TempRampColor(c.Temperature())
	fb.String(rightX, 16, tempStr, f, tempColor)

	diskStr := fmt.Sprintf("D:%d%%", int(c.DiskPercent()))
	diskColor := theme.ThresholdColor(c.DiskPercent(), theme.DiskWarn, theme.DiskCrit)
	dx := st7735.Width - len([]rune(diskStr))*f.Width - 2
	fb.String(dx, 16, diskStr, f, diskColor)
}

// drawSparklineGraph renders 13 vertical bars in the graph area.
// Each bar is 5px wide with 1px gap, starting at x offset xOff.
// Graph area: y=35 to y=58 (24px tall).
func drawSparklineGraph(fb *st7735.Framebuffer, xOff int, history []float64, warn, crit float64) {
	const (
		barW     = 5
		barGap   = 1
		graphY   = 35
		graphH   = 24
		graphEnd = 59 // graphY + graphH - 1
	)

	for i, val := range history {
		x := xOff + i*(barW+barGap)
		color := theme.ThresholdColor(val, warn, crit)

		// Bar height proportional to value (0-100%)
		h := int(val * float64(graphH) / 100.0)
		if h < 0 {
			h = 0
		}
		if h > graphH {
			h = graphH
		}

		// Draw from bottom up
		if h > 0 {
			fb.Rect(x, graphEnd-h+1, barW, h, color)
		}
	}
}

// drawCPURAMValues renders the CPU and RAM percentage labels at y=60.
func drawCPURAMValues(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector) {
	const (
		leftX  = 2
		rightX = 82
		y      = 60
	)

	cpu := clampMin(c.CPUPercent(), 1)
	ram := clampMin(c.RAMPercent(), 1)

	cpuColor := theme.ThresholdColor(c.CPUPercent(), theme.CPUWarn, theme.CPUCrit)
	ramColor := theme.ThresholdColor(c.RAMPercent(), theme.RAMWarn, theme.RAMCrit)

	fb.String(leftX, y, fmt.Sprintf("CPU %d%%", int(cpu)), f, cpuColor)
	fb.String(rightX, y, fmt.Sprintf("RAM %d%%", int(ram)), f, ramColor)
}

// drawIORow renders network and disk I/O at y=76 (partial, clipped to 80px height).
func drawIORow(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector) {
	const y = 64

	net := c.NetBandwidth()
	rxStr := fmt.Sprintf("\u2193%s", format.Rate(net.RxBytesPerSec))
	txStr := fmt.Sprintf("\u2191%s", format.Rate(net.TxBytesPerSec))
	fb.String(2, y, rxStr, f, theme.ColorFG)
	txX := 2 + len([]rune(rxStr))*f.Width + f.Width
	fb.String(txX, y, txStr, f, theme.ColorFG)

	dio := c.DiskIO()
	ioStr := fmt.Sprintf("%s/%s", format.Rate(dio.ReadBytesPerSec), format.Rate(dio.WriteBytesPerSec))
	ix := st7735.Width - len([]rune(ioStr))*f.Width - 2
	fb.String(ix, y, ioStr, f, theme.ColorFG)
}
