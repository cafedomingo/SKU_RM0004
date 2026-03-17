#include "diagnostic.h"
#include "rpiInfo.h"
#include "st7735.h"
#include <stdarg.h>
#include <stdio.h>
#include <string.h>

typedef struct {
    char label[12];
    char value[24];
    uint16_t color; /* value color; labels are always gray */
} diag_row_t;

static diag_row_t rows[DIAG_TOTAL_ROWS];

static void format_rate(uint64_t bps, char *buf, size_t len) {
    if (bps >= 1048576)
        snprintf(buf, len, "%.1fM", (double)bps / 1048576.0);
    else if (bps >= 1024)
        snprintf(buf, len, "%.1fK", (double)bps / 1024.0);
    else
        snprintf(buf, len, "%luB", (unsigned long)bps);
}

static void format_uptime(uint32_t secs, char *buf, size_t len) {
    uint32_t d = secs / 86400;
    uint32_t h = (secs % 86400) / 3600;
    uint32_t m = (secs % 3600) / 60;
    if (d > 0)
        snprintf(buf, len, "%ud %uh", d, h);
    else if (h > 0)
        snprintf(buf, len, "%uh %um", h, m);
    else
        snprintf(buf, len, "%um", m);
}

static uint16_t threshold_color(uint8_t val) {
    if (val < 60) return ST7735_GREEN;
    if (val < 80) return ST7735_YELLOW;
    if (val < 90) return ST7735_ORANGE;
    return ST7735_RED;
}

static uint16_t temp_color(uint8_t celsius) {
    if (celsius < 50) return ST7735_GREEN;
    if (celsius < 60) return ST7735_YELLOW;
    if (celsius < 70) return ST7735_ORANGE;
    return ST7735_RED;
}

/*
 * Set a row with a left-aligned label and a right-aligned value.
 * If label is empty, the value is rendered left-aligned instead (for header rows).
 */
static void set_row(int idx, const char *label, uint16_t color, const char *fmt, ...) {
    snprintf(rows[idx].label, sizeof(rows[idx].label), "%s", label);
    va_list ap;
    va_start(ap, fmt);
    vsnprintf(rows[idx].value, sizeof(rows[idx].value), fmt, ap);
    va_end(ap);
    rows[idx].color = color;
}

void diag_refresh_data(void) {
    char r[12], w[12];
    int i = 0;

    /* Page 1: System overview */
    set_row(i++, "", ST7735_WHITE, "%s", get_hostname());
    set_row(i++, "", ST7735_VIOLET, "%s", get_ip_address());
    set_row(i++, "", ST7735_VIOLET, "%s", get_ip6_suffix());

    format_uptime(get_uptime_secs(), r, sizeof(r));
    set_row(i++, "Uptime", ST7735_WHITE, "%s", r);

    uint8_t cpu = get_cpu_percent();
    cpu_freq_t freq = get_cpu_freq();
    set_row(i++, "CPU", threshold_color(cpu), "%d%% %dMHz", cpu, freq.cur_mhz);

    uint8_t temp = get_temperature();
    set_row(i++, "Temp", temp_color(temp), "%dC / %dF", temp, (int)temp * 9 / 5 + 32);

    uint8_t ram = get_ram_percent();
    set_row(i++, "RAM", threshold_color(ram), "%d%%", ram);

    uint32_t thr = get_cpu_throttle_status();
    if ((thr & THROTTLE_CURRENT_MASK) != 0)
        set_row(i++, "Throttle", ST7735_RED, "ACTIVE");
    else if ((thr & THROTTLE_PAST_MASK) != 0)
        set_row(i++, "Throttle", ST7735_YELLOW, "past");
    else
        set_row(i++, "Throttle", ST7735_GREEN, "OK");

    /* Page 2: I/O + Updates */
    uint8_t disk = get_disk_percent();
    set_row(i++, "Disk", threshold_color(disk), "%d%%", disk);

    net_bandwidth_t net = get_net_bandwidth();
    format_rate(net.rx_bytes_per_sec, r, sizeof(r));
    format_rate(net.tx_bytes_per_sec, w, sizeof(w));
    set_row(i++, "Net RX", ST7735_WHITE, "%s", r);
    set_row(i++, "Net TX", ST7735_WHITE, "%s", w);

    disk_io_t dio = get_disk_io();
    format_rate(dio.read_bytes_per_sec, r, sizeof(r));
    format_rate(dio.write_bytes_per_sec, w, sizeof(w));
    set_row(i++, "IO R/W", ST7735_WHITE, "%s/%s", r, w);
    set_row(i++, "IOPS R/W", ST7735_WHITE, "%u/%u", dio.read_iops, dio.write_iops);

    int ds = get_dietpi_update_status();
    if (ds == 2)
        set_row(i++, "DietPi", ST7735_RED, "update!");
    else if (ds == 1)
        set_row(i++, "DietPi", ST7735_GREEN, "OK");
    else
        set_row(i++, "DietPi", ST7735_GRAY, "N/A");

    int apt = get_apt_update_count();
    if (apt > 0)
        set_row(i++, "APT", ST7735_YELLOW, "%d updates", apt);
    else if (apt == 0)
        set_row(i++, "APT", ST7735_GREEN, "up to date");
    else
        set_row(i++, "APT", ST7735_GRAY, "N/A");
}

/*
 * Page layout (max 8 rows per page):
 *   Page 0: System identity + CPU/thermal/RAM  (rows 0-7)
 *   Page 1: I/O + Updates                      (rows 8-14)
 */
static const int page_start[DIAG_NUM_PAGES] = {0, 8};
static const int page_len[DIAG_NUM_PAGES] = {8, 7};

void lcd_display_diagnostic_page(int page) {
    static uint8_t fb[ST7735_WIDTH * ST7735_HEIGHT * 2];
    uint16_t row_h = Font_7x10.height;
    uint16_t char_w = Font_7x10.width;
    int first = page_start[page];
    int count = page_len[page];

    memset(fb, 0, sizeof(fb));

    for (int v = 0; v < count; v++) {
        int idx = first + v;
        uint16_t y = (uint16_t)(v * row_h);
        const char *label = rows[idx].label;
        const char *value = rows[idx].value;
        uint16_t color = rows[idx].color;

        if (label[0] == '\0') {
            /* Header row: value rendered left-aligned in its color */
            lcd_fb_string(fb, 0, y, value, Font_7x10, color);
        } else {
            /* Label left-aligned in gray */
            lcd_fb_string(fb, 0, y, label, Font_7x10, ST7735_GRAY);
            /* Value right-aligned in color (fall back to left-align if too wide) */
            uint16_t vlen = strlen(value);
            uint16_t vx = (vlen * char_w <= ST7735_WIDTH) ? ST7735_WIDTH - vlen * char_w : 0;
            lcd_fb_string(fb, vx, y, value, Font_7x10, color);
        }
    }

    lcd_draw_fullscreen(fb);
}
