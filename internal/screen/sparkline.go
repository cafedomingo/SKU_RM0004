package screen

import (
	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/font"
	"github.com/cafedomingo/SKU_RM0004/internal/format"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
	"github.com/cafedomingo/SKU_RM0004/internal/theme"
)

// SparklineHistory is the number of historical samples stored per metric.
const SparklineHistory = 13

// Sparkline layout constants shared across draw functions.
const (
	rightColX = 82 // x offset where the right column starts
	sepY      = 35 // y coordinate of separator between header and graph
	badgeGap  = 3  // pixel gap between adjacent badges/elements
)

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

	f := font.Spleen6x12

	drawTicker(fb, f, s.collector, s)
	drawUptimeRow(fb, f, s.collector)
	drawFreqRow(fb, f, s.collector, cfg)
	fb.Rect(0, sepY, st7735.Width, 1, theme.ColorSep)
	drawSparklineGraph(fb, 0, s.cpuHistory[:], theme.CPUWarn, theme.CPUCrit)
	drawSparklineGraph(fb, rightColX, s.ramHistory[:], theme.RAMWarn, theme.RAMCrit)
	drawCPURAMValues(fb, f, s.collector)
	drawIORow(fb, f, s.collector)
}

func (s *sparklineScreen) Draw() {
	drawChanged(s.disp, &s.front, &s.back)
}

// drawTicker renders the cycling ticker row at y=0.
// All phases use the same color since they share the row.
// Phase 0 = hostname, phase 1 = IPv4, phase 2 = IPv6.
func drawTicker(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector, s *sparklineScreen) {
	var text string

	switch s.tickerPhase {
	case 0:
		text = c.Hostname()
	case 1:
		text = c.IPv4Address()
	case 2:
		text = c.IPv6Suffix()
	}

	fb.String(0, 1, text, f, theme.ColorIdentity)

	// Advance: hostname -> ipv4 -> ipv6 (if available) -> hostname
	s.tickerPhase++
	hasIPv6 := c.IPv6Suffix() != "" && c.IPv6Suffix() != sysinfo.NoIPv6
	if s.tickerPhase > 2 || (s.tickerPhase == 2 && !hasIPv6) {
		s.tickerPhase = 0
	}
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
		badgeWidth := format.StringWidth(badge, f)
		rightEdge -= badgeWidth
		fb.String(rightEdge, y, badge, f, theme.APTColor(c.APTUpdateCount()))
	}

	// DietPi diamond — left of APT badge with gap, built from right edge inward
	if c.DietPiStatus() == sysinfo.DietPiUpdateAvail {
		rightEdge -= badgeGap
		rightEdge -= f.Width
		fb.Char(rightEdge, y, font.Diamond, f, theme.ColorAlert)
	}
}

// drawFreqRow renders CPU freq + throttle on the left, temp | D:N% on the right at y=25.
// Matches original C layout: right side built from right edge inward.
func drawFreqRow(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector, cfg config.Config) {
	const (
		colWidth  = 78
		y         = 25
		diskLabel = "D:"
	)

	// Left: CPU frequency
	freq := c.CPUFreq()
	freqStr := format.Freq(freq.Cur)
	fb.String(0, y, freqStr, f, theme.ColorFG)

	// Throttle indicator right after freq
	throttle := c.ThrottleStatus()
	if throttle&sysinfo.ThrottleCurrentMask != 0 {
		throttleX := format.StringWidth(freqStr, f)
		fb.String(throttleX, y, "!", f, theme.ColorAlert)
	}

	// Right side: build from right edge inward
	// D:N% — label in white, value in threshold color, right-aligned
	diskVal := format.Pct(c.DiskPercent())
	diskColor := theme.DiskColor(c.DiskPercent())
	diskLabelW := format.StringWidth(diskLabel, f)
	diskValW := format.StringWidth(diskVal, f)
	diskX := rightColX + colWidth - diskLabelW - diskValW
	fb.String(diskX, y, diskLabel, f, theme.ColorFG)
	fb.String(diskX+diskLabelW, y, diskVal, f, diskColor)

	// Pipe separator with equal gaps: temp [gap] | [gap] D:N%
	pipeX := diskX - badgeGap - f.Width
	fb.String(pipeX, y, "|", f, theme.ColorSep)

	// Temperature right-aligned before the gap
	tempStr := format.Temp(c.Temperature(), cfg.TempUnit == config.TempFahrenheit)
	tempColor := theme.TempColor(c.Temperature())
	tempW := format.StringWidth(tempStr, f)
	fb.String(pipeX-badgeGap-tempW, y, tempStr, f, tempColor)
}

// drawSparklineGraph renders 13 vertical bars in the graph area.
// Each bar is 5px wide with 1px gap, starting at x offset xOff.
// Graph area: y=37 to y=54 (18px tall).
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

	cpu := max(c.CPUPercent(), 1)
	ram := max(c.RAMPercent(), 1)

	cpuColor := theme.CPUColor(c.CPUPercent())
	ramColor := theme.RAMColor(c.RAMPercent())

	const (
		cpuX     = 0
		ramX     = rightColX
		cpuLabel = "CPU:"
		ramLabel = "RAM:"
	)

	fb.String(cpuX, y, cpuLabel, f, theme.ColorFG)
	fb.String(cpuX+format.StringWidth(cpuLabel, f), y, format.Pct(cpu), f, cpuColor)

	fb.String(ramX, y, ramLabel, f, theme.ColorFG)
	fb.String(ramX+format.StringWidth(ramLabel, f), y, format.Pct(ram), f, ramColor)
}

// drawIORow renders network rx/tx and disk R/W at y=68.
// Network fills left column (0-78), disk fills right column (82-160).
func drawIORow(fb *st7735.Framebuffer, f *font.Font, c sysinfo.Collector) {
	const (
		y      = 68
		rxX    = 0
		txX    = 40
		readX  = rightColX
		writeX = 122
	)

	net := c.NetBandwidth()
	linkSpeed := c.LinkSpeedMbps()

	// Network: arrow+rx on left, arrow+tx centered in left column
	fb.Char(rxX, y, font.ArrowDown, f, theme.ColorFG)
	fb.String(rxX+f.Width, y, format.Rate(net.RxBytesPerSec), f, theme.NetColor(net.RxBytesPerSec, linkSpeed))

	fb.Char(txX, y, font.ArrowUp, f, theme.ColorFG)
	fb.String(txX+f.Width, y, format.Rate(net.TxBytesPerSec), f, theme.NetColor(net.TxBytesPerSec, linkSpeed))

	// Disk: R+read on left of right column, W+write at midpoint
	dio := c.DiskIO()
	fb.String(readX, y, "R", f, theme.ColorFG)
	fb.String(readX+f.Width, y, format.Rate(dio.ReadBytesPerSec), f, theme.DiskIOColor(dio.ReadBytesPerSec))

	fb.String(writeX, y, "W", f, theme.ColorFG)
	fb.String(writeX+f.Width, y, format.Rate(dio.WriteBytesPerSec), f, theme.DiskIOColor(dio.WriteBytesPerSec))
}
