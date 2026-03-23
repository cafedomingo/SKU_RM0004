package format

import (
	"fmt"
	"time"
)

const (
	KB = 1024
	MB = 1024 * 1024

	secsPerHour = 60 * 60
	secsPerDay  = 24 * secsPerHour

	aptBadgeMax = 99
)

// Rate formats a byte rate for display: "0B", "1.0K", "10K", "1.0M", "10M"
func Rate(bytes uint64) string {
	switch {
	case bytes < KB:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 10*KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/KB)
	case bytes < MB:
		return fmt.Sprintf("%dK", bytes/KB)
	case bytes < 10*MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/MB)
	default:
		return fmt.Sprintf("%dM", bytes/MB)
	}
}

// Freq formats CPU frequency: "600MHz" or "1.8GHz"
func Freq(mhz uint16) string {
	if mhz < 1000 {
		return fmt.Sprintf("%dMHz", mhz)
	}
	return fmt.Sprintf("%.1fGHz", float64(mhz)/1000)
}

// Uptime formats a duration: "1d 2h", "5h 12m", "42m"
func Uptime(d time.Duration) string {
	total := int(d.Seconds())
	days := total / secsPerDay
	hours := (total % secsPerDay) / secsPerHour
	minutes := (total % secsPerHour) / 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}

// Temp formats temperature for display: "52°C" or "125°F".
func Temp(celsius float64, unit string) string {
	val := celsius
	if unit == "F" {
		val = CelsiusToF(celsius)
	}
	return fmt.Sprintf("%2d°%s", int(val), unit)
}

// CelsiusToF converts Celsius to Fahrenheit
func CelsiusToF(c float64) float64 {
	return c*9/5 + 32
}

// RuneLen returns the number of runes in a string.
// Use instead of len() when calculating pixel widths, since len() counts bytes
// and multi-byte characters like ° would give wrong results.
func RuneLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// APTBadge formats APT update count: "^3" (capped at 99). Returns "" for count <= 0.
func APTBadge(count int) string {
	if count <= 0 {
		return ""
	}
	if count > aptBadgeMax {
		count = aptBadgeMax
	}
	return fmt.Sprintf("^%d", count)
}
