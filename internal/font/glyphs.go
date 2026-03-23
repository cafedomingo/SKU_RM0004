package font

// AddGlyph registers a custom bitmap glyph for a rune.
// Used as fallback when Spleen doesn't cover a needed symbol.
func (f *Font) AddGlyph(r rune, data []byte) {
	f.Glyphs[r] = data
}
