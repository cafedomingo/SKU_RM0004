#ifndef __THEME_H__
#define __THEME_H__

#include <stdint.h>

typedef struct {
    uint16_t bg, fg, sep, ip, alert;
    uint16_t ok, warn, crit;
    uint16_t tempRamp[4];
} Theme;

extern const Theme theme;

/* Threshold constants — percentage metrics */
#define TH_CPU_WARN  60
#define TH_CPU_CRIT  80
#define TH_RAM_WARN  60
#define TH_RAM_CRIT  80
#define TH_DISK_WARN 70
#define TH_DISK_CRIT 90

/* Threshold constants — I/O throughput (bytes/s) */
#define TH_NET_WARN 1048576  /* 1 MB/s */
#define TH_NET_CRIT 10485760 /* 10 MB/s */
#define TH_DIO_WARN 524288   /* 512 KB/s */
#define TH_DIO_CRIT 5242880  /* 5 MB/s */

/* APT badge: warn→crit color threshold */
#define TH_APT_CRIT 10

/* Temperature ramp breakpoints (Celsius) */
#define TEMP_COLD 30
#define TEMP_COOL 50
#define TEMP_WARM 65
#define TEMP_HOT  85

uint16_t threshold_color(uint32_t value, uint32_t warn_th, uint32_t crit_th);
uint16_t temp_ramp_color(uint8_t temp_c);

#endif /* __THEME_H__ */
