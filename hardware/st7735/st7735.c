/* vim: set ai et ts=4 sw=4: */
#include "st7735.h"
#include "time.h"
#include <stdio.h>
#include <string.h>
#include <sys/sysinfo.h>
#include <sys/vfs.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/ioctl.h>
#include <netinet/in.h>
#include <net/if.h>
#include <unistd.h>
#include <arpa/inet.h>
#include <sys/ioctl.h>
#include <linux/i2c.h>
#include <linux/i2c-dev.h>
#include <fcntl.h>
#include "rpiInfo.h"

int i2cd;

/*
 * Set display coordinates
 */
void lcd_set_address_window(uint8_t x0, uint8_t y0, uint8_t x1, uint8_t y1)
{
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
void lcd_write_char(uint16_t x, uint16_t y, char ch, FontDef font, uint16_t color, uint16_t bgcolor)
{
    uint8_t buff[16 * 26 * 2]; /* max font size: 16x26 */
    uint32_t i, b, j, idx;

    for (i = 0; i < font.height; i++)
    {
        b = font.data[(ch - 32) * font.height + i];
        for (j = 0; j < font.width; j++)
        {
            idx = (i * font.width + j) * 2;
            uint16_t c = ((b << j) & 0x8000) ? color : bgcolor;
            buff[idx]     = c >> 8;
            buff[idx + 1] = c & 0xFF;
        }
    }

    lcd_draw_image(x, y, font.width, font.height, buff);
}

void lcd_write_ch(uint16_t x, uint16_t y, char ch, FontType font, uint16_t color, uint16_t bgcolor)
{
    switch (font)
    {
    case FontType_7x10:
        lcd_write_char(x, y, ch, Font_7x10, color, bgcolor);
        break;
    case FontType_8x16:
        lcd_write_char(x, y, ch, Font_8x16, color, bgcolor);
        break;
    case FontType_11x18:
        lcd_write_char(x, y, ch, Font_11x18, color, bgcolor);
        break;
    case FontType_16x26:
        lcd_write_char(x, y, ch, Font_16x26, color, bgcolor);
        break;
    }
}

/*
 * display string
 */
void lcd_write_string(uint16_t x, uint16_t y, char *str, FontDef font, uint16_t color, uint16_t bgcolor)
{

    while (*str)
    {
        if (x + font.width >= ST7735_WIDTH)
        {
            x = 0;
            y += font.height;
            if (y + font.height >= ST7735_HEIGHT)
            {
                break;
            }

            if (*str == ' ')
            {
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

void lcd_write_str(uint16_t x, uint16_t y, char *str, FontType font, uint16_t color, uint16_t bgcolor)
{
    switch (font)
    {
    case FontType_7x10:
        lcd_write_string(x, y, str, Font_7x10, color, bgcolor);
        break;
    case FontType_8x16:
        lcd_write_string(x, y, str, Font_8x16, color, bgcolor);
        break;
    case FontType_11x18:
        lcd_write_string(x, y, str, Font_11x18, color, bgcolor);
        break;
    case FontType_16x26:
        lcd_write_string(x, y, str, Font_16x26, color, bgcolor);
        break;
    }
}

/*
 * fill rectangle
 */
void lcd_fill_rectangle(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint16_t color)
{
    uint8_t buff[320] = {0};
    uint16_t count = 0;
    // clipping
    if ((x >= ST7735_WIDTH) || (y >= ST7735_HEIGHT))
        return;
    if ((x + w) >= ST7735_WIDTH)
        w = ST7735_WIDTH - x;
    if ((y + h) >= ST7735_HEIGHT)
        h = ST7735_HEIGHT - y;
    lcd_set_address_window(x, y, x + w - 1, y + h - 1);

    for (count = 0; count < w; count++)
    {
        buff[count * 2] = color >> 8;
        buff[count * 2 + 1] = color & 0xFF;
    }
    for (y = h; y > 0; y--)
    {
        i2c_burst_transfer(buff, sizeof(uint16_t) * w);
    }
}

/*
 * fill screen
 */

void lcd_fill_screen(uint16_t color)
{
    lcd_fill_rectangle(0, 0, ST7735_WIDTH, ST7735_HEIGHT, color);
    i2c_write_command(SYNC_REG, 0x00, 0x01);
}

void lcd_draw_image(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint8_t *data)
{
    lcd_set_address_window(x, y, x + w - 1, y + h - 1);
    i2c_burst_transfer(data, sizeof(uint16_t) * w * h);
}

uint8_t lcd_begin(void)
{
    char i2c[20] = "/dev/i2c-1";
    // I2C Init
    i2cd = open(i2c, O_RDWR);
    if (i2cd < 0)
    {
        fprintf(stderr, "Device I2C-1 failed to initialize\n");
        return 1;
    }
    if (ioctl(i2cd, I2C_SLAVE_FORCE, I2C_ADDRESS) < 0)
    {
        return 1;
    }
    return 0;
}

void i2c_write_data(uint8_t high, uint8_t low)
{
    uint8_t msg[3] = {WRITE_DATA_REG, high, low};
    if (write(i2cd, msg, 3) < 0) { /* best-effort I2C */ }
    usleep(10);
}

void i2c_write_command(uint8_t command, uint8_t high, uint8_t low)
{
    uint8_t msg[3] = {command, high, low};
    if (write(i2cd, msg, 3) < 0) { /* best-effort I2C */ }
    usleep(10);
}

void i2c_burst_transfer(uint8_t *buff, uint32_t length)
{
    uint32_t count = 0;
    i2c_write_command(BURST_WRITE_REG, 0x00, 0x01);
    while (length > count)
    {
        if ((length - count) > BURST_MAX_LENGTH)
        {
            if (write(i2cd, buff + count, BURST_MAX_LENGTH) < 0) { /* best-effort */ }
            count += BURST_MAX_LENGTH;
        }
        else
        {
            if (write(i2cd, buff + count, length - count) < 0) { /* best-effort */ }
            count += (length - count);
        }
        usleep(700);
    }
    i2c_write_command(BURST_WRITE_REG, 0x00, 0x00);
    i2c_write_command(SYNC_REG, 0x00, 0x01);
}

void lcd_display(uint8_t symbol)
{
    switch (symbol)
    {
    case 0:
        lcd_display_cpuLoad();
        break;
    case 1:
        lcd_display_ram();
        break;
    case 2:
        lcd_display_temp();
        break;
    case 3:
        lcd_display_disk();
        break;
    case 4:
        lcd_display_all();
        break;
    default:
        break;
    }
}

void lcd_display_percentage(uint8_t val, uint16_t color)
{
    uint8_t count = 0;
    uint8_t xCoordinate = 30;
    val += 10;
    if (val >= 100)
    {
        val = 100;
    }
    val /= 10;
    for (count = 0; count < val; count++)
    {
        lcd_fill_rectangle(xCoordinate, 60, 6, 10, color);
        xCoordinate += 10;
    }
    for (count = 0; count < 10 - val; count++)
    {
        lcd_fill_rectangle(xCoordinate, 60, 6, 10, ST7735_GRAY);
        xCoordinate += 10;
    }
}

void lcd_display_cpuLoad(void)
{
    char iPSource[20] = {0};
    uint8_t cpuLoad = 0;
    char cpuStr[10] = {0};
    lcd_fill_screen(ST7735_BLACK);
    cpuLoad = get_cpu_message();
    sprintf(cpuStr, "%d", cpuLoad);
    lcd_fill_rectangle(0, 20, ST7735_WIDTH, 5, ST7735_BLUE);
    if (IP_SWITCH == IP_DISPLAY_OPEN)
    {
        lcd_write_string(0, 0, "IP:", Font_8x16, ST7735_WHITE, ST7735_BLACK);
        strcpy(iPSource, get_ip_address());
        lcd_write_string(24, 0, iPSource, Font_8x16, ST7735_WHITE, ST7735_BLACK);
    }
    else
    {
        lcd_write_string(0, 0, CUSTOM_DISPLAY, Font_8x16, ST7735_WHITE, ST7735_BLACK);
    }
    lcd_write_string(36, 35, "CPU:", Font_11x18, ST7735_WHITE, ST7735_BLACK);
    lcd_write_string(80, 35, cpuStr, Font_11x18, ST7735_WHITE, ST7735_BLACK);
    lcd_write_string(113, 35, "%", Font_11x18, ST7735_WHITE, ST7735_BLACK);
    lcd_display_percentage(cpuLoad, ST7735_GREEN);
}

void lcd_display_ram(void)
{
    float Totalram = 0.0;
    float availram = 0.0;
    uint8_t residue = 0;
    char residueStr[10] = {0};
    get_cpu_memory(&Totalram, &availram);
    residue = (Totalram - availram) / Totalram * 100;
    sprintf(residueStr, "%d", residue);
    lcd_fill_rectangle(0, 35, ST7735_WIDTH, 20, ST7735_BLACK);
    lcd_write_string(36, 35, "RAM:", Font_11x18, ST7735_WHITE, ST7735_BLACK);
    lcd_write_string(80, 35, residueStr, Font_11x18, ST7735_WHITE, ST7735_BLACK);
    lcd_write_string(113, 35, "%", Font_11x18, ST7735_WHITE, ST7735_BLACK);
    lcd_display_percentage(residue, ST7735_YELLOW);
}

void lcd_display_temp(void)
{
    uint16_t temp = 0;
    char tempStr[10] = {0};
    temp = get_temperature();
    sprintf(tempStr, "%d", temp);
    lcd_fill_rectangle(0, 35, ST7735_WIDTH, 20, ST7735_BLACK);
    lcd_write_string(30, 35, "TEMP:", Font_11x18, ST7735_WHITE, ST7735_BLACK);
    lcd_write_string(85, 35, tempStr, Font_11x18, ST7735_WHITE, ST7735_BLACK);
    if (TEMPERATURE_TYPE == FAHRENHEIT)
    {
        lcd_write_string(118, 35, "F", Font_11x18, ST7735_WHITE, ST7735_BLACK);
    }
    else
    {
        lcd_write_string(118, 35, "C", Font_11x18, ST7735_WHITE, ST7735_BLACK);
    }
    if (TEMPERATURE_TYPE == FAHRENHEIT)
    {
        temp -= 32;
        temp /= 1.8;
    }
    lcd_display_percentage((uint16_t)temp, ST7735_RED);
}

void lcd_display_disk(void)
{

    uint16_t diskMemSize = 0;
    uint16_t diskUseMemSize = 0;
    uint32_t sdMemSize = 0;
    uint32_t sdUseMemSize = 0;

    uint16_t memTotal = 0;
    uint16_t useMemTotal = 0;
    uint16_t residue = 0;
    char residueStr[10] = {0};

    get_sd_memory(&sdMemSize, &sdUseMemSize);
    get_hard_disk_memory(&diskMemSize, &diskUseMemSize);

    memTotal = sdMemSize + diskMemSize;
    useMemTotal = sdUseMemSize + diskUseMemSize;
    residue = useMemTotal * 1.0 / memTotal * 100;
    sprintf(residueStr, "%d", residue);

    lcd_fill_rectangle(0, 35, ST7735_WIDTH, 20, ST7735_BLACK);
    lcd_write_string(30, 35, "DISK:", Font_11x18, ST7735_WHITE, ST7735_BLACK);
    lcd_write_string(85, 35, residueStr, Font_11x18, ST7735_WHITE, ST7735_BLACK);
    lcd_write_string(118, 35, "%", Font_11x18, ST7735_WHITE, ST7735_BLACK);
    lcd_display_percentage(residue, ST7735_BLUE);
}

void lcd_display_mini_bar(uint16_t x, uint16_t y, uint16_t w, uint16_t h, uint8_t val, uint16_t color)
{
    uint16_t filled = (uint16_t)val * w / 100;
    if (filled > w) filled = w;
    if (filled > 0)
        lcd_fill_rectangle(x, y, filled, h, color);
    if (filled < w)
        lcd_fill_rectangle(x + filled, y, w - filled, h, ST7735_GRAY);
}

static uint16_t threshold_color(uint8_t val)
{
    if (val < 60)       return ST7735_GREEN;
    if (val < 80)       return ST7735_YELLOW;
    if (val < 90)       return ST7735_COLOR565(255, 165, 0); /* orange */
    return ST7735_RED;
}

static uint16_t temp_threshold_color(uint8_t celsius)
{
    if (celsius < 40)       return ST7735_CYAN;
    if (celsius < 50)       return ST7735_GREEN;
    if (celsius < 60)       return ST7735_YELLOW;
    if (celsius < 70)       return ST7735_COLOR565(255, 165, 0); /* orange */
    return ST7735_RED;
}

#define ST7735_VIOLET ST7735_COLOR565(180, 130, 255)

void lcd_display_all(void)
{
    char buf[24];
    uint8_t cpuLoad;
    float totalRam = 0.0, availRam = 0.0;
    uint8_t ramPercent;
    uint16_t temp;
    uint32_t sdMemSize = 0, sdUseMemSize = 0;
    uint16_t diskMemSize = 0, diskUseMemSize = 0;
    uint16_t memTotal, useMemTotal, diskPercent;
    uint8_t tempForBar;
    uint16_t color;
    char ipBuf[20];
    char hostBuf[17];

    /* Gather all data */
    cpuLoad = get_cpu_message();
    get_cpu_memory(&totalRam, &availRam);
    ramPercent = (totalRam > 0) ? (uint8_t)((totalRam - availRam) / totalRam * 100) : 0;
    temp = get_temperature();
    get_sd_memory(&sdMemSize, &sdUseMemSize);
    get_hard_disk_memory(&diskMemSize, &diskUseMemSize);
    memTotal = sdMemSize + diskMemSize;
    useMemTotal = sdUseMemSize + diskUseMemSize;
    diskPercent = (memTotal > 0) ? useMemTotal * 100 / memTotal : 0;

    lcd_fill_screen(ST7735_BLACK);

    /* Row 1: Hostname (truncated to leave room for badge) */
    strncpy(hostBuf, get_hostname(), 16);
    hostBuf[16] = '\0';
    lcd_write_string(2, 0, hostBuf, Font_8x16, ST7735_WHITE, ST7735_BLACK);

    /* Row 2: IP address */
    strcpy(ipBuf, get_ip_address());
    lcd_write_string(2, 18, ipBuf, Font_7x10, ST7735_VIOLET, ST7735_BLACK);

    /* Separator line */
    lcd_fill_rectangle(0, 30, ST7735_WIDTH, 1, ST7735_BLUE);

    /* DietPi status dot — red when update needed, hidden otherwise */
    int dietpi_status = get_dietpi_update_status();
    if (dietpi_status == 2) {
        lcd_fill_rectangle(154, 5, 2, 1, ST7735_RED);
        lcd_fill_rectangle(153, 6, 4, 1, ST7735_RED);
        lcd_fill_rectangle(152, 7, 6, 1, ST7735_RED);
        lcd_fill_rectangle(152, 8, 6, 1, ST7735_RED);
        lcd_fill_rectangle(153, 9, 4, 1, ST7735_RED);
        lcd_fill_rectangle(154, 10, 2, 1, ST7735_RED);
    }

    /* APT update count — right-aligned on IP row */
    int apt_count = get_apt_update_count();
    lcd_fill_rectangle(124, 18, 36, 10, ST7735_BLACK);
    if (apt_count > 0) {
        uint16_t badge_color = (apt_count >= 10) ? ST7735_RED : ST7735_YELLOW;
        char badge_buf[8];
        sprintf(badge_buf, "^%d", apt_count > 99 ? 99 : apt_count);
        int len = strlen(badge_buf);
        uint16_t bx = 160 - (len * 7) - 2;
        lcd_write_string(bx, 19, badge_buf, Font_7x10, badge_color, ST7735_BLACK);
    }

    /* CPU (left column, row 1) */
    color = threshold_color(cpuLoad);
    lcd_write_string(2, 34, "CPU:", Font_7x10, ST7735_WHITE, ST7735_BLACK);
    sprintf(buf, "%3d%%", cpuLoad);
    lcd_write_string(30, 34, buf, Font_7x10, color, ST7735_BLACK);
    lcd_display_mini_bar(2, 46, 65, 6, cpuLoad, color);

    /* RAM (left column, row 2) */
    color = threshold_color(ramPercent);
    lcd_write_string(2, 56, "RAM:", Font_7x10, ST7735_WHITE, ST7735_BLACK);
    sprintf(buf, "%3d%%", ramPercent);
    lcd_write_string(30, 56, buf, Font_7x10, color, ST7735_BLACK);
    lcd_display_mini_bar(2, 68, 65, 6, ramPercent, color);

    /* Temperature (right column, row 1) */
    tempForBar = temp;
    if (TEMPERATURE_TYPE == FAHRENHEIT) {
        tempForBar = (temp - 32) / 1.8;
    }
    color = temp_threshold_color(tempForBar);
    lcd_write_string(84, 34, "TEMP:", Font_7x10, ST7735_WHITE, ST7735_BLACK);
    sprintf(buf, "%3d%c", temp, TEMPERATURE_TYPE == FAHRENHEIT ? 'F' : 'C');
    lcd_write_string(119, 34, buf, Font_7x10, color, ST7735_BLACK);
    lcd_display_mini_bar(84, 46, 65, 6, tempForBar > 100 ? 100 : tempForBar, color);

    /* Disk (right column, row 2) */
    color = threshold_color(diskPercent > 100 ? 100 : (uint8_t)diskPercent);
    lcd_write_string(84, 56, "DISK:", Font_7x10, ST7735_WHITE, ST7735_BLACK);
    sprintf(buf, "%3d%%", diskPercent > 999 ? 999 : diskPercent);
    lcd_write_string(119, 56, buf, Font_7x10, color, ST7735_BLACK);
    lcd_display_mini_bar(84, 68, 65, 6, diskPercent > 100 ? 100 : (uint8_t)diskPercent, color);
}
