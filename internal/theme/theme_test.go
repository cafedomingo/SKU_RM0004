package theme_test

import (
	"testing"

	"github.com/cafedomingo/SKU_RM0004/internal/theme"
)

func TestThresholdColor(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		warn  float64
		crit  float64
		want  uint16
	}{
		{"below warn", 30, 60, 80, theme.ColorOK},
		{"at warn", 60, 60, 80, theme.ColorWarn},
		{"between warn and crit", 70, 60, 80, theme.ColorWarn},
		{"at crit", 80, 60, 80, theme.ColorCrit},
		{"above crit", 90, 60, 80, theme.ColorCrit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := theme.ThresholdColor(tt.value, tt.warn, tt.crit)
			if got != tt.want {
				t.Errorf("ThresholdColor(%v, %v, %v) = 0x%04X, want 0x%04X", tt.value, tt.warn, tt.crit, got, tt.want)
			}
		})
	}
}

func TestTempRampColor(t *testing.T) {
	tests := []struct {
		name    string
		celsius float64
		want    uint16
	}{
		{"below 40 is cyan", 39.9, theme.TempCyan},
		{"at 0 is cyan", 0, theme.TempCyan},
		{"at 40 is green", 40, theme.TempGreen},
		{"between 40 and 50 is between green and yellow", 45, 0}, // interpolated, checked separately
		{"at 50 is yellow", 50, theme.TempYellow},
		{"between 50 and 60 is between yellow and orange", 55, 0}, // interpolated, checked separately
		{"at 60 is orange", 60, theme.TempOrange},
		{"between 60 and 70 is between orange and red", 65, 0}, // interpolated, checked separately
		{"at 70 is red", 70, theme.TempRed},
		{"above 70 is red", 90, theme.TempRed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := theme.TempRampColor(tt.celsius)
			// For interpolated values, skip exact match
			if tt.want == 0 {
				return
			}
			if got != tt.want {
				t.Errorf("TempRampColor(%v) = 0x%04X, want 0x%04X", tt.celsius, got, tt.want)
			}
		})
	}
}

// TestTempRampColorInterpolation verifies the ramp is monotonically progressing
// through the hue range (no sudden jumps back).
func TestTempRampColorInterpolation(t *testing.T) {
	// At midpoints the result must be strictly between the two boundary values
	// We verify by checking the interpolated value differs from both endpoints.
	tests := []struct {
		celsius float64
		lo      uint16
		hi      uint16
	}{
		{45, theme.TempGreen, theme.TempYellow},
		{55, theme.TempYellow, theme.TempOrange},
		{65, theme.TempOrange, theme.TempRed},
	}
	for _, tt := range tests {
		got := theme.TempRampColor(tt.celsius)
		if got == tt.lo || got == tt.hi {
			t.Errorf("TempRampColor(%v) = 0x%04X; expected value interpolated between 0x%04X and 0x%04X", tt.celsius, got, tt.lo, tt.hi)
		}
	}
}

func TestNetThresholds(t *testing.T) {
	// 1 Gbps link: max = 1000 * 125_000 = 125_000_000 bytes/s
	// warn = 40% = 50_000_000, crit = 80% = 100_000_000
	const mbps1G = 1000
	warn, crit := theme.NetThresholds(mbps1G)
	wantWarn := uint64(50_000_000)
	wantCrit := uint64(100_000_000)
	if warn != wantWarn {
		t.Errorf("NetThresholds(1000) warn = %d, want %d", warn, wantWarn)
	}
	if crit != wantCrit {
		t.Errorf("NetThresholds(1000) crit = %d, want %d", crit, wantCrit)
	}
}

func TestNetThresholdsFallback(t *testing.T) {
	// 0 (unknown speed): fall back to 100 Mbps assumption
	// max = 100 * 125_000 = 12_500_000 bytes/s
	// warn = 40% = 5_000_000, crit = 80% = 10_000_000
	warn, crit := theme.NetThresholds(0)
	wantWarn := uint64(5_000_000)
	wantCrit := uint64(10_000_000)
	if warn != wantWarn {
		t.Errorf("NetThresholds(0) warn = %d, want %d", warn, wantWarn)
	}
	if crit != wantCrit {
		t.Errorf("NetThresholds(0) crit = %d, want %d", crit, wantCrit)
	}
}
