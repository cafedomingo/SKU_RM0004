/* vim: set ai et ts=4 sw=4: */
#include "st7735.h"
#include "rpiInfo.h"
#include <fcntl.h>
#include <linux/i2c-dev.h>
#include <linux/i2c.h>
#include <stdio.h>
#include <string.h>
#include <sys/ioctl.h>
#include <unistd.h>

int i2cd;

/*
 * Set display coordinates
 */
void lcd_set_address_window(uint8_t x0, uint8_t y0, uint8_t x1, uint8_t y1) {
    // col address set
    i2c_write_command(X_COORDINATE_REG, x0 + ST7735_XSTART, x1 + ST7735_XSTART);
    // row address set
    i2c_write_command(Y_COORDINATE_REG, y0 + ST7735_YSTART, y1 + ST7735_YSTART);
    // write to RAM
    i2c_write_command(CHAR_DATA_REG, 0x00, 0x00);

    i2c_write_command(SYNC_REG, 0x00, 0x01);
}

/*
 * Display a single character
 */
void lcd_write_char(uint16_t x, uint16_t y, char ch, FontDef font,
                    uint16_t color, uint16_t bgcolor) {
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
void lcd_write_string(uint16_t x, uint16_t y, char *str, FontDef font,
                      uint16_t color, uint16_t bgcolor) {

    while (*str) {
        if (x + font.width >= ST7735_WIDTH) {
            x = 0;
            y += font.height;
            if (y + font.height >= ST7735_HEIGHT) {
                break;
            }

            if (*str == ' ') {
                // skip spaces in the beginning of the new line
                str++;
                continue;
            }
        }

        lcd_write_char(x, y, *str, font, color, bgcolor);
        x += font.width;
        str++;
    }
}

/*
 * Fill rectangle
 */
void lcd_fill_rectangle(uint16_t x, uint16_t y, uint16_t w, uint16_t h,
                        uint16_t color) {
    uint8_t buff[320] = {0};
    uint16_t count = 0;
    // clipping
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
 * fill screen
 */

void lcd_fill_screen(uint16_t color) {
    lcd_fill_rectangle(0, 0, ST7735_WIDTH, ST7735_HEIGHT, color);
    i2c_write_command(SYNC_REG, 0x00, 0x01);
}

void lcd_draw_image(uint16_t x, uint16_t y, uint16_t w, uint16_t h,
                    uint8_t *data) {
    lcd_set_address_window(x, y, x + w - 1, y + h - 1);
    i2c_burst_transfer(data, sizeof(uint16_t) * w * h);
}

uint8_t lcd_begin(void) {
    char i2c_path[] = "/dev/i2c-1";

    i2cd = open(i2c_path, O_RDWR);
    if (i2cd < 0) {
        fprintf(stderr, "Device I2C-1 failed to initialize\n");
        return 1;
    }
    if (ioctl(i2cd, I2C_SLAVE_FORCE, I2C_ADDRESS) < 0) {
        return 1;
    }
    return 0;
}

void i2c_write_data(uint8_t high, uint8_t low) {
    uint8_t msg[3] = {WRITE_DATA_REG, high, low};
    (void)write(i2cd, msg, 3);
    usleep(10);
}

void i2c_write_command(uint8_t command, uint8_t high, uint8_t low) {
    uint8_t msg[3] = {command, high, low};
    (void)write(i2cd, msg, 3);
    usleep(10);
}

void i2c_burst_transfer(uint8_t *buff, uint32_t length) {
    uint32_t count = 0;
    i2c_write_command(BURST_WRITE_REG, 0x00, 0x01);
    while (length > count) {
        if ((length - count) > BURST_MAX_LENGTH) {
            (void)write(i2cd, buff + count, BURST_MAX_LENGTH);
            count += BURST_MAX_LENGTH;
        } else {
            (void)write(i2cd, buff + count, length - count);
            count += (length - count);
        }
        usleep(700);
    }
    i2c_write_command(BURST_WRITE_REG, 0x00, 0x00);
    i2c_write_command(SYNC_REG, 0x00, 0x01);
}

void lcd_display_mini_bar(uint16_t x, uint16_t y, uint16_t w, uint16_t h,
                          uint8_t val, uint16_t color) {
    uint16_t filled = (uint16_t)val * w / 100;
    if (filled > w) filled = w;
    if (filled > 0) lcd_fill_rectangle(x, y, filled, h, color);
    if (filled < w)
        lcd_fill_rectangle(x + filled, y, w - filled, h, ST7735_GRAY);
}

static uint16_t threshold_color(uint8_t val) {
    if (val < 60) return ST7735_GREEN;
    if (val < 80) return ST7735_YELLOW;
    if (val < 90) return ST7735_COLOR565(255, 165, 0); /* orange */
    return ST7735_RED;
}

static uint16_t temp_threshold_color(uint8_t celsius) {
    if (celsius < 40) return ST7735_CYAN;
    if (celsius < 50) return ST7735_GREEN;
    if (celsius < 60) return ST7735_YELLOW;
    if (celsius < 70) return ST7735_COLOR565(255, 165, 0); /* orange */
    return ST7735_RED;
}

#define ST7735_VIOLET ST7735_COLOR565(180, 130, 255)
#define BAR_WIDTH 65
#define BAR_HEIGHT 6

static void draw_metric(uint16_t x, uint16_t y, const char *label,
                        const char *value, uint8_t bar_pct, uint16_t color) {
    uint16_t val_x = x + BAR_WIDTH -
                     strlen(value) * Font_7x10.width; /* right-align with bar */
    lcd_write_string(x, y, (char *)label, Font_7x10, ST7735_WHITE,
                     ST7735_BLACK);
    lcd_write_string(val_x, y, (char *)value, Font_7x10, color, ST7735_BLACK);
    lcd_display_mini_bar(x, y + 12, BAR_WIDTH, BAR_HEIGHT, bar_pct, color);
}

void lcd_display_all(void) {
    char buf[24];
    char hostBuf[17];
    uint8_t tempForBar;
    uint16_t color;

    /* Gather all data */
    uint8_t cpuPercent = get_cpu_percent();
    uint8_t ramPercent = get_ram_percent();
    uint8_t temp = get_temperature();
    uint8_t diskPercent = get_disk_percent();
    const char *hostname = get_hostname();
    const char *ip = get_ip_address();
    int dietpi_status = get_dietpi_update_status();
    int apt_count = get_apt_update_count();

    /* Header: hostname, IP, separator */
    strncpy(hostBuf, hostname, 16);
    hostBuf[16] = '\0';
    lcd_write_string(2, 0, hostBuf, Font_8x16, ST7735_WHITE, ST7735_BLACK);

    lcd_write_string(2, 18, (char *)ip, Font_7x10, ST7735_VIOLET, ST7735_BLACK);

    lcd_fill_rectangle(0, 30, ST7735_WIDTH, 1, ST7735_BLUE);

    /* DietPi status dot — red when update needed, hidden otherwise */
    if (dietpi_status == 2) {
        lcd_fill_rectangle(154, 5, 2, 1, ST7735_RED);
        lcd_fill_rectangle(153, 6, 4, 1, ST7735_RED);
        lcd_fill_rectangle(152, 7, 6, 1, ST7735_RED);
        lcd_fill_rectangle(152, 8, 6, 1, ST7735_RED);
        lcd_fill_rectangle(153, 9, 4, 1, ST7735_RED);
        lcd_fill_rectangle(154, 10, 2, 1, ST7735_RED);
    }

    /* APT update count — right-aligned on IP row */
    lcd_fill_rectangle(124, 18, 36, 10, ST7735_BLACK);
    if (apt_count > 0) {
        char badge[5];
        sprintf(badge, "^%d", apt_count > 99 ? 99 : apt_count);
        uint16_t color = (apt_count >= 10) ? ST7735_RED : ST7735_YELLOW;
        uint16_t bx = ST7735_WIDTH - strlen(badge) * Font_7x10.width - 2;
        lcd_write_string(bx, 19, badge, Font_7x10, color, ST7735_BLACK);
    }

    /* CPU */
    color = threshold_color(cpuPercent);
    sprintf(buf, "%3d%%", cpuPercent);
    draw_metric(2, 34, "CPU:", buf, cpuPercent, color);

    /* RAM */
    color = threshold_color(ramPercent);
    sprintf(buf, "%3d%%", ramPercent);
    draw_metric(2, 56, "RAM:", buf, ramPercent, color);

    /* Temperature */
    tempForBar = temp;
    if (TEMPERATURE_TYPE == FAHRENHEIT) tempForBar = (temp - 32) / 1.8;
    color = temp_threshold_color(tempForBar);
    sprintf(buf, "%3d%c", temp, TEMPERATURE_TYPE == FAHRENHEIT ? 'F' : 'C');
    draw_metric(84, 34, "TEMP:", buf, tempForBar > 100 ? 100 : tempForBar,
                color);

    /* Disk */
    color = threshold_color(diskPercent);
    sprintf(buf, "%3d%%", diskPercent);
    draw_metric(84, 56, "DISK:", buf, diskPercent, color);
}
