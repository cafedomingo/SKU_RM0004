#include "runtime_config.h"
#include "log.h"
#include "rpiInfo.h"
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>

static void parse_config(runtime_config_t *cfg) {
    /* Defaults */
    strncpy(cfg->screen, SCREEN_DASHBOARD, sizeof(cfg->screen));
    cfg->screen[sizeof(cfg->screen) - 1] = '\0';
    cfg->refresh = REFRESH_INTERVAL_SECS;

    FILE *fp = fopen(CONFIG_PATH, "r");
    if (!fp) return;

    char line[128];
    while (fgets(line, sizeof(line), fp)) {
        /* Skip comments and blank lines */
        if (line[0] == '#' || line[0] == '\n') continue;

        char *eq = strchr(line, '=');
        if (!eq) continue;

        *eq = '\0';
        char *val = eq + 1;

        /* Strip trailing newline */
        char *nl = strchr(val, '\n');
        if (nl) *nl = '\0';

        if (strcmp(line, "screen") == 0) {
            if (strcmp(val, SCREEN_DASHBOARD) != 0 && strcmp(val, SCREEN_DIAGNOSTIC) != 0) continue;
            strncpy(cfg->screen, val, sizeof(cfg->screen));
            cfg->screen[sizeof(cfg->screen) - 1] = '\0';
        } else if (strcmp(line, "refresh") == 0) {
            char *end;
            long v = strtol(val, &end, 10);
            if (end != val && *end == '\0' && v >= REFRESH_MIN_SECS && v <= REFRESH_MAX_SECS) cfg->refresh = (uint8_t)v;
        }
    }
    fclose(fp);
}

void load_runtime_config(runtime_config_t *cfg) {
    static runtime_config_t prev = {.screen = "", .refresh = 0};
    static time_t last_mtime = 0;

    struct stat st;
    int have_file = (stat(CONFIG_PATH, &st) == 0);
    if (have_file && st.st_mtime == last_mtime) {
        *cfg = prev;
        return;
    }
    last_mtime = have_file ? st.st_mtime : 0;

    parse_config(cfg);

    if (strcmp(cfg->screen, prev.screen) != 0) LOG_INFO("screen=%s", cfg->screen);
    if (cfg->refresh != prev.refresh) LOG_INFO("refresh=%ds", cfg->refresh);
    prev = *cfg;
}
