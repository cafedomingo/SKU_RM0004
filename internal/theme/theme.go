// Package theme provides centralized color constants and threshold logic
// for the SKU_RM0004 LCD display screens.
//
// Colors are expressed as RGB565 values, matching the ST7735 display format.
// The RGB565 encoding is: ((r >> 3) << 11) | ((g >> 2) << 5) | (b >> 3)
package theme

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

	ColorTempCool uint16 = 0x07FF // cyan       (0x00,0xFF,0xFF)
	ColorTempOK          = ColorOK
	ColorTempWarn        = ColorWarn
	ColorTempHot  uint16 = 0xFC00 // orange     (0xFF,0x80,0x00)
	ColorTempCrit        = ColorCrit
)

// Temperature ramp: color stops paired with °C thresholds (DietPi-style).
// Below the first stop: that stop's color. At or above the last: that color.
// Between stops: linearly interpolated.
var tempRamp = []struct {
	celsius float64
	color   uint16
}{
	{30, ColorTempCool},
	{40, ColorTempOK},
	{50, ColorTempWarn},
	{60, ColorTempHot},
	{70, ColorTempCrit},
}

// Metric thresholds (warn / crit percentages or absolute values).
const (
	CPUWarn  = 60.0
	CPUCrit  = 80.0
	RAMWarn  = 60.0
	RAMCrit  = 80.0
	DiskWarn = 70.0
	DiskCrit = 90.0

	// Disk I/O in bytes/s (MiB thresholds)
	mib        = 1024 * 1024
	DiskIOWarn = 25 * mib
	DiskIOCrit = 75 * mib

	// APT pending upgrades
	APTWarn = 1
	APTCrit = 10

	// Network bandwidth
	netDefaultMbps  = 100           // assumed link speed when unknown
	netWarnPct      = 40            // warn at 40% of link capacity
	netCritPct      = 80            // crit at 80% of link capacity
	mbpsToBytesPerS = 1_000_000 / 8 // bits/s to bytes/s
)

// Convenience color functions for common metrics.
func CPUColor(pct float64) uint16  { return ThresholdColor(pct, CPUWarn, CPUCrit) }
func RAMColor(pct float64) uint16  { return ThresholdColor(pct, RAMWarn, RAMCrit) }
func DiskColor(pct float64) uint16 { return ThresholdColor(pct, DiskWarn, DiskCrit) }
func DiskIOColor(v uint64) uint16  { return ThresholdColor(float64(v), DiskIOWarn, DiskIOCrit) }
func APTColor(count int) uint16    { return ThresholdColor(float64(count), APTWarn, APTCrit) }
func NetColor(v uint64, linkSpeedMbps int) uint16 {
	warn, crit := NetThresholds(linkSpeedMbps)
	return ThresholdColor(float64(v), float64(warn), float64(crit))
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

// TempColor returns an interpolated RGB565 color for a CPU/GPU temperature.
// Below the first ramp stop it returns that stop's color. At or above the
// last stop it returns that stop's color. Between stops it linearly interpolates.
func TempColor(celsius float64) uint16 {
	if celsius <= tempRamp[0].celsius {
		return tempRamp[0].color
	}
	for i := 1; i < len(tempRamp); i++ {
		if celsius < tempRamp[i].celsius {
			lo, hi := tempRamp[i-1], tempRamp[i]
			t := float32((celsius - lo.celsius) / (hi.celsius - lo.celsius))
			return LerpColor(lo.color, hi.color, t)
		}
	}
	return tempRamp[len(tempRamp)-1].color
}

// NetThresholds returns warn and crit byte-per-second thresholds for a NIC
// with the given link speed in Mbps. The thresholds are 40% (warn) and 80%
// (crit) of the theoretical maximum throughput. If linkSpeedMbps is 0 (speed
// unknown) it falls back to a default assumption.
func NetThresholds(linkSpeedMbps int) (warn, crit uint64) {
	if linkSpeedMbps <= 0 {
		linkSpeedMbps = netDefaultMbps
	}
	maxBytesPerSec := uint64(linkSpeedMbps) * mbpsToBytesPerS
	warn = maxBytesPerSec * netWarnPct / 100
	crit = maxBytesPerSec * netCritPct / 100
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
