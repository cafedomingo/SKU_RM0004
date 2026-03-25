package format

import (
	"fmt"
	"time"

	"github.com/cafedomingo/SKU_RM0004/internal/font"
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
	seconds := int(d.Seconds())
	days := seconds / secsPerDay
	hours := (seconds % secsPerDay) / secsPerHour
	minutes := (seconds % secsPerHour) / 60

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
func Temp(celsius float64, toFahrenheit bool) string {
	if toFahrenheit {
		return fmt.Sprintf("%2d°F", int(CelsiusToF(celsius)))
	}
	return fmt.Sprintf("%2d°C", int(celsius))
}

// Pct formats a percentage for display: " 47%", "100%".
func Pct(v float64) string {
	return fmt.Sprintf("%3d%%", int(v))
}

// CelsiusToF converts Celsius to Fahrenheit
func CelsiusToF(c float64) float64 {
	return c*9/5 + 32
}

// StringWidth returns the pixel width of s when rendered in the given font.
// Use instead of len()*font.Width, since len() counts bytes and multi-byte
// characters like ° would give wrong results.
func StringWidth(s string, f *font.Font) int {
	n := 0
	for range s {
		n++
	}
	return n * f.Width
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
