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

type sparklineScreen struct {
	disp        st7735.Display
	collector   sysinfo.Collector
	front, back st7735.Framebuffer
	cpuHistory  [SparklineHistory]float64
	ramHistory  [SparklineHistory]float64
	tickerPhase int
}

func (s *sparklineScreen) Buffer() *st7735.Framebuffer { return &s.back }

func (s *sparklineScreen) Update(cfg config.Config) {
	s.back.Fill(theme.ColorBG)
	fb := &s.back

	// Shift history left, add new values
	copy(s.cpuHistory[0:], s.cpuHistory[1:])
	s.cpuHistory[SparklineHistory-1] = s.collector.CPUPercent()
	copy(s.ramHistory[0:], s.ramHistory[1:])
	s.ramHistory[SparklineHistory-1] = s.collector.RAMPercent()

	sm := font.Spleen6x12

	drawTicker(fb, sm, s.collector, s)
	drawUptimeRow(fb, sm, s.collector)
	drawFreqRow(fb, sm, s.collector, cfg)
	fb.Rect(0, 35, st7735.Width, 1, theme.ColorSep)
	drawSparklineGraph(fb, 0, s.cpuHistory[:], theme.CPUWarn, theme.CPUCrit)
	drawSparklineGraph(fb, 82, s.ramHistory[:], theme.RAMWarn, theme.RAMCrit)
	drawCPURAMValues(fb, sm, s.collector)
	drawIORow(fb, sm, s.collector)
}

func (s *sparklineScreen) Draw() {
	drawChanged(s.disp, &s.front, &s.back)
}

// drawTicker renders the cycling ticker row at y=0.
// Phase 0 = hostname (white), phase 1 = IPv4 (IP color), phase 2 = IPv6 (IP color).
// If IPv6 is empty, phase 2 is skipped.
func drawTicker(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector, s *sparklineScreen) {
	var text string
	var color uint16

	switch s.tickerPhase {
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
	if ipv6 == "" || ipv6 == sysinfo.NoIPv6 {
		maxPhase = 1
	}
	s.tickerPhase = (s.tickerPhase + 1) % (maxPhase + 1)
}

// drawUptimeRow renders uptime on the left, update badges on the right at y=14.
func drawUptimeRow(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector) {
	const y = 14

	fb.String(0, y, format.Uptime(c.Uptime()), f, theme.ColorFG)

	// Build badges from right edge inward
	rightEdge := st7735.Width

	// APT badge
	badge := format.APTBadge(c.APTUpdateCount())
	if badge != "" {
		badgeColor := theme.ColorWarn
		if c.APTUpdateCount() >= theme.APTCrit {
			badgeColor = theme.ColorCrit
		}
		badgeWidth := len(badge) * f.Width
		rightEdge -= badgeWidth
		fb.String(rightEdge, y, badge, f, badgeColor)
	}

	// DietPi diamond — left of APT badge with gap, built from right edge inward
	if c.DietPiStatus() == sysinfo.DietPiUpdateAvail {
		rightEdge -= 3 // gap
		rightEdge -= f.Width
		fb.Char(rightEdge, y, font.Diamond, f, theme.ColorAlert)
	}
}

// drawFreqRow renders CPU freq + throttle on the left, temp | D:N% on the right at y=25.
// Matches original C layout: right side built from right edge inward.
func drawFreqRow(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector, cfg config.Config) {
	const (
		colRightX = 82
		colWidth  = 78
		y         = 25
	)

	// Left: CPU frequency
	freq := c.CPUFreq()
	freqStr := format.Freq(freq.Cur)
	fb.String(0, y, freqStr, f, theme.ColorFG)

	// Throttle indicator right after freq
	throttle := c.ThrottleStatus()
	if throttle&sysinfo.ThrottleCurrentMask != 0 {
		throttleX := len(freqStr) * f.Width
		fb.String(throttleX, y, "!", f, theme.ColorAlert)
	}

	// Right side: build from right edge inward
	// D:N% — "D:" label in white, value in threshold color, right-aligned
	diskVal := fmt.Sprintf("%d%%", int(c.DiskPercent()))
	diskColor := theme.DiskColor(c.DiskPercent())
	labelWidth := 2 * f.Width // "D:"
	valueWidth := len(diskVal) * f.Width
	diskX := colRightX + colWidth - labelWidth - valueWidth
	fb.String(diskX, y, "D:", f, theme.ColorFG)
	fb.String(diskX+labelWidth, y, diskVal, f, diskColor)

	// Pipe separator with equal gaps: temp [gap] | [gap] D:N%
	const gap = 3 // pixels between pipe and adjacent text
	pipeX := diskX - gap - f.Width
	fb.String(pipeX, y, "|", f, theme.ColorSep)

	// Temperature right-aligned before the gap
	tempStr := format.Temp(c.Temperature(), cfg.TempUnit == config.TempFahrenheit)
	tempColor := theme.TempRampColor(c.Temperature())
	tempW := format.RuneLen(tempStr) * f.Width
	fb.String(pipeX-gap-tempW, y, tempStr, f, tempColor)
}

// drawSparklineGraph renders 13 vertical bars in the graph area.
// Each bar is 5px wide with 1px gap, starting at x offset xOff.
// Graph area: y=19 to y=48 (30px tall).
func drawSparklineGraph(fb *st7735.Framebuffer, xOff int, history []float64, warn, crit float64) {
	const (
		barW     = 5
		barGap   = 1
		graphY   = 37
		graphH   = 18
		graphEnd = 54 // graphY + graphH - 1
	)

	for i, val := range history {
		if val == 0 {
			continue
		}
		x := xOff + i*(barW+barGap)
		color := theme.ThresholdColor(val, warn, crit)

		// Bar height proportional to value, with rounding and 1px minimum
		h := int((val*float64(graphH) + 50) / 100)
		if h < 1 {
			h = 1
		}
		if h > graphH {
			h = graphH
		}

		fb.Rect(x, graphEnd-h+1, barW, h, color)
	}
}

// drawCPURAMValues renders CPU and RAM labels (white) + values (threshold color) at y=56.
func drawCPURAMValues(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector) {
	const y = 56

	cpu := format.ClampMin(c.CPUPercent(), 1)
	ram := format.ClampMin(c.RAMPercent(), 1)

	cpuColor := theme.CPUColor(c.CPUPercent())
	ramColor := theme.RAMColor(c.RAMPercent())

	// CPU label (white) + value (colored)
	fb.String(0, y, "CPU:", f, theme.ColorFG)
	cpuVal := fmt.Sprintf("%d%%", int(cpu))
	fb.String(4*f.Width, y, cpuVal, f, cpuColor)

	// RAM label (white) + value (colored)
	fb.String(82, y, "RAM:", f, theme.ColorFG)
	ramVal := fmt.Sprintf("%d%%", int(ram))
	fb.String(82+4*f.Width, y, ramVal, f, ramColor)
}

// drawIORow renders network rx/tx and disk R/W at y=68.
// Network fills left column (0-78), disk fills right column (82-160).
func drawIORow(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector) {
	const y = 68

	net := c.NetBandwidth()
	linkSpeed := c.LinkSpeedMbps()

	// Network: arrow+rx on left, arrow+tx centered in left column
	fb.Char(0, y, font.ArrowDown, f, theme.ColorFG)
	rxStr := format.Rate(net.RxBytesPerSec)
	fb.String(f.Width, y, rxStr, f, theme.NetColor(float64(net.RxBytesPerSec), linkSpeed))

	txX := 40
	fb.Char(txX, y, font.ArrowUp, f, theme.ColorFG)
	txStr := format.Rate(net.TxBytesPerSec)
	fb.String(txX+f.Width, y, txStr, f, theme.NetColor(float64(net.TxBytesPerSec), linkSpeed))

	// Disk: R+read on left of right column, W+write at midpoint
	dio := c.DiskIO()
	fb.String(82, y, "R", f, theme.ColorFG)
	fb.String(82+f.Width, y, format.Rate(dio.ReadBytesPerSec), f, theme.DiskIOColor(float64(dio.ReadBytesPerSec)))

	wX := 122
	fb.String(wX, y, "W", f, theme.ColorFG)
	fb.String(wX+f.Width, y, format.Rate(dio.WriteBytesPerSec), f, theme.DiskIOColor(float64(dio.WriteBytesPerSec)))
}
