package st7735

// Region describes a dirty rectangle of the display.
type Region struct {
	X int // start column
	Y int // start row
	W int // number of columns
	H int // number of rows
}

// splitGapMinPixels is the minimum clean-gap area (gap width x strip height,
// in pixels) worth splitting a strip into two regions. Each extra region
// costs ~30 bytes of command traffic on the wire; a gap is only skipped when
// not sending it saves more than that (one pixel = 2 bytes).
const splitGapMinPixels = 32

// DiffRegions compares two framebuffers and returns coalesced dirty
// rectangles. Consecutive dirty rows form a strip; within each strip the
// dirty x-extent is trimmed, and clean vertical gaps large enough to pay for
// the extra region overhead split the strip into side-by-side regions.
func DiffRegions(front, back *Framebuffer) []Region {
	var regions []Region
	dirtyStart := -1

	for y := 0; y < Height; y++ {
		rowOffset := y * Width
		dirty := rowDirty(front.Pixels[rowOffset:rowOffset+Width], back.Pixels[rowOffset:rowOffset+Width])

		if dirty && dirtyStart == -1 {
			dirtyStart = y
		} else if !dirty && dirtyStart != -1 {
			regions = appendStripRegions(regions, front, back, dirtyStart, y-dirtyStart)
			dirtyStart = -1
		}
	}

	if dirtyStart != -1 {
		regions = appendStripRegions(regions, front, back, dirtyStart, Height-dirtyStart)
	}

	return regions
}

// appendStripRegions splits the strip of rows [y, y+h) into regions trimmed
// to the dirty columns, keeping clean gaps only when they are too small to
// be worth a separate region.
func appendStripRegions(regions []Region, front, back *Framebuffer, y, h int) []Region {
	var colDirty [Width]bool
	for row := y; row < y+h; row++ {
		offset := row * Width
		for x := 0; x < Width; x++ {
			if front.Pixels[offset+x] != back.Pixels[offset+x] {
				colDirty[x] = true
			}
		}
	}

	runStart := -1 // start of the current region's columns
	runEnd := -1   // last dirty column seen
	for x := 0; x < Width; x++ {
		if !colDirty[x] {
			continue
		}
		if runStart == -1 {
			runStart = x
		} else if (x-runEnd-1)*h >= splitGapMinPixels {
			regions = append(regions, Region{X: runStart, Y: y, W: runEnd - runStart + 1, H: h})
			runStart = x
		}
		runEnd = x
	}
	// A strip only exists because at least one row differed, so there is
	// always a final run to emit.
	return append(regions, Region{X: runStart, Y: y, W: runEnd - runStart + 1, H: h})
}

func rowDirty(a, b []uint16) bool {
	for i := range a {
		if a[i] != b[i] {
			return true
		}
	}
	return false
}
