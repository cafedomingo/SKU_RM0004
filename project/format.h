#ifndef __FORMAT_H__
#define __FORMAT_H__

#include <stddef.h>
#include <stdint.h>

/* Format bytes/s as human-readable rate: 0B, 1.2K, 45K, 1.2M, 12M */
void format_rate(uint64_t bytes, char *buf, size_t len);

/* Format CPU frequency: "1.8GHz" when >= 1000 MHz, "600MHz" otherwise */
void format_freq(uint16_t mhz, char *buf, size_t len);

/* Format uptime: "3d 2h", "5h 12m", or "42m" */
void format_uptime(uint32_t secs, char *buf, size_t len);

#endif /* __FORMAT_H__ */
