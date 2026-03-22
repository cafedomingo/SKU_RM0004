#include "diagnostic.h"
#include "format.h"
#include "rpiInfo.h"
#include "st7735.h"
#include "theme.h"
#include <stdarg.h>
#include <stdio.h>
#include <string.h>

typedef struct {
    char label[12];
    char value[24];
    uint16_t color; /* value color; labels are always gray */
} diag_row_t;

static diag_row_t rows[DIAG_TOTAL_ROWS];

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
    set_row(i++, "", theme.fg, "%s", get_hostname());
    set_row(i++, "", theme.ip, "%s", get_ip_address());
    set_row(i++, "", theme.ip, "%s", get_ip6_suffix());

    format_uptime(get_uptime_secs(), r, sizeof(r));
    set_row(i++, "Uptime", theme.fg, "%s", r);

    uint8_t cpu = get_cpu_percent();
    cpu_freq_t freq = get_cpu_freq();
    char freq_str[10];
    format_freq(freq.cur_mhz, freq_str, sizeof(freq_str));
    set_row(i++, "CPU", threshold_color(cpu, TH_CPU_WARN, TH_CPU_CRIT), "%d%% %s", cpu, freq_str);

    uint8_t temp = get_temperature();
    set_row(i++, "Temp", temp_ramp_color(temp), "%dC / %dF", temp, celsius_to_f(temp));

    uint8_t ram = get_ram_percent();
    set_row(i++, "RAM", threshold_color(ram, TH_RAM_WARN, TH_RAM_CRIT), "%d%%", ram);

    uint32_t thr = get_cpu_throttle_status();
    if ((thr & THROTTLE_CURRENT_MASK) != 0)
        set_row(i++, "Throttle", theme.crit, "ACTIVE");
    else if ((thr & THROTTLE_PAST_MASK) != 0)
        set_row(i++, "Throttle", theme.warn, "past");
    else
        set_row(i++, "Throttle", theme.ok, "OK");

    /* Page 2: I/O + Updates */
    uint8_t disk = get_disk_percent();
    set_row(i++, "Disk", threshold_color(disk, TH_DISK_WARN, TH_DISK_CRIT), "%d%%", disk);

    net_bandwidth_t net = get_net_bandwidth();
    format_rate(net.rx_bytes_per_sec, r, sizeof(r));
    format_rate(net.tx_bytes_per_sec, w, sizeof(w));
    set_row(i++, "Net RX", theme.fg, "%s", r);
    set_row(i++, "Net TX", theme.fg, "%s", w);

    disk_io_t dio = get_disk_io();
    format_rate(dio.read_bytes_per_sec, r, sizeof(r));
    format_rate(dio.write_bytes_per_sec, w, sizeof(w));
    set_row(i++, "IO R/W", theme.fg, "%s/%s", r, w);
    set_row(i++, "IOPS R/W", theme.fg, "%u/%u", dio.read_iops, dio.write_iops);

    int ds = get_dietpi_update_status();
    if (ds == DIETPI_UPDATE_AVAIL)
        set_row(i++, "DietPi", theme.alert, "update!");
    else if (ds == DIETPI_UP_TO_DATE)
        set_row(i++, "DietPi", theme.ok, "OK");
    else
        set_row(i++, "DietPi", theme.muted, "N/A");

    int apt = get_apt_update_count();
    if (apt > 0)
        set_row(i++, "APT", theme.warn, "%d updates", apt);
    else if (apt == 0)
        set_row(i++, "APT", theme.ok, "up to date");
    else
        set_row(i++, "APT", theme.muted, "N/A");
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
