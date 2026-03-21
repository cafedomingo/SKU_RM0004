#include "dashboard.h"
#include "format.h"
#include "rpiInfo.h"
#include "st7735.h"
#include "theme.h"
#include <stdio.h>
#include <string.h>

#define METRIC_BAR_WIDTH  65
#define METRIC_BAR_HEIGHT 6

/*
 * Draw a labeled metric with a right-aligned value and progress bar.
 */
static void draw_metric(uint16_t x, uint16_t y, const char *label, const char *value, uint8_t bar_pct, uint16_t color) {
    uint16_t val_x = x + METRIC_BAR_WIDTH - strlen(value) * Font_7x10.width; /* right-align with bar */
    lcd_write_string(x, y, label, Font_7x10, theme.fg, theme.bg);
    lcd_write_string(val_x, y, value, Font_7x10, color, theme.bg);
    lcd_draw_bar(x, y + 12, METRIC_BAR_WIDTH, METRIC_BAR_HEIGHT, bar_pct, color);
}

/*
 * Gather system metrics and render the full dashboard display.
 */
void lcd_display_dashboard(void) {
    char buf[24];

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
    char hostBuf[17];
    snprintf(hostBuf, sizeof(hostBuf), "%s", hostname);
    lcd_write_string(2, 0, hostBuf, Font_8x16, theme.fg, theme.bg);

    lcd_write_string(2, 18, ip, Font_7x10, theme.ip, theme.bg);

    lcd_fill_rectangle(0, 30, ST7735_WIDTH, 1, theme.sep);

    /* DietPi diamond — alert color when update needed */
    if (dietpi_status == 2) {
        lcd_draw_diamond(152, 5, theme.alert);
    }

    /* APT update count — right-aligned on IP row */
    lcd_fill_rectangle(124, 18, 36, 10, theme.bg);
    char badge[5];
    if (format_apt_badge(apt_count, badge, sizeof(badge))) {
        uint16_t color = (apt_count >= 10) ? theme.crit : theme.warn;
        uint16_t bx = ST7735_WIDTH - strlen(badge) * Font_7x10.width - 2;
        lcd_write_string(bx, 19, badge, Font_7x10, color, theme.bg);
    }

    /* CPU */
    uint16_t color = threshold_color(cpuPercent, TH_CPU_WARN, TH_CPU_CRIT);
    snprintf(buf, sizeof(buf), "%3d%%", cpuPercent);
    draw_metric(2, 34, "CPU:", buf, cpuPercent, color);

    /* RAM */
    color = threshold_color(ramPercent, TH_RAM_WARN, TH_RAM_CRIT);
    snprintf(buf, sizeof(buf), "%3d%%", ramPercent);
    draw_metric(2, 56, "RAM:", buf, ramPercent, color);

    /* Temperature */
    color = temp_ramp_color(temp);
    format_temp(temp, buf, sizeof(buf));
    draw_metric(84, 34, "TEMP:", buf, temp > 100 ? 100 : temp, color);

    /* Disk */
    color = threshold_color(diskPercent, TH_DISK_WARN, TH_DISK_CRIT);
    snprintf(buf, sizeof(buf), "%3d%%", diskPercent);
    draw_metric(84, 56, "DISK:", buf, diskPercent, color);
}
