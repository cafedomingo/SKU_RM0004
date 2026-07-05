# ST7735 TFT LCD Driver

## Hardware

- UCTRONICS SKU_RM0004 front panel: 160x80 pixel ST7735 TFT LCD
- The panel is not wired to the Pi directly: a bridge MCU translates I2C to the panel
- Color format: RGB565, 2 bytes per pixel; full framebuffer 25,600 bytes (160 x 80 x 2)
- The controller addresses a 160x160 panel; our 160x80 window starts at row 24 (`yOffset`)

```
 Pi (this driver)  ──/dev/i2c-1 @ 400kHz──►  bridge MCU (addr 0x18)  ──►  ST7735 panel
```

- Bus config: `dtparam=i2c_arm=on,i2c_arm_baudrate=400000` in `/boot/firmware/config.txt`
  - **Gotcha:** bare `i2c_arm_baudrate=X` lines (without the `dtparam=` prefix) are silently ignored

## Bridge protocol

Commands are 3-byte I2C writes `[register, hi, lo]`. Pixel data flows in burst mode
as raw chunks of at most `burstMaxLen` (160) bytes, big-endian RGB565.

| Register | Meaning                         |
|----------|---------------------------------|
| `0x2A`   | column window `[x0, x1]`        |
| `0x2B`   | row window `[y0+24, y1+24]`     |
| `0x2C`   | pixel data start                |
| `0x03`   | sync (command commit)           |
| `0x01`   | burst mode on (`1`) / off (`0`) |

One `SendRegion` on the wire:

```
0x2A x0 x1        ┐
0x2B y0 y1        │ address window
0x2C 0  0         │
0x03 0  1         ┘ sync
0x01 0  1         burst on
<=160-byte chunk  ┐
450µs delay       │ repeated until the region's pixels are sent
...               ┘
0x01 0  0         burst off
0x03 0  1         sync
```

## Timing (measured on hardware, 2026-07)

| What                    | Nominal @ 400kHz | Measured           |
|-------------------------|------------------|--------------------|
| 160-byte chunk write(2) | 3.6ms            | 7.7ms p50          |
| 450µs `time.Sleep`      | 450µs            | ~1.15ms            |
| 3-byte command write    | ~110µs           | ~200µs             |
| Full frame (160 chunks) | ~650ms           | ~1.3s              |

- The bridge MCU clock-stretches SCL ~26µs/byte while it processes pixels, on top of
  the 22.5µs/byte transfer. Effective throughput is ~19 kB/s and is set by the MCU,
  not by anything host-side.
- Per-region fixed overhead: 7 command writes + trailing delay ≈ 2.55ms,
  equivalent to ~26px of pixel data. This calibrates `splitGapMinPixels`.

## Burst tuning (do not change without hardware testing)

- `burstMaxLen = 160`: hardware limit, larger chunks scramble the panel
- `burstDelayUS = 450`, tuned empirically at 400kHz:
  - 300µs: garbled output
  - 400µs: intermittent color errors
  - **450µs: stable (current value)**
  - 500µs: stable but unnecessarily slow
- The tuning above went through `time.Sleep` overshoot, so the proven floor is the
  ~1.1ms *actual* gap. Do not "fix" the sleep with a precise timer or busy-wait:
  a true 450µs gap is below the floor ever tested stable.

## Dead ends (tested on hardware, do not retry)

- **Sync is not a latch.** The bridge streams pixels to the panel as chunks arrive;
  deferring or omitting all `0x03` writes changes nothing visually and produces no
  I2C errors. Atomic (tear-free) updates are impossible with this firmware.
- **The delay cannot be tightened.** See timing above; clock stretching dominates.

Consequence: redraws are visibly progressive. The only lever is sending fewer bytes,
which is what `DiffRegions` is for.

## Diffing

`drawChanged` (internal/screen) keeps a front buffer (what the panel shows) and a
back buffer (what it should show):

```
Screen.Update() ──► back buffer
Screen.Draw()   ──► DiffRegions(front, back) ──► []Region ──► SendRegion(r, back) each
                    front = back
```

`DiffRegions` produces trimmed dirty rectangles in two passes:

```
1. Rows: consecutive dirty rows coalesce into a strip.

      row dirty?                      strips
   y=14  yes   ┐
   y=15  yes   ├──►  {Y:14, H:3}
   y=16  yes   ┘
   y=17  no
   y=37  yes   ───►  {Y:37, H:1}

2. Columns, per strip: dirty columns form runs; a clean gap splits the strip
   when gapWidth x stripHeight >= splitGapMinPixels (32), i.e. when skipping
   the gap saves more than another region's ~2.35ms overhead.

   columns:   0........77  78..81  82........159
   strip:     ██████████    gap    ████████████
              └ region 1 ┘         └ region 2 ┘

   Small gaps stay merged:   ████ .. ████   ──►  one region (cheaper to resend
   the gap than to re-frame)
```

The two-column screen layouts (gutter at x=78..81, 18px tall graph strips:
gap area 72 >= 32) split naturally into per-column regions.

`SendRegion` serializes the rectangle row by row straight from the framebuffer
(rows are `Width` pixels apart in memory), so callers never copy pixels.
