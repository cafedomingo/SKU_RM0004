#ifndef __FORMAT_H__
#define __FORMAT_H__

#include <stddef.h>
#include <stdint.h>

#define KB 1024
#define MB (1024 * 1024)

/* Format bytes/s as human-readable rate: 0B, 1.2K, 45K, 1.2M, 12M */
void format_rate(uint64_t bytes, char *buf, size_t len);

/* Format CPU frequency: "1.8GHz" when >= 1000 MHz, "600MHz" otherwise */
void format_freq(uint16_t mhz, char *buf, size_t len);

/* Format uptime: "3d 2h", "5h 12m", or "42m" */
void format_uptime(uint32_t secs, char *buf, size_t len);

/* Celsius to Fahrenheit conversion */
static inline int celsius_to_f(uint8_t c) { return (int)c * 9 / 5 + 32; }

/* Format temperature with unit: "52C" or "125F" depending on TEMPERATURE_TYPE */
void format_temp(uint8_t celsius, char *buf, size_t len);

#define APT_BADGE_MAX 99

/* Format APT badge: "^3", capped at APT_BADGE_MAX. Returns 0 if count <= 0. */
int format_apt_badge(int count, char *buf, size_t len);

#endif /* __FORMAT_H__ */
