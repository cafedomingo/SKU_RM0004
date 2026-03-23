package screen

import (
	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
	"github.com/cafedomingo/SKU_RM0004/internal/theme"
)

// Screen renders a display mode and manages its own framebuffers.
type Screen interface {
	Update(c sysinfo.Collector, cfg config.Config)
	Send(disp st7735.Display)
	NeedsRefresh() bool
	Buffer() *st7735.Framebuffer
}

// New returns a Screen for the given screen name.
// If disp is non-nil, the display is blanked on creation.
func New(name string, disp st7735.Display) Screen {
	if disp != nil {
		var blank st7735.Framebuffer
		blank.Fill(theme.ColorBG)
		disp.SendFull(blank.Pixels[:])
	}

	switch name {
	case config.ScreenDiagnostic:
		return &diagnosticScreen{}
	case config.ScreenSparkline:
		return &sparklineScreen{}
	default:
		return &dashboardScreen{}
	}
}
