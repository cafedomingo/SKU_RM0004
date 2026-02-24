/*
 * UCTRONICS ST7735 LCD display driver for Raspberry Pi
 */
#include <stdio.h>
#include <unistd.h>
#include "st7735.h"
#include "rpiInfo.h"

int main(void)
{
	if (lcd_begin())
		return 1;
	sleep(1);
	while (1) {
		lcd_display_all();
		sleep(REFRESH_INTERVAL_SECS);
	}
	return 0;
}
