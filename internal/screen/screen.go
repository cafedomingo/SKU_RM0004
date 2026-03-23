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
	Send(disp st7735.Display, front, back *st7735.Framebuffer)
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

// sendDirty compares front and back, sends only changed regions.
func sendDirty(disp st7735.Display, front, back *st7735.Framebuffer) {
	for _, r := range st7735.DiffRegions(front, back) {
		disp.SendRegion(0, r.Y, st7735.Width, r.H,
			back.Pixels[r.Y*st7735.Width:(r.Y+r.H)*st7735.Width])
	}
	*front = *back
}

// sendFull sends the entire back buffer to the display.
func sendFull(disp st7735.Display, front, back *st7735.Framebuffer) {
	disp.SendFull(back.Pixels[:])
	*front = *back
}
