#include "format.h"
#include <stdio.h>

void format_rate(uint64_t bytes, char *buf, size_t len) {
    if (bytes >= 10485760)
        snprintf(buf, len, "%uM", (unsigned)(bytes / 1048576));
    else if (bytes >= 1048576)
        snprintf(buf, len, "%.1fM", (double)bytes / 1048576.0);
    else if (bytes >= 10240)
        snprintf(buf, len, "%uK", (unsigned)(bytes / 1024));
    else if (bytes >= 1024)
        snprintf(buf, len, "%.1fK", (double)bytes / 1024.0);
    else if (bytes > 0)
        snprintf(buf, len, "%luB", (unsigned long)bytes);
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
