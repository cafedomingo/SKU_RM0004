/******
Demo for ssd1306 i2c driver for  Raspberry Pi 
******/
#include <stdio.h>
#include "st7735.h"
#include "time.h"
#include <unistd.h>



int main(void) 
{
	uint8_t symbol = 4;

	if(lcd_begin())      //LCD Screen initialization
	{
		return 0;
	}
	sleep(1);
	while(1)
	{
		lcd_display(symbol);
		sleep(5);
	}
	return 0;
}
