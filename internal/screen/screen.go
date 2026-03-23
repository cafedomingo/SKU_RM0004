package screen

import (
	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
	"github.com/cafedomingo/SKU_RM0004/internal/theme"
)

// Screen renders a display mode and manages its own framebuffers.
type Screen interface {
	Update(cfg config.Config)
	Draw()
	Buffer() *st7735.Framebuffer
}

// New returns a Screen for the given screen name.
// If disp is non-nil, the display is blanked on creation.
func New(name string, disp st7735.Display, collector sysinfo.Collector) Screen {
	if disp != nil {
		var blank st7735.Framebuffer
		blank.Fill(theme.ColorBG)
		disp.SendFull(blank.Pixels[:])
	}

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
