package st7735

// Region describes a dirty rectangle of the display.
type Region struct {
	X int // start column
	Y int // start row
	W int // number of columns
	H int // number of rows
}

// splitGapMinPixels is the minimum clean-gap area (gap width x strip height)
// worth splitting a strip: smaller gaps cost less to resend than an extra
// region's command overhead.
const splitGapMinPixels = 32

// DiffRegions compares two framebuffers and returns coalesced dirty
// rectangles: strips of dirty rows, trimmed and split by dirty column runs.
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

// appendStripRegions appends the strip of rows [y, y+h) as regions trimmed
// to its dirty columns, splitting at gaps of at least splitGapMinPixels.
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
	// The strip has at least one dirty row, so a final run always exists.
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
