/*
 * UCTRONICS ST7735 LCD display driver for Raspberry Pi
 */
#include "dashboard.h"
#include "diagnostic.h"
#include "log.h"
#include "runtime_config.h"
#include "sparkline.h"
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

static void sleep_remaining(long before, uint8_t refresh) {
    long elapsed_s = (now_ms() - before) / 1000;
    if (elapsed_s < refresh) sleep(refresh - elapsed_s);
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
    char prev_screen[16] = "";
    int diag_page = 0;
    sparkline_state_t spark_state = {0};

    while (1) {
        load_runtime_config(&cfg);

        /* Clear display on screen change */
        if (strcmp(cfg.screen, prev_screen) != 0) {
            lcd_fill_screen(ST7735_BLACK);
            snprintf(prev_screen, sizeof(prev_screen), "%s", cfg.screen);
            diag_page = 0;
            sparkline_invalidate();
        }

        if (strcmp(cfg.screen, SCREEN_DIAGNOSTIC) == 0) {
            if (diag_page == 0) diag_refresh_data();
            lcd_display_diagnostic_page(diag_page);
            diag_page = (diag_page + 1) % DIAG_NUM_PAGES;
            sleep(cfg.refresh);
        } else if (strcmp(cfg.screen, SCREEN_SPARKLINE) == 0) {
            diag_page = 0;
            long before = now_ms();
            lcd_display_sparkline(&spark_state);
            sleep_remaining(before, cfg.refresh);
        } else {
            diag_page = 0;
            long before = now_ms();
            lcd_display_dashboard();
            sleep_remaining(before, cfg.refresh);
        }
    }
    return 0;
}
