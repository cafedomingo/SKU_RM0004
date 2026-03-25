package theme

import "testing"

func TestThresholdColor(t *testing.T) {
	t.Run("at zero", func(t *testing.T) {
		got := ThresholdColor(0, 60, 80)
		if got != ColorOK {
			t.Errorf("got 0x%04X, want ColorOK 0x%04X", got, ColorOK)
		}
	})
	t.Run("at warn", func(t *testing.T) {
		got := ThresholdColor(60, 60, 80)
		if got != ColorWarn {
			t.Errorf("got 0x%04X, want ColorWarn 0x%04X", got, ColorWarn)
		}
	})
	t.Run("at crit", func(t *testing.T) {
		got := ThresholdColor(80, 60, 80)
		if got != ColorCrit {
			t.Errorf("got 0x%04X, want ColorCrit 0x%04X", got, ColorCrit)
		}
	})
	t.Run("above crit", func(t *testing.T) {
		got := ThresholdColor(90, 60, 80)
		if got != ColorCrit {
			t.Errorf("got 0x%04X, want ColorCrit 0x%04X", got, ColorCrit)
		}
	})

	t.Run("below warn is lerped", func(t *testing.T) {
		got := ThresholdColor(30, 60, 80)
		if got == ColorOK || got == ColorWarn {
			t.Errorf("got 0x%04X, expected interpolated between OK and Warn", got)
		}
	})
	t.Run("between warn and crit is lerped", func(t *testing.T) {
		got := ThresholdColor(70, 60, 80)
		if got == ColorWarn || got == ColorCrit {
			t.Errorf("got 0x%04X, expected interpolated between Warn and Crit", got)
		}
	})
}

func TestLerpColor(t *testing.T) {
	if got := LerpColor(0x0000, 0xFFFF, 0); got != 0x0000 {
		t.Errorf("LerpColor(0, FFFF, 0) = 0x%04X, want 0x0000", got)
	}
	if got := LerpColor(0x0000, 0xFFFF, 1); got != 0xFFFF {
		t.Errorf("LerpColor(0, FFFF, 1) = 0x%04X, want 0xFFFF", got)
	}
	mid := LerpColor(0x0000, 0xFFFF, 0.5)
	if mid == 0x0000 || mid == 0xFFFF {
		t.Errorf("LerpColor(0, FFFF, 0.5) = 0x%04X, expected midpoint", mid)
	}
	if got := LerpColor(0x1234, 0x1234, 0.5); got != 0x1234 {
		t.Errorf("LerpColor(same, same, 0.5) = 0x%04X, want 0x1234", got)
	}
}

func TestTempRampOrdering(t *testing.T) {
	if len(tempRamp) < 2 {
		t.Fatal("temp ramp must have at least 2 stops")
	}
	for i := 1; i < len(tempRamp); i++ {
		if tempRamp[i].celsius <= tempRamp[i-1].celsius {
			t.Errorf("temp ramp not ascending: stop[%d]=%v <= stop[%d]=%v",
				i, tempRamp[i].celsius, i-1, tempRamp[i-1].celsius)
		}
	}
}

func TestTempColor(t *testing.T) {
	cool := TempColor(30)
	optimal := TempColor(40)
	warm := TempColor(50)
	hot := TempColor(60)
	critical := TempColor(70)

	tests := []struct {
		name    string
		celsius float64
		want    uint16
	}{
		{"below first stop is cool", 0, cool},
		{"at 30 is cool", 30, cool},
		{"at 40 is optimal", 40, optimal},
		{"at 50 is warm", 50, warm},
		{"at 60 is hot", 60, hot},
		{"at 70 is critical", 70, critical},
		{"above 70 is critical", 90, critical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TempColor(tt.celsius)
			if got != tt.want {
				t.Errorf("TempColor(%v) = 0x%04X, want 0x%04X", tt.celsius, got, tt.want)
			}
		})
	}

	stops := []uint16{cool, optimal, warm, hot, critical}
	for i := 1; i < len(stops); i++ {
		if stops[i] == stops[i-1] {
			t.Errorf("stop %d and %d have same color 0x%04X", i-1, i, stops[i])
		}
	}
}

func TestTempColorInterpolation(t *testing.T) {
	tests := []struct {
		celsius float64
		lo, hi  float64
	}{
		{35, 30, 40},
		{45, 40, 50},
		{55, 50, 60},
		{65, 60, 70},
	}
	for _, tt := range tests {
		got := TempColor(tt.celsius)
		loColor := TempColor(tt.lo)
		hiColor := TempColor(tt.hi)
		if got == loColor || got == hiColor {
			t.Errorf("TempColor(%v) = 0x%04X; expected interpolated between 0x%04X and 0x%04X", tt.celsius, got, loColor, hiColor)
		}
	}
}

func TestNetThresholds(t *testing.T) {
	const mbps1G = 1000
	warn, crit := NetThresholds(mbps1G)
	if warn != 50_000_000 {
		t.Errorf("NetThresholds(1000) warn = %d, want 50000000", warn)
	}
	if crit != 100_000_000 {
		t.Errorf("NetThresholds(1000) crit = %d, want 100000000", crit)
	}
}

func TestNetThresholdsFallback(t *testing.T) {
	warn, crit := NetThresholds(0)
	if warn != 5_000_000 {
		t.Errorf("NetThresholds(0) warn = %d, want 5000000", warn)
	}
	if crit != 10_000_000 {
		t.Errorf("NetThresholds(0) crit = %d, want 10000000", crit)
	}
}
