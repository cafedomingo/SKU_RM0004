/*
 * UCTRONICS ST7735 LCD display driver for Raspberry Pi
 */
#include "dashboard.h"
#include "diagnostic.h"
#include "log.h"
#include "rpiInfo.h"
#include "runtime_config.h"
#include "st7735.h"
#include <arpa/inet.h>
#include <errno.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

#define I2C_EXPECTED_HZ     400000
#define I2C_CLOCK_FREQ_PATH "/proc/device-tree/soc/i2c@7e804000/clock-frequency"

static void check_i2c_speed(void) {
    FILE *f = fopen(I2C_CLOCK_FREQ_PATH, "rb");
    if (!f) {
        LOG_WARN("could not open %s: %s", I2C_CLOCK_FREQ_PATH, strerror(errno));
        return;
    }
    uint32_t freq_be;
    if (fread(&freq_be, sizeof(freq_be), 1, f) != 1) {
        fclose(f);
        LOG_WARN("could not read %s", I2C_CLOCK_FREQ_PATH);
        return;
    }
    fclose(f);
    uint32_t freq = ntohl(freq_be);
    if (freq == I2C_EXPECTED_HZ) {
        LOG_INFO("I2C bus speed: %u Hz", freq);
    } else {
        LOG_WARN("I2C bus speed is %u Hz (expected %u Hz)", freq, I2C_EXPECTED_HZ);
    }
}

static long now_ms(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1000L + ts.tv_nsec / 1000000L;
}

int main(void) {
    LOG_INFO("starting");
    check_i2c_speed();
    if (lcd_begin()) {
        LOG_ERROR("lcd_begin failed, exiting");
        return 1;
    }
    lcd_fill_screen(ST7735_BLACK);

    runtime_config_t cfg;
    uint8_t scroll_offset = 0;
    long last_refresh_ms = 0;

    while (1) {
        load_runtime_config(&cfg);

        if (strcmp(cfg.screen, SCREEN_DIAGNOSTIC) == 0) {
            long refresh_ms = cfg.refresh * 1000L;
            if (now_ms() - last_refresh_ms >= refresh_ms) {
                diag_refresh_data();
                last_refresh_ms = now_ms();
            }
            lcd_display_diagnostic(scroll_offset);
            scroll_offset = (scroll_offset + 1) % DIAG_TOTAL_ROWS;
        } else {
            scroll_offset = 0;
            last_refresh_ms = 0;
            long before = now_ms();
            lcd_display_dashboard();
            long elapsed_s = (now_ms() - before) / 1000;
            if (elapsed_s < cfg.refresh) sleep(cfg.refresh - elapsed_s);
        }
    }
    return 0;
}
