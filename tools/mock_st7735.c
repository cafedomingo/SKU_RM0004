#include "st7735.h"
#include <string.h>

/* Global display framebuffer — captures all rendering for screenshot export */
uint8_t display_fb[ST7735_WIDTH * ST7735_HEIGHT * 2];

uint8_t lcd_begin(void) { return 0; }

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
