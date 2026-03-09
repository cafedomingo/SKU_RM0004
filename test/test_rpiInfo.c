#include "rpiInfo.h"
#include <stdio.h>
#include <string.h>
#include <unistd.h>

static int tests_run = 0;
static int tests_failed = 0;

#define ASSERT(cond, msg)                                                                                              \
    do {                                                                                                               \
        tests_run++;                                                                                                   \
        if (!(cond)) {                                                                                                 \
            fprintf(stderr, "  FAIL: %s (%s:%d)\n", msg, __FILE__, __LINE__);                                          \
            tests_failed++;                                                                                            \
        } else {                                                                                                       \
            printf("  ok: %s\n", msg);                                                                                 \
        }                                                                                                              \
    } while (0)

static void test_cpu_percent(void) {
    /* Delta-based: may return 0 on first call if idle ticks haven't changed */
    uint8_t pct = get_cpu_percent();
    ASSERT(pct <= 100, "get_cpu_percent returns 0-100");
}

static void test_ram_percent(void) {
    uint8_t pct = get_ram_percent();
    ASSERT(pct > 0 && pct <= 100, "get_ram_percent returns 1-100");
}

static void test_disk_percent(void) {
    uint8_t pct = get_disk_percent();
    ASSERT(pct > 0 && pct <= 100, "get_disk_percent returns 1-100");
}

static void test_ip_address(void) {
    char *ip = get_ip_address();
    ASSERT(ip != NULL, "get_ip_address returns non-NULL");
    ASSERT(strlen(ip) > 0, "get_ip_address returns non-empty string");
}

static void test_hostname(void) {
    char *name = get_hostname();
    ASSERT(name != NULL, "get_hostname returns non-NULL");
    ASSERT(strlen(name) > 0, "get_hostname returns non-empty string");
}

static void test_temperature(void) {
    uint8_t temp = get_temperature();
    /* May be 0 on CI (no thermal zone), but should never be absurdly high */
    ASSERT(temp <= 150, "get_temperature returns 0-150");
}

static void test_uptime(void) {
    uint32_t secs = get_uptime_secs();
    ASSERT(secs > 0, "get_uptime_secs returns > 0");
}

static void test_cpu_freq(void) {
    /* May return zeros on CI if no cpufreq driver — just verify no crash */
    cpu_freq_t freq = get_cpu_freq();
    ASSERT(freq.cur_mhz <= 10000, "get_cpu_freq cur_mhz is reasonable");
    ASSERT(freq.min_mhz <= 10000 && freq.max_mhz <= 10000, "get_cpu_freq min/max are reasonable");
}

static void test_cpu_throttle_status(void) {
    uint32_t status = get_cpu_throttle_status();
    if (access("/dev/vcio", R_OK | W_OK) == 0) {
        /* On a real Pi with permissions, the result is a valid throttle bitmask.
         * Only the defined bits (0-3, 16-19) should be set. */
        ASSERT((status & ~0x000F000Fu) == 0, "get_cpu_throttle_status returns valid bitmask");
    } else {
        (void)status;
        ASSERT(1, "get_cpu_throttle_status does not crash (no vcio access)");
    }
}

static void test_dietpi_update_status(void) {
    int status = get_dietpi_update_status();
    ASSERT(status >= 0 && status <= 2, "get_dietpi_update_status returns 0-2");
}

static void test_apt_update_count(void) {
    int count = get_apt_update_count();
    /* -1 if no DietPi cache, >= 0 otherwise */
    ASSERT(count >= -1, "get_apt_update_count returns >= -1");
}

static void test_net_bandwidth(void) {
    net_bandwidth_t bw1 = get_net_bandwidth();
    ASSERT(bw1.rx_bytes_per_sec == 0 && bw1.tx_bytes_per_sec == 0, "get_net_bandwidth first call returns zeros");

    usleep(100000); /* 100ms */

    net_bandwidth_t bw2 = get_net_bandwidth();
    /* Second call should succeed (values may be 0 if no traffic) */
    (void)bw2;
    ASSERT(1, "get_net_bandwidth second call does not crash");
}

static void test_disk_io(void) {
    disk_io_t io1 = get_disk_io();
    ASSERT(io1.read_bytes_per_sec == 0 && io1.write_bytes_per_sec == 0 && io1.read_iops == 0 && io1.write_iops == 0,
           "get_disk_io first call returns zeros");

    usleep(100000); /* 100ms */

    disk_io_t io2 = get_disk_io();
    /* Second call should succeed (values may be 0 if no I/O) */
    (void)io2;
    ASSERT(1, "get_disk_io second call does not crash");
}

int main(void) {
    printf("rpiInfo tests:\n");

    test_cpu_percent();
    test_ram_percent();
    test_disk_percent();
    test_ip_address();
    test_hostname();
    test_temperature();
    test_uptime();
    test_cpu_freq();
    test_cpu_throttle_status();
    test_dietpi_update_status();
    test_apt_update_count();
    test_net_bandwidth();
    test_disk_io();

    printf("\n%d/%d tests passed.\n", tests_run - tests_failed, tests_run);
    return tests_failed > 0 ? 1 : 0;
}
