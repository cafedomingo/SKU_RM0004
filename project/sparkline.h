#ifndef __SPARKLINE_H__
#define __SPARKLINE_H__

#include <stdint.h>

#define SPARKLINE_HISTORY 13

typedef struct {
    uint8_t cpu_history[SPARKLINE_HISTORY];
    uint8_t ram_history[SPARKLINE_HISTORY];
    uint8_t ticker_phase;
} sparkline_state_t;

void lcd_display_sparkline(sparkline_state_t *state);

#endif /* __SPARKLINE_H__ */
