#include "st7735.h"
#include <stdio.h>
#include <string.h>

static int tests_run = 0;
static int tests_failed = 0;

#define ASSERT(cond, msg)                                                                                              \
    do {                                                                                                               \
        tests_run++;                                                                                                   \
        if (!(cond)) {                                                                                                 \
            fprintf(stderr, "  FAIL: %s (%s:%d)\n", msg, __FILE__, __LINE__);                                          \
            tests_failed++;                                                                                            \
        } else {                                                                                                       \
            printf("  ok: %s\n", msg);                                                                                 \
        }                                                                                                              \
    } while (0)

#define FB_SIZE (ST7735_WIDTH * ST7735_HEIGHT * 2)

static uint8_t fb[FB_SIZE];

static uint16_t fb_get(uint16_t x, uint16_t y) {
    uint32_t off = (y * ST7735_WIDTH + x) * 2;
    return (uint16_t)(fb[off] << 8) | fb[off + 1];
}

/* ── lcd_fb_fill ────────────────────────────────────────────────── */

static void test_fb_fill(void) {
    lcd_fb_fill(fb, 0x1234);
    ASSERT(fb_get(0, 0) == 0x1234, "fill sets first pixel");
    ASSERT(fb_get(ST7735_WIDTH - 1, ST7735_HEIGHT - 1) == 0x1234, "fill sets last pixel");
    ASSERT(fb_get(80, 40) == 0x1234, "fill sets middle pixel");
}

/* ── lcd_fb_pixel ───────────────────────────────────────────────── */

static void test_fb_pixel(void) {
    lcd_fb_fill(fb, ST7735_BLACK);

    lcd_fb_pixel(fb, 10, 20, 0xABCD);
    ASSERT(fb_get(10, 20) == 0xABCD, "pixel writes correct color");
    ASSERT(fb_get(11, 20) == ST7735_BLACK, "pixel does not bleed right");
    ASSERT(fb_get(10, 21) == ST7735_BLACK, "pixel does not bleed down");

    /* Out-of-bounds should be silently ignored */
    lcd_fb_pixel(fb, ST7735_WIDTH, 0, 0xFFFF);
    lcd_fb_pixel(fb, 0, ST7735_HEIGHT, 0xFFFF);
    ASSERT(fb_get(0, 0) == ST7735_BLACK, "out-of-bounds pixel is ignored");
}

/* ── lcd_fb_rect ────────────────────────────────────────────────── */

static void test_fb_rect(void) {
    lcd_fb_fill(fb, ST7735_BLACK);

    lcd_fb_rect(fb, 5, 10, 3, 2, ST7735_WHITE);
    ASSERT(fb_get(5, 10) == ST7735_WHITE, "rect top-left corner");
    ASSERT(fb_get(7, 11) == ST7735_WHITE, "rect bottom-right corner");
    ASSERT(fb_get(4, 10) == ST7735_BLACK, "rect does not bleed left");
    ASSERT(fb_get(8, 10) == ST7735_BLACK, "rect does not bleed right");
    ASSERT(fb_get(5, 9) == ST7735_BLACK, "rect does not bleed up");
    ASSERT(fb_get(5, 12) == ST7735_BLACK, "rect does not bleed down");

    /* Rect clipped at screen edge */
    lcd_fb_fill(fb, ST7735_BLACK);
    lcd_fb_rect(fb, ST7735_WIDTH - 2, ST7735_HEIGHT - 2, 10, 10, ST7735_RED);
    ASSERT(fb_get(ST7735_WIDTH - 1, ST7735_HEIGHT - 1) == ST7735_RED, "rect clipped at edge draws");
    ASSERT(fb_get(ST7735_WIDTH - 3, ST7735_HEIGHT - 1) == ST7735_BLACK, "rect clipped does not bleed");
}

/* ── lcd_fb_diamond ─────────────────────────────────────────────── */

static void test_fb_diamond(void) {
    lcd_fb_fill(fb, ST7735_BLACK);

    lcd_fb_diamond(fb, 20, 30, ST7735_GREEN);

    /* Top row: 2 pixels wide starting at x+2 */
    ASSERT(fb_get(22, 30) == ST7735_GREEN, "diamond top-left pixel");
    ASSERT(fb_get(23, 30) == ST7735_GREEN, "diamond top-right pixel");
    ASSERT(fb_get(21, 30) == ST7735_BLACK, "diamond top row left edge clear");

    /* Middle row: 6 pixels wide starting at x+0 */
    ASSERT(fb_get(20, 32) == ST7735_GREEN, "diamond mid-left pixel");
    ASSERT(fb_get(25, 32) == ST7735_GREEN, "diamond mid-right pixel");

    /* Bottom row: 2 pixels wide starting at x+2 */
    ASSERT(fb_get(22, 35) == ST7735_GREEN, "diamond bottom-left pixel");
    ASSERT(fb_get(21, 35) == ST7735_BLACK, "diamond bottom row left edge clear");

    /* Outside diamond */
    ASSERT(fb_get(20, 30) == ST7735_BLACK, "diamond corner is empty");
}

/* ── lcd_fb_string ──────────────────────────────────────────────── */

static void test_fb_string(void) {
    lcd_fb_fill(fb, ST7735_BLACK);

    lcd_fb_string(fb, 0, 0, "A", Font_7x10, ST7735_WHITE);

    /* Font_7x10 'A' should have some white pixels in its bounding box */
    int found = 0;
    for (uint16_t y = 0; y < 10; y++)
        for (uint16_t x = 0; x < 7; x++)
            if (fb_get(x, y) == ST7735_WHITE) found++;
    ASSERT(found > 0, "string renders visible pixels");

    /* Outside the glyph bounding box should be black */
    ASSERT(fb_get(7, 0) == ST7735_BLACK, "string does not bleed past glyph width");
    ASSERT(fb_get(0, 10) == ST7735_BLACK, "string does not bleed past glyph height");

    /* Multi-char: second char starts at x=7 */
    lcd_fb_fill(fb, ST7735_BLACK);
    lcd_fb_string(fb, 0, 0, "AB", Font_7x10, ST7735_WHITE);
    int found2 = 0;
    for (uint16_t y = 0; y < 10; y++)
        for (uint16_t x = 7; x < 14; x++)
            if (fb_get(x, y) == ST7735_WHITE) found2++;
    ASSERT(found2 > 0, "second char renders visible pixels");

    /* Unprintable char replaced with '?' */
    lcd_fb_fill(fb, ST7735_BLACK);
    lcd_fb_string(fb, 0, 0, "\x01", Font_7x10, ST7735_WHITE);
    int found_q = 0;
    for (uint16_t y = 0; y < 10; y++)
        for (uint16_t x = 0; x < 7; x++)
            if (fb_get(x, y) == ST7735_WHITE) found_q++;
    ASSERT(found_q > 0, "unprintable char renders as '?'");
}

int main(void) {
    printf("st7735_fb tests:\n");

    test_fb_fill();
    test_fb_pixel();
    test_fb_rect();
    test_fb_diamond();
    test_fb_string();

    printf("\n%d/%d tests passed.\n", tests_run - tests_failed, tests_run);
    return tests_failed > 0 ? 1 : 0;
}
