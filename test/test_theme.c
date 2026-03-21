#include "theme.h"
#include <stdio.h>

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

/* ── threshold_color ─────────────────────────────────────────────── */

static void test_threshold_color(void) {
    /* Standard thresholds (60/80) */
    ASSERT(threshold_color(0, 60, 80) == theme.ok, "threshold_color 0 → ok");
    ASSERT(threshold_color(59, 60, 80) == theme.ok, "threshold_color 59 → ok");
    ASSERT(threshold_color(60, 60, 80) == theme.warn, "threshold_color 60 → warn");
    ASSERT(threshold_color(79, 60, 80) == theme.warn, "threshold_color 79 → warn");
    ASSERT(threshold_color(80, 60, 80) == theme.crit, "threshold_color 80 → crit");
    ASSERT(threshold_color(100, 60, 80) == theme.crit, "threshold_color 100 → crit");

    /* Different thresholds (disk: 70/90) */
    ASSERT(threshold_color(69, 70, 90) == theme.ok, "threshold_color 69 (70/90) → ok");
    ASSERT(threshold_color(70, 70, 90) == theme.warn, "threshold_color 70 (70/90) → warn");
    ASSERT(threshold_color(89, 70, 90) == theme.warn, "threshold_color 89 (70/90) → warn");
    ASSERT(threshold_color(90, 70, 90) == theme.crit, "threshold_color 90 (70/90) → crit");

    /* Large values (I/O thresholds use bytes/s) */
    ASSERT(threshold_color(500000, 1048576, 10485760) == theme.ok, "threshold_color 500K I/O → ok");
    ASSERT(threshold_color(5000000, 1048576, 10485760) == theme.warn, "threshold_color 5M I/O → warn");
    ASSERT(threshold_color(20000000, 1048576, 10485760) == theme.crit, "threshold_color 20M I/O → crit");

    /* Edge: warn == crit */
    ASSERT(threshold_color(49, 50, 50) == theme.ok, "threshold_color warn==crit below → ok");
    ASSERT(threshold_color(50, 50, 50) == theme.crit, "threshold_color warn==crit at → crit");
}

/* ── temp_ramp_color ─────────────────────────────────────────────── */

static void test_temp_ramp_color(void) {
    /* Cold end should be the first ramp color */
    ASSERT(temp_ramp_color(0) == theme.tempRamp[0], "temp_ramp_color 0 → tempRamp[0]");

    /* Hot end should be the last ramp color */
    ASSERT(temp_ramp_color(90) == theme.tempRamp[3], "temp_ramp_color 90 → tempRamp[3]");
    ASSERT(temp_ramp_color(255) == theme.tempRamp[3], "temp_ramp_color 255 → tempRamp[3]");

    /* Exact breakpoints: endpoint of one segment = start of the next */
    ASSERT(temp_ramp_color(TEMP_COLD) == theme.tempRamp[0], "temp_ramp_color TEMP_COLD → tempRamp[0]");
    ASSERT(temp_ramp_color(TEMP_COOL) == theme.tempRamp[1], "temp_ramp_color TEMP_COOL → tempRamp[1]");
    ASSERT(temp_ramp_color(TEMP_WARM) == theme.tempRamp[2], "temp_ramp_color TEMP_WARM → tempRamp[2]");
    ASSERT(temp_ramp_color(TEMP_HOT) == theme.tempRamp[3], "temp_ramp_color TEMP_HOT → tempRamp[3]");

    /* Mid-range should not be an endpoint */
    uint16_t mid = temp_ramp_color(57);
    ASSERT(mid != theme.tempRamp[0] && mid != theme.tempRamp[3], "temp_ramp_color 57 is interpolated");

    /* Monotonic: hotter temps should produce different colors than cooler ones */
    ASSERT(temp_ramp_color(40) != temp_ramp_color(70), "temp_ramp_color 40 ≠ 70");
}

int main(void) {
    printf("theme tests:\n");

    test_threshold_color();
    test_temp_ramp_color();

    printf("\n%d/%d tests passed.\n", tests_run - tests_failed, tests_run);
    return tests_failed > 0 ? 1 : 0;
}
