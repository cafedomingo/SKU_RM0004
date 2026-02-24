#ifndef  __RPIINFO_H
#define  __RPIINFO_H

#include <stdint.h>
/**********Select display temperature type**************/
#define CELSIUS       0
#define FAHRENHEIT    1
#define TEMPERATURE_TYPE  CELSIUS
/**********Select display temperature type**************/

/***************Refresh interval (seconds)***************/
#define REFRESH_INTERVAL_SECS  5
/***************Refresh interval (seconds)***************/


char* get_ip_address(void);
void get_sd_memory(uint32_t *MemSize, uint32_t *freesize);
void get_cpu_memory(float *Totalram, float *availram);
uint8_t get_temperature(void);
uint8_t get_cpu_message(void);
uint8_t get_hard_disk_memory(uint16_t *diskMemSize, uint16_t *useMemSize);
char* get_hostname(void);
int get_dietpi_update_status(void);
int get_apt_update_count(void);

#endif /*__RPIINFO_H*/