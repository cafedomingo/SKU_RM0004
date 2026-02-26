/*
 * UCTRONICS ST7735 LCD display driver for Raspberry Pi
 */
#include "rpiInfo.h"
#include "st7735.h"
#include <stdio.h>
#include <unistd.h>

int main(void) {
    if (lcd_begin()) return 1;
    lcd_fill_screen(ST7735_BLACK);
    while (1) {
        lcd_display_dashboard();
        sleep(REFRESH_INTERVAL_SECS);
    }
    return 0;
}
