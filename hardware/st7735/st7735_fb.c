#include "st7735.h"

/* Framebuffer drawing primitives — pure computation, no hardware dependency */

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

/* 6x6 diamond shape data: {x_offset, width} per row */
const uint8_t diamond_shape[][2] = {
    {2, 2}, {1, 4}, {0, 6}, {0, 6}, {1, 4}, {2, 2},
};

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
