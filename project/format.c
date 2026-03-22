#include "format.h"
#include "rpiInfo.h"
#include <inttypes.h>
#include <stdio.h>

void format_rate(uint64_t bytes, char *buf, size_t len) {
    if (bytes >= 10 * MB)
        snprintf(buf, len, "%" PRIu64 "M", bytes / MB);
    else if (bytes >= MB)
        snprintf(buf, len, "%.1fM", (double)bytes / MB);
    else if (bytes >= 10 * KB)
        snprintf(buf, len, "%" PRIu64 "K", bytes / KB);
    else if (bytes >= KB)
        snprintf(buf, len, "%.1fK", (double)bytes / KB);
    else if (bytes > 0)
        snprintf(buf, len, "%" PRIu64 "B", bytes);
    else
        snprintf(buf, len, "0B");
}

void format_freq(uint16_t mhz, char *buf, size_t len) {
    if (mhz >= 1000)
        snprintf(buf, len, "%.1fGHz", mhz / 1000.0);
    else
        snprintf(buf, len, "%uMHz", mhz);
}

void format_uptime(uint32_t secs, char *buf, size_t len) {
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

void format_temp(uint8_t celsius, char *buf, size_t len) {
    if (TEMPERATURE_TYPE == FAHRENHEIT)
        snprintf(buf, len, "%3dF", celsius_to_f(celsius));
    else
        snprintf(buf, len, "%3dC", celsius);
}

int format_apt_badge(int count, char *buf, size_t len) {
    if (count <= 0) return 0;
    int capped = count > APT_BADGE_MAX ? APT_BADGE_MAX : count;
    snprintf(buf, len, "^%d", capped);
    return 1;
}
