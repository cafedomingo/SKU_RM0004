#include "rpiInfo.h"
#include <arpa/inet.h>
#include <net/if.h>
#include <netinet/in.h>
#include <stdio.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <sys/vfs.h>
#include <unistd.h>

static inline int has_prefix(const char *s, const char *prefix) { return strncmp(s, prefix, strlen(prefix)) == 0; }

/*
 * Get the IP address of the default-route interface.
 * Auto-detects the interface via /proc/net/route instead of
 * hardcoding eth0/wlan0, so it works on any Linux system
 * (Armbian end0, USB gadgets, etc).
 * Inspired by darkgrue/SKU_RM0004.
 */
char *get_ip_address(void) {
    FILE *fp;
    char line[256], iface[64], dest[16];
    char *default_iface = NULL;
    int fd;
    struct ifreq ifr;

    /* Find the interface that carries the default route */
    fp = fopen("/proc/net/route", "r");
    if (fp) {
        /* Skip header line */
        if (fgets(line, sizeof(line), fp)) {
            while (fgets(line, sizeof(line), fp)) {
                if (sscanf(line, "%63s %15s", iface, dest) == 2) {
                    if (strcmp(dest, "00000000") == 0) {
                        default_iface = iface;
                        break;
                    }
                }
            }
        }
        fclose(fp);
    }

    if (!default_iface) return "no network";

    fd = socket(AF_INET, SOCK_DGRAM, 0);
    if (fd < 0) return "no network";

    ifr.ifr_addr.sa_family = AF_INET;
    strncpy(ifr.ifr_name, default_iface, IFNAMSIZ - 1);
    ifr.ifr_name[IFNAMSIZ - 1] = '\0';

    if (ioctl(fd, SIOCGIFADDR, &ifr) != 0) {
        close(fd);
        return "no network";
    }
    close(fd);

    return inet_ntoa(((struct sockaddr_in *)&ifr.ifr_addr)->sin_addr);
}

/*
 * Get RAM usage as a percentage (0-100)
 */
uint8_t get_ram_percent(void) {
    unsigned int value = 0;
    unsigned int total = 0, avail = 0;
    char buffer[128], label[32];

    FILE *fp = fopen("/proc/meminfo", "r");
    if (!fp) return 0;

    while (fgets(buffer, sizeof(buffer), fp)) {
        if (sscanf(buffer, "%31s %u", label, &value) != 2) continue;
        if (strcmp(label, "MemTotal:") == 0)
            total = value;
        else if (strcmp(label, "MemAvailable:") == 0)
            avail = value;
    }
    fclose(fp);

    if (total == 0) return 0;
    return (uint8_t)((uint64_t)(total - avail) * 100 / total);
}

/*
 * Get SD card usage in GiB
 */
static void get_sd_memory(uint32_t *total_gib, uint32_t *used_gib) {
    struct statfs info;
    if (statfs("/", &info) != 0) {
        *total_gib = 0;
        *used_gib = 0;
        return;
    }
    unsigned long long block = info.f_bsize;
    unsigned long long total = block * info.f_blocks;
    unsigned long long used = total - block * info.f_bfree;
    *total_gib = (uint32_t)(total >> 30);
    *used_gib = (uint32_t)(used >> 30);
}

/*
 * Get hard disk usage in GiB via /proc/mounts + statfs
 */
static void get_hard_disk_memory(uint32_t *total_gib, uint32_t *used_gib) {
    *total_gib = 0;
    *used_gib = 0;
    char line[512], device[256], mountpoint[256];
    struct statfs info;

    FILE *fp = fopen("/proc/mounts", "r");
    if (!fp) return;

    while (fgets(line, sizeof(line), fp)) {
        if (sscanf(line, "%255s %255s", device, mountpoint) == 2) {
            if (has_prefix(device, "/dev/sda") || has_prefix(device, "/dev/nvme")) {
                if (statfs(mountpoint, &info) == 0) {
                    unsigned long long block = info.f_bsize;
                    unsigned long long total = block * info.f_blocks;
                    unsigned long long used = total - (block * info.f_bfree);
                    *total_gib += (uint32_t)(total >> 30);
                    *used_gib += (uint32_t)(used >> 30);
                }
            }
        }
    }
    fclose(fp);
}

/*
 * Get combined disk usage (SD + hard disk) as a percentage (0-100)
 */
uint8_t get_disk_percent(void) {
    uint32_t sdTotal = 0, sdUsed = 0;
    uint32_t diskTotal = 0, diskUsed = 0;

    get_sd_memory(&sdTotal, &sdUsed);
    get_hard_disk_memory(&diskTotal, &diskUsed);

    uint32_t total = sdTotal + diskTotal;
    uint32_t used = sdUsed + diskUsed;

    if (total == 0) return 0;
    uint32_t pct = used * 100 / total;
    return (uint8_t)(pct > 100 ? 100 : pct);
}

/*
 * Get CPU temperature in configured units (C or F)
 */
uint8_t get_temperature(void) {
    unsigned int millideg;
    char buf[10];

    FILE *fp = fopen("/sys/class/thermal/thermal_zone0/temp", "r");
    if (!fp) return 0;
    if (!fgets(buf, sizeof(buf), fp)) {
        fclose(fp);
        return 0;
    }
    fclose(fp);
    if (sscanf(buf, "%u", &millideg) != 1) return 0;

    unsigned int celsius = millideg / 1000;
    if (TEMPERATURE_TYPE == FAHRENHEIT) return (uint8_t)(celsius * 9 / 5 + 32);
    return (uint8_t)celsius;
}

/*
 * Read aggregate CPU idle and total ticks from /proc/stat
 */
static int read_cpu_stat(unsigned long long *idle, unsigned long long *total) {
    unsigned long long user, nice, system, idle_val, iowait, irq, softirq, steal;
    FILE *fp = fopen("/proc/stat", "r");
    if (!fp) return -1;
    if (fscanf(fp, "cpu %llu %llu %llu %llu %llu %llu %llu %llu", &user, &nice, &system, &idle_val, &iowait, &irq,
               &softirq, &steal) != 8) {
        fclose(fp);
        return -1;
    }
    fclose(fp);
    *idle = idle_val + iowait;
    *total = user + nice + system + idle_val + iowait + irq + softirq + steal;
    return 0;
}

/*
 * Get CPU usage as a percentage (0-100) via /proc/stat delta
 */
uint8_t get_cpu_percent(void) {
    static unsigned long long prev_idle = 0, prev_total = 0;
    static int initialized = 0;
    unsigned long long idle, total;

    if (!initialized) {
        if (read_cpu_stat(&prev_idle, &prev_total) != 0) return 0;
        usleep(100000);
        initialized = 1;
    }

    if (read_cpu_stat(&idle, &total) != 0) return 0;

    unsigned long long diff_idle = idle - prev_idle;
    unsigned long long diff_total = total - prev_total;
    prev_idle = idle;
    prev_total = total;

    if (diff_total == 0) return 0;
    return (uint8_t)((100 * (diff_total - diff_idle) + diff_total / 2) / diff_total);
}

/*
 * Get hostname
 */
char *get_hostname(void) {
    static char hostname[65]; /* HOST_NAME_MAX is typically 64 */
    if (gethostname(hostname, sizeof(hostname)) != 0) {
        strncpy(hostname, "unknown", sizeof(hostname));
    }
    hostname[sizeof(hostname) - 1] = '\0';
    return hostname;
}

/*
 * Get DietPi core update status
 * Returns: 0 = not DietPi, 1 = up to date, 2 = update available
 */
int get_dietpi_update_status(void) {
    if (access("/run/dietpi", F_OK) != 0) return 0;
    if (access("/run/dietpi/.update_available", F_OK) == 0) return 2;
    return 1;
}

/*
 * Get APT upgradable package count from DietPi cache
 * Returns: -1 if file missing, otherwise the count
 */
int get_apt_update_count(void) {
    int count = 0;
    FILE *fp = fopen("/run/dietpi/.apt_updates", "r");
    if (!fp) return -1;
    if (fscanf(fp, "%d", &count) != 1) count = 0;
    fclose(fp);
    return count;
}