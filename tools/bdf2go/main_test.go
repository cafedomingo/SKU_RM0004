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
	wanted := map[rune]bool{32: true, 33: true, 63: true}
	glyphs, err := ParseBDF(strings.NewReader(testBDF), wanted)
	if err != nil {
		t.Fatal(err)
	}

	if len(glyphs) != 3 {
		t.Fatalf("got %d glyphs, want 3", len(glyphs))
	}

	// Check space is all zeros
	for _, g := range glyphs {
		if g.Encoding == ' ' {
			for i, b := range g.Bitmap {
				if b != 0 {
					t.Errorf("space byte %d = %#02x, want 0x00", i, b)
				}
			}
			if len(g.Bitmap) != 16 {
				t.Errorf("space bitmap len = %d, want 16", len(g.Bitmap))
			}
		}
	}

	// Check '!' has 0x18 at offset 3
	for _, g := range glyphs {
		if g.Encoding == '!' {
			if len(g.Bitmap) != 16 {
				t.Fatalf("'!' bitmap len = %d, want 16", len(g.Bitmap))
			}
			if g.Bitmap[3] != 0x18 {
				t.Errorf("'!' byte 3 = %#02x, want 0x18", g.Bitmap[3])
			}
		}
	}
}

func TestParseBDFFiltersUnwanted(t *testing.T) {
	// Only request space
	wanted := map[rune]bool{32: true}
	glyphs, err := ParseBDF(strings.NewReader(testBDF), wanted)
	if err != nil {
		t.Fatal(err)
	}
	if len(glyphs) != 1 {
		t.Fatalf("got %d glyphs, want 1", len(glyphs))
	}
	if glyphs[0].Encoding != ' ' {
		t.Errorf("got encoding %d, want 32", glyphs[0].Encoding)
	}
}
