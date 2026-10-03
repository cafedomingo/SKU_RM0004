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

// scrubRows is the height of the band resent each tick to repair silent
// mis-latches; see internal/st7735/README.md.
const scrubRows = 4

// panelSync tracks what the panel shows and keeps it in step with a back buffer.
type panelSync struct {
	front  st7735.Framebuffer
	scrubY int
}

// drawChanged sends the regions that differ from the front buffer, then
// resends the next scrub band.
func (p *panelSync) drawChanged(disp st7735.Display, back *st7735.Framebuffer) {
	if disp == nil {
		return
	}
	for _, r := range st7735.DiffRegions(&p.front, back) {
		disp.SendRegion(r, back)
	}
	h := min(scrubRows, st7735.Height-p.scrubY)
	disp.SendRegion(st7735.Region{X: 0, Y: p.scrubY, W: st7735.Width, H: h}, back)
	p.scrubY = (p.scrubY + h) % st7735.Height
	p.front = *back
}

// drawAll sends the entire back buffer to the display.
func drawAll(disp st7735.Display, back *st7735.Framebuffer) {
	if disp == nil {
		return
	}
	disp.SendFull(back)
}
