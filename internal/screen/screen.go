package screen

import (
	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
)

// Screen renders a display mode onto a framebuffer.
type Screen interface {
	Render(fb *st7735.Framebuffer, c sysinfo.Collector, cfg config.Config)
	NeedsRefresh() bool
	FullRedraw() bool
}

// New returns a Screen for the given screen name.
func New(name string) Screen {
	switch name {
	case config.ScreenDiagnostic:
		return &diagnosticScreen{}
	case config.ScreenSparkline:
		return &sparklineScreen{}
	default:
		return &dashboardScreen{}
	}
}
