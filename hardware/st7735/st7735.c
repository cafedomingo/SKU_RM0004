#include "st7735.h"
#include "log.h"
#include <fcntl.h>
#include <linux/i2c-dev.h>
#include <linux/i2c.h>
#include <string.h>
#include <sys/ioctl.h>
#include <unistd.h>

/* I2C configuration */
#define I2C_ADDRESS      0x18
#define BURST_MAX_LENGTH 160
#define BURST_DELAY_US   450

/* I2C bridge register addresses */
#define WRITE_DATA_REG     0x00
#define BURST_WRITE_REG    0x01
#define SYNC_REG           0x03
#define X_COORDINATE_REG   0x2A
#define Y_COORDINATE_REG   0x2B
#define CHAR_DATA_REG      0x2C
#define SCAN_DIRECTION_REG 0x36

/* MADCTL flags */
#define ST7735_MADCTL_MY  0x80
#define ST7735_MADCTL_MX  0x40
#define ST7735_MADCTL_MV  0x20
#define ST7735_MADCTL_ML  0x10
#define ST7735_MADCTL_RGB 0x00
#define ST7735_MADCTL_BGR 0x08
#define ST7735_MADCTL_MH  0x04

/* Display origin offsets */
#define ST7735_XSTART   0
#define ST7735_YSTART   24
#define ST7735_ROTATION (ST7735_MADCTL_MY | ST7735_MADCTL_MV | ST7735_MADCTL_BGR)

/* ST7735 command set */
#define ST7735_NOP     0x00
#define ST7735_SWRESET 0x01
#define ST7735_RDDID   0x04
#define ST7735_RDDST   0x09

#define ST7735_SLPIN  0x10
#define ST7735_SLPOUT 0x11
#define ST7735_PTLON  0x12
#define ST7735_NORON  0x13

#define ST7735_INVOFF  0x20
#define ST7735_INVON   0x21
#define ST7735_DISPOFF 0x28
#define ST7735_DISPON  0x29
#define ST7735_CASET   0x2A
#define ST7735_RASET   0x2B
#define ST7735_RAMWR   0x2C
#define ST7735_RAMRD   0x2E

#define ST7735_PTLAR  0x30
#define ST7735_COLMOD 0x3A
#define ST7735_MADCTL 0x36

#define ST7735_FRMCTR1 0xB1
#define ST7735_FRMCTR2 0xB2
#define ST7735_FRMCTR3 0xB3
#define ST7735_INVCTR  0xB4
#define ST7735_DISSET5 0xB6

#define ST7735_PWCTR1 0xC0
#define ST7735_PWCTR2 0xC1
#define ST7735_PWCTR3 0xC2
#define ST7735_PWCTR4 0xC3
#define ST7735_PWCTR5 0xC4
#define ST7735_VMCTR1 0xC5

#define ST7735_RDID1 0xDA
#define ST7735_RDID2 0xDB
#define ST7735_RDID3 0xDC
#define ST7735_RDID4 0xDD

#define ST7735_PWCTR6 0xFC

#define ST7735_GMCTRP1 0xE0
#define ST7735_GMCTRN1 0xE1

static int i2cd;

/* Lifecycle */

/*
 * Open the I2C bus and configure the LCD slave address.
 * Returns 0 on success, 1 on failure.
 */
uint8_t lcd_begin(void) {
    i2cd = open("/dev/i2c-1", O_RDWR);
    if (i2cd < 0) {
        LOG_ERROR("I2C-1 failed to initialize");
        return 1;
    }
    if (ioctl(i2cd, I2C_SLAVE_FORCE, I2C_ADDRESS) < 0) {
        LOG_ERROR("ioctl I2C_SLAVE_FORCE failed");
        return 1;
    }
    return 0;
}

/* I2C transport */

/*
 * Write a command with a two-byte argument over I2C.
 */
static void i2c_write_command(uint8_t command, uint8_t high, uint8_t low) {
    uint8_t msg[3] = {command, high, low};
    if (write(i2cd, msg, 3) != 3) LOG_ERROR("i2c_write_command failed");
    usleep(10);
}

/*
 * Set the display address window for subsequent pixel writes.
 */
static void lcd_set_address_window(uint8_t x0, uint8_t y0, uint8_t x1, uint8_t y1) {
    i2c_write_command(X_COORDINATE_REG, x0 + ST7735_XSTART, x1 + ST7735_XSTART);
    i2c_write_command(Y_COORDINATE_REG, y0 + ST7735_YSTART, y1 + ST7735_YSTART);
    i2c_write_command(CHAR_DATA_REG, 0x00, 0x00);
    i2c_write_command(SYNC_REG, 0x00, 0x01);
}

/*
 * Burst transfer: open once, send data in chunks, close once.
 * The ST7735 auto-increments through the address window, so all
 * data sent within a burst session fills the window sequentially.
 */
static void burst_begin(void) { i2c_write_command(BURST_WRITE_REG, 0x00, 0x01); }

static void burst_send(const uint8_t *buf, uint32_t length) {
    uint32_t count = 0;
    while (count < length) {
        uint32_t chunk = length - count;
        if (chunk > BURST_MAX_LENGTH) chunk = BURST_MAX_LENGTH;
        ssize_t written = write(i2cd, buf + count, chunk);
        if (written < 0) {
            LOG_ERROR("burst write failed at offset %u", count);
            break;
        }
        count += (uint32_t)written;
        usleep(BURST_DELAY_US);
    }
}

static void burst_end(void) {
    i2c_write_command(BURST_WRITE_REG, 0x00, 0x00);
    i2c_write_command(SYNC_REG, 0x00, 0x01);
}

/* Drawing primitives */

/*
 * Fill a rectangle with a solid color.
 */
void lcd_fill_rectangle(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint16_t color) {
    if (x >= ST7735_WIDTH || y >= ST7735_HEIGHT) return;
    if (x + w > ST7735_WIDTH) w = ST7735_WIDTH - x;
    if (y + h > ST7735_HEIGHT) h = ST7735_HEIGHT - y;

    uint8_t row[ST7735_WIDTH * 2];
    for (uint16_t i = 0; i < w; i++) {
        row[i * 2] = color >> 8;
        row[i * 2 + 1] = color & 0xFF;
    }

    lcd_set_address_window(x, y, x + w - 1, y + h - 1);
    burst_begin();
    for (uint16_t r = 0; r < h; r++)
        burst_send(row, w * 2);
    burst_end();
}

/*
 * Fill the entire screen with a solid color.
 */
void lcd_fill_screen(uint16_t color) { lcd_fill_rectangle(0, 0, ST7735_WIDTH, ST7735_HEIGHT, color); }

/*
 * Send a pre-rendered full-screen pixel buffer in a single I2C burst.
 */
void lcd_draw_fullscreen(uint8_t *buf) {
    lcd_set_address_window(0, 0, ST7735_WIDTH - 1, ST7735_HEIGHT - 1);
    burst_begin();
    burst_send(buf, ST7735_WIDTH * ST7735_HEIGHT * 2);
    burst_end();
}

/*
 * Send a rectangular region from a full-screen framebuffer.
 * For full-width strips the data is contiguous, so it goes as one burst.
 */
void lcd_draw_region(uint8_t *buf, uint16_t x, uint16_t y, uint16_t w, uint16_t h) {
    if (x >= ST7735_WIDTH || y >= ST7735_HEIGHT) return;
    if (x + w > ST7735_WIDTH) w = ST7735_WIDTH - x;
    if (y + h > ST7735_HEIGHT) h = ST7735_HEIGHT - y;

    lcd_set_address_window(x, y, x + w - 1, y + h - 1);
    burst_begin();
    if (x == 0 && w == ST7735_WIDTH) {
        burst_send(buf + y * ST7735_WIDTH * 2, w * h * 2);
    } else {
        for (uint16_t row = y; row < y + h; row++)
            burst_send(buf + (row * ST7735_WIDTH + x) * 2, w * 2);
    }
    burst_end();
}

/*
 * Draw a horizontal progress bar (0-100%).
 */
void lcd_draw_bar(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint8_t val, uint16_t color) {
    if (x >= ST7735_WIDTH || y >= ST7735_HEIGHT) return;
    if (x + w > ST7735_WIDTH) w = ST7735_WIDTH - x;
    if (y + h > ST7735_HEIGHT) h = ST7735_HEIGHT - y;

    uint16_t filled = (uint16_t)val * w / 100;
    if (filled > w) filled = w;

    /* Build one row with both colors */
    uint8_t row[ST7735_WIDTH * 2];
    for (uint16_t i = 0; i < filled; i++) {
        row[i * 2] = color >> 8;
        row[i * 2 + 1] = color & 0xFF;
    }
    for (uint16_t i = filled; i < w; i++) {
        row[i * 2] = ST7735_GRAY >> 8;
        row[i * 2 + 1] = ST7735_GRAY & 0xFF;
    }

    /* Single address window, single burst for the whole bar */
    lcd_set_address_window(x, y, x + w - 1, y + h - 1);
    burst_begin();
    for (uint16_t r = 0; r < h; r++)
        burst_send(row, w * 2);
    burst_end();
}

/* Text */

/*
 * Render a string and send it as a single I2C burst.
 */
void lcd_write_string(uint16_t x, uint16_t y, const char *str, FontDef font, uint16_t color, uint16_t bgcolor) {
    /* Count how many characters fit on this line */
    uint16_t len = 0;
    while (str[len] && x + (len + 1) * font.width <= ST7735_WIDTH)
        len++;
    if (len == 0) return;

    uint16_t w = len * font.width;
    uint16_t h = font.height;

    /* Render all characters into one buffer */
    uint8_t buf[ST7735_WIDTH * 26 * 2]; /* max: full width × tallest font */
    for (uint16_t c = 0; c < len; c++) {
        uint16_t cx = c * font.width;
        char ch = (str[c] >= 32 && str[c] < 127) ? str[c] : '?';
        for (uint16_t row = 0; row < h; row++) {
            uint16_t bits = font.data[(ch - 32) * h + row];
            for (uint16_t col = 0; col < font.width; col++) {
                uint16_t px = ((bits << col) & 0x8000) ? color : bgcolor;
                uint32_t off = (row * w + cx + col) * 2;
                buf[off] = px >> 8;
                buf[off + 1] = px & 0xFF;
            }
        }
    }

    lcd_set_address_window(x, y, x + w - 1, y + h - 1);
    burst_begin();
    burst_send(buf, w * h * 2);
    burst_end();
}

/* Framebuffer drawing primitives */

void lcd_fb_pixel(uint8_t *fb, uint16_t x, uint16_t y, uint16_t color) {
    if (x >= ST7735_WIDTH || y >= ST7735_HEIGHT) return;
    uint32_t off = (y * ST7735_WIDTH + x) * 2;
    fb[off] = color >> 8;
    fb[off + 1] = color & 0xFF;
}

/* 6x6 diamond shape data: {x_offset, width} per row */
static const uint8_t diamond_shape[][2] = {
    {2, 2}, {1, 4}, {0, 6}, {0, 6}, {1, 4}, {2, 2},
};
#define DIAMOND_ROWS (sizeof(diamond_shape) / sizeof(diamond_shape[0]))

void lcd_draw_diamond(uint16_t x, uint16_t y, uint16_t color) {
    for (uint16_t r = 0; r < DIAMOND_ROWS; r++)
        lcd_fill_rectangle(x + diamond_shape[r][0], y + r, diamond_shape[r][1], 1, color);
}

void lcd_fb_rect(uint8_t *fb, uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint16_t color) {
    for (uint16_t row = y; row < y + h && row < ST7735_HEIGHT; row++)
        for (uint16_t col = x; col < x + w && col < ST7735_WIDTH; col++)
            lcd_fb_pixel(fb, col, row, color);
}

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
