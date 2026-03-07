#include "st7735.h"
#include <fcntl.h>
#include <linux/i2c-dev.h>
#include <linux/i2c.h>
#include <stdio.h>
#include <string.h>
#include <sys/ioctl.h>
#include <unistd.h>

/* I2C configuration */
#define I2C_ADDRESS      0x18
#define BURST_MAX_LENGTH 160

/* Display coordinate limits */
#define X_COORDINATE_MAX 160
#define X_COORDINATE_MIN 0
#define Y_COORDINATE_MAX 80
#define Y_COORDINATE_MIN 0

/* I2C register addresses */
#define X_COORDINATE_REG   0X2A
#define Y_COORDINATE_REG   0X2B
#define CHAR_DATA_REG      0X2C
#define SCAN_DIRECTION_REG 0x36
#define WRITE_DATA_REG     0x00
#define BURST_WRITE_REG    0X01
#define SYNC_REG           0X03

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
    char i2c_path[] = "/dev/i2c-1";

    i2cd = open(i2c_path, O_RDWR);
    if (i2cd < 0) {
        fprintf(stderr, "Device I2C-1 failed to initialize\n");
        return 1;
    }
    if (ioctl(i2cd, I2C_SLAVE_FORCE, I2C_ADDRESS) < 0) {
        fprintf(stderr, "st7735: ioctl I2C_SLAVE_FORCE failed\n");
        return 1;
    }
    return 0;
}

/* I2C transport */

/*
 * Write a two-byte data value over I2C.
 */
static void i2c_write_data(uint8_t high, uint8_t low) {
    uint8_t msg[3] = {WRITE_DATA_REG, high, low};
    if (write(i2cd, msg, 3) != 3) fprintf(stderr, "st7735: i2c_write_data failed\n");
    usleep(10);
}

/*
 * Write a command with a two-byte argument over I2C.
 */
static void i2c_write_command(uint8_t command, uint8_t high, uint8_t low) {
    uint8_t msg[3] = {command, high, low};
    if (write(i2cd, msg, 3) != 3) fprintf(stderr, "st7735: i2c_write_command failed\n");
    usleep(10);
}

/*
 * Transfer a large buffer over I2C in BURST_MAX_LENGTH-byte chunks.
 */
static void i2c_burst_transfer(uint8_t *buff, uint32_t length) {
    uint32_t count = 0;
    i2c_write_command(BURST_WRITE_REG, 0x00, 0x01);
    while (length > count) {
        uint32_t chunk = ((length - count) > BURST_MAX_LENGTH) ? BURST_MAX_LENGTH : (length - count);
        ssize_t written = write(i2cd, buff + count, chunk);
        if (written < 0) {
            fprintf(stderr, "st7735: burst write failed at offset %u\n", count);
            break;
        }
        count += (uint32_t)written;
        usleep(700);
    }
    i2c_write_command(BURST_WRITE_REG, 0x00, 0x00);
    i2c_write_command(SYNC_REG, 0x00, 0x01);
}

/* Drawing primitives */

/*
 * Set display coordinates
 */
static void lcd_set_address_window(uint8_t x0, uint8_t y0, uint8_t x1, uint8_t y1) {
    /* col address set */
    i2c_write_command(X_COORDINATE_REG, x0 + ST7735_XSTART, x1 + ST7735_XSTART);
    /* row address set */
    i2c_write_command(Y_COORDINATE_REG, y0 + ST7735_YSTART, y1 + ST7735_YSTART);
    /* write to RAM */
    i2c_write_command(CHAR_DATA_REG, 0x00, 0x00);

    i2c_write_command(SYNC_REG, 0x00, 0x01);
}

/*
 * Draw a rectangular image from a raw pixel buffer.
 */
static void lcd_draw_image(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint8_t *data) {
    lcd_set_address_window(x, y, x + w - 1, y + h - 1);
    i2c_burst_transfer(data, sizeof(uint16_t) * w * h);
}

/*
 * Fill rectangle
 */
void lcd_fill_rectangle(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint16_t color) {
    uint8_t buff[320] = {0};
    uint16_t count = 0;
    /* clipping */
    if ((x >= ST7735_WIDTH) || (y >= ST7735_HEIGHT)) return;
    if ((x + w) >= ST7735_WIDTH) w = ST7735_WIDTH - x;
    if ((y + h) >= ST7735_HEIGHT) h = ST7735_HEIGHT - y;
    lcd_set_address_window(x, y, x + w - 1, y + h - 1);

    for (count = 0; count < w; count++) {
        buff[count * 2] = color >> 8;
        buff[count * 2 + 1] = color & 0xFF;
    }
    for (y = h; y > 0; y--) {
        i2c_burst_transfer(buff, sizeof(uint16_t) * w);
    }
}

/*
 * Fill screen
 */
void lcd_fill_screen(uint16_t color) {
    lcd_fill_rectangle(0, 0, ST7735_WIDTH, ST7735_HEIGHT, color);
    i2c_write_command(SYNC_REG, 0x00, 0x01);
}

/*
 * Draw a small horizontal progress bar (0-100%).
 */
void lcd_draw_bar(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint8_t val, uint16_t color) {
    uint16_t filled = (uint16_t)val * w / 100;
    if (filled > w) filled = w;
    if (filled > 0) lcd_fill_rectangle(x, y, filled, h, color);
    if (filled < w) lcd_fill_rectangle(x + filled, y, w - filled, h, ST7735_GRAY);
}

/* Text */

/*
 * Display a single character
 */
static void lcd_write_char(uint16_t x, uint16_t y, char ch, FontDef font, uint16_t color, uint16_t bgcolor) {
    uint8_t buff[16 * 26 * 2]; /* max font size: 16x26 */
    uint32_t i, b, j, idx;

    for (i = 0; i < font.height; i++) {
        b = font.data[(ch - 32) * font.height + i];
        for (j = 0; j < font.width; j++) {
            idx = (i * font.width + j) * 2;
            uint16_t c = ((b << j) & 0x8000) ? color : bgcolor;
            buff[idx] = c >> 8;
            buff[idx + 1] = c & 0xFF;
        }
    }

    lcd_draw_image(x, y, font.width, font.height, buff);
}

/*
 * Display a string
 */
void lcd_write_string(uint16_t x, uint16_t y, char *str, FontDef font, uint16_t color, uint16_t bgcolor) {

    while (*str) {
        if (x + font.width >= ST7735_WIDTH) {
            x = 0;
            y += font.height;
            if (y + font.height >= ST7735_HEIGHT) {
                break;
            }

            if (*str == ' ') {
                /* skip spaces in the beginning of the new line */
                str++;
                continue;
            }
        }

        lcd_write_char(x, y, *str, font, color, bgcolor);
        x += font.width;
        str++;
    }
}
