# Go Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the UCTRONICS SKU_RM0004 display firmware from C to idiomatic Go, maintaining all functionality with clean licensing.

**Architecture:** Layered design — renderers draw into an in-memory framebuffer, a diff engine detects changes, and a thin I2C driver sends only dirty regions to the display. System metrics come from gopsutil where possible, with direct reads for Pi-specific hardware.

**Tech Stack:** Go 1.22+, gopsutil v4, periph.io v3, log/slog, image/png (stdlib)

**Spec:** `docs/superpowers/specs/2026-03-22-go-rewrite-design.md`

---

## File Structure

```
SKU_RM0004/
├── go.mod
├── go.sum
├── LICENSE                                 # Clean MIT
├── THIRD_PARTY_LICENSES                    # Spleen BSD-2-Clause
├── README.md
├── install.sh                              # Updated for Go binary
├── .editorconfig
├── .gitattributes
├── .gitignore
├── .golangci.yml
│
├── cmd/
│   ├── display/main.go                     # Entry point, main loop, signal handling
│   └── screenshot/main.go                  # PNG doc generator with mock data
│
├── internal/
│   ├── config/
│   │   ├── config.go                       # Config struct, Load(), mtime caching
│   │   └── config_test.go                  # Missing file, partial, invalid, mtime
│   ├── font/
│   │   ├── font.go                         # Font type definition, glyph lookup by rune
│   │   ├── font_test.go                    # Glyph lookup, bounds, missing runes
│   │   ├── glyphs.go                       # Custom glyph fallback infrastructure
│   │   ├── spleen.go                       # Generated: Spleen 8x16 font data
│   │   └── generate.go                     # //go:generate directive
│   ├── format/
│   │   ├── format.go                       # Rate, Freq, Uptime, Temp, APTBadge
│   │   └── format_test.go                  # Boundary values, conversions, edges
│   ├── screen/
│   │   ├── dashboard.go                    # Single-page status renderer
│   │   ├── dashboard_test.go               # Integration: mock collector → pixel checks
│   │   ├── diagnostic.go                   # Multi-page detail renderer
│   │   ├── diagnostic_test.go              # Integration: page cycling, data rows
│   │   ├── sparkline.go                    # Scrolling history renderer
│   │   └── sparkline_test.go               # Integration: history, ticker, blocks
│   ├── sysinfo/
│   │   ├── sysinfo.go                      # Collector interface + types (CPUFreq, etc.)
│   │   ├── collector.go                    # Live implementation: gopsutil + direct reads
│   │   ├── collector_test.go               # Tests with mock /proc files
│   │   ├── mock.go                         # MockCollector for tests + screenshot
│   │   ├── pi.go                           # Throttle (vcio ioctl), DietPi, CPU freq
│   │   └── pi_test.go                      # Tests for Pi-specific reads
│   ├── st7735/
│   │   ├── driver.go                       # Display interface, I2C implementation
│   │   ├── framebuffer.go                  # 160x80 RGB565 buffer, drawing methods
│   │   ├── framebuffer_test.go             # Pixel-by-pixel drawing validation
│   │   ├── diff.go                         # Double-buffer diff + region coalescing
│   │   ├── diff_test.go                    # Diff detection, coalescing tests
│   │   └── README.md                       # Hardware reference (moved from hardware/)
│   └── theme/
│       ├── theme.go                        # Colors, thresholds, ThresholdColor, TempRampColor
│       └── theme_test.go                   # Threshold boundaries, ramp interpolation
│
├── tools/
│   └── bdf2go/
│       ├── main.go                         # BDF parser → Go source generator
│       └── main_test.go                    # Parse known BDF input, verify output
│
├── .github/workflows/build.yml             # CI: lint, test, build, screenshot, release
│
└── docs/
    ├── dashboard.png                       # Generated
    ├── diagnostic.png                      # Generated
    └── sparkline.png                       # Generated
```

---

## Task 0: Project Scaffolding

**Files:**
- Create: `go.mod`, `LICENSE`, `THIRD_PARTY_LICENSES`, `.editorconfig`, `.gitattributes`, `.gitignore`, `.golangci.yml`

This task sets up the Go module, removes all C source files, and establishes the clean break.

- [ ] **Step 1: Create a new branch for the rewrite**

```bash
git checkout -b go-rewrite
```

- [ ] **Step 2: Preserve hardware README before deleting C source**

```bash
mkdir -p internal/st7735
cp hardware/st7735/README.md internal/st7735/README.md
```

- [ ] **Step 3: Remove all C source files, build artifacts, and old license**

Remove these directories and files entirely:
- `hardware/` (all C source — README already moved to `internal/st7735/`)
- `project/` (all C source)
- `test/` (all C tests)
- `tools/` (C screenshot tool and mocks)
- `obj/`
- `Makefile`
- `LICENSE`
- `.clang-format`

Keep:
- `.github/` (will be updated in Task 14)
- `docs/` (screenshots will be regenerated)
- `install.sh` (will be updated in Task 15)
- `README.md` (will be rewritten in Task 16)

- [ ] **Step 4: Initialize Go module**

```bash
go mod init github.com/cafedomingo/SKU_RM0004
```

- [ ] **Step 5: Create LICENSE**

Create `LICENSE` with a clean MIT license:

```
MIT License

Copyright (c) 2025 Patrick Sunday

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 6: Create THIRD_PARTY_LICENSES**

Include the Spleen font BSD 2-Clause license text. Fetch from https://github.com/fcambus/spleen/blob/master/LICENSE.

- [ ] **Step 7: Create .editorconfig**

```ini
root = true

[*]
end_of_line = lf
insert_final_newline = true
charset = utf-8

[*.go]
indent_style = tab
indent_size = 4

[*.{yml,yaml,json,md}]
indent_style = space
indent_size = 2

[Makefile]
indent_style = tab
```

- [ ] **Step 8: Create .gitattributes**

```
* text=auto
*.go text eol=lf
internal/font/spleen.go linguist-generated
```

- [ ] **Step 9: Create .gitignore**

```
/display
/screenshot
*.png
!docs/*.png
```

- [ ] **Step 10: Create .golangci.yml**

```yaml
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - ineffassign
    - goconst
    - gofmt
    - goimports

linters-settings:
  goimports:
    local-prefixes: github.com/cafedomingo/SKU_RM0004
```

- [ ] **Step 11: Create directory structure**

```bash
mkdir -p cmd/display cmd/screenshot
mkdir -p internal/config internal/font internal/format
mkdir -p internal/screen internal/sysinfo internal/st7735
mkdir -p internal/theme
mkdir -p tools/bdf2go
```

- [ ] **Step 12: Commit**

```bash
git add -A
git commit -m "chore: scaffold Go project, remove C codebase"
```

---

## Task 1: Theme Package

**Files:**
- Create: `internal/theme/theme.go`
- Test: `internal/theme/theme_test.go`

The theme package has no dependencies on other internal packages, making it an ideal starting point.

- [ ] **Step 1: Write theme tests**

Create `internal/theme/theme_test.go` with table-driven tests:

```go
package theme

import "testing"

func TestThresholdColor(t *testing.T) {
    tests := []struct {
        name     string
        value    float64
        warn     float64
        crit     float64
        expected uint16
    }{
        {"below warn", 30, 60, 80, ColorOK},
        {"at warn", 60, 60, 80, ColorWarn},
        {"between warn and crit", 70, 60, 80, ColorWarn},
        {"at crit", 80, 60, 80, ColorCrit},
        {"above crit", 95, 60, 80, ColorCrit},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ThresholdColor(tt.value, tt.warn, tt.crit)
            if got != tt.expected {
                t.Errorf("ThresholdColor(%v, %v, %v) = 0x%04X, want 0x%04X",
                    tt.value, tt.warn, tt.crit, got, tt.expected)
            }
        })
    }
}

func TestTempRampColor(t *testing.T) {
    // DietPi thresholds: <40 cyan, 40 green, 50 yellow, 60 orange, 70+ red
    tests := []struct {
        name string
        temp float64
        want uint16
    }{
        {"freezing", 10, TempCyan},
        {"cool boundary", 39, TempCyan},   // lerp near cyan
        {"optimal", 40, TempGreen},
        {"warm", 50, TempYellow},
        {"hot", 60, TempOrange},
        {"critical", 70, TempRed},
        {"above critical", 85, TempRed},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := TempRampColor(tt.temp)
            // At exact boundaries, color should match. Between boundaries, it's interpolated.
            // For boundary tests, exact match. For in-between, just check non-zero.
            if tt.temp == 40 || tt.temp == 50 || tt.temp == 60 || tt.temp >= 70 {
                if got != tt.want {
                    t.Errorf("TempRampColor(%v) = 0x%04X, want 0x%04X", tt.temp, got, tt.want)
                }
            }
        })
    }
}

func TestNetThresholds(t *testing.T) {
    warn, crit := NetThresholds(1000) // 1Gbps
    if warn != 50_000_000 {           // 40% of 125MB/s
        t.Errorf("NetThresholds(1000) warn = %d, want 50000000", warn)
    }
    if crit != 100_000_000 { // 80% of 125MB/s
        t.Errorf("NetThresholds(1000) crit = %d, want 100000000", crit)
    }
}

func TestNetThresholdsFallback(t *testing.T) {
    warn, crit := NetThresholds(0) // unknown speed, fallback to 100Mbps
    if warn != 5_000_000 {         // 40% of 12.5MB/s
        t.Errorf("NetThresholds(0) warn = %d, want 5000000", warn)
    }
    if crit != 10_000_000 { // 80% of 12.5MB/s
        t.Errorf("NetThresholds(0) crit = %d, want 10000000", crit)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/pq/src/SKU_RM0004 && go test ./internal/theme/
```

Expected: compilation errors (package doesn't exist yet).

- [ ] **Step 3: Implement theme.go**

Create `internal/theme/theme.go`:

```go
package theme

// RGB565 color constants
const (
    ColorBG    uint16 = 0x0000 // black
    ColorFG    uint16 = 0xFFFF // white
    ColorMuted uint16 = 0x8410 // gray
    ColorSep   uint16 = 0x39E7 // dark gray
    ColorIP    uint16 = 0x7D5F // light blue
    ColorAlert uint16 = 0xFC0B // orange
    ColorOK    uint16 = 0x45E4 // green
    ColorWarn  uint16 = 0xCDE0 // yellow
    ColorCrit  uint16 = 0xFC0B // red-orange

    // Temperature ramp (matches DietPi)
    TempCyan   uint16 = 0x2D7F // <40C cool
    TempGreen  uint16 = 0x069A // 40C optimal
    TempYellow uint16 = 0xCDE0 // 50C warm
    TempOrange uint16 = 0xFC0B // 60C hot
    TempRed    uint16 = 0xF800 // 70C+ critical
)

// Thresholds
const (
    CPUWarn  = 60.0
    CPUCrit  = 80.0
    RAMWarn  = 60.0
    RAMCrit  = 80.0
    DiskWarn = 70.0
    DiskCrit = 90.0
    DIOWarn  = 25_000_000.0 // 25 MB/s
    DIOCrit  = 75_000_000.0 // 75 MB/s
    APTCrit  = 10
)

// ThresholdColor returns ok/warn/crit color based on value vs thresholds.
func ThresholdColor(value, warn, crit float64) uint16 {
    if value >= crit {
        return ColorCrit
    }
    if value >= warn {
        return ColorWarn
    }
    return ColorOK
}

// TempRampColor returns an interpolated color for the given temperature in Celsius.
// Matches DietPi breakpoints: <40 cyan, 40 green, 50 yellow, 60 orange, 70+ red.
// Hard step at 40 (cyan→green), interpolated between other breakpoints.
func TempRampColor(celsius float64) uint16 {
    if celsius < 40 {
        return TempCyan
    }
    ramp := []struct {
        temp  float64
        color uint16
    }{
        {40, TempGreen},
        {50, TempYellow},
        {60, TempOrange},
        {70, TempRed},
    }

    for i := 1; i < len(ramp); i++ {
        if celsius <= ramp[i].temp {
            t := (celsius - ramp[i-1].temp) / (ramp[i].temp - ramp[i-1].temp)
            return lerpColor(ramp[i-1].color, ramp[i].color, float32(t))
        }
    }
    return ramp[len(ramp)-1].color
}

// NetThresholds returns warn/crit byte-rate thresholds for a given link speed in Mbps.
// Falls back to 100 Mbps if speed is 0 (unknown).
func NetThresholds(linkSpeedMbps int) (warn, crit uint64) {
    if linkSpeedMbps <= 0 {
        linkSpeedMbps = 100
    }
    maxBytesPerSec := uint64(linkSpeedMbps) * 1_000_000 / 8
    return maxBytesPerSec * 40 / 100, maxBytesPerSec * 80 / 100
}

func lerpColor(a, b uint16, t float32) uint16 {
    ar, ag, ab := (a>>11)&0x1F, (a>>5)&0x3F, a&0x1F
    br, bg, bb := (b>>11)&0x1F, (b>>5)&0x3F, b&0x1F
    r := float32(ar) + float32(int(br)-int(ar))*t
    g := float32(ag) + float32(int(bg)-int(ag))*t
    bv := float32(ab) + float32(int(bb)-int(ab))*t
    return uint16(r)<<11 | uint16(g)<<5 | uint16(bv)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/theme/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/theme/
git commit -m "feat: add theme package with colors, thresholds, and temp ramp"
```

---

## Task 2: Format Package

**Files:**
- Create: `internal/format/format.go`
- Test: `internal/format/format_test.go`

No internal dependencies. Port all formatting functions from `project/format.c`.

- [ ] **Step 1: Write format tests**

Port all tests from `test/test_format.c` to Go table-driven tests. Include:
- `TestRate`: 0, 1, 500, KB-1, KB, 5KB, 10KB, MB-1, MB, 10MB, 100MB
- `TestFreq`: 0, 600, 999, 1000, 1800, 2400
- `TestUptime`: 0s, 59s, 60s, 120s, 3600s, 3700s, 86400s, 90061s
- `TestTemp`: 0/C, 52/C, 100/C, 52/F, 0/F, 100/F — use `°` character
- `TestCelsiusToF`: 0→32, 100→212, 50→122
- `TestAPTBadge`: 0→"", -1→"", 1→"^1", 3→"^3", 99→"^99", 100→"^99"

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/format/
```

- [ ] **Step 3: Implement format.go**

Functions:
- `Rate(bytes uint64) string`
- `Freq(mhz uint16) string`
- `Uptime(d time.Duration) string`
- `Temp(celsius float64, unit string) string` — uses `°` symbol
- `CelsiusToF(c float64) float64`
- `APTBadge(count int) string` — returns "" for count <= 0, capped at 99

Constants: `KB = 1024`, `MB = 1024 * 1024`

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/format/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/format/
git commit -m "feat: add format package for metric string formatting"
```

---

## Task 3: Config Package

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write config tests**

Tests using `t.TempDir()` to create temporary config files:
- `TestDefaults`: no file → dashboard, 5s, "C"
- `TestFullConfig`: all three keys present
- `TestPartialConfig`: only `screen=sparkline` → defaults for refresh and temp_unit
- `TestInvalidScreen`: `screen=invalid` → falls back to "dashboard"
- `TestRefreshBounds`: `refresh=1` → clamps to 2, `refresh=50` → clamps to 30
- `TestComments`: lines starting with `#` are ignored
- `TestEmptyFile`: empty file → all defaults
- `TestTempUnit`: `temp_unit=F` → "F"
- `TestMtimeCaching`: load twice without change → second load uses cache
- `TestMtimeReload`: modify file between loads → second load picks up changes

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/config/
```

- [ ] **Step 3: Implement config.go**

```go
package config

import (
    "bufio"
    "log/slog"
    "os"
    "strconv"
    "strings"
    "time"
)

const (
    DefaultPath    = "/etc/uctronics-display.conf"
    DefaultScreen  = "dashboard"
    DefaultRefresh = 5 * time.Second
    DefaultTempUnit = "C"

    MinRefresh = 2 * time.Second
    MaxRefresh = 30 * time.Second
)

type Config struct {
    Screen   string
    Refresh  time.Duration
    TempUnit string
}

type Loader struct {
    path      string
    logger    *slog.Logger
    lastMtime time.Time
    cached    Config
}

func NewLoader(path string, logger *slog.Logger) *Loader { ... }
func (l *Loader) Load() Config { ... }
```

`Load()`:
1. Check mtime via `os.Stat`. If unchanged, return cached.
2. If file doesn't exist, return defaults.
3. Parse line by line: split on `=`, trim, validate.
4. Log changes when values differ from previous.
5. Cache result + mtime.

Valid screens: "dashboard", "diagnostic", "sparkline". Invalid → default, log warning.
Refresh: parse int, clamp to 2-30, multiply by `time.Second`.
TempUnit: "C" or "F". Invalid → "C", log warning.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/config/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package for runtime INI config loading"
```

---

## Task 4: Font Package + BDF Converter

**Files:**
- Create: `internal/font/font.go`, `internal/font/font_test.go`, `internal/font/generate.go`
- Create: `tools/bdf2go/main.go`, `tools/bdf2go/main_test.go`
- Generate: `internal/font/spleen.go`

This task has two parts: the font type definition and the BDF converter tool.

### Part A: Font type and tests

- [ ] **Step 1: Write font type tests**

Create `internal/font/font_test.go`:
- `TestGlyphLookupASCII`: printable ASCII chars (32-126) return non-nil glyph data
- `TestGlyphLookupUnicode`: `°` (U+00B0), `◆` (U+25C6), `▲` (U+25B2), `▼` (U+25BC), `█` (U+2588) return non-nil
- `TestGlyphLookupMissing`: rune 0, rune 999999 return the '?' replacement glyph
- `TestFontDimensions`: width=8, height=16 for Spleen8x16

- [ ] **Step 2: Implement font.go**

```go
package font

// Font holds a bitmap font with fixed-width glyphs.
type Font struct {
    Width  int
    Height int
    glyphs map[rune][]byte // height bytes per glyph, MSB = leftmost pixel
}

// Glyph returns the bitmap data for a rune, or '?' if not found.
func (f *Font) Glyph(r rune) []byte { ... }
```

Note: Spleen 8x16 glyphs are 8 pixels wide, so each row is 1 byte (MSB = leftmost pixel). Each glyph is 16 bytes.

- [ ] **Step 3: Create glyphs.go**

Create `internal/font/glyphs.go` — custom glyph fallback infrastructure:

```go
package font

// AddGlyph registers a custom bitmap glyph for a rune.
// Used as fallback when Spleen doesn't cover a needed symbol.
func (f *Font) AddGlyph(r rune, data []byte) {
    f.glyphs[r] = data
}
```

This keeps the infrastructure available without defining any custom glyphs upfront — Spleen should cover our needs (◆, ▲, ▼, °, block elements).

- [ ] **Step 4: Commit font type**

```bash
git add internal/font/font.go internal/font/font_test.go internal/font/glyphs.go
git commit -m "feat: add font type definition with rune-based glyph lookup"
```

### Part B: BDF converter

- [ ] **Step 5: Write bdf2go tests**

Create `tools/bdf2go/main_test.go`:
- `TestParseBDF`: feed a minimal BDF snippet (2-3 characters), verify parsed glyph count and dimensions
- `TestGenerateGo`: verify the generated Go source compiles (check for syntax markers)

- [ ] **Step 6: Implement bdf2go**

Create `tools/bdf2go/main.go`:
- `const defaultSpleenVersion = "2.1.0"`
- Downloads Spleen release tarball from GitHub if BDF files aren't cached
- Parses BDF format: reads `ENCODING`, `BBX`, `BITMAP` sections
- Extracts ASCII 32-126 + specific Unicode codepoints: U+00B0 (°), U+25B2 (▲), U+25BC (▼), U+25C6 (◆), U+2581-U+2588 (block elements)
- Outputs `internal/font/spleen.go` with a header comment noting source and license
- Generated file defines `var Spleen8x16 = &Font{Width: 8, Height: 16, glyphs: map[rune][]byte{...}}`

- [ ] **Step 7: Create generate.go**

Create `internal/font/generate.go`:

```go
package font

//go:generate go run ../../tools/bdf2go
```

- [ ] **Step 8: Run go generate and verify**

```bash
go generate ./internal/font/
go test ./internal/font/ -v
```

Expected: spleen.go is generated, font tests pass.

- [ ] **Step 9: Commit**

```bash
git add tools/bdf2go/ internal/font/
git commit -m "feat: add BDF-to-Go converter and generate Spleen 8x16 font"
```

---

## Task 5: Framebuffer Package

**Files:**
- Create: `internal/st7735/framebuffer.go`
- Test: `internal/st7735/framebuffer_test.go`

- [ ] **Step 1: Write framebuffer tests**

Port tests from `test/test_st7735_fb.c` to Go. Table-driven where possible:

- `TestFill`: fill with color, verify all 12800 pixels match
- `TestSetPixel`: set pixel at (0,0), (159,79), (80,40), verify value and bounds checking
- `TestSetPixelOutOfBounds`: set pixel at (-1,0), (160,0), (0,80) — no panic, no change
- `TestRect`: draw 10x5 rect at (20,30), verify filled pixels and surrounding unchanged
- `TestChar`: render 'A' with Spleen8x16, verify some known foreground pixels are set and background pixels are untouched
- `TestCharUnknown`: render rune 0 → falls back to '?' glyph
- `TestString`: render "Hi", verify characters are side by side at correct positions
- `TestStringClipping`: string that extends past width=160 is clipped
- `TestBar`: draw 50% bar at (10,20,60,5) — first 30px in fg color, last 30px in bg color
- `TestBar0Percent`: 0% bar — all bg color
- `TestBar100Percent`: 100% bar — all fg color

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/st7735/
```

- [ ] **Step 3: Implement framebuffer.go**

```go
package st7735

import "github.com/cafedomingo/SKU_RM0004/internal/font"

const (
    Width  = 160
    Height = 80
)

type Framebuffer struct {
    Pixels [Width * Height]uint16
}

func (fb *Framebuffer) Fill(color uint16) { ... }
func (fb *Framebuffer) SetPixel(x, y int, color uint16) { ... }
func (fb *Framebuffer) Rect(x, y, w, h int, color uint16) { ... }
func (fb *Framebuffer) Char(x, y int, ch rune, f *font.Font, color uint16) { ... }
func (fb *Framebuffer) String(x, y int, s string, f *font.Font, color uint16) { ... }
func (fb *Framebuffer) Bar(x, y, w, h int, pct int, fg, bg uint16) { ... }
```

Key details:
- `SetPixel`: bounds check, silently ignore out-of-bounds
- `Char`: foreground-only drawing (only set pixels where glyph bit is 1)
- `String`: iterate runes with `range`, advance x by font.Width per character, clip at Width
- `Bar`: filled portion in `fg`, remainder in `bg` (used for progress bars)

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/st7735/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/st7735/framebuffer.go internal/st7735/framebuffer_test.go
git commit -m "feat: add framebuffer with drawing primitives"
```

---

## Task 6: Double-Buffer Diff Engine

**Files:**
- Create: `internal/st7735/diff.go`
- Test: `internal/st7735/diff_test.go`

- [ ] **Step 1: Write diff tests**

```go
package st7735

import "testing"

func TestDiffIdentical(t *testing.T) {
    // Two identical buffers → no dirty regions
    var front, back Framebuffer
    front.Fill(0x0000)
    back.Fill(0x0000)
    regions := DiffRegions(&front, &back)
    if len(regions) != 0 {
        t.Errorf("expected 0 regions, got %d", len(regions))
    }
}

func TestDiffSingleRow(t *testing.T) {
    // Change one row → one region covering that row
    var front, back Framebuffer
    back.SetPixel(80, 40, 0xFFFF)
    regions := DiffRegions(&front, &back)
    if len(regions) != 1 {
        t.Fatalf("expected 1 region, got %d", len(regions))
    }
    if regions[0].Y != 40 || regions[0].H != 1 {
        t.Errorf("region = %+v, want Y=40 H=1", regions[0])
    }
}

func TestDiffCoalesceAdjacentRows(t *testing.T) {
    // Change rows 10-12 → one coalesced region
    var front, back Framebuffer
    for y := 10; y <= 12; y++ {
        back.SetPixel(0, y, 0xFFFF)
    }
    regions := DiffRegions(&front, &back)
    if len(regions) != 1 {
        t.Fatalf("expected 1 region, got %d", len(regions))
    }
    if regions[0].Y != 10 || regions[0].H != 3 {
        t.Errorf("region = %+v, want Y=10 H=3", regions[0])
    }
}

func TestDiffNonAdjacentRows(t *testing.T) {
    // Change rows 5 and 50 → two separate regions
    var front, back Framebuffer
    back.SetPixel(0, 5, 0xFFFF)
    back.SetPixel(0, 50, 0xFFFF)
    regions := DiffRegions(&front, &back)
    if len(regions) != 2 {
        t.Fatalf("expected 2 regions, got %d", len(regions))
    }
}

func TestDiffFullScreen(t *testing.T) {
    // Completely different → one region covering all 80 rows
    var front, back Framebuffer
    back.Fill(0xFFFF)
    regions := DiffRegions(&front, &back)
    if len(regions) != 1 {
        t.Fatalf("expected 1 region, got %d", len(regions))
    }
    if regions[0].Y != 0 || regions[0].H != Height {
        t.Errorf("region = %+v, want Y=0 H=80", regions[0])
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/st7735/ -run TestDiff
```

- [ ] **Step 3: Implement diff.go**

```go
package st7735

// Region describes a full-width horizontal strip of the display.
type Region struct {
    Y int // start row
    H int // number of rows
}

// DiffRegions compares two framebuffers row-by-row and returns
// coalesced regions where they differ. All regions are full-width.
func DiffRegions(front, back *Framebuffer) []Region { ... }
```

Algorithm (matches C `flush_dirty`):
1. For each row 0-79, compare `Width` uint16 values
2. Track `dirtyStart`. When a dirty row starts a new run, record start.
3. When a clean row ends a run (or at row 80), emit a Region.
4. Return slice of Regions.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/st7735/ -run TestDiff -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/st7735/diff.go internal/st7735/diff_test.go
git commit -m "feat: add double-buffer diff engine with row coalescing"
```

---

## Task 7: ST7735 I2C Driver

**Files:**
- Create: `internal/st7735/driver.go`

This task implements the Display interface. Cannot be unit tested (requires hardware), so no test file. The hardware README was already moved to `internal/st7735/README.md` in Task 0.

- [ ] **Step 1: Implement driver.go**

```go
package st7735

import (
    "encoding/binary"
    "fmt"
    "log/slog"
    "time"

    "periph.io/x/conn/v3/i2c"
    "periph.io/x/conn/v3/i2c/i2creg"
    "periph.io/x/host/v3"
)

const (
    i2cAddress    = 0x18
    burstMaxLen   = 160
    burstDelayUS  = 450
    yOffset       = 24

    regWriteData  = 0x00
    regBurstWrite = 0x01
    regSync       = 0x03
    regXCoord     = 0x2A
    regYCoord     = 0x2B
    regCharData   = 0x2C
)

type Display interface {
    SendRegion(x, y, w, h int, pixels []uint16)
    SendFull(pixels []uint16)
    Close() error
}

type display struct {
    dev    *i2c.Dev
    bus    i2c.BusCloser
    logger *slog.Logger
}

func NewDisplay(busName string, logger *slog.Logger) (Display, error) { ... }
func (d *display) SendRegion(x, y, w, h int, pixels []uint16) { ... }
func (d *display) SendFull(pixels []uint16) { ... }
func (d *display) Close() error { ... }
```

Key implementation details:
- `NewDisplay`: call `host.Init()`, open bus via `i2creg.Open(busName)`, create `i2c.Dev` with address 0x18
- `SendRegion`/`SendFull`: convert `[]uint16` to big-endian `[]byte`, set address window (add yOffset to Y coordinates), burst send in 160-byte chunks with 450μs delays
- I2C command format: 3 bytes `[register, high, low]`
- Burst protocol: `burst_begin` (write regBurstWrite 0x00 0x01), send data chunks, `burst_end` (write regBurstWrite 0x00 0x00, then sync)

If `periph.io` can't achieve the timing, we'll refactor to use raw `os.File` + `ioctl` syscalls. The `Display` interface stays the same.

- [ ] **Step 2: Add periph.io dependencies**

```bash
go get periph.io/x/conn/v3 periph.io/x/host/v3
```

- [ ] **Step 3: Commit**

```bash
git add internal/st7735/driver.go go.mod go.sum
git commit -m "feat: add ST7735 I2C display driver via periph.io"
```

---

## Task 8: Sysinfo Package

**Files:**
- Create: `internal/sysinfo/sysinfo.go`, `internal/sysinfo/collector.go`, `internal/sysinfo/pi.go`, `internal/sysinfo/mock.go`
- Test: `internal/sysinfo/collector_test.go`, `internal/sysinfo/pi_test.go`

### Part A: Interface + Mock

- [ ] **Step 1: Define Collector interface and types**

Create `internal/sysinfo/sysinfo.go`:

```go
package sysinfo

import "time"

type CPUFreq struct {
    Cur, Min, Max uint16 // MHz
}

type NetBandwidth struct {
    RxBytesPerSec, TxBytesPerSec uint64
}

type DiskIO struct {
    ReadBytesPerSec, WriteBytesPerSec uint64
    ReadIOPS, WriteIOPS               uint32
}

type DietPiStatus int

const (
    DietPiNotInstalled DietPiStatus = iota
    DietPiUpToDate
    DietPiUpdateAvail
)

type Collector interface {
    CPUPercent() float64
    RAMPercent() float64
    DiskPercent() float64
    Temperature() float64
    Hostname() string
    IPAddress() string
    IPv6Suffix() string
    CPUFreq() CPUFreq
    NetBandwidth() NetBandwidth
    DiskIO() DiskIO
    Uptime() time.Duration
    ThrottleStatus() uint32
    DietPiStatus() DietPiStatus
    APTUpdateCount() int
    LinkSpeedMbps() int // detected from default-route interface
    Refresh()
}
```

- [ ] **Step 2: Create MockCollector**

Create `internal/sysinfo/mock.go` with all fields exported for test configuration:

```go
package sysinfo

import "time"

type MockCollector struct {
    CPU       float64
    RAM       float64
    Disk      float64
    Temp      float64
    Host      string
    IP        string
    IPv6      string
    Freq      CPUFreq
    Net       NetBandwidth
    DIO       DiskIO
    Up        time.Duration
    Throttle  uint32
    DietPi    DietPiStatus
    APT       int
    LinkSpeed int
}

func (m *MockCollector) CPUPercent() float64        { return m.CPU }
func (m *MockCollector) RAMPercent() float64         { return m.RAM }
// ... etc for all interface methods
func (m *MockCollector) Refresh()                    {}
```

- [ ] **Step 3: Commit interface and mock**

```bash
git add internal/sysinfo/sysinfo.go internal/sysinfo/mock.go
git commit -m "feat: add sysinfo Collector interface and MockCollector"
```

### Part B: Live collector

- [ ] **Step 4: Write collector tests**

Create `internal/sysinfo/collector_test.go`. These test the gopsutil-based metrics. Since gopsutil reads real system files, these are more like smoke tests:

- `TestCPUPercentRange`: call twice with 100ms between, result in 0-100
- `TestRAMPercentRange`: result in 1-100
- `TestHostnameNonEmpty`: non-empty string
- `TestUptimePositive`: > 0
- `TestTemperatureRange`: 0-120 (or 0 if no thermal zone)

- [ ] **Step 5: Implement collector.go**

Create `internal/sysinfo/collector.go`:
- Uses `gopsutil/v4` for: `cpu.Percent`, `mem.VirtualMemory`, `host.Info` (hostname, uptime), `sensors.TemperaturesWithContext`
- Custom: `DiskPercent` (aggregate `/` + `/dev/sda*` + `/dev/nvme*` via `syscall.Statfs`), `IPAddress`/`IPv6Suffix` (parse `/proc/net/route` for default interface), `NetBandwidth` (gopsutil `net.IOCounters` with delta), `LinkSpeedMbps` (read `/sys/class/net/<iface>/speed`)

- [ ] **Step 6: Run tests**

```bash
go test ./internal/sysinfo/ -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/sysinfo/collector.go internal/sysinfo/collector_test.go
git commit -m "feat: add live sysinfo collector with gopsutil"
```

### Part C: Pi-specific

- [ ] **Step 8: Write Pi-specific tests**

Create `internal/sysinfo/pi_test.go`:
- `TestCPUFreqRead`: reads `/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq` — returns > 0 on Pi, 0 elsewhere
- `TestDietPiDetection`: check `/run/dietpi` — returns expected status
- `TestAPTUpdateCount`: returns >= -1

Note: throttle status (`/dev/vcio` ioctl) can only be tested on Pi hardware. Test is skipped on non-Pi.

- [ ] **Step 9: Implement pi.go**

Create `internal/sysinfo/pi.go`:
- `readCPUFreq() CPUFreq` — parse sysfs files
- `readThrottleStatus() uint32` — open `/dev/vcio`, ioctl with mailbox property buffer (tag 0x00030046, aligned 16-byte buffer), parse response
- `readDietPiStatus() DietPiStatus` — check `/run/dietpi` dir, `/run/dietpi/.update_available`
- `readAPTUpdateCount() int` — read `/run/dietpi/.apt_updates`, fallback to `/boot/dietpi/.version`

- [ ] **Step 10: Run tests**

```bash
go test ./internal/sysinfo/ -v
```

- [ ] **Step 11: Commit**

```bash
git add internal/sysinfo/pi.go internal/sysinfo/pi_test.go
git commit -m "feat: add Pi-specific sysinfo (throttle, dietpi, cpu freq)"
```

---

## Task 9: Dashboard Screen

**Files:**
- Create: `internal/screen/dashboard.go`
- Test: `internal/screen/dashboard_test.go`

- [ ] **Step 1: Write dashboard tests**

Create `internal/screen/dashboard_test.go`:
- `TestDashboardRenders`: mock collector with known values, render to framebuffer, verify:
  - Hostname text pixels present at top
  - CPU bar region has correct threshold color
  - Temperature uses `°` character
  - Background pixels are ColorBG
- `TestDashboardThresholds`: render with values at warn/crit boundaries, verify bar colors change
- `TestDashboardDisplayFloor`: CPU=0, RAM=0 → bars show 1% (display floor), not 0%
- `TestDashboardDietPiDiamond`: with DietPiUpdateAvail → ◆ character pixels present
- `TestDashboardAPTBadge`: apt_count=3 → "^3" text pixels present

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/screen/ -run Dashboard
```

- [ ] **Step 3: Implement dashboard.go**

```go
package screen

import (
    "github.com/cafedomingo/SKU_RM0004/internal/config"
    "github.com/cafedomingo/SKU_RM0004/internal/font"
    "github.com/cafedomingo/SKU_RM0004/internal/format"
    "github.com/cafedomingo/SKU_RM0004/internal/st7735"
    "github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
    "github.com/cafedomingo/SKU_RM0004/internal/theme"
)

func RenderDashboard(fb *st7735.Framebuffer, c sysinfo.Collector, cfg config.Config) { ... }
```

Layout (adjusted for 8x16 font, 80px height):
- y=0: Hostname (8x16, ColorFG) + ◆ at top-right if DietPi update
- y=16: IP (8x16, ColorIP) + APT badge right-aligned
- y=33: separator line (1px)
- y=35: CPU label + value (8x16) + bar below (~1px gap, 4-5px bar height)
- y=56: RAM label + value (8x16) + bar below
- Right column mirrors: TEMP and DISK at same Y positions

Display floor: clamp CPU/RAM/Disk to min 1 for bar rendering.
Temperature: use `format.Temp(celsius, cfg.TempUnit)` with `°` character.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/screen/ -run Dashboard -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/screen/dashboard.go internal/screen/dashboard_test.go
git commit -m "feat: add dashboard screen renderer"
```

---

## Task 10: Diagnostic Screen

**Files:**
- Create: `internal/screen/diagnostic.go`
- Test: `internal/screen/diagnostic_test.go`

- [ ] **Step 1: Write diagnostic tests**

- `TestDiagnosticPageCount`: with 15 rows and 5 per page → 3 pages
- `TestDiagnosticPage0Content`: mock collector, render page 0, verify hostname row present at y=0, CPU row with threshold color
- `TestDiagnosticPage1Content`: render page 1, verify disk/net rows
- `TestDiagnosticPage2Content`: render page 2, verify DietPi/APT rows
- `TestDiagnosticTempBothUnits`: temp row shows both °C and °F regardless of config
- `TestDiagnosticThrottleStates`: active (CRIT), past (WARN), none (OK)

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/screen/ -run Diagnostic
```

- [ ] **Step 3: Implement diagnostic.go**

```go
package screen

type DiagState struct {
    Page int // 0, 1, or 2
}

func RenderDiagnostic(fb *st7735.Framebuffer, c sysinfo.Collector, cfg config.Config, state *DiagState) { ... }
```

Layout: 5 rows per page at 16px each = 80px.
- Page 0 (rows 0-4): hostname, IPv4, IPv6, uptime, CPU% + freq
- Page 1 (rows 5-9): temp (both °C/°F), RAM%, throttle, disk%, net RX
- Page 2 (rows 10-14): net TX, IO R/W, IOPS R/W, DietPi, APT

Each row: label left-aligned in ColorMuted, value right-aligned in its color.
Header rows (hostname, IPs): no label, value left-aligned.

State management: `state.Page` increments after render, wraps at 3. Data refresh only on page 0.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/screen/ -run Diagnostic -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/screen/diagnostic.go internal/screen/diagnostic_test.go
git commit -m "feat: add diagnostic screen renderer (3 pages)"
```

---

## Task 11: Sparkline Screen

**Files:**
- Create: `internal/screen/sparkline.go`
- Test: `internal/screen/sparkline_test.go`

- [ ] **Step 1: Write sparkline tests**

- `TestSparklineHistoryShift`: after render, history arrays shift left, new value at end
- `TestSparklineTickerCycle`: ticker advances 0→1→2→0 (or 0→1→0 if no IPv6)
- `TestSparklineBlockElements`: verify sparkline bars use block element characters (▁-█) for rendering, or custom rect drawing based on final implementation
- `TestSparklineThresholdColors`: CPU at 70% renders in warn color
- `TestSparklineIORow`: net/disk rates rendered with threshold colors
- `TestSparklineDisplayFloor`: CPU=0 → rendered as 1%

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/screen/ -run Sparkline
```

- [ ] **Step 3: Implement sparkline.go**

```go
package screen

const sparklineHistory = 13

type SparklineState struct {
    CPUHistory  [sparklineHistory]float64
    RAMHistory  [sparklineHistory]float64
    TickerPhase int
}

func RenderSparkline(fb *st7735.Framebuffer, c sysinfo.Collector, cfg config.Config, state *SparklineState) { ... }
```

Layout (adjusted for 8x16 font):
- y=0: ticker (hostname/IPv4/IPv6 cycling)
- y=16: uptime + update badges
- y=32: separator
- y=34: sparkline graph area (reduced height vs C version)
- y=58: CPU% | RAM% labels
- y=74: I/O rates (if they fit, otherwise merge with line above)

Sparkline rendering options:
1. Use `▁▂▃▄▅▆▇█` block characters if the graph uses character-cell columns
2. Use `Framebuffer.Rect()` for pixel-precise bars (same as C)

The implementation should try block elements first; fall back to rects if the visual doesn't work with the font height.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/screen/ -run Sparkline -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/screen/sparkline.go internal/screen/sparkline_test.go
git commit -m "feat: add sparkline screen renderer with history charts"
```

---

## Task 12: Main Loop

**Files:**
- Create: `cmd/display/main.go`

- [ ] **Step 1: Implement main.go**

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/cafedomingo/SKU_RM0004/internal/config"
    "github.com/cafedomingo/SKU_RM0004/internal/screen"
    "github.com/cafedomingo/SKU_RM0004/internal/st7735"
    "github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
    "github.com/cafedomingo/SKU_RM0004/internal/theme"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
    logger.Info("starting")

    // Check I2C speed
    checkI2CSpeed(logger)

    // Open display
    disp, err := st7735.NewDisplay("/dev/i2c-1", logger)
    if err != nil {
        logger.Error("failed to open display", "error", err)
        os.Exit(1)
    }
    defer disp.Close()

    // Init
    collector := sysinfo.NewCollector(logger)
    cfgLoader := config.NewLoader(config.DefaultPath, logger)
    var front, back st7735.Framebuffer
    var diagState screen.DiagState
    var sparkState screen.SparklineState
    prevScreen := ""

    // Signal handling
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
    defer stop()

    // Fill screen black
    back.Fill(theme.ColorBG)
    disp.SendFull(back.Pixels[:])
    front = back

    // Main loop
    for {
        start := time.Now()
        cfg := cfgLoader.Load()

        // Screen change: reset state, full redraw
        if cfg.Screen != prevScreen {
            diagState = screen.DiagState{}
            sparkState = screen.SparklineState{}
            front.Fill(theme.ColorBG)
            disp.SendFull(front.Pixels[:])
            prevScreen = cfg.Screen
        }

        // Render
        back.Fill(theme.ColorBG)
        switch cfg.Screen {
        case "diagnostic":
            if diagState.Page == 0 {
                collector.Refresh()
            }
            screen.RenderDiagnostic(&back, collector, cfg, &diagState)
            disp.SendFull(back.Pixels[:]) // diagnostic always full redraw
            front = back
        case "sparkline":
            collector.Refresh()
            screen.RenderSparkline(&back, collector, cfg, &sparkState)
            regions := st7735.DiffRegions(&front, &back)
            for _, r := range regions {
                disp.SendRegion(0, r.Y, st7735.Width, r.H, back.Pixels[r.Y*st7735.Width:(r.Y+r.H)*st7735.Width])
            }
            front = back
        default: // dashboard
            collector.Refresh()
            screen.RenderDashboard(&back, collector, cfg)
            regions := st7735.DiffRegions(&front, &back)
            for _, r := range regions {
                disp.SendRegion(0, r.Y, st7735.Width, r.H, back.Pixels[r.Y*st7735.Width:(r.Y+r.H)*st7735.Width])
            }
            front = back
        }

        // Sleep remaining
        elapsed := time.Since(start)
        if remaining := cfg.Refresh - elapsed; remaining > 0 {
            select {
            case <-ctx.Done():
                logger.Info("shutting down")
                return
            case <-time.After(remaining):
            }
        }

        // Check for shutdown between iterations
        select {
        case <-ctx.Done():
            logger.Info("shutting down")
            return
        default:
        }
    }
}

func checkI2CSpeed(logger *slog.Logger) {
    // Read /proc/device-tree/soc/i2c@7e804000/clock-frequency (big-endian uint32)
    // Log info if 400000, warn otherwise
    ...
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./cmd/display/
```

Expected: compiles without error (won't run without Pi hardware).

- [ ] **Step 3: Commit**

```bash
git add cmd/display/main.go
git commit -m "feat: add main display loop with signal handling and double buffering"
```

---

## Task 13: Screenshot Tool

**Files:**
- Create: `cmd/screenshot/main.go`

- [ ] **Step 1: Implement screenshot main.go**

```go
package main

import (
    "image"
    "image/color"
    "image/png"
    "os"
    "time"

    "github.com/cafedomingo/SKU_RM0004/internal/config"
    "github.com/cafedomingo/SKU_RM0004/internal/screen"
    "github.com/cafedomingo/SKU_RM0004/internal/st7735"
    "github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
)

const scale = 5

func main() {
    mock := &sysinfo.MockCollector{
        CPU:    47, RAM: 63, Disk: 42, Temp: 52,
        Host:   "raspberrypi",
        IP:     "192.168.1.100",
        IPv6:   "::a8f1",
        Freq:   sysinfo.CPUFreq{Cur: 1800, Min: 600, Max: 2400},
        Net:    sysinfo.NetBandwidth{RxBytesPerSec: 15360, TxBytesPerSec: 2048},
        DIO:    sysinfo.DiskIO{ReadBytesPerSec: 1048576, WriteBytesPerSec: 524288, ReadIOPS: 150, WriteIOPS: 80},
        Up:     3*24*time.Hour + 2*time.Hour,
        DietPi: sysinfo.DietPiUpToDate,
        APT:    3,
        LinkSpeed: 1000,
    }
    cfg := config.Config{Screen: "dashboard", Refresh: 5 * time.Second, TempUnit: "C"}

    // Dashboard
    var fb st7735.Framebuffer
    screen.RenderDashboard(&fb, mock, cfg)
    writePNG("docs/dashboard.png", &fb)

    // Diagnostic (page 0)
    fb = st7735.Framebuffer{}
    diagState := screen.DiagState{Page: 0}
    screen.RenderDiagnostic(&fb, mock, cfg, &diagState)
    writePNG("docs/diagnostic.png", &fb)

    // Sparkline
    fb = st7735.Framebuffer{}
    state := screen.SparklineState{
        CPUHistory: [13]float64{22, 35, 28, 45, 52, 38, 61, 73, 55, 42, 68, 50, 47},
        RAMHistory: [13]float64{40, 42, 45, 48, 50, 53, 55, 57, 58, 60, 61, 62, 63},
    }
    screen.RenderSparkline(&fb, mock, cfg, &state)
    writePNG("docs/sparkline.png", &fb)
}

func writePNG(path string, fb *st7735.Framebuffer) {
    w := st7735.Width * scale
    h := st7735.Height * scale

    // Use image.Paletted for indexed color (small file size)
    // Collect unique colors from framebuffer
    palette := collectPalette(fb)
    img := image.NewPaletted(image.Rect(0, 0, w, h), palette)

    for y := 0; y < st7735.Height; y++ {
        for x := 0; x < st7735.Width; x++ {
            c := rgb565ToRGBA(fb.Pixels[y*st7735.Width+x])
            for dy := 0; dy < scale; dy++ {
                for dx := 0; dx < scale; dx++ {
                    img.Set(x*scale+dx, y*scale+dy, c)
                }
            }
        }
    }

    f, _ := os.Create(path)
    defer f.Close()
    png.Encode(f, img)
}

func rgb565ToRGBA(c uint16) color.RGBA {
    r5 := (c >> 11) & 0x1F
    g6 := (c >> 5) & 0x3F
    b5 := c & 0x1F
    return color.RGBA{
        R: uint8(r5<<3 | r5>>2),
        G: uint8(g6<<2 | g6>>4),
        B: uint8(b5<<3 | b5>>2),
        A: 255,
    }
}

func collectPalette(fb *st7735.Framebuffer) color.Palette {
    seen := map[uint16]bool{}
    for _, px := range fb.Pixels {
        seen[px] = true
    }
    p := make(color.Palette, 0, len(seen))
    for c := range seen {
        p = append(p, rgb565ToRGBA(c))
    }
    return p
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./cmd/screenshot/
```

- [ ] **Step 3: Commit**

```bash
git add cmd/screenshot/main.go
git commit -m "feat: add screenshot tool for PNG doc generation"
```

---

## Task 14: CI Pipeline

**Files:**
- Modify: `.github/workflows/build.yml`

- [ ] **Step 1: Write build.yml**

```yaml
name: Build

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Check formatting
        run: |
          test -z "$(gofmt -l ./...)" || { gofmt -l ./... ; exit 1; }

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

      - name: Lint shell scripts
        run: shellcheck install.sh

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run tests
        run: go test ./...

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build display binary (arm64)
        run: GOOS=linux GOARCH=arm64 go build -o display ./cmd/display

      - name: Build screenshot tool (native)
        run: go build -o screenshot ./cmd/screenshot

      - name: Generate screenshots
        run: ./screenshot

      - name: Upload artifacts
        uses: actions/upload-artifact@v7
        with:
          name: build
          path: |
            display
            docs/*.png

  release:
    needs: [lint, test, build]
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v6
        with:
          sparse-checkout: install.sh
          sparse-checkout-cone-mode: false

      - uses: actions/download-artifact@v8
        with:
          name: build

      - name: Create release
        env:
          GH_TOKEN: ${{ github.token }}
          GH_REPO: ${{ github.repository }}
        run: |
          chmod +x display install.sh
          TAG=$(date -u +%Y.%m.%d.%H%M)
          gh release create "$TAG" display install.sh \
            --title "$TAG" \
            --notes "Built from commit ${GITHUB_SHA::7}."
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/build.yml
git commit -m "feat: update CI pipeline for Go (lint, test, cross-compile, release)"
```

---

## Task 15: Install Script Update

**Files:**
- Modify: `install.sh`

- [ ] **Step 1: Update install.sh**

Changes needed:
- The binary name stays `display` — no changes to install path logic
- The `Makefile` check for local binary detection changes to check for `go.mod` instead
- Add `temp_unit=C` to the default config file creation
- No other changes needed — the script already handles stopping/starting the service, I2C config, GPIO overlay

Update the local binary detection:
```bash
# Developer path: use local binary if run from a repo clone
if [ -f "./${BINARY}" ] && [ -f "./go.mod" ]; then
```

Update install_config:
```bash
install_config() {
    if [ ! -f /etc/uctronics-display.conf ]; then
        log "Creating default config at /etc/uctronics-display.conf"
        cat > /etc/uctronics-display.conf <<CONF
# UCTRONICS LCD display configuration
screen=dashboard
refresh=5
temp_unit=C
CONF
    fi
}
```

- [ ] **Step 2: Run shellcheck**

```bash
shellcheck install.sh
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add install.sh
git commit -m "feat: update install script for Go binary"
```

---

## Task 16: README and Final Polish

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Rewrite README.md**

Update for Go:
- Project description
- Screenshot images (dashboard, sparkline)
- Installation: `curl -sL ... | sudo bash`
- Configuration: document all three config keys (screen, refresh, temp_unit)
- Building from source: `go build ./cmd/display`
- Development: `go test ./...`, `go generate ./internal/font/`, screenshot tool
- Credits: Spleen font acknowledgment
- License: MIT

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: rewrite README for Go version"
```

---

## Task 17: Integration Verification

This task runs all tests, lint, and build to verify everything works together.

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -v
```

Expected: all tests pass.

- [ ] **Step 2: Run linter**

```bash
golangci-lint run
```

Expected: no errors.

- [ ] **Step 3: Check formatting**

```bash
test -z "$(gofmt -l ./...)"
```

Expected: no files listed.

- [ ] **Step 4: Build binaries**

```bash
GOOS=linux GOARCH=arm64 go build -o display ./cmd/display
go build -o screenshot ./cmd/screenshot
```

Expected: both compile without errors.

- [ ] **Step 5: Generate screenshots**

```bash
./screenshot
ls -la docs/*.png
```

Expected: dashboard.png and sparkline.png generated, reasonable file sizes.

- [ ] **Step 6: Verify install.sh**

```bash
shellcheck install.sh
```

Expected: no errors.

- [ ] **Step 7: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "fix: integration verification fixes"
```

(Only if fixes were needed in previous steps.)
