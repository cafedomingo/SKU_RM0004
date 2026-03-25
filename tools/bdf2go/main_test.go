package main

import (
	"strings"
	"testing"
)

const testBDF = `STARTFONT 2.1
FONT -Misc-Test-Medium-R-Normal--16-160-72-72-C-80-ISO10646-1
SIZE 16 72 72
FONTBOUNDINGBOX 8 16 0 -4
STARTPROPERTIES 2
FONT_ASCENT 12
FONT_DESCENT 4
ENDPROPERTIES
CHARS 3
STARTCHAR space
ENCODING 32
SWIDTH 500 0
DWIDTH 8 0
BBX 8 16 0 -4
BITMAP
00
00
00
00
00
00
00
00
00
00
00
00
00
00
00
00
ENDCHAR
STARTCHAR exclam
ENCODING 33
SWIDTH 500 0
DWIDTH 8 0
BBX 8 16 0 -4
BITMAP
00
00
00
18
18
18
18
18
18
00
18
18
00
00
00
00
ENDCHAR
STARTCHAR question
ENCODING 63
SWIDTH 500 0
DWIDTH 8 0
BBX 8 16 0 -4
BITMAP
00
00
3c
66
66
06
0c
18
18
00
18
18
00
00
00
00
ENDCHAR
ENDFONT
`

func TestParseBDF(t *testing.T) {
	wanted := map[rune]bool{' ': true, '!': true, '?': true}
	glyphs, err := ParseBDF(strings.NewReader(testBDF), wanted)
	if err != nil {
		t.Fatal(err)
	}

	if len(glyphs) != 3 {
		t.Fatalf("got %d glyphs, want 3", len(glyphs))
	}

	byRune := make(map[rune]BDFGlyph, len(glyphs))
	for _, g := range glyphs {
		byRune[g.Encoding] = g
	}

	space := byRune[' ']
	if len(space.Bitmap) != 16 {
		t.Fatalf("space bitmap len = %d, want 16", len(space.Bitmap))
	}
	for i, b := range space.Bitmap {
		if b != 0 {
			t.Errorf("space byte %d = %#02x, want 0x00", i, b)
		}
	}

	exclam := byRune['!']
	if len(exclam.Bitmap) != 16 {
		t.Fatalf("'!' bitmap len = %d, want 16", len(exclam.Bitmap))
	}
	if exclam.Bitmap[3] != 0x18 {
		t.Errorf("'!' byte 3 = %#02x, want 0x18", exclam.Bitmap[3])
	}
}

func TestParseBDFFiltersUnwanted(t *testing.T) {
	wanted := map[rune]bool{' ': true}
	glyphs, err := ParseBDF(strings.NewReader(testBDF), wanted)
	if err != nil {
		t.Fatal(err)
	}
	if len(glyphs) != 1 {
		t.Fatalf("got %d glyphs, want 1", len(glyphs))
	}
	if glyphs[0].Encoding != ' ' {
		t.Errorf("got encoding %q, want %q", glyphs[0].Encoding, ' ')
	}
}
