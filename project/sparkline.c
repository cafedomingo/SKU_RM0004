#include "sparkline.h"
#include "rpiInfo.h"
#include "st7735.h"
#include "theme.h"
#include <math.h>
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
#define ROW_SPARK_BOT 56
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

    /* Format uptime as "Nd Nh" */
    uint32_t secs = get_uptime_secs();
    uint32_t days = secs / 86400;
    uint32_t hours = (secs % 86400) / 3600;
    snprintf(d->uptime, sizeof(d->uptime), "%ud %uh", days, hours);

    d->apt_count = (uint8_t)(get_apt_update_count() > 0 ? get_apt_update_count() : 0);
    d->dietpi_update = (get_dietpi_update_status() == 2);

    uint32_t thr = get_cpu_throttle_status();
    d->throttled = (thr & THROTTLE_THROTTLED) != 0;
}

static void format_rate(uint32_t bytes, char *buf, size_t len) {
    if (bytes >= 10485760)
        snprintf(buf, len, "%uM", bytes / 1048576);
    else if (bytes >= 1048576)
        snprintf(buf, len, "%.1fM", bytes / 1048576.0);
    else if (bytes >= 10240)
        snprintf(buf, len, "%uK", bytes / 1024);
    else if (bytes >= 1024)
        snprintf(buf, len, "%.1fK", bytes / 1024.0);
    else if (bytes > 0)
        snprintf(buf, len, "%uB", bytes);
    else
        snprintf(buf, len, "0B");
}

static void fb_string_right(uint8_t *buf, uint16_t rx, uint16_t y, const char *str, uint16_t color) {
    uint16_t w = strlen(str) * Font_7x10.width;
    uint16_t x = (rx + 1 >= w) ? rx + 1 - w : 0;
    lcd_fb_string(buf, x, y, str, Font_7x10, color);
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

    /* Alert badges — build from right edge inward */
    uint16_t ax = ST7735_WIDTH; /* next available x (moving left) */

    /* APT badge: ^N */
    if (d->apt_count > 0) {
        char badge[5];
        int capped = d->apt_count > 99 ? 99 : d->apt_count;
        snprintf(badge, sizeof(badge), "^%d", capped);
        uint16_t color = (d->apt_count >= 10) ? theme.crit : theme.warn;
        uint16_t bw = strlen(badge) * Font_7x10.width;
        ax -= bw;
        lcd_fb_string(fb, ax, ROW_TICKER, badge, Font_7x10, color);
    }

    /* DietPi diamond (4x4 pixel art, 3px gap left of APT badge) */
    if (d->dietpi_update) {
        ax -= 3; /* gap */
        uint16_t dx = ax - 4;
        lcd_fb_pixel(fb, dx + 1, ROW_TICKER + 3, theme.alert);
        lcd_fb_pixel(fb, dx + 2, ROW_TICKER + 3, theme.alert);
        lcd_fb_pixel(fb, dx + 0, ROW_TICKER + 4, theme.alert);
        lcd_fb_pixel(fb, dx + 1, ROW_TICKER + 4, theme.alert);
        lcd_fb_pixel(fb, dx + 2, ROW_TICKER + 4, theme.alert);
        lcd_fb_pixel(fb, dx + 3, ROW_TICKER + 4, theme.alert);
        lcd_fb_pixel(fb, dx + 0, ROW_TICKER + 5, theme.alert);
        lcd_fb_pixel(fb, dx + 1, ROW_TICKER + 5, theme.alert);
        lcd_fb_pixel(fb, dx + 2, ROW_TICKER + 5, theme.alert);
        lcd_fb_pixel(fb, dx + 3, ROW_TICKER + 5, theme.alert);
        lcd_fb_pixel(fb, dx + 1, ROW_TICKER + 6, theme.alert);
        lcd_fb_pixel(fb, dx + 2, ROW_TICKER + 6, theme.alert);
    }
}

/*
 * Row 2 (y=12): uptime left, temperature right-aligned.
 */
static void draw_uptime_temp(const SystemData *d) {
    lcd_fb_string(fb, 0, ROW_UPTIME, d->uptime, Font_7x10, theme.fg);

    char temp_str[6];
    snprintf(temp_str, sizeof(temp_str), "%uC", d->temp_c);
    fb_string_right(fb, ST7735_WIDTH - 1, ROW_UPTIME, temp_str, temp_ramp_color(d->temp_c));
}

/*
 * Row 3 (y=23): frequency+throttle left, D:N% right-aligned.
 */
static void draw_freq_disk(const SystemData *d) {
    /* Frequency */
    char freq_str[10];
    snprintf(freq_str, sizeof(freq_str), "%uMHz", d->freq_mhz);
    lcd_fb_string(fb, 0, ROW_FREQ, freq_str, Font_7x10, theme.fg);
    if (d->throttled) {
        uint16_t bang_x = strlen(freq_str) * Font_7x10.width;
        lcd_fb_char(fb, bang_x, ROW_FREQ, '!', Font_7x10, theme.alert);
    }

    /* Disk: "D:N%" right-aligned */
    char disk_val[5];
    snprintf(disk_val, sizeof(disk_val), "%u%%", d->disk_pct);
    uint16_t lbl_w = 2 * Font_7x10.width; /* "D:" */
    uint16_t val_w = strlen(disk_val) * Font_7x10.width;
    uint16_t dx = COL_RIGHT_X + COL_WIDTH - lbl_w - val_w;
    lcd_fb_string(fb, dx, ROW_FREQ, "D:", Font_7x10, theme.fg);
    lcd_fb_string(fb, dx + lbl_w, ROW_FREQ, disk_val, Font_7x10,
                  threshold_color(d->disk_pct, TH_DISK_WARN, TH_DISK_CRIT));
}

/*
 * Draw a single sparkline chart (13 bars growing upward from ROW_SPARK_BOT).
 */
static void draw_sparkline(uint16_t start_x, const uint8_t *history, uint32_t warn_th, uint32_t crit_th) {
    for (int i = 0; i < SPARKLINE_HISTORY; i++) {
        uint8_t val = history[i];
        if (val == 0) continue;
        int col_height = (int)round(val * SPARK_HEIGHT / 100.0);
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

/*
 * Pixel-art down arrow: 5px wide x 7px tall, drawn at (x, y+2) to center in 10px row.
 */
static void draw_arrow_down(uint16_t x, uint16_t y, uint16_t color) {
    uint16_t by = y + 2;
    /* Stem: 1px wide x 4px */
    for (int r = 0; r < 4; r++)
        lcd_fb_pixel(fb, x + 2, by + r, color);
    /* Bar: 5px wide */
    for (int c = 0; c < 5; c++)
        lcd_fb_pixel(fb, x + c, by + 4, color);
    /* Mid: 3px wide */
    for (int c = 1; c < 4; c++)
        lcd_fb_pixel(fb, x + c, by + 5, color);
    /* Tip: 1px */
    lcd_fb_pixel(fb, x + 2, by + 6, color);
}

/*
 * Pixel-art up arrow: 5px wide x 7px tall, drawn at (x, y+2) to center in 10px row.
 */
static void draw_arrow_up(uint16_t x, uint16_t y, uint16_t color) {
    uint16_t by = y + 2;
    /* Tip: 1px */
    lcd_fb_pixel(fb, x + 2, by, color);
    /* Mid: 3px wide */
    for (int c = 1; c < 4; c++)
        lcd_fb_pixel(fb, x + c, by + 1, color);
    /* Bar: 5px wide */
    for (int c = 0; c < 5; c++)
        lcd_fb_pixel(fb, x + c, by + 2, color);
    /* Stem: 1px wide x 4px */
    for (int r = 3; r < 7; r++)
        lcd_fb_pixel(fb, x + 2, by + r, color);
}

/*
 * I/O row (y=69): net ↓↑ left column, disk R/W right column.
 */
static void draw_io_row(const SystemData *d) {
    char buf[6];

    /* Network: ↓rx ↑tx */
    draw_arrow_down(0, ROW_IO, theme.fg);
    format_rate(d->net_rx, buf, sizeof(buf));
    lcd_fb_string(fb, 7, ROW_IO, buf, Font_7x10, threshold_color(d->net_rx, TH_NET_WARN, TH_NET_CRIT));

    draw_arrow_up(38, ROW_IO, theme.fg);
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
    draw_uptime_temp(&data);
    draw_freq_disk(&data);
    lcd_fb_rect(fb, 0, ROW_DIVIDER, ST7735_WIDTH, 1, theme.sep);

    /* Live activity zone (bottom half) */
    draw_sparkline(COL_LEFT_X, state->cpu_history, TH_CPU_WARN, TH_CPU_CRIT);
    draw_sparkline(COL_RIGHT_X, state->ram_history, TH_RAM_WARN, TH_RAM_CRIT);
    draw_cpu_ram_values(&data);
    draw_io_row(&data);

    lcd_draw_fullscreen(fb);
}
