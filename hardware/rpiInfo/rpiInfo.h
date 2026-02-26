#ifndef __RPIINFO_H
#define __RPIINFO_H

#include <stdint.h>

/* Temperature unit: CELSIUS or FAHRENHEIT */
#define CELSIUS          0
#define FAHRENHEIT       1
#define TEMPERATURE_TYPE CELSIUS

/* Seconds between display refreshes */
#define REFRESH_INTERVAL_SECS 5

int get_apt_update_count(void);
uint8_t get_cpu_percent(void);
int get_dietpi_update_status(void);
uint8_t get_disk_percent(void);
char *get_hostname(void);
char *get_ip_address(void);
uint8_t get_ram_percent(void);
uint8_t get_temperature(void);

#endif /* __RPIINFO_H */