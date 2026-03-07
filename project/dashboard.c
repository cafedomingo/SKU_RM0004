#include "dashboard.h"
#include "rpiInfo.h"
#include "st7735.h"
#include <stdio.h>
#include <string.h>

#define METRIC_BAR_WIDTH  65
#define METRIC_BAR_HEIGHT 6

/*
 * Map a percentage value to a green/yellow/orange/red color.
 */
static uint16_t threshold_color(uint8_t val) {
    if (val < 60) return ST7735_GREEN;
    if (val < 80) return ST7735_YELLOW;
    if (val < 90) return ST7735_ORANGE;
    return ST7735_RED;
}

/*
 * Map a temperature in Celsius to a cyan-to-red color scale.
 */
static uint16_t temp_threshold_color(uint8_t celsius) {
    if (celsius < 40) return ST7735_CYAN;
    if (celsius < 50) return ST7735_GREEN;
    if (celsius < 60) return ST7735_YELLOW;
    if (celsius < 70) return ST7735_ORANGE;
    return ST7735_RED;
}

/*
 * Draw a labeled metric with a right-aligned value and progress bar.
 */
static void draw_metric(uint16_t x, uint16_t y, const char *label, const char *value, uint8_t bar_pct, uint16_t color) {
    uint16_t val_x = x + METRIC_BAR_WIDTH - strlen(value) * Font_7x10.width; /* right-align with bar */
    lcd_write_string(x, y, (char *)label, Font_7x10, ST7735_WHITE, ST7735_BLACK);
    lcd_write_string(val_x, y, (char *)value, Font_7x10, color, ST7735_BLACK);
    lcd_draw_bar(x, y + 12, METRIC_BAR_WIDTH, METRIC_BAR_HEIGHT, bar_pct, color);
}

/*
 * Gather system metrics and render the full dashboard display.
 */
void lcd_display_dashboard(void) {
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
    draw_metric(84, 34, "TEMP:", buf, tempForBar > 100 ? 100 : tempForBar, color);

    /* Disk */
    color = threshold_color(diskPercent);
    sprintf(buf, "%3d%%", diskPercent);
    draw_metric(84, 56, "DISK:", buf, diskPercent, color);
}
