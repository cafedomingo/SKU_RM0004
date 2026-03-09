/*
 * UCTRONICS ST7735 LCD display driver for Raspberry Pi
 */
#include "dashboard.h"
#include "rpiInfo.h"
#include "st7735.h"
#include <arpa/inet.h>
#include <stdio.h>
#include <unistd.h>

#define I2C_EXPECTED_HZ     400000
#define I2C_CLOCK_FREQ_PATH "/proc/device-tree/soc/i2c@7e804000/clock-frequency"

static void check_i2c_speed(void) {
    FILE *f = fopen(I2C_CLOCK_FREQ_PATH, "rb");
    if (!f) {
        fprintf(stderr, "display: WARNING: could not read I2C bus speed\n");
        return;
    }
    uint32_t freq_be;
    if (fread(&freq_be, sizeof(freq_be), 1, f) != 1) {
        fclose(f);
        fprintf(stderr, "display: WARNING: could not read I2C bus speed\n");
        return;
    }
    fclose(f);
    uint32_t freq = ntohl(freq_be);
    if (freq == I2C_EXPECTED_HZ) {
        fprintf(stderr, "display: I2C bus speed: %u Hz\n", freq);
    } else {
        fprintf(stderr,
                "display: WARNING: I2C bus speed is %u Hz"
                " (expected %u Hz)\n",
                freq, I2C_EXPECTED_HZ);
    }
}

int main(void) {
    fprintf(stderr, "display: starting (refresh every %ds)\n", REFRESH_INTERVAL_SECS);
    check_i2c_speed();
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
