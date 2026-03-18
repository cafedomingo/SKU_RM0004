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

static const Theme modus_vivendi_tinted = {
    .bg    = ST7735_COLOR565(0x0D, 0x0E, 0x1C),
    .fg    = ST7735_COLOR565(0xFF, 0xFF, 0xFF),
    .sep   = ST7735_COLOR565(0x3A, 0x3A, 0x50),
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

static const Theme catppuccin_mocha = {
    .bg    = ST7735_COLOR565(0x1E, 0x1E, 0x2E),
    .fg    = ST7735_COLOR565(0xCD, 0xD6, 0xF4),
    .sep   = ST7735_COLOR565(0x44, 0x46, 0x5E),
    .ip    = ST7735_COLOR565(0xB4, 0xBE, 0xFE),
    .alert = ST7735_COLOR565(0xF3, 0x8B, 0xA8),
    .ok    = ST7735_COLOR565(0xA6, 0xE3, 0xA1),
    .warn  = ST7735_COLOR565(0xF9, 0xE2, 0xAF),
    .crit  = ST7735_COLOR565(0xF3, 0x8B, 0xA8),
    .tempRamp = {
        ST7735_COLOR565(0x89, 0xB4, 0xFA),
        ST7735_COLOR565(0x94, 0xE2, 0xD5),
        ST7735_COLOR565(0xF9, 0xE2, 0xAF),
        ST7735_COLOR565(0xF3, 0x8B, 0xA8),
    },
};

static const Theme rose_pine = {
    .bg    = ST7735_COLOR565(0x19, 0x17, 0x24),
    .fg    = ST7735_COLOR565(0xE0, 0xDE, 0xF4),
    .sep   = ST7735_COLOR565(0x3A, 0x37, 0x55),
    .ip    = ST7735_COLOR565(0xEB, 0xBC, 0xBA),
    .alert = ST7735_COLOR565(0xEB, 0x6F, 0x92),
    .ok    = ST7735_COLOR565(0x9C, 0xCF, 0xD8),
    .warn  = ST7735_COLOR565(0xF6, 0xC1, 0x77),
    .crit  = ST7735_COLOR565(0xEB, 0x6F, 0x92),
    .tempRamp = {
        ST7735_COLOR565(0x4A, 0x90, 0xA8),
        ST7735_COLOR565(0x9C, 0xCF, 0xD8),
        ST7735_COLOR565(0xF6, 0xC1, 0x77),
        ST7735_COLOR565(0xEB, 0x6F, 0x92),
    },
};

static const Theme selenized_dark = {
    .bg    = ST7735_COLOR565(0x10, 0x3C, 0x48),
    .fg    = ST7735_COLOR565(0xAD, 0xBC, 0xBC),
    .sep   = ST7735_COLOR565(0x28, 0x64, 0x70),
    .ip    = ST7735_COLOR565(0x7E, 0xB8, 0xDA),
    .alert = ST7735_COLOR565(0xFF, 0x80, 0x60),
    .ok    = ST7735_COLOR565(0x75, 0xB9, 0x38),
    .warn  = ST7735_COLOR565(0xDB, 0xB3, 0x2D),
    .crit  = ST7735_COLOR565(0xFF, 0x80, 0x60),
    .tempRamp = {
        ST7735_COLOR565(0x46, 0x95, 0xF7),
        ST7735_COLOR565(0x41, 0xC7, 0xB9),
        ST7735_COLOR565(0xDB, 0xB3, 0x2D),
        ST7735_COLOR565(0xFF, 0x80, 0x60),
    },
};
/* clang-format on */

/* Suppress unused-variable warnings for themes not selected */
static const void *theme_refs[] __attribute__((unused)) = {
    &modus_vivendi, &modus_vivendi_tinted, &catppuccin_mocha, &rose_pine, &selenized_dark,
};

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
