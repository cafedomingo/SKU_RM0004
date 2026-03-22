#ifndef __ST7735_H__
#define __ST7735_H__

#include "fonts.h"
#include <stdbool.h>

/* ST7735 160x80 display dimensions */
#define ST7735_WIDTH  160
#define ST7735_HEIGHT 80

/* Color definitions */
#define ST7735_COLOR565(r, g, b) (((r & 0xF8) << 8) | ((g & 0xFC) << 3) | ((b & 0xF8) >> 3))

#define ST7735_BLACK   0x0000
#define ST7735_BLUE    0x001F
#define ST7735_CYAN    0x07FF
#define ST7735_GRAY    0x8410
#define ST7735_GREEN   0x07E0
#define ST7735_MAGENTA 0xF81F
#define ST7735_ORANGE  0xFD20
#define ST7735_RED     0xF800
#define ST7735_VIOLET  0xB41F
#define ST7735_WHITE   0xFFFF
#define ST7735_YELLOW  0xFFE0

/* Lifecycle */
extern uint8_t lcd_begin(void);

/* Drawing primitives */
extern void lcd_draw_bar(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint8_t val, uint16_t color);
extern void lcd_fill_rectangle(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint16_t color);
extern void lcd_fill_screen(uint16_t color);

/* Bulk transfer */
extern void lcd_draw_fullscreen(uint8_t *buf);
extern void lcd_draw_region(uint8_t *buf, uint16_t x, uint16_t y, uint16_t w, uint16_t h);

/* Text */
extern void lcd_write_string(uint16_t x, uint16_t y, const char *str, FontDef font, uint16_t color, uint16_t bgcolor);

extern void lcd_draw_diamond(uint16_t x, uint16_t y, uint16_t color);

/* 6x6 diamond shape data: {x_offset, width} per row */
extern const uint8_t diamond_shape[][2];
#define DIAMOND_ROWS 6

/* Framebuffer drawing primitives */
extern void lcd_fb_fill(uint8_t *fb, uint16_t color);
extern void lcd_fb_pixel(uint8_t *fb, uint16_t x, uint16_t y, uint16_t color);
extern void lcd_fb_rect(uint8_t *fb, uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint16_t color);
extern void lcd_fb_char(uint8_t *fb, uint16_t x, uint16_t y, char ch, FontDef font, uint16_t color);
extern void lcd_fb_string(uint8_t *fb, uint16_t x, uint16_t y, const char *str, FontDef font, uint16_t color);
extern void lcd_fb_diamond(uint8_t *fb, uint16_t x, uint16_t y, uint16_t color);

#endif /* __ST7735_H__ */
