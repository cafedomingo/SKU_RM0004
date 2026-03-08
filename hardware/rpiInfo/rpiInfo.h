#ifndef __RPIINFO_H
#define __RPIINFO_H

#include <stdint.h>

/* Temperature unit: CELSIUS or FAHRENHEIT */
#define CELSIUS          0
#define FAHRENHEIT       1
#define TEMPERATURE_TYPE CELSIUS

/* Seconds between display refreshes */
#define REFRESH_INTERVAL_SECS 5

/* ── Network ─────────────────────────────────────────────────────── */

typedef struct {
    uint64_t rx_bytes_per_sec;
    uint64_t tx_bytes_per_sec;
} net_bandwidth_t;

char *get_ip_address(void);
net_bandwidth_t get_net_bandwidth(void);

/* ── CPU ─────────────────────────────────────────────────────────── */

typedef struct {
    uint16_t cur_mhz;
    uint16_t min_mhz;
    uint16_t max_mhz;
} cpu_freq_t;

/* Throttle status bitmask — current conditions (bits 0-3) */
#define THROTTLE_UNDERVOLTAGE    (1 << 0) /* Under-voltage detected */
#define THROTTLE_FREQ_CAPPED     (1 << 1) /* ARM frequency capped */
#define THROTTLE_THROTTLED       (1 << 2) /* Currently throttled */
#define THROTTLE_SOFT_TEMP_LIMIT (1 << 3) /* Soft temperature limit active */

/* Throttle status bitmask — has occurred since boot (bits 16-19) */
#define THROTTLE_UNDERVOLTAGE_PAST    (1 << 16) /* Under-voltage has occurred */
#define THROTTLE_FREQ_CAPPED_PAST     (1 << 17) /* Frequency capping has occurred */
#define THROTTLE_THROTTLED_PAST       (1 << 18) /* Throttling has occurred */
#define THROTTLE_SOFT_TEMP_LIMIT_PAST (1 << 19) /* Soft temp limit has occurred */

#define THROTTLE_CURRENT_MASK 0x0000000F
#define THROTTLE_PAST_MASK    0x000F0000

uint8_t get_cpu_percent(void);
cpu_freq_t get_cpu_freq(void);
uint32_t get_cpu_throttle_status(void);

/* ── Disk ────────────────────────────────────────────────────────── */

typedef struct {
    uint64_t read_bytes_per_sec;
    uint64_t write_bytes_per_sec;
    uint32_t read_iops;
    uint32_t write_iops;
} disk_io_t;

uint8_t get_disk_percent(void);
disk_io_t get_disk_io(void);

/* ── System ──────────────────────────────────────────────────────── */

uint8_t get_ram_percent(void);
uint8_t get_temperature(void);
char *get_hostname(void);

/* ── DietPi ──────────────────────────────────────────────────────── */

int get_dietpi_update_status(void);
int get_apt_update_count(void);

#endif /* __RPIINFO_H */
