package st7735

// Region describes a rectangle of the display.
type Region struct {
	X int // start column
	Y int // start row
	W int // number of columns
	H int // number of rows
}

// DiffRegions compares two framebuffers row-by-row and returns
// coalesced regions where they differ. All regions are full-width (Width=160).
func DiffRegions(front, back *Framebuffer) []Region {
	var regions []Region
	dirtyStart := -1

	for y := 0; y < Height; y++ {
		rowOffset := y * Width
		dirty := rowDirty(front.Pixels[rowOffset:rowOffset+Width], back.Pixels[rowOffset:rowOffset+Width])

		if dirty && dirtyStart == -1 {
			dirtyStart = y
		} else if !dirty && dirtyStart != -1 {
			regions = append(regions, Region{X: 0, Y: dirtyStart, W: Width, H: y - dirtyStart})
			dirtyStart = -1
		}
	}

	if dirtyStart != -1 {
		regions = append(regions, Region{X: 0, Y: dirtyStart, W: Width, H: Height - dirtyStart})
	}

	return regions
}

func rowDirty(a, b []uint16) bool {
	for i := range a {
		if a[i] != b[i] {
			return true
		}
	}
	return false
}
