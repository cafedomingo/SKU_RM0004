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

func TestDiffSingleRow(t *testing.T) {
	var front, back Framebuffer
	back.SetPixel(0, 40, 0xFFFF)
	regions := DiffRegions(&front, &back)
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %v", regions)
	}
	if regions[0].Y != 40 || regions[0].H != 1 {
		t.Fatalf("expected Y=40 H=1, got %+v", regions[0])
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
	if regions[0].Y != 10 || regions[0].H != 3 {
		t.Fatalf("expected Y=10 H=3, got %+v", regions[0])
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
	if regions[0].Y != 0 || regions[0].H != Height {
		t.Fatalf("expected Y=0 H=%d, got %+v", Height, regions[0])
	}
}
