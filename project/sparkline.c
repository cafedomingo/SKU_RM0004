#include "sparkline.h"
#include "format.h"
#include "rpiInfo.h"
#include "st7735.h"
#include "theme.h"
#include <stdbool.h>
#include <stdio.h>
#include <string.h>

/* Two-column layout */
#define COL_LEFT_X  0
#define COL_RIGHT_X 82
#define COL_WIDTH   78

/* Row Y positions */
#define ROW_TICKER    1
#define ROW_UPTIME    12
#define ROW_FREQ      23
#define ROW_DIVIDER   33
#define ROW_SPARK_TOP 35
#define SPARK_HEIGHT  22
#define ROW_CPU_RAM   58
#define ROW_IO        69

/* Sparkline bar geometry */
#define SPARK_BAR_W   5
#define SPARK_BAR_GAP 1

typedef struct {
    uint8_t cpu_pct;
    uint8_t ram_pct;
    uint8_t disk_pct;
    uint8_t temp_c;
    uint16_t freq_mhz;
    uint32_t net_rx;
    uint32_t net_tx;
    uint32_t disk_read;
    uint32_t disk_write;
    char hostname[17];
    char ipv4[16];
    char ipv6[40];
    char uptime[12];
    uint8_t apt_count;
    bool dietpi_update;
    bool throttled;
    bool has_ipv6;
} SystemData;

static uint8_t fb[ST7735_WIDTH * ST7735_HEIGHT * 2];
static uint8_t prev_fb[ST7735_WIDTH * ST7735_HEIGHT * 2];
static bool first_frame = true;

/*
 * Compare current and previous framebuffers row-by-row, coalesce adjacent
 * dirty rows into full-width strips, and send only those strips to the display.
 * On first frame, send everything.
 */
static void flush_dirty(void) {
    if (first_frame) {
        lcd_draw_fullscreen(fb);
        memcpy(prev_fb, fb, sizeof(fb));
        first_frame = false;
        return;
    }

    int row_bytes = ST7735_WIDTH * 2;
    int dirty_start = -1;

    for (int y = 0; y <= ST7735_HEIGHT; y++) {
        bool dirty = (y < ST7735_HEIGHT) && memcmp(fb + y * row_bytes, prev_fb + y * row_bytes, row_bytes) != 0;
        if (dirty && dirty_start < 0) {
            dirty_start = y;
        } else if (!dirty && dirty_start >= 0) {
            lcd_draw_region(fb, 0, dirty_start, ST7735_WIDTH, y - dirty_start);
            dirty_start = -1;
        }
    }

    memcpy(prev_fb, fb, sizeof(fb));
}

void sparkline_invalidate(void) { first_frame = true; }

static void collect_data(SystemData *d) {
    d->cpu_pct = get_cpu_percent();
    d->ram_pct = get_ram_percent();
    d->disk_pct = get_disk_percent();
    d->temp_c = get_temperature();

    cpu_freq_t freq = get_cpu_freq();
    d->freq_mhz = freq.cur_mhz;

    net_bandwidth_t net = get_net_bandwidth();
    d->net_rx = (uint32_t)net.rx_bytes_per_sec;
    d->net_tx = (uint32_t)net.tx_bytes_per_sec;

    disk_io_t dio = get_disk_io();
    d->disk_read = (uint32_t)dio.read_bytes_per_sec;
    d->disk_write = (uint32_t)dio.write_bytes_per_sec;

    snprintf(d->hostname, sizeof(d->hostname), "%s", get_hostname());
    snprintf(d->ipv4, sizeof(d->ipv4), "%s", get_ip_address());

    const char *ip6 = get_ip6_suffix();
    d->has_ipv6 = (strcmp(ip6, "no IPv6") != 0);
    snprintf(d->ipv6, sizeof(d->ipv6), "%s", ip6);

    format_uptime(get_uptime_secs(), d->uptime, sizeof(d->uptime));

    int apt = get_apt_update_count();
    d->apt_count = (uint8_t)(apt > 0 ? apt : 0);
    d->dietpi_update = (get_dietpi_update_status() == 2);

    uint32_t thr = get_cpu_throttle_status();
    d->throttled = (thr & THROTTLE_CURRENT_MASK) != 0;
}

/*
 * Row 1 (y=1): ticker cycling hostname/ipv4/ipv6 + alert badges on right.
 */
static void draw_ticker(sparkline_state_t *state, const SystemData *d) {
    /* Advance ticker */
    state->ticker_phase++;
    int max_phase = d->has_ipv6 ? 3 : 2;
    if (state->ticker_phase >= max_phase) state->ticker_phase = 0;

    switch (state->ticker_phase) {
    case 0:
        lcd_fb_string(fb, 0, ROW_TICKER, d->hostname, Font_7x10, theme.fg);
        break;
    case 1:
        lcd_fb_string(fb, 0, ROW_TICKER, d->ipv4, Font_7x10, theme.ip);
        break;
    case 2:
        lcd_fb_string(fb, 0, ROW_TICKER, d->ipv6, Font_7x10, theme.ip);
        break;
    }
}

/*
 * Row 2 (y=12): uptime left, update badges right-aligned.
 */
static void draw_uptime_updates(const SystemData *d) {
    lcd_fb_string(fb, 0, ROW_UPTIME, d->uptime, Font_7x10, theme.fg);

    /* Update badges — build from right edge inward */
    uint16_t ax = ST7735_WIDTH;

    /* APT badge: ^N */
    if (d->apt_count > 0) {
        char badge[5];
        int capped = d->apt_count > 99 ? 99 : d->apt_count;
        snprintf(badge, sizeof(badge), "^%d", capped);
        uint16_t color = (d->apt_count >= 10) ? theme.crit : theme.warn;
        uint16_t bw = strlen(badge) * Font_7x10.width;
        ax -= bw;
        lcd_fb_string(fb, ax, ROW_UPTIME, badge, Font_7x10, color);
    }

    /* DietPi diamond */
    if (d->dietpi_update) {
        ax -= 3; /* gap */
        lcd_fb_diamond(fb, ax - 6, ROW_UPTIME + 2, theme.alert);
    }
}

/*
 * Row 3 (y=23): frequency+throttle left, D:N% right-aligned.
 */
static void draw_freq_disk(const SystemData *d) {
    /* Frequency */
    char freq_str[10];
    format_freq(d->freq_mhz, freq_str, sizeof(freq_str));
    lcd_fb_string(fb, 0, ROW_FREQ, freq_str, Font_7x10, theme.fg);
    if (d->throttled) {
        uint16_t bang_x = strlen(freq_str) * Font_7x10.width;
        lcd_fb_char(fb, bang_x, ROW_FREQ, '!', Font_7x10, theme.alert);
    }

    /* Right side: temp | D:N% — build from right edge inward */
    char disk_val[5];
    snprintf(disk_val, sizeof(disk_val), "%u%%", d->disk_pct);
    uint16_t lbl_w = 2 * Font_7x10.width; /* "D:" */
    uint16_t val_w = strlen(disk_val) * Font_7x10.width;
    uint16_t dx = COL_RIGHT_X + COL_WIDTH - lbl_w - val_w;
    lcd_fb_string(fb, dx, ROW_FREQ, "D:", Font_7x10, theme.fg);
    lcd_fb_string(fb, dx + lbl_w, ROW_FREQ, disk_val, Font_7x10,
                  threshold_color(d->disk_pct, TH_DISK_WARN, TH_DISK_CRIT));

    /* Temperature | pipe */
    char temp_str[6];
    snprintf(temp_str, sizeof(temp_str), "%uC", d->temp_c);
    uint16_t temp_w = strlen(temp_str) * Font_7x10.width;
    uint16_t pipe_x = dx - 2 - Font_7x10.width - 2;
    lcd_fb_char(fb, pipe_x + 2, ROW_FREQ, '|', Font_7x10, theme.sep);
    lcd_fb_string(fb, pipe_x - temp_w, ROW_FREQ, temp_str, Font_7x10, temp_ramp_color(d->temp_c));
}

/*
 * Draw a single sparkline chart (13 bars growing upward from ROW_SPARK_BOT).
 */
static void draw_sparkline(uint16_t start_x, const uint8_t *history, uint32_t warn_th, uint32_t crit_th) {
    for (int i = 0; i < SPARKLINE_HISTORY; i++) {
        uint8_t val = history[i];
        if (val == 0) continue;
        int col_height = (val * SPARK_HEIGHT + 50) / 100;
        if (col_height < 1) col_height = 1;
        uint16_t cx = start_x + i * (SPARK_BAR_W + SPARK_BAR_GAP);
        uint16_t cy = ROW_SPARK_TOP + SPARK_HEIGHT - col_height;
        uint16_t color = threshold_color(val, warn_th, crit_th);
        lcd_fb_rect(fb, cx, cy, SPARK_BAR_W, col_height, color);
    }
}

/*
 * CPU N% (left) and RAM N% (right) at y=58.
 */
static void draw_cpu_ram_values(const SystemData *d) {
    /* CPU label + value */
    lcd_fb_string(fb, COL_LEFT_X, ROW_CPU_RAM, "CPU", Font_7x10, theme.fg);
    char buf[5];
    snprintf(buf, sizeof(buf), "%u%%", d->cpu_pct);
    lcd_fb_string(fb, COL_LEFT_X + 3 * Font_7x10.width + 1, ROW_CPU_RAM, buf, Font_7x10,
                  threshold_color(d->cpu_pct, TH_CPU_WARN, TH_CPU_CRIT));

    /* RAM label + value */
    lcd_fb_string(fb, COL_RIGHT_X, ROW_CPU_RAM, "RAM", Font_7x10, theme.fg);
    snprintf(buf, sizeof(buf), "%u%%", d->ram_pct);
    lcd_fb_string(fb, COL_RIGHT_X + 3 * Font_7x10.width + 1, ROW_CPU_RAM, buf, Font_7x10,
                  threshold_color(d->ram_pct, TH_RAM_WARN, TH_RAM_CRIT));
}

/* 5px wide x 6px tall arrow shapes: {x_offset, width} per row */
static const uint8_t arrow_up[][2] = {
    {2, 1}, {1, 3}, {0, 5}, {2, 1}, {2, 1}, {2, 1},
};
static const uint8_t arrow_down[][2] = {
    {2, 1}, {2, 1}, {2, 1}, {0, 5}, {1, 3}, {2, 1},
};
#define ARROW_ROWS 6

static void draw_arrow(uint16_t x, uint16_t y, int dir, uint16_t color) {
    uint16_t by = y + (dir > 0 ? 1 : 2);
    const uint8_t (*shape)[2] = dir > 0 ? arrow_down : arrow_up;
    for (int r = 0; r < ARROW_ROWS; r++)
        lcd_fb_rect(fb, x + shape[r][0], by + r, shape[r][1], 1, color);
}

/*
 * I/O row (y=69): net ↓↑ left column, disk R/W right column.
 */
static void draw_io_row(const SystemData *d) {
    char buf[8];

    /* Network: ↓rx ↑tx */
    draw_arrow(0, ROW_IO, 1, theme.fg);
    format_rate(d->net_rx, buf, sizeof(buf));
    lcd_fb_string(fb, 7, ROW_IO, buf, Font_7x10, threshold_color(d->net_rx, TH_NET_WARN, TH_NET_CRIT));

    draw_arrow(38, ROW_IO, -1, theme.fg);
    format_rate(d->net_tx, buf, sizeof(buf));
    lcd_fb_string(fb, 45, ROW_IO, buf, Font_7x10, threshold_color(d->net_tx, TH_NET_WARN, TH_NET_CRIT));

    /* Disk: R/W */
    lcd_fb_char(fb, 82, ROW_IO, 'R', Font_7x10, theme.fg);
    format_rate(d->disk_read, buf, sizeof(buf));
    lcd_fb_string(fb, 89, ROW_IO, buf, Font_7x10, threshold_color(d->disk_read, TH_DIO_WARN, TH_DIO_CRIT));

    lcd_fb_char(fb, 120, ROW_IO, 'W', Font_7x10, theme.fg);
    format_rate(d->disk_write, buf, sizeof(buf));
    lcd_fb_string(fb, 127, ROW_IO, buf, Font_7x10, threshold_color(d->disk_write, TH_DIO_WARN, TH_DIO_CRIT));
}

void lcd_display_sparkline(sparkline_state_t *state) {
    SystemData data;
    collect_data(&data);

    /* Update sparkline history */
    memmove(state->cpu_history, state->cpu_history + 1, SPARKLINE_HISTORY - 1);
    state->cpu_history[SPARKLINE_HISTORY - 1] = data.cpu_pct;
    memmove(state->ram_history, state->ram_history + 1, SPARKLINE_HISTORY - 1);
    state->ram_history[SPARKLINE_HISTORY - 1] = data.ram_pct;

    /* Fill background */
    uint16_t bg = theme.bg;
    uint8_t hi = bg >> 8, lo = bg & 0xFF;
    for (uint32_t i = 0; i < sizeof(fb); i += 2) {
        fb[i] = hi;
        fb[i + 1] = lo;
    }

    /* System state zone (top half) */
    draw_ticker(state, &data);
    draw_uptime_updates(&data);
    draw_freq_disk(&data);
    lcd_fb_rect(fb, 0, ROW_DIVIDER, ST7735_WIDTH, 1, theme.sep);

    /* Live activity zone (bottom half) */
    draw_sparkline(COL_LEFT_X, state->cpu_history, TH_CPU_WARN, TH_CPU_CRIT);
    draw_sparkline(COL_RIGHT_X, state->ram_history, TH_RAM_WARN, TH_RAM_CRIT);
    draw_cpu_ram_values(&data);
    draw_io_row(&data);

    flush_dirty();
}
