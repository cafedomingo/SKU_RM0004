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
		if r.X == 0 && r.W == st7735.Width {
			// Full-width regions are contiguous in the framebuffer.
			disp.SendRegion(0, r.Y, r.W, r.H,
				back.Pixels[r.Y*st7735.Width:(r.Y+r.H)*st7735.Width])
			continue
		}
		pixels := make([]uint16, r.W*r.H)
		for row := 0; row < r.H; row++ {
			src := (r.Y+row)*st7735.Width + r.X
			copy(pixels[row*r.W:(row+1)*r.W], back.Pixels[src:src+r.W])
		}
		disp.SendRegion(r.X, r.Y, r.W, r.H, pixels)
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
