package st7735

import (
	"slices"
	"testing"
)

// assertRegions fails unless DiffRegions(front, back) returns exactly want.
func assertRegions(t *testing.T, front, back *Framebuffer, want ...Region) {
	t.Helper()
	got := DiffRegions(front, back)
	if !slices.Equal(got, want) {
		t.Fatalf("DiffRegions = %+v, want %+v", got, want)
	}
}

func TestDiffIdentical(t *testing.T) {
	var front, back Framebuffer
	front.Fill(0x1234)
	back.Fill(0x1234)
	assertRegions(t, &front, &back)
}

func TestDiffSinglePixel(t *testing.T) {
	var front, back Framebuffer
	back.SetPixel(25, 40, 0xFFFF)
	assertRegions(t, &front, &back, Region{X: 25, Y: 40, W: 1, H: 1})
}

func TestDiffTrimsRowExtent(t *testing.T) {
	var front, back Framebuffer
	back.Rect(10, 40, 11, 1, 0xFFFF)
	assertRegions(t, &front, &back, Region{X: 10, Y: 40, W: 11, H: 1})
}

func TestDiffCoalesceAdjacentRows(t *testing.T) {
	var front, back Framebuffer
	back.Rect(0, 10, 1, 3, 0xFFFF)
	assertRegions(t, &front, &back, Region{X: 0, Y: 10, W: 1, H: 3})
}

func TestDiffNonAdjacentRows(t *testing.T) {
	var front, back Framebuffer
	back.SetPixel(0, 5, 0xFFFF)
	back.SetPixel(0, 50, 0xFFFF)
	assertRegions(t, &front, &back,
		Region{X: 0, Y: 5, W: 1, H: 1},
		Region{X: 0, Y: 50, W: 1, H: 1})
}

func TestDiffFullScreen(t *testing.T) {
	var front, back Framebuffer
	front.Fill(0x0000)
	back.Fill(0xFFFF)
	assertRegions(t, &front, &back, Region{X: 0, Y: 0, W: Width, H: Height})
}

// A strip dirty on both sides of the two-column gutter must split in two.
func TestDiffSplitsColumns(t *testing.T) {
	var front, back Framebuffer
	back.Rect(0, 37, 78, 18, 0xF800)
	back.Rect(82, 37, 78, 18, 0x001F)
	assertRegions(t, &front, &back,
		Region{X: 0, Y: 37, W: 78, H: 18},
		Region{X: 82, Y: 37, W: 78, H: 18})
}

// A gap too small to pay for another region's overhead must not split.
func TestDiffKeepsSmallGap(t *testing.T) {
	var front, back Framebuffer
	// Single row, 10px gap: area below splitGapMinPixels.
	back.Rect(0, 40, 21, 1, 0xFFFF)
	back.Rect(31, 40, 20, 1, 0xFFFF)
	assertRegions(t, &front, &back, Region{X: 0, Y: 40, W: 51, H: 1})
}

// Rows with different dirty extents coalesce into their union.
func TestDiffUnionExtents(t *testing.T) {
	var front, back Framebuffer
	back.Rect(5, 10, 5, 1, 0xFFFF)
	back.Rect(8, 11, 7, 1, 0xFFFF)
	assertRegions(t, &front, &back, Region{X: 5, Y: 10, W: 10, H: 2})
}
