package screen

import (
	"slices"
	"testing"

	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
)

// recordingDisplay records the regions sent to it.
type recordingDisplay struct {
	regions []st7735.Region
}

func (r *recordingDisplay) SendRegion(reg st7735.Region, _ *st7735.Framebuffer) {
	r.regions = append(r.regions, reg)
}

func (r *recordingDisplay) SendFull(_ *st7735.Framebuffer) {
	r.regions = append(r.regions, st7735.Region{X: 0, Y: 0, W: st7735.Width, H: st7735.Height})
}

func (r *recordingDisplay) Close() error { return nil }

func scrubBand(y int) st7735.Region {
	return st7735.Region{X: 0, Y: y, W: st7735.Width, H: scrubRows}
}

// An unchanged frame still sends the next scrub band.
func TestDrawChangedScrubsWhenClean(t *testing.T) {
	var p panelSync
	var back st7735.Framebuffer
	disp := &recordingDisplay{}

	p.drawChanged(disp, &back)
	p.drawChanged(disp, &back)

	want := []st7735.Region{scrubBand(0), scrubBand(scrubRows)}
	if !slices.Equal(disp.regions, want) {
		t.Fatalf("regions = %+v, want %+v", disp.regions, want)
	}
}

// Diffed regions go out before the scrub band, and the front buffer catches up.
func TestDrawChangedSendsDiffThenScrub(t *testing.T) {
	var p panelSync
	var back st7735.Framebuffer
	back.SetPixel(10, 40, 0xFFFF)
	disp := &recordingDisplay{}

	p.drawChanged(disp, &back)

	want := []st7735.Region{{X: 10, Y: 40, W: 1, H: 1}, scrubBand(0)}
	if !slices.Equal(disp.regions, want) {
		t.Fatalf("regions = %+v, want %+v", disp.regions, want)
	}
	if p.front != back {
		t.Fatal("front buffer not updated to back")
	}
}

// The scrub covers every row once per cycle, then wraps.
func TestDrawChangedScrubCycle(t *testing.T) {
	var p panelSync
	var back st7735.Framebuffer
	disp := &recordingDisplay{}

	for range st7735.Height/scrubRows + 1 {
		p.drawChanged(disp, &back)
	}

	var covered [st7735.Height]bool
	for _, r := range disp.regions[:len(disp.regions)-1] {
		for y := r.Y; y < r.Y+r.H; y++ {
			covered[y] = true
		}
	}
	for y, ok := range covered {
		if !ok {
			t.Errorf("row %d never scrubbed", y)
		}
	}
	if last := disp.regions[len(disp.regions)-1]; last != scrubBand(0) {
		t.Errorf("scrub did not wrap: got %+v", last)
	}
}

func TestDrawChangedNilDisplay(t *testing.T) {
	var p panelSync
	var back st7735.Framebuffer
	back.SetPixel(0, 0, 0xFFFF)
	p.drawChanged(nil, &back)
	if p.scrubY != 0 {
		t.Errorf("scrubY = %d, want 0 with no display", p.scrubY)
	}
}
