package st7735

import "testing"

func TestDiffIdentical(t *testing.T) {
	var front, back Framebuffer
	front.Fill(0x1234)
	back.Fill(0x1234)
	regions := DiffRegions(&front, &back)
	if len(regions) != 0 {
		t.Fatalf("expected no regions, got %v", regions)
	}
}

func TestDiffSinglePixel(t *testing.T) {
	var front, back Framebuffer
	back.SetPixel(25, 40, 0xFFFF)
	regions := DiffRegions(&front, &back)
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %v", regions)
	}
	want := Region{X: 25, Y: 40, W: 1, H: 1}
	if regions[0] != want {
		t.Fatalf("expected %+v, got %+v", want, regions[0])
	}
}

func TestDiffTrimsRowExtent(t *testing.T) {
	var front, back Framebuffer
	for x := 10; x <= 20; x++ {
		back.SetPixel(x, 40, 0xFFFF)
	}
	regions := DiffRegions(&front, &back)
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %v", regions)
	}
	want := Region{X: 10, Y: 40, W: 11, H: 1}
	if regions[0] != want {
		t.Fatalf("expected %+v, got %+v", want, regions[0])
	}
}

func TestDiffCoalesceAdjacentRows(t *testing.T) {
	var front, back Framebuffer
	back.SetPixel(0, 10, 0xFFFF)
	back.SetPixel(0, 11, 0xFFFF)
	back.SetPixel(0, 12, 0xFFFF)
	regions := DiffRegions(&front, &back)
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %v", regions)
	}
	want := Region{X: 0, Y: 10, W: 1, H: 3}
	if regions[0] != want {
		t.Fatalf("expected %+v, got %+v", want, regions[0])
	}
}

func TestDiffNonAdjacentRows(t *testing.T) {
	var front, back Framebuffer
	back.SetPixel(0, 5, 0xFFFF)
	back.SetPixel(0, 50, 0xFFFF)
	regions := DiffRegions(&front, &back)
	if len(regions) != 2 {
		t.Fatalf("expected 2 regions, got %v", regions)
	}
	if regions[0].Y != 5 || regions[0].H != 1 {
		t.Fatalf("expected first region Y=5 H=1, got %+v", regions[0])
	}
	if regions[1].Y != 50 || regions[1].H != 1 {
		t.Fatalf("expected second region Y=50 H=1, got %+v", regions[1])
	}
}

func TestDiffFullScreen(t *testing.T) {
	var front, back Framebuffer
	front.Fill(0x0000)
	back.Fill(0xFFFF)
	regions := DiffRegions(&front, &back)
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %v", regions)
	}
	want := Region{X: 0, Y: 0, W: Width, H: Height}
	if regions[0] != want {
		t.Fatalf("expected %+v, got %+v", want, regions[0])
	}
}

// TestDiffSplitsColumns mirrors the two-column screen layouts: a strip dirty
// on both sides of the x=78..81 gutter must split into two regions when the
// gap is worth the extra region overhead.
func TestDiffSplitsColumns(t *testing.T) {
	var front, back Framebuffer
	for y := 37; y <= 54; y++ {
		for x := 0; x <= 77; x++ {
			back.SetPixel(x, y, 0xF800)
		}
		for x := 82; x < Width; x++ {
			back.SetPixel(x, y, 0x001F)
		}
	}
	regions := DiffRegions(&front, &back)
	if len(regions) != 2 {
		t.Fatalf("expected 2 regions, got %v", regions)
	}
	wantLeft := Region{X: 0, Y: 37, W: 78, H: 18}
	wantRight := Region{X: 82, Y: 37, W: 78, H: 18}
	if regions[0] != wantLeft {
		t.Fatalf("expected left %+v, got %+v", wantLeft, regions[0])
	}
	if regions[1] != wantRight {
		t.Fatalf("expected right %+v, got %+v", wantRight, regions[1])
	}
}

// TestDiffKeepsSmallGap verifies a gap too small to pay for another region's
// command overhead does not split the strip.
func TestDiffKeepsSmallGap(t *testing.T) {
	var front, back Framebuffer
	// Single row, 10px gap: gap area 10 px < splitGapMinPixels.
	for x := 0; x <= 20; x++ {
		back.SetPixel(x, 40, 0xFFFF)
	}
	for x := 31; x <= 50; x++ {
		back.SetPixel(x, 40, 0xFFFF)
	}
	regions := DiffRegions(&front, &back)
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %v", regions)
	}
	want := Region{X: 0, Y: 40, W: 51, H: 1}
	if regions[0] != want {
		t.Fatalf("expected %+v, got %+v", want, regions[0])
	}
}

// TestDiffUnionExtents verifies rows with different dirty extents coalesce
// into runs covering their union.
func TestDiffUnionExtents(t *testing.T) {
	var front, back Framebuffer
	for x := 5; x <= 9; x++ {
		back.SetPixel(x, 10, 0xFFFF)
	}
	for x := 8; x <= 14; x++ {
		back.SetPixel(x, 11, 0xFFFF)
	}
	regions := DiffRegions(&front, &back)
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %v", regions)
	}
	want := Region{X: 5, Y: 10, W: 10, H: 2}
	if regions[0] != want {
		t.Fatalf("expected %+v, got %+v", want, regions[0])
	}
}
