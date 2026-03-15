#include "diagnostic.h"
#include "rpiInfo.h"
#include "st7735.h"
#include <stdarg.h>
#include <stdio.h>
#include <string.h>

typedef struct {
    char text[24];
    uint16_t color;
} diag_row_t;

static diag_row_t rows[DIAG_TOTAL_ROWS];

static void format_bytes(uint64_t bps, char *buf, size_t len) {
    if (bps >= 1048576)
        snprintf(buf, len, "%.1f MB/s", (double)bps / 1048576.0);
    else if (bps >= 1024)
        snprintf(buf, len, "%.1f KB/s", (double)bps / 1024.0);
    else
        snprintf(buf, len, "%lu B/s", (unsigned long)bps);
}

static void format_uptime(uint32_t secs, char *buf, size_t len) {
    uint32_t d = secs / 86400;
    uint32_t h = (secs % 86400) / 3600;
    uint32_t m = (secs % 3600) / 60;
    if (d > 0)
        snprintf(buf, len, "%ud %uh %um", d, h, m);
    else if (h > 0)
        snprintf(buf, len, "%uh %um", h, m);
    else
        snprintf(buf, len, "%um", m);
}

static void set_row(int idx, uint16_t color, const char *fmt, ...) {
    va_list ap;
    va_start(ap, fmt);
    vsnprintf(rows[idx].text, sizeof(rows[idx].text), fmt, ap);
    va_end(ap);
    rows[idx].color = color;
}

static void set_separator(int idx) {
    snprintf(rows[idx].text, sizeof(rows[idx].text), "--------------------");
    rows[idx].color = ST7735_GRAY;
}

void diag_refresh_data(void) {
    char tmp[24];
    int i = 0;

    /* System info */
    set_row(i++, ST7735_WHITE, "%s", get_hostname());
    set_row(i++, ST7735_VIOLET, "IP: %s", get_ip_address());

    uint32_t up = get_uptime_secs();
    format_uptime(up, tmp, sizeof(tmp));
    set_row(i++, ST7735_WHITE, "Up: %s", tmp);

    set_separator(i++);

    /* CPU / thermal / RAM */
    uint8_t cpu = get_cpu_percent();
    cpu_freq_t freq = get_cpu_freq();
    set_row(i++, ST7735_GREEN, "CPU: %3d%%  %dMHz", cpu, freq.cur_mhz);
    set_row(i++, ST7735_GREEN, "Freq: %d-%d MHz", freq.min_mhz, freq.max_mhz);

    uint8_t temp = get_temperature();
    int fahr = (int)temp * 9 / 5 + 32;
    set_row(i++, ST7735_CYAN, "Temp: %dC / %dF", temp, fahr);

    uint8_t ram = get_ram_percent();
    set_row(i++, ST7735_GREEN, "RAM: %3d%%", ram);

    uint32_t thr = get_cpu_throttle_status();
    if ((thr & THROTTLE_CURRENT_MASK) == 0 && (thr & THROTTLE_PAST_MASK) == 0)
        set_row(i++, ST7735_GREEN, "Throttle: OK");
    else if ((thr & THROTTLE_CURRENT_MASK) != 0)
        set_row(i++, ST7735_RED, "Throttle: ACTIVE");
    else
        set_row(i++, ST7735_YELLOW, "Throttle: past");

    set_separator(i++);

    /* Network */
    net_bandwidth_t net = get_net_bandwidth();
    format_bytes(net.rx_bytes_per_sec, tmp, sizeof(tmp));
    set_row(i++, ST7735_CYAN, "Net RX: %s", tmp);
    format_bytes(net.tx_bytes_per_sec, tmp, sizeof(tmp));
    set_row(i++, ST7735_CYAN, "Net TX: %s", tmp);

    set_separator(i++);

    /* Disk */
    uint8_t disk = get_disk_percent();
    set_row(i++, ST7735_GREEN, "Disk: %3d%%", disk);

    disk_io_t dio = get_disk_io();
    format_bytes(dio.read_bytes_per_sec, tmp, sizeof(tmp));
    set_row(i++, ST7735_GREEN, "IO R: %s", tmp);
    format_bytes(dio.write_bytes_per_sec, tmp, sizeof(tmp));
    set_row(i++, ST7735_GREEN, "IO W: %s", tmp);
    set_row(i++, ST7735_GREEN, "IOPS R: %u", dio.read_iops);
    set_row(i++, ST7735_GREEN, "IOPS W: %u", dio.write_iops);

    set_separator(i++);

    /* DietPi */
    int ds = get_dietpi_update_status();
    if (ds == 0)
        set_row(i++, ST7735_GRAY, "DietPi: N/A");
    else if (ds == 1)
        set_row(i++, ST7735_GREEN, "DietPi: OK");
    else
        set_row(i++, ST7735_RED, "DietPi: update!");

    int apt = get_apt_update_count();
    if (apt < 0)
        set_row(i++, ST7735_GRAY, "APT: N/A");
    else if (apt == 0)
        set_row(i++, ST7735_GREEN, "APT: up to date");
    else
        set_row(i++, ST7735_YELLOW, "APT: %d updates", apt);
}

void lcd_display_diagnostic(uint8_t scroll_offset) {
    static uint8_t fb[ST7735_WIDTH * ST7735_HEIGHT * 2];
    uint16_t w = ST7735_WIDTH;
    uint16_t row_h = Font_7x10.height;

    /* Clear framebuffer to black */
    memset(fb, 0, sizeof(fb));

    /* Render each visible row into the framebuffer */
    for (int v = 0; v < DIAG_VISIBLE_ROWS; v++) {
        int idx = (scroll_offset + v) % DIAG_TOTAL_ROWS;
        uint16_t y = (uint16_t)(v * row_h);
        const char *str = rows[idx].text;
        uint16_t color = rows[idx].color;

        uint16_t x = 0;
        while (*str && x + Font_7x10.width <= w) {
            uint16_t b;
            for (uint16_t row = 0; row < row_h; row++) {
                b = Font_7x10.data[(*str - 32) * row_h + row];
                for (uint16_t col = 0; col < Font_7x10.width; col++) {
                    if ((b << col) & 0x8000) {
                        uint32_t off = ((y + row) * w + x + col) * 2;
                        fb[off] = color >> 8;
                        fb[off + 1] = color & 0xFF;
                    }
                }
            }
            x += Font_7x10.width;
            str++;
        }
    }

    lcd_draw_fullscreen(fb);
}
