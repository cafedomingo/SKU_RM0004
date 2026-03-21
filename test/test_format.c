#include "format.h"
#include <stdio.h>
#include <string.h>

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

#define ASSERT_STR(actual, expected, msg)                                                                              \
    do {                                                                                                               \
        tests_run++;                                                                                                   \
        if (strcmp(actual, expected) != 0) {                                                                           \
            fprintf(stderr, "  FAIL: %s: got \"%s\", expected \"%s\" (%s:%d)\n", msg, actual, expected, __FILE__,      \
                    __LINE__);                                                                                         \
            tests_failed++;                                                                                            \
        } else {                                                                                                       \
            printf("  ok: %s\n", msg);                                                                                 \
        }                                                                                                              \
    } while (0)

/* ── format_rate ─────────────────────────────────────────────────── */

static void test_format_rate(void) {
    char buf[16];

    format_rate(0, buf, sizeof(buf));
    ASSERT_STR(buf, "0B", "format_rate 0 → 0B");

    format_rate(1, buf, sizeof(buf));
    ASSERT_STR(buf, "1B", "format_rate 1 → 1B");

    format_rate(500, buf, sizeof(buf));
    ASSERT_STR(buf, "500B", "format_rate 500 → 500B");

    format_rate(1023, buf, sizeof(buf));
    ASSERT_STR(buf, "1023B", "format_rate 1023 → 1023B");

    format_rate(1024, buf, sizeof(buf));
    ASSERT_STR(buf, "1.0K", "format_rate 1024 → 1.0K");

    format_rate(5120, buf, sizeof(buf));
    ASSERT_STR(buf, "5.0K", "format_rate 5120 → 5.0K");

    format_rate(10239, buf, sizeof(buf));
    ASSERT_STR(buf, "10.0K", "format_rate 10239 → 10.0K (just below integer-K)");

    format_rate(10240, buf, sizeof(buf));
    ASSERT_STR(buf, "10K", "format_rate 10240 → 10K");

    format_rate(1048575, buf, sizeof(buf));
    ASSERT_STR(buf, "1023K", "format_rate 1048575 → 1023K (just below M)");

    format_rate(1048576, buf, sizeof(buf));
    ASSERT_STR(buf, "1.0M", "format_rate 1048576 → 1.0M");

    format_rate(10485760, buf, sizeof(buf));
    ASSERT_STR(buf, "10M", "format_rate 10485760 → 10M");

    format_rate(104857600, buf, sizeof(buf));
    ASSERT_STR(buf, "100M", "format_rate 100MB → 100M");
}

/* ── format_freq ─────────────────────────────────────────────────── */

static void test_format_freq(void) {
    char buf[16];

    format_freq(0, buf, sizeof(buf));
    ASSERT_STR(buf, "0MHz", "format_freq 0 → 0MHz");

    format_freq(600, buf, sizeof(buf));
    ASSERT_STR(buf, "600MHz", "format_freq 600 → 600MHz");

    format_freq(999, buf, sizeof(buf));
    ASSERT_STR(buf, "999MHz", "format_freq 999 → 999MHz");

    format_freq(1000, buf, sizeof(buf));
    ASSERT_STR(buf, "1.0GHz", "format_freq 1000 → 1.0GHz");

    format_freq(1800, buf, sizeof(buf));
    ASSERT_STR(buf, "1.8GHz", "format_freq 1800 → 1.8GHz");

    format_freq(2400, buf, sizeof(buf));
    ASSERT_STR(buf, "2.4GHz", "format_freq 2400 → 2.4GHz");
}

/* ── format_uptime ───────────────────────────────────────────────── */

static void test_format_uptime(void) {
    char buf[16];

    format_uptime(0, buf, sizeof(buf));
    ASSERT_STR(buf, "0m", "format_uptime 0s → 0m");

    format_uptime(59, buf, sizeof(buf));
    ASSERT_STR(buf, "0m", "format_uptime 59s → 0m");

    format_uptime(60, buf, sizeof(buf));
    ASSERT_STR(buf, "1m", "format_uptime 60s → 1m");

    format_uptime(120, buf, sizeof(buf));
    ASSERT_STR(buf, "2m", "format_uptime 120s → 2m");

    format_uptime(3600, buf, sizeof(buf));
    ASSERT_STR(buf, "1h 0m", "format_uptime 3600s → 1h 0m");

    format_uptime(3700, buf, sizeof(buf));
    ASSERT_STR(buf, "1h 1m", "format_uptime 3700s → 1h 1m");

    format_uptime(86400, buf, sizeof(buf));
    ASSERT_STR(buf, "1d 0h", "format_uptime 86400s → 1d 0h");

    format_uptime(90061, buf, sizeof(buf));
    ASSERT_STR(buf, "1d 1h", "format_uptime 90061s → 1d 1h");
}

/* ── format_temp ─────────────────────────────────────────────────── */

static void test_format_temp(void) {
    char buf[8];

    /* TEMPERATURE_TYPE is CELSIUS at compile time */
    format_temp(52, buf, sizeof(buf));
    ASSERT_STR(buf, "52C", "format_temp 52 → 52C");

    format_temp(0, buf, sizeof(buf));
    ASSERT_STR(buf, "0C", "format_temp 0 → 0C");
}

/* ── celsius_to_f ────────────────────────────────────────────────── */

static void test_celsius_to_f(void) {
    ASSERT(celsius_to_f(0) == 32, "celsius_to_f 0 → 32");
    ASSERT(celsius_to_f(100) == 212, "celsius_to_f 100 → 212");
    ASSERT(celsius_to_f(50) == 122, "celsius_to_f 50 → 122");
}

/* ── format_apt_badge ────────────────────────────────────────────── */

static void test_format_apt_badge(void) {
    char buf[8];

    ASSERT(format_apt_badge(0, buf, sizeof(buf)) == 0, "format_apt_badge 0 returns 0");
    ASSERT(format_apt_badge(-1, buf, sizeof(buf)) == 0, "format_apt_badge -1 returns 0");

    ASSERT(format_apt_badge(3, buf, sizeof(buf)) == 1, "format_apt_badge 3 returns 1");
    ASSERT_STR(buf, "^3", "format_apt_badge 3 → ^3");

    format_apt_badge(100, buf, sizeof(buf));
    ASSERT_STR(buf, "^99", "format_apt_badge 100 capped at 99");

    format_apt_badge(1, buf, sizeof(buf));
    ASSERT_STR(buf, "^1", "format_apt_badge 1 → ^1");

    format_apt_badge(99, buf, sizeof(buf));
    ASSERT_STR(buf, "^99", "format_apt_badge 99 → ^99");
}

int main(void) {
    printf("format tests:\n");

    test_format_rate();
    test_format_freq();
    test_format_uptime();
    test_format_temp();
    test_celsius_to_f();
    test_format_apt_badge();

    printf("\n%d/%d tests passed.\n", tests_run - tests_failed, tests_run);
    return tests_failed > 0 ? 1 : 0;
}
