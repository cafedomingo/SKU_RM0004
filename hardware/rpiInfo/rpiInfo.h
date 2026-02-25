#ifndef __RPIINFO_H
#define __RPIINFO_H

#include <stdint.h>
/**********Select display temperature type**************/
#define CELSIUS 0
#define FAHRENHEIT 1
#define TEMPERATURE_TYPE CELSIUS
/**********Select display temperature type**************/

/***************Refresh interval (seconds)***************/
#define REFRESH_INTERVAL_SECS 5
/***************Refresh interval (seconds)***************/

char *get_ip_address(void);
uint8_t get_cpu_percent(void);
uint8_t get_ram_percent(void);
uint8_t get_temperature(void);
uint8_t get_disk_percent(void);
char *get_hostname(void);
int get_dietpi_update_status(void);
int get_apt_update_count(void);

#endif /*__RPIINFO_H*/