package screen

import (
	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
)

// Screen renders a display mode and manages its own framebuffers.
type Screen interface {
	Update(cfg config.Config)
	Draw()
	Buffer() *st7735.Framebuffer
}

// New returns a Screen for the given screen name.
// disp may be nil for off-screen rendering (e.g. screenshot generation);
// Draw() becomes a no-op and only Update()/Buffer() are usable.
func New(name string, disp st7735.Display, collector sysinfo.Collector) Screen {
	switch name {
	case config.ScreenDiagnostic:
		return &diagnosticScreen{disp: disp, collector: collector}
	case config.ScreenSparkline:
		return &sparklineScreen{disp: disp, collector: collector}
	default:
		return &dashboardScreen{disp: disp, collector: collector}
	}
}

// drawChanged compares front and back buffers and sends only changed regions.
func drawChanged(disp st7735.Display, front, back *st7735.Framebuffer) {
	if disp == nil {
		return
	}
	for _, r := range st7735.DiffRegions(front, back) {
		disp.SendRegion(0, r.Y, st7735.Width, r.H,
			back.Pixels[r.Y*st7735.Width:(r.Y+r.H)*st7735.Width])
	}
	*front = *back
}

// drawAll sends the entire back buffer to the display.
func drawAll(disp st7735.Display, back *st7735.Framebuffer) {
	if disp == nil {
		return
	}
	disp.SendFull(back.Pixels[:])
}
