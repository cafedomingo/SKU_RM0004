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
// Layout (160x80, Spleen 6x12 for text):
//
//	y=1:   Ticker (hostname -> IPv4 -> IPv6, cycling)     | badges right-aligned
//	y=12:  Uptime                                         | DietPi/APT badges
//	y=23:  CPU freq + throttle                            | Temp | D:N%
//	y=33:  --- separator (1px) ---
//	y=35:  [sparkline graph area -- CPU bars left, RAM bars right]
//	       Graph height: 20px (y=35 to y=54)
//	y=56:  CPU N% (left)                                  RAM N% (right)
//	y=68:  down-arrow rx up-arrow tx (left)               R disk W disk (right)
func RenderSparkline(fb *st7735.Framebuffer, c sysinfo.Collector, cfg config.Config, state *SparklineState) {
	sm := font.Spleen6x12

	// Shift history left, add new values
	copy(state.CPUHistory[0:], state.CPUHistory[1:])
	state.CPUHistory[SparklineHistory-1] = c.CPUPercent()
	copy(state.RAMHistory[0:], state.RAMHistory[1:])
	state.RAMHistory[SparklineHistory-1] = c.RAMPercent()

	// Row 1: Ticker
	drawTicker(fb, sm, c, state)

	// Row 2: Uptime + badges
	drawUptimeRow(fb, sm, c)

	// Row 3: Freq + temp/disk
	drawFreqRow(fb, sm, c, cfg)

	// Separator
	fb.Rect(0, 33, st7735.Width, 1, theme.ColorSep)

	// Sparkline graphs
	drawSparklineGraph(fb, 0, state.CPUHistory[:], theme.CPUWarn, theme.CPUCrit)
	drawSparklineGraph(fb, 82, state.RAMHistory[:], theme.RAMWarn, theme.RAMCrit)

	// CPU/RAM summary
	drawCPURAMValues(fb, sm, c)

	// I/O row
	drawIORow(fb, sm, c)
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

	fb.String(0, 1, text, f, color)

	// Advance ticker phase
	maxPhase := 2
	ipv6 := c.IPv6Suffix()
	if ipv6 == "" || ipv6 == "no IPv6" {
		maxPhase = 1
	}
	state.TickerPhase = (state.TickerPhase + 1) % (maxPhase + 1)
}

// drawUptimeRow renders uptime on the left, update badges on the right at y=12.
func drawUptimeRow(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector) {
	const y = 12

	fb.String(0, y, format.Uptime(c.Uptime()), f, theme.ColorFG)

	// Build badges from right edge inward
	ax := st7735.Width

	// APT badge
	badge := format.APTBadge(c.APTUpdateCount())
	if badge != "" {
		badgeColor := theme.ColorWarn
		if c.APTUpdateCount() >= theme.APTCrit {
			badgeColor = theme.ColorCrit
		}
		bw := len(badge) * f.Width
		ax -= bw
		fb.String(ax, y, badge, f, badgeColor)
	}

	// DietPi diamond (use 8x16 for the symbol, positioned on ticker row)
	big := font.Spleen8x16
	if c.DietPiStatus() == sysinfo.DietPiUpdateAvail {
		fb.Char(152, 1, '\u25C6', big, theme.ColorAlert)
	}
}

// drawFreqRow renders CPU freq + throttle on the left, temp | D:N% on the right at y=23.
func drawFreqRow(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector, cfg config.Config) {
	const y = 23

	// Left: CPU frequency
	freq := c.CPUFreq()
	freqStr := format.Freq(freq.Cur)
	fb.String(0, y, freqStr, f, theme.ColorFG)

	// Throttle indicator right after freq
	throttle := c.ThrottleStatus()
	if throttle&0x0000000F != 0 {
		tx := len(freqStr) * f.Width
		fb.String(tx, y, "!", f, theme.ColorAlert)
	}

	// Right side: build from right edge inward
	// D:N%
	diskStr := fmt.Sprintf("D:%d%%", int(c.DiskPercent()))
	diskColor := theme.ThresholdColor(c.DiskPercent(), theme.DiskWarn, theme.DiskCrit)
	dx := st7735.Width - len(diskStr)*f.Width
	fb.String(dx, y, diskStr, f, diskColor)

	// Pipe separator
	pipeX := dx - f.Width - 2
	fb.String(pipeX+2, y, "|", f, theme.ColorSep)

	// Temperature before the pipe
	tempStr := format.Temp(c.Temperature(), cfg.TempUnit)
	tempColor := theme.TempRampColor(c.Temperature())
	tempW := len(tempStr) * f.Width
	fb.String(pipeX-tempW, y, tempStr, f, tempColor)
}

// drawSparklineGraph renders 13 vertical bars in the graph area.
// Each bar is 5px wide with 1px gap, starting at x offset xOff.
// Graph area: y=19 to y=48 (30px tall).
func drawSparklineGraph(fb *st7735.Framebuffer, xOff int, history []float64, warn, crit float64) {
	const (
		barW     = 5
		barGap   = 1
		graphY   = 35
		graphH   = 20
		graphEnd = 54 // graphY + graphH - 1
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

// drawCPURAMValues renders CPU and RAM labels (white) + values (threshold color) at y=56.
func drawCPURAMValues(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector) {
	const y = 56

	cpu := clampMin(c.CPUPercent(), 1)
	ram := clampMin(c.RAMPercent(), 1)

	cpuColor := theme.ThresholdColor(c.CPUPercent(), theme.CPUWarn, theme.CPUCrit)
	ramColor := theme.ThresholdColor(c.RAMPercent(), theme.RAMWarn, theme.RAMCrit)

	// CPU label (white) + value (colored)
	fb.String(0, y, "CPU", f, theme.ColorFG)
	cpuVal := fmt.Sprintf("%d%%", int(cpu))
	fb.String(3*f.Width+1, y, cpuVal, f, cpuColor)

	// RAM label (white) + value (colored)
	fb.String(82, y, "RAM", f, theme.ColorFG)
	ramVal := fmt.Sprintf("%d%%", int(ram))
	fb.String(82+3*f.Width+1, y, ramVal, f, ramColor)
}

// drawIORow renders network rx/tx and disk R/W at y=68.
// Uses custom arrow indicators and separate R/W labels matching original C layout.
func drawIORow(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector) {
	const y = 68

	net := c.NetBandwidth()
	netWarn, netCrit := theme.NetThresholds(c.LinkSpeedMbps())

	// Network: down-arrow + rx, up-arrow + tx
	fb.Char(0, y, font.ArrowDown, f, theme.ColorFG)
	rxStr := format.Rate(net.RxBytesPerSec)
	fb.String(f.Width, y, rxStr, f, theme.ThresholdColor(float64(net.RxBytesPerSec), float64(netWarn), float64(netCrit)))

	txX := (1 + len(rxStr)) * f.Width + f.Width
	fb.Char(txX, y, font.ArrowUp, f, theme.ColorFG)
	txStr := format.Rate(net.TxBytesPerSec)
	fb.String(txX+f.Width, y, txStr, f, theme.ThresholdColor(float64(net.TxBytesPerSec), float64(netWarn), float64(netCrit)))

	// Disk: R + read rate, W + write rate
	dio := c.DiskIO()
	fb.String(82, y, "R", f, theme.ColorFG)
	rStr := format.Rate(dio.ReadBytesPerSec)
	fb.String(82+f.Width, y, rStr, f, theme.ThresholdColor(float64(dio.ReadBytesPerSec), theme.DiskIOWarn, theme.DiskIOCrit))

	wX := 82 + (1 + len(rStr)) * f.Width + f.Width
	fb.String(wX, y, "W", f, theme.ColorFG)
	wStr := format.Rate(dio.WriteBytesPerSec)
	fb.String(wX+f.Width, y, wStr, f, theme.ThresholdColor(float64(dio.WriteBytesPerSec), theme.DiskIOWarn, theme.DiskIOCrit))
}
