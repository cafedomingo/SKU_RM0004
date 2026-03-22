#include "st7735.h"
#include <string.h>

/* Global display framebuffer — captures all rendering for screenshot export */
uint8_t display_fb[ST7735_WIDTH * ST7735_HEIGHT * 2];

uint8_t lcd_begin(void) { return 0; }

/*
 * Framebuffer primitives below are copied from hardware/st7735/st7735.c.
 * Keep in sync if the real driver's fb functions change.
 */

void lcd_fb_fill(uint8_t *fb, uint16_t color) {
    uint8_t hi = color >> 8, lo = color & 0xFF;
    for (uint32_t i = 0; i < ST7735_WIDTH * ST7735_HEIGHT * 2; i += 2) {
        fb[i] = hi;
        fb[i + 1] = lo;
    }
}

void lcd_fb_pixel(uint8_t *fb, uint16_t x, uint16_t y, uint16_t color) {
    if (x >= ST7735_WIDTH || y >= ST7735_HEIGHT) return;
    uint32_t off = (y * ST7735_WIDTH + x) * 2;
    fb[off] = color >> 8;
    fb[off + 1] = color & 0xFF;
}

void lcd_fb_rect(uint8_t *fb, uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint16_t color) {
    for (uint16_t row = y; row < y + h && row < ST7735_HEIGHT; row++)
        for (uint16_t col = x; col < x + w && col < ST7735_WIDTH; col++)
            lcd_fb_pixel(fb, col, row, color);
}

static const uint8_t diamond_shape[][2] = {
    {2, 2}, {1, 4}, {0, 6}, {0, 6}, {1, 4}, {2, 2},
};
#define DIAMOND_ROWS (sizeof(diamond_shape) / sizeof(diamond_shape[0]))

void lcd_fb_diamond(uint8_t *fb, uint16_t x, uint16_t y, uint16_t color) {
    for (uint16_t r = 0; r < DIAMOND_ROWS; r++)
        lcd_fb_rect(fb, x + diamond_shape[r][0], y + r, diamond_shape[r][1], 1, color);
}

void lcd_fb_char(uint8_t *fb, uint16_t x, uint16_t y, char ch, FontDef font, uint16_t color) {
    if (ch < 32 || ch >= 127) ch = '?';
    for (uint16_t row = 0; row < font.height; row++) {
        uint16_t bits = font.data[(ch - 32) * font.height + row];
        for (uint16_t col = 0; col < font.width; col++) {
            if ((bits << col) & 0x8000) lcd_fb_pixel(fb, x + col, y + row, color);
        }
    }
}

void lcd_fb_string(uint8_t *fb, uint16_t x, uint16_t y, const char *str, FontDef font, uint16_t color) {
    while (*str && x + font.width <= ST7735_WIDTH) {
        lcd_fb_char(fb, x, y, *str++, font, color);
        x += font.width;
    }
}

/* ── Display operations (render to global display_fb) ────────────── */

void lcd_fill_screen(uint16_t color) { lcd_fb_fill(display_fb, color); }

void lcd_fill_rectangle(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint16_t color) {
    lcd_fb_rect(display_fb, x, y, w, h, color);
}

void lcd_draw_fullscreen(uint8_t *buf) { memcpy(display_fb, buf, ST7735_WIDTH * ST7735_HEIGHT * 2); }

void lcd_draw_region(uint8_t *buf, uint16_t x, uint16_t y, uint16_t w, uint16_t h) {
    if (x >= ST7735_WIDTH || y >= ST7735_HEIGHT) return;
    if (x + w > ST7735_WIDTH) w = ST7735_WIDTH - x;
    if (y + h > ST7735_HEIGHT) h = ST7735_HEIGHT - y;

    if (x == 0 && w == ST7735_WIDTH) {
        memcpy(display_fb + y * ST7735_WIDTH * 2, buf + y * ST7735_WIDTH * 2, w * h * 2);
    } else {
        for (uint16_t row = y; row < y + h; row++) {
            uint32_t off = (row * ST7735_WIDTH + x) * 2;
            memcpy(display_fb + off, buf + off, w * 2);
        }
    }
}

void lcd_draw_bar(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint8_t val, uint16_t color) {
    if (x >= ST7735_WIDTH || y >= ST7735_HEIGHT) return;
    if (x + w > ST7735_WIDTH) w = ST7735_WIDTH - x;
    if (y + h > ST7735_HEIGHT) h = ST7735_HEIGHT - y;

    uint16_t filled = (uint16_t)val * w / 100;
    if (filled > w) filled = w;
    if (filled > 0) lcd_fb_rect(display_fb, x, y, filled, h, color);
    if (filled < w) lcd_fb_rect(display_fb, x + filled, y, w - filled, h, ST7735_GRAY);
}

void lcd_write_string(uint16_t x, uint16_t y, const char *str, FontDef font, uint16_t color, uint16_t bgcolor) {
    uint16_t len = 0;
    while (str[len] && x + (len + 1) * font.width <= ST7735_WIDTH)
        len++;
    if (len == 0) return;

    lcd_fb_rect(display_fb, x, y, len * font.width, font.height, bgcolor);
    for (uint16_t i = 0; i < len; i++)
        lcd_fb_char(display_fb, x + i * font.width, y, str[i], font, color);
}

void lcd_draw_diamond(uint16_t x, uint16_t y, uint16_t color) { lcd_fb_diamond(display_fb, x, y, color); }
