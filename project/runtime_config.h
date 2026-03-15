#ifndef __RUNTIME_CONFIG_H__
#define __RUNTIME_CONFIG_H__

#include <stdint.h>

#define CONFIG_PATH       "/etc/uctronics-display.conf"
#define SCREEN_DASHBOARD  "dashboard"
#define SCREEN_DIAGNOSTIC "diagnostic"
#define REFRESH_MIN_SECS  1
#define REFRESH_MAX_SECS  30

typedef struct {
    char screen[16];
    uint8_t refresh;
} runtime_config_t;

void load_runtime_config(runtime_config_t *cfg);

#endif /* __RUNTIME_CONFIG_H__ */
