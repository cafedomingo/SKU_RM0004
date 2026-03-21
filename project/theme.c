#include "theme.h"
#include "st7735.h"

/* clang-format off */
static const Theme modus_vivendi = {
    .bg    = ST7735_COLOR565(0x00, 0x00, 0x00),
    .fg    = ST7735_COLOR565(0xFF, 0xFF, 0xFF),
    .sep   = ST7735_COLOR565(0x3A, 0x3A, 0x3A),
    .ip    = ST7735_COLOR565(0x79, 0xA8, 0xFF),
    .alert = ST7735_COLOR565(0xFF, 0x80, 0x59),
    .ok    = ST7735_COLOR565(0x44, 0xBC, 0x44),
    .warn  = ST7735_COLOR565(0xD0, 0xBC, 0x00),
    .crit  = ST7735_COLOR565(0xFF, 0x80, 0x59),
    .tempRamp = {
        ST7735_COLOR565(0x2F, 0xAF, 0xFF),
        ST7735_COLOR565(0x00, 0xD3, 0xD0),
        ST7735_COLOR565(0xD0, 0xBC, 0x00),
        ST7735_COLOR565(0xFF, 0x80, 0x59),
    },
};

/* clang-format on */

const Theme theme = modus_vivendi;

uint16_t threshold_color(uint32_t value, uint32_t warn_th, uint32_t crit_th) {
    if (value >= crit_th) return theme.crit;
    if (value >= warn_th) return theme.warn;
    return theme.ok;
}

uint16_t lerp_color(uint16_t a, uint16_t b, float t) {
    int ar = (a >> 11) & 0x1F, ag = (a >> 5) & 0x3F, ab = a & 0x1F;
    int br = (b >> 11) & 0x1F, bg = (b >> 5) & 0x3F, bb = b & 0x1F;
    int r = ar + (int)((br - ar) * t);
    int g = ag + (int)((bg - ag) * t);
    int bv = ab + (int)((bb - ab) * t);
    return (r << 11) | (g << 5) | bv;
}

static float clampf(float v, float lo, float hi) { return v < lo ? lo : v > hi ? hi : v; }

uint16_t temp_ramp_color(uint8_t temp_c) {
    if (temp_c <= TEMP_COOL)
        return lerp_color(theme.tempRamp[0], theme.tempRamp[1],
                          clampf(((int)temp_c - TEMP_COLD) / (float)(TEMP_COOL - TEMP_COLD), 0.0f, 1.0f));
    if (temp_c <= TEMP_WARM)
        return lerp_color(theme.tempRamp[1], theme.tempRamp[2], (temp_c - TEMP_COOL) / (float)(TEMP_WARM - TEMP_COOL));
    if (temp_c <= TEMP_HOT)
        return lerp_color(theme.tempRamp[2], theme.tempRamp[3], (temp_c - TEMP_WARM) / (float)(TEMP_HOT - TEMP_WARM));
    return theme.tempRamp[3];
}
