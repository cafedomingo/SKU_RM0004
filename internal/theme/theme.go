// Package theme provides centralized color constants and threshold logic
// for the SKU_RM0004 LCD display screens.
//
// Colors are expressed as RGB565 values, matching the ST7735 display format.
// The RGB565 encoding is: ((r >> 3) << 11) | ((g >> 2) << 5) | (b >> 3)
package theme

// color565 encodes an RGB triplet into an RGB565 uint16.
// It is only used at package init time; constants are pre-computed below.
func color565(r, g, b uint8) uint16 {
	return (uint16(r>>3) << 11) | (uint16(g>>2) << 5) | uint16(b>>3)
}

// UI palette — sourced from the original C firmware color struct.
// Values are pre-computed via the RGB565 formula: ((r>>3)<<11)|((g>>2)<<5)|(b>>3)
const (
	ColorBG       uint16 = 0x0000 // black      (0x00,0x00,0x00)
	ColorFG       uint16 = 0xFFFF // white      (0xFF,0xFF,0xFF)
	ColorMuted    uint16 = 0x8410 // gray       (0x80,0x80,0x80)
	ColorSep      uint16 = 0x39C7 // dark gray  (0x3A,0x3A,0x3A)
	ColorIdentity uint16 = 0x7D5F // lt blue    (0x79,0xA8,0xFF)
	ColorAlert    uint16 = 0xFC0B // orange     (0xFF,0x80,0x59)
	ColorOK       uint16 = 0x45E8 // green      (0x44,0xBC,0x44)
	ColorWarn     uint16 = 0xD5E0 // yellow     (0xD0,0xBC,0x00)
	ColorCrit     uint16 = 0xFC0B // red-orange (same as alert)
)

// Temperature ramp colors (DietPi-style breakpoints).
const (
	TempCyan   uint16 = 0x07FF // <40 °C  (0x00,0xFF,0xFF)
	TempGreen  uint16 = 0x07E0 // 40 °C   (0x00,0xFF,0x00)
	TempYellow uint16 = 0xFFE0 // 50 °C   (0xFF,0xFF,0x00)
	TempOrange uint16 = 0xFC00 // 60 °C   (0xFF,0x80,0x00)
	TempRed    uint16 = 0xF800 // 70+ °C  (0xFF,0x00,0x00)
)

// Metric thresholds (warn / crit percentages or absolute values).
const (
	CPUWarn  = 60.0
	CPUCrit  = 80.0
	RAMWarn  = 60.0
	RAMCrit  = 80.0
	DiskWarn = 70.0
	DiskCrit = 90.0

	// Disk I/O in bytes/s
	DiskIOWarn = 25 * 1024 * 1024
	DiskIOCrit = 75 * 1024 * 1024

	// APT pending upgrades
	APTCrit = 10
)

// Convenience color functions for common metrics.
func CPUColor(pct float64) uint16    { return ThresholdColor(pct, CPUWarn, CPUCrit) }
func RAMColor(pct float64) uint16    { return ThresholdColor(pct, RAMWarn, RAMCrit) }
func DiskColor(pct float64) uint16   { return ThresholdColor(pct, DiskWarn, DiskCrit) }
func DiskIOColor(v float64) uint16   { return ThresholdColor(v, DiskIOWarn, DiskIOCrit) }
func NetColor(v float64, linkSpeedMbps int) uint16 {
	warn, crit := NetThresholds(linkSpeedMbps)
	return ThresholdColor(v, float64(warn), float64(crit))
}

// ThresholdColor returns a color interpolated across three zones:
//   - below warn: ok → warn (lerp from 0 to warn)
//   - warn to crit: warn → crit (lerp)
//   - above crit: crit (solid)
func ThresholdColor(value, warn, crit float64) uint16 {
	if value >= crit {
		return ColorCrit
	}
	if value >= warn {
		t := float32((value - warn) / (crit - warn))
		return LerpColor(ColorWarn, ColorCrit, t)
	}
	if warn > 0 && value > 0 {
		t := float32(value / warn)
		return LerpColor(ColorOK, ColorWarn, t)
	}
	return ColorOK
}

// tempRamp holds the four breakpoint colors for temperatures 40–70 °C.
// Index 0 = 40 °C, index 3 = 70 °C. Values below 40 are handled as a hard
// step (cyan), so this slice starts at 40 to avoid division by zero.
var tempRamp = [4]uint16{TempGreen, TempYellow, TempOrange, TempRed}

// TempColor returns an interpolated RGB565 color for a CPU/GPU temperature.
// Below 40 °C it returns TempCyan. At and above 70 °C it returns TempRed.
// Between breakpoints it linearly interpolates the RGB565 components.
func TempColor(celsius float64) uint16 {
	if celsius < 40 {
		return TempCyan
	}
	if celsius >= 70 {
		return TempRed
	}

	// Map [40, 70) onto [0, 3) in 10-degree segments.
	pos := (celsius - 40) / 10.0 // 0.0 … <3.0
	idx := int(pos)               // segment index: 0, 1, or 2
	t := float32(pos - float64(idx))

	return LerpColor(tempRamp[idx], tempRamp[idx+1], t)
}

// NetThresholds returns warn and crit byte-per-second thresholds for a NIC
// with the given link speed in Mbps. The thresholds are 40% (warn) and 80%
// (crit) of the theoretical maximum throughput. If linkSpeedMbps is 0 (speed
// unknown) it falls back to a 100 Mbps assumption.
func NetThresholds(linkSpeedMbps int) (warn, crit uint64) {
	if linkSpeedMbps <= 0 {
		linkSpeedMbps = 100
	}
	// Convert Mbps → bytes/s: Mbps * 1_000_000 / 8 = Mbps * 125_000
	maxBytesPerSec := uint64(linkSpeedMbps) * 125_000
	warn = maxBytesPerSec * 40 / 100
	crit = maxBytesPerSec * 80 / 100
	return
}

// LerpColor linearly interpolates between two RGB565 colors.
// t=0 returns a, t=1 returns b.
func LerpColor(a, b uint16, t float32) uint16 {
	// Extract RGB565 channels.
	rA := uint8((a >> 11) & 0x1F)
	gA := uint8((a >> 5) & 0x3F)
	bA := uint8(a & 0x1F)

	rB := uint8((b >> 11) & 0x1F)
	gB := uint8((b >> 5) & 0x3F)
	bB := uint8(b & 0x1F)

	r := uint16(float32(rA) + t*float32(int(rB)-int(rA)))
	g := uint16(float32(gA) + t*float32(int(gB)-int(gA)))
	bl := uint16(float32(bA) + t*float32(int(bB)-int(bA)))

	return (r << 11) | (g << 5) | bl
}
