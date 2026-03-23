package screen

import (
	"testing"

	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
	"github.com/cafedomingo/SKU_RM0004/internal/theme"
)

func sparkMock() *sysinfo.MockCollector {
	return &sysinfo.MockCollector{
		Host: "sparkhost",
		IPv4: "10.0.0.1",
		IPv6: "::a8f1:23bc:abcd",
		CPU:  47,
		RAM:  63,
		Disk: 42,
		Temp: 52,
		Freq: sysinfo.CPUFreq{Cur: 1800},
		Net:  sysinfo.NetBandwidth{RxBytesPerSec: 1024, TxBytesPerSec: 512},
		DIO:  sysinfo.DiskIO{ReadBytesPerSec: 1024 * 1024, WriteBytesPerSec: 512 * 1024},
	}
}

func sparkCfg() config.Config {
	return config.Config{TempUnit: "C"}
}

// TestSparklineHistoryShift verifies that after a render the history arrays
// shift left and the newest value appears at the end.
func TestSparklineHistoryShift(t *testing.T) {

	m := sparkMock()
	m.CPU = 55
	m.RAM = 72

	state := &sparklineScreen{collector: m}
	// Pre-fill with known values
	for i := 0; i < SparklineHistory; i++ {
		state.cpuHistory[i] = float64(i * 5)
		state.ramHistory[i] = float64(i * 3)
	}

	state.Update(sparkCfg())

	// After render, element [0] should be the old element [1]
	if state.cpuHistory[0] != 5 {
		t.Errorf("CPU history[0] = %v, want 5 (shifted from [1])", state.cpuHistory[0])
	}
	// Last element should be the current CPU value
	if state.cpuHistory[SparklineHistory-1] != 55 {
		t.Errorf("CPU history[last] = %v, want 55", state.cpuHistory[SparklineHistory-1])
	}

	if state.ramHistory[0] != 3 {
		t.Errorf("RAM history[0] = %v, want 3 (shifted from [1])", state.ramHistory[0])
	}
	if state.ramHistory[SparklineHistory-1] != 72 {
		t.Errorf("RAM history[last] = %v, want 72", state.ramHistory[SparklineHistory-1])
	}
}

// TestSparklineTickerCycle verifies the ticker advances through phases and
// wraps correctly. With IPv6 present: 0->1->2->0. Without IPv6: 0->1->0.
func TestSparklineTickerCycle(t *testing.T) {
	t.Run("with_ipv6", func(t *testing.T) {
		m := sparkMock()
		m.IPv6 = "::a8f1:23bc:abcd"
		state := &sparklineScreen{collector: m, tickerPhase: 0}

		// Phase 0 -> renders hostname, advances to 1
		state.Update(sparkCfg())
		if state.tickerPhase != 1 {
			t.Errorf("after phase 0: got %d, want 1", state.tickerPhase)
		}

		// Phase 1 -> renders IPv4, advances to 2
		state.Update(sparkCfg())
		if state.tickerPhase != 2 {
			t.Errorf("after phase 1: got %d, want 2", state.tickerPhase)
		}

		// Phase 2 -> renders IPv6, wraps to 0
		state.Update(sparkCfg())
		if state.tickerPhase != 0 {
			t.Errorf("after phase 2: got %d, want 0", state.tickerPhase)
		}
	})

	t.Run("without_ipv6", func(t *testing.T) {
		m := sparkMock()
		m.IPv6 = ""
		state := &sparklineScreen{collector: m, tickerPhase: 0}

		// Phase 0 -> advances to 1
		state.Update(sparkCfg())
		if state.tickerPhase != 1 {
			t.Errorf("after phase 0: got %d, want 1", state.tickerPhase)
		}

		// Phase 1 -> wraps to 0 (no phase 2)
		state.Update(sparkCfg())
		if state.tickerPhase != 0 {
			t.Errorf("after phase 1: got %d, want 0", state.tickerPhase)
		}
	})
}

// TestSparklineThresholdColors verifies that sparkline bars use the correct
// threshold color based on their value.
func TestSparklineThresholdColors(t *testing.T) {

	m := sparkMock()
	m.CPU = 70 // above CPUWarn (60), below CPUCrit (80) -> warn color
	m.RAM = 90 // above RAMCrit (80) -> crit color

	state := &sparklineScreen{collector: m}
	state.Update(sparkCfg())

	// CPU graph is at x=0..77, y=37..54
	// The last bar (newest) at 70% should have visible colored pixels (lerped between warn and crit)
	// Last bar x = 12 * 6 = 72, width 5 -> x=72..76
	if !hasNonBGInRegion(state.Buffer(), 72, 37, 5, 18) {
		t.Error("expected CPU sparkline bar at 70% to have colored pixels")
	}

	// RAM graph is at x=82..159, y=37..54
	// At 90% (above crit), should be exact ColorCrit
	// Last bar x = 82 + 12*6 = 154, width 5 -> x=154..158
	if !hasColorInRegion(state.Buffer(), 154, 37, 5, 18, theme.ColorCrit) {
		t.Error("expected RAM sparkline bar at 90% to use crit color")
	}
}

// TestSparklineDisplayFloor verifies that CPU and RAM values of 0 are
// displayed as 1% in the text labels.
func TestSparklineDisplayFloor(t *testing.T) {

	m := sparkMock()
	m.CPU = 0
	m.RAM = 0

	state := &sparklineScreen{collector: m}
	state.Update(sparkCfg())

	// The CPU/RAM labels at y=56 should show 1% not 0%.
	if !hasNonBGInRegion(state.Buffer(), 0, 56, 40, 12) {
		t.Error("expected CPU label pixels at y=56 even when CPU=0")
	}

	if !hasNonBGInRegion(state.Buffer(), 82, 56, 40, 12) {
		t.Error("expected RAM label pixels at y=56 even when RAM=0")
	}
}

// TestSparklineRenders verifies the overall screen renders without panics
// and produces expected visual regions.
func TestSparklineRenders(t *testing.T) {

	m := sparkMock()
	state := &sparklineScreen{collector: m}
	state.Update(sparkCfg())

	// Ticker row at y=1 should have white pixels (hostname, 6x12 font)
	if !hasColorInRegion(state.Buffer(), 0, 1, 60, 12, theme.ColorFG) {
		t.Error("expected ticker text pixels at y=1")
	}

	// Separator at y=35
	if !hasColorInRegion(state.Buffer(), 0, 35, st7735.Width, 1, theme.ColorSep) {
		t.Error("expected separator line at y=35")
	}

	// Sparkline graph area should have colored bars (y=37..54, 18px tall)
	if !hasNonBGInRegion(state.Buffer(), 0, 37, 78, 18) {
		t.Error("expected CPU sparkline bars in graph area")
	}
	if !hasNonBGInRegion(state.Buffer(), 82, 37, 78, 18) {
		t.Error("expected RAM sparkline bars in graph area")
	}
}
