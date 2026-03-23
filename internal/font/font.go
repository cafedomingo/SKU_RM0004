package font

// Font holds a bitmap font with fixed-width glyphs.
type Font struct {
	Width  int
	Height int
	Glyphs map[rune][]byte // height bytes per glyph, MSB = leftmost pixel
}

func init() {
	Spleen6x12.AddArrowGlyphs()
}

// Glyph returns the bitmap data for a rune, or '?' if not found.
func (f *Font) Glyph(r rune) []byte {
	if g, ok := f.Glyphs[r]; ok {
		return g
	}
	if g, ok := f.Glyphs['?']; ok {
		return g
	}
	return nil
}
