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

void lcd_display_sparkline(sparkline_state_t *state) {
    SystemData data;
    collect_data(&data);

    /* Fill background */
    uint16_t bg = theme.bg;
    uint8_t hi = bg >> 8, lo = bg & 0xFF;
    for (uint32_t i = 0; i < sizeof(fb); i += 2) {
        fb[i] = hi;
        fb[i + 1] = lo;
    }

    lcd_draw_fullscreen(fb);
}
