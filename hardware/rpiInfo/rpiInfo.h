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

uint8_t get_cpu_percent(void);

/* ── Disk ────────────────────────────────────────────────────────── */

uint8_t get_disk_percent(void);

/* ── System ──────────────────────────────────────────────────────── */

uint8_t get_ram_percent(void);
uint8_t get_temperature(void);
char *get_hostname(void);

/* ── DietPi ──────────────────────────────────────────────────────── */

int get_dietpi_update_status(void);
int get_apt_update_count(void);

#endif /* __RPIINFO_H */
