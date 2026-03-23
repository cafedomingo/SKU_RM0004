# Go Rewrite Design Spec

**Date:** 2026-03-22
**Status:** Draft
**Scope:** Complete rewrite of UCTRONICS SKU_RM0004 display firmware from C to Go

## Overview

Rewrite the ~3,760-line C codebase as idiomatic Go. Maintain all existing functionality (three screen modes, runtime config, system metrics, I2C display driver) while improving readability, testability, and maintainability. This is a clean break — all C source, current fonts, and the existing license are replaced.

### Goals

- Idiomatic Go, not a direct C port
- Use libraries (`gopsutil`, `periph.io`) instead of raw `/proc`/`/sys` parsing where possible
- Comprehensive test coverage including renderer integration tests
- Runtime config compatibility (same file path and format, no restart required)
- Clean MIT license with no UCTRONICS attribution (no original code remains)
- Replace bitmap fonts with Spleen (BSD 2-Clause, clean provenance)
- Install cleanly over existing C installation

### Non-Goals

- Multi-platform support beyond Raspberry Pi (Linux arm64)
- GUI or web interface
- APT update count on non-DietPi systems (deferred to future work)

## Architecture

```
┌──────────────────────────────────┐
│  cmd/display          Main loop  │
│  cmd/screenshot       PNG docs   │
├──────────────────────────────────┤
│  screen/   Renderers draw into   │
│            framebuffer only      │
├──────────────────────────────────┤
│  st7735/   Framebuffer (draw) +  │
│            Driver (I2C transfer) │
├──────────────────────────────────┤
│  sysinfo/  gopsutil + Pi-native  │
├──────────────────────────────────┤
│  config/   Runtime config reader │
│  theme/    Colors + thresholds   │
│  format/   Metric formatting     │
│  font/     Spleen data + glyphs  │
└──────────────────────────────────┘
```

Renderers never touch hardware. They draw into a framebuffer. The main loop diffs the framebuffer against what's on screen and sends only changed regions to the display.

## Project Structure

```
SKU_RM0004/
├── go.mod
├── go.sum
├── LICENSE                             # Clean MIT
├── THIRD_PARTY_LICENSES                # Spleen BSD-2-Clause
├── README.md
├── install.sh                          # Updated for Go binary
├── .editorconfig
├── .gitattributes                      # Line endings, linguist-generated markers
├── .gitignore                          # Build artifacts, binaries
├── .golangci.yml
│
├── cmd/
│   ├── display/main.go                 # Entry point, main loop
│   └── screenshot/main.go              # PNG doc generator
│
├── internal/
│   ├── config/config.go                # Runtime config loader
│   ├── font/
│   │   ├── font.go                     # Font type, glyph lookup
│   │   ├── spleen.go                   # Generated Spleen font data
│   │   └── glyphs.go                   # Custom glyphs (diamond, arrow)
│   ├── format/format.go                # Rate, freq, uptime, temp formatting
│   ├── screen/
│   │   ├── dashboard.go                # Single-page status view
│   │   ├── diagnostic.go               # Two-page detail view
│   │   └── sparkline.go                # Scrolling history charts
│   ├── sysinfo/
│   │   ├── sysinfo.go                  # Collector interface + gopsutil impl
│   │   └── pi.go                       # Pi-specific: throttle, dietpi
│   ├── st7735/
│   │   ├── driver.go                   # I2C display driver (periph.io)
│   │   ├── framebuffer.go             # 160x80 RGB565 buffer + drawing
│   │   └── README.md                   # Hardware reference (I2C protocol, timing)
│   └── theme/theme.go                  # Colors, thresholds, temp ramp
│
├── tools/
│   └── bdf2go/main.go                 # BDF → Go source converter
│
├── .github/workflows/build.yml
│
└── docs/                               # Generated screenshots
```

## Key Dependencies

| Dependency | Purpose |
|---|---|
| `github.com/shirou/gopsutil/v4` | CPU, RAM, disk, network, temperature, uptime, hostname |
| `periph.io/x/conn/v3` + `periph.io/x/host/v3` | I2C bus access |
| `log/slog` (stdlib) | Structured logging |
| `image/png` (stdlib) | Screenshot generation |

## Component Details

### ST7735 Driver (`internal/st7735/driver.go`)

Thin I2C transport layer:

```go
type Display interface {
    SendRegion(x, y, w, h int, pixels []uint16)
    SendFull(pixels []uint16)
    Close() error
}

// Constructor opens the I2C bus and initializes the display
func NewDisplay(bus string, logger *slog.Logger) (Display, error)
```

Implementation details (from hardware testing — do not change without hardware verification):
- I2C address: `0x18`
- I2C bus: `/dev/i2c-1` at 400kHz
- Burst chunk size: 160 bytes max (hardware limit)
- Inter-chunk delay: 450μs (empirically tuned)
- Full-screen transfer: ~720ms (160 chunks × 450μs)
- Y-offset: 24 pixels (controller is 160x160, display is 160x80)

Start with `periph.io` for I2C access. Fall back to raw ioctl if the burst timing can't be achieved through the library. The interface stays the same either way.

### Framebuffer (`internal/st7735/framebuffer.go`)

All drawing happens in memory. 160×80 RGB565 pixel buffer.

```go
type Framebuffer struct {
    Pixels [160 * 80]uint16
}
```

Drawing methods:
- `Fill(color uint16)`
- `SetPixel(x, y int, color uint16)`
- `Rect(x, y, w, h int, color uint16)`
- `Char(x, y int, ch rune, f Font, color uint16)` — foreground only, background untouched; rune supports Unicode
- `String(x, y int, s string, f Font, color uint16)` — foreground only; iterates runes

Renderers clear the framebuffer with `Fill(bg)` first, then draw foreground pixels. This matches the C framebuffer behavior where `lcd_fb_char` only sets foreground pixels.

**Byte order:** The framebuffer stores pixels as native `uint16` for fast comparison during diffing. The `Display.SendRegion()`/`SendFull()` implementation converts to big-endian bytes before I2C transfer, since the ST7735 expects MSB-first RGB565 on the wire.

### Double Buffering & Diff-Based Updates

The main loop maintains two framebuffers: `front` (what's on the display) and `back` (what renderers draw into). After each render:

1. Renderer draws full frame into `back`
2. Diff `back` vs `front` to find changed rows
3. Coalesce adjacent dirty rows into regions
4. Send only dirty regions via `Display.SendRegion()`
5. Copy `back` → `front`

This is automatic — renderers don't track dirty state. The diff scan (comparing 25,600 bytes in memory) is negligible vs. I2C transfer cost. A single changed bar might be ~800 bytes instead of 25,600.

Exception: diagnostic screen does a full redraw when flipping pages (different content entirely).

### System Metrics (`internal/sysinfo/sysinfo.go`)

```go
type Collector interface {
    CPUPercent() float64
    RAMPercent() float64
    DiskPercent() float64
    Temperature() float64       // Always Celsius internally
    Hostname() string
    IPAddress() string
    IPv6Suffix() string
    CPUFreq() CPUFreq           // Cur, Min, Max MHz
    NetBandwidth() NetBandwidth // RX, TX bytes/sec
    DiskIO() DiskIO             // Read, Write bytes/sec, IOPS
    Uptime() time.Duration
    ThrottleStatus() uint32
    DietPiStatus() DietPiStatus
    APTUpdateCount() int
    LinkSpeedMbps() int         // Detected from default-route interface
    Refresh()                   // Update delta-based metrics
}
```

**gopsutil handles:** CPUPercent, RAMPercent, Temperature, Hostname, Uptime, NetBandwidth counters

**Custom implementation required (not suitable for gopsutil):**
- DiskPercent: aggregates root filesystem (`/`) + any `/dev/sda*` or `/dev/nvme*` mount points via `/proc/mounts` + `statfs()`. This is Pi-specific behavior — gopsutil's `disk.Usage("/")` only reports root. The Go code must replicate this aggregation.
- IPAddress / IPv6Suffix: detect default-route interface by parsing `/proc/net/route`, then read that interface's addresses. gopsutil's `net.Interfaces()` doesn't have default-route detection. Multi-homed systems must show the same IP as the C version.

**Direct reads / ioctl (Pi-specific):**
- Throttle status: reads via VideoCore mailbox (`/dev/vcio` ioctl, tag `0x00030046`). Requires a 16-byte-aligned buffer and `_IOWR(100, 0, char*)` ioctl. Neither gopsutil nor periph.io provides this — raw ioctl required.
- CPU frequency (cur/min/max): `/sys/devices/system/cpu/cpu0/cpufreq/*`
- DietPi detection: check `/run/dietpi` directory exists
- DietPi update status: check `/run/dietpi/.update_available` exists (present = update available)
- APT update count: read integer from `/run/dietpi/.apt_updates`. If file missing, check `/boot/dietpi/.version` exists (DietPi but no updates = 0). If neither exists, not DietPi — return -1. Non-DietPi systems show no APT badge (daily check deferred to future work).
- IPv6 suffix: parsed from default-route network interface

A `MockCollector` implementation provides fixed values for tests and the screenshot tool.

### Runtime Config (`internal/config/config.go`)

**File:** `/etc/uctronics-display.conf` (INI-style)

```ini
# All settings optional. Missing keys or missing file = defaults.
screen=dashboard       # dashboard | diagnostic | sparkline
refresh=5              # 2-30 seconds
temp_unit=C            # C | F
```

```go
type Config struct {
    Screen   string        // "dashboard", "diagnostic", "sparkline"
    Refresh  time.Duration // 2s-30s
    TempUnit string        // "C" or "F"
}
```

Behavior:
- No file → all defaults (dashboard, 5s, Celsius)
- Partial file → defaults for missing keys
- Invalid values → ignored, default used, warning logged
- File checked every refresh cycle via mtime — no restart needed
- New `temp_unit` setting replaces the old compile-time `TEMPERATURE_TYPE` flag

### Fonts (`internal/font/`)

**Spleen fonts** (BSD 2-Clause, github.com/fcambus/spleen):

| Size | Use case |
|---|---|
| 5x8 | Available as fallback if 8x16 is too large somewhere |
| 8x16 | Primary font — all text across all screens |
| 12x24 | Available if needed |
| 16x32 | Available if needed |

**Font size strategy:** Use Spleen 8x16 as the single primary font, replacing both the old 7x10 (metric labels, values) and 8x16 (hostname). This simplifies rendering — one font size for everything — but requires layout adjustments since 8x16 is 6px taller than 7x10.

**Layout adjustments:**
- **Dashboard:** Tighten spacing around the separator, reduce gap between label text and progress bar, shrink bar height by 1-2px. The 2x2 metric grid (CPU/Temp, RAM/Disk) fits with these tweaks.
- **Sparkline:** Reduce graph height to reclaim vertical space for taller text rows.
- **Diagnostic:** With 8x16, only 5 rows fit in 80px. Diagnostic will need 3 pages instead of 2, or a reduced row count. Acceptable since diagnostic is for detailed inspection, not at-a-glance monitoring.

Final layouts will be validated visually using the screenshot tool before merging.

Character ranges: ASCII 32-126 (printable), plus selected Unicode blocks (Latin-1 Supplement, Box Drawing, Block Elements, Geometric Shapes). The BDF converter extracts only the codepoints we use to keep the generated file compact.

**BDF → Go converter** (`tools/bdf2go/main.go`):
- Reads Spleen BDF files from a release archive
- Outputs Go source with font data as byte arrays
- Spleen version stored as a constant in the tool for easy updates:
  ```go
  const defaultSpleenVersion = "2.1.0"
  ```
- Invoked via `go generate` in `internal/font/`
- Generated `spleen.go` is committed to the repo — normal builds need no network access

**Unicode symbols from Spleen** (replacing custom glyphs where possible):
- `◆` (U+25C6) — DietPi update indicator (replaces custom diamond glyph)
- `▲` (U+25B2) / `▼` (U+25BC) — trend indicators (replaces custom arrow glyph)
- `°` (U+00B0) — degree sign for temperature ("52°C" instead of "52C")
- `▁▂▃▄▅▆▇█` (U+2581–U+2588) — block elements for sparkline bar rendering

**Custom glyphs** (`internal/font/glyphs.go`):
- Infrastructure kept as a fallback for anything Spleen doesn't cover
- BDF converter must handle multi-byte Unicode codepoints, not just ASCII 32-126

**Attribution:** Spleen license included in `THIRD_PARTY_LICENSES` at repo root. Source noted in generated `spleen.go` header comment. Brief credit in README.

### Screen Renderers (`internal/screen/`)

Each renderer is a function that draws into a `Framebuffer` using data from a `Collector` and `Config`.

**Dashboard** (`dashboard.go`) — single-page status:
- Hostname, IP, APT badge
- CPU bar + temp bar
- RAM bar + disk bar
- DietPi ◆ indicator

Layout adjustments for 8x16 font (was 7x10 for metrics):
- Reduce gap between label text and progress bar (bar immediately below text, ~1px gap instead of 2px)
- Shrink bar height by 1-2px (4-5px instead of 6px)
- Tighten spacing around the separator line
- Temperature uses `°` character ("52°C")

**Diagnostic** (`diagnostic.go`) — multi-page detail:
- Same data as before: hostname, IPs, uptime, CPU%, temp, RAM%, throttle, disk%, net RX/TX, disk I/O, IOPS, DietPi, APT
- With 8x16 font, 5 rows fit per page (80px / 16px). Currently 15 rows total → 3 pages instead of 2.
- Alternates pages each refresh. Full screen redraw each time.
- State: page counter (0/1/2)
- Data refresh: metrics collected only on page 0. Subsequent pages display the same data snapshot. This ensures delta-based metrics (net bandwidth, disk I/O) measure a full refresh interval.
- Temperature row always shows both °C and °F regardless of `temp_unit` config. The `temp_unit` config only affects dashboard and sparkline formatting.

**Sparkline** (`sparkline.go`) — scrolling history:
- Ticker cycling hostname → IPv4 → IPv6
- CPU/RAM rolling history
- Sparkline charts using `▁▂▃▄▅▆▇█` block elements + I/O stats
- State: history buffers + ticker phase

Layout adjustments for 8x16 font:
- Reduce sparkline graph height to reclaim vertical space for taller text rows
- History sample count may decrease with shorter graphs (fewer pixel rows = fewer distinct values to display)
- Block elements give 8 distinct heights per character cell, so even shorter graphs maintain resolution

Dashboard and sparkline benefit from double-buffer diffing — most refreshes only change a few bars or values. Diagnostic always redraws fully.

### Theme (`internal/theme/theme.go`)

Colors (RGB565) and threshold logic, centralized:

**Thresholds:**
- CPU: warn 60%, crit 80%
- RAM: warn 60%, crit 80%
- Disk: warn 70%, crit 90%
- Net: dynamic — detect link speed of default-route interface via `/sys/class/net/<iface>/speed`, warn at 40%, crit at 80% of link capacity. If speed unavailable (some WiFi drivers), fall back to 100 Mbps assumption.
- Disk I/O: warn 25 MB/s, crit 75 MB/s
- APT: crit 10+ updates
- Temp ramp (matches DietPi): <40°C cyan (cool) → 40°C green (optimal) → 50°C yellow (warm) → 60°C orange (hot) → 70°C red (critical)

Functions: `ThresholdColor(value, warn, crit)`, `TempRampColor(celsius)`

### Format (`internal/format/format.go`)

Metric string formatting:
- `Rate(bytes)` → "0B", "1.2K", "45K", "1.2M"
- `Freq(mhz)` → "600MHz", "1.8GHz"
- `Uptime(duration)` → "3d 2h", "5h 12m", "42m"
- `Temp(celsius, unit)` → "52°C", "125°F"
- `APTBadge(count)` → "^3" (capped at 99)

**Display floor:** When rendering percent values (CPU, RAM, disk), clamp to a minimum of 1% for display. The collector returns the real value (including 0); the floor is applied at the rendering layer only. A system that's truly at 0% CPU or 0% RAM isn't realistic in practice, and showing "0%" looks like a read failure.

### Logging

`log/slog` (stdlib), structured, leveled.

**Log what:**
- INFO: startup details (I2C bus speed, config loaded, screen mode), config changes, screen switches
- WARN: config parse issues, I2C speed mismatch, missing network interface
- ERROR: I2C failures, fatal startup issues

**Don't log:** every refresh cycle, metric values, successful I2C writes.

Logger created in `main()`, passed via struct fields (not global). Output: stderr (systemd/journald captures it).

### Main Loop (`cmd/display/main.go`)

```
1. Initialize logger
2. Open I2C display (or fail with clear error)
3. Log I2C bus speed, warn if not 400kHz
4. Create Collector (gopsutil + Pi-specific)
5. Allocate front + back framebuffers
6. Register signal handler (SIGTERM, SIGINT) for graceful shutdown
7. Loop:
   a. Load config (check mtime)
   b. Log if config changed
   c. Refresh metrics (diagnostic: only on page 0)
   d. Clear back buffer
   e. Render current screen into back buffer
   f. Diff back vs front → dirty regions
   g. Send dirty regions to display
   h. Copy back → front
   i. Sleep remaining time to hit refresh interval
8. On shutdown signal: close I2C, log exit, return
```

**Signal handling:** The main loop listens for `SIGTERM`/`SIGINT` via `signal.NotifyContext`. On signal, the loop exits cleanly, closing the I2C connection. No display blanking on exit — the last frame stays visible (same as current C behavior).

### Screenshot Tool (`cmd/screenshot/main.go`)

- Uses `MockCollector` with fixed metric values
- Renders each screen into a `Framebuffer`
- Converts `[]uint16` RGB565 → `image.RGBA`
- Writes PNGs at 5x scale for readability
- Optimize PNG output for small file size (indexed color palette where possible, maximum compression). The display only uses ~15 distinct colors, so indexed PNGs should be very compact.
- Builds and runs natively on any platform (no cross-compilation needed)
- CI runs this on the build runner (x86 Linux) to generate docs

## Testing Strategy

### Unit Tests

| Package | What's tested |
|---|---|
| `config` | Missing file, partial file, invalid values, mtime caching |
| `format` | Boundary values, unit conversions, edge cases |
| `theme` | Threshold colors, temp ramp interpolation |
| `font` | Glyph lookup, character range bounds |
| `st7735/framebuffer` | Fill, pixel, rect, char, string, glyph — pixel-by-pixel validation |
| `sysinfo` | Mock `/proc`/`/sys` file content → expected parsed values |

### Renderer Integration Tests

Each screen renderer tested against a `MockCollector` and a real `Framebuffer`:
- Provide known metric values
- Render into framebuffer
- Assert specific pixels/regions have expected colors
- Table-driven tests for different metric combinations (normal, warn, crit thresholds)

### What's NOT Tested

- I2C driver (requires hardware)
- Main loop integration (tested manually on Pi)

## CI Pipeline

```yaml
jobs:
  lint:
    - gofmt -l ./...
    - goimports -l ./...
    - golangci-lint run
    - shellcheck install.sh

  test:
    - go test ./...

  build:
    - GOOS=linux GOARCH=arm64 go build -o display ./cmd/display
    - go build -o screenshot ./cmd/screenshot   # native, for doc generation
    - ./screenshot                                # generate PNGs

  release:   # on push to main
    - Build arm64 display binary
    - Generate screenshots (native)
    - Create dated release (YYYY.MM.DD.HHMM tag)
    - Attach: display binary, install.sh
```

Cross-compilation is built into Go — no external toolchain needed.

## Installation

`install.sh` updated to install the Go binary. Same behavior:
1. Detect Pi model (4 or 5)
2. Enable I2C in boot config (`dtparam=i2c_arm=on,i2c_arm_baudrate=400000`)
3. Configure GPIO shutdown overlay (`gpio-shutdown,gpio_pin=4`, Pi4/Pi5-specific params)
4. Create config file (if not exists)
5. Download binary from latest release
6. Install systemd service
7. Enable and start service

Installs cleanly over existing C installation — same binary path, same service name, same config file. The new `temp_unit` config key is optional, so existing config files work unchanged.

Supports `curl | bash` installation pattern.

## License

- `LICENSE`: Clean MIT license, copyright Patrick Sunday
- `THIRD_PARTY_LICENSES`: Spleen font BSD 2-Clause license text
- No UCTRONICS attribution (no original code remains)

## Hardware Reference

`internal/st7735/README.md` — moved from `hardware/st7735/` to live alongside the driver code it documents. Preserved and updated as needed. Documents I2C protocol details, timing constraints, and hardware gotchas that inform driver implementation.
