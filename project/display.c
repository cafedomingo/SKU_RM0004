/*
 * UCTRONICS ST7735 LCD display driver for Raspberry Pi
 */
#include "dashboard.h"
#include "rpiInfo.h"
#include "st7735.h"
#include <stdio.h>
#include <unistd.h>

int main(void) {
    fprintf(stderr, "display: starting (refresh every %ds)\n", REFRESH_INTERVAL_SECS);
    if (lcd_begin()) {
        fprintf(stderr, "display: lcd_begin failed, exiting\n");
        return 1;
    }
    lcd_fill_screen(ST7735_BLACK);
    while (1) {
        lcd_display_dashboard();
        sleep(REFRESH_INTERVAL_SECS);
    }
    return 0;
}
