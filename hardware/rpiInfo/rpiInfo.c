#include "rpiInfo.h"
#include "log.h"
#include <arpa/inet.h>
#include <fcntl.h>
#include <ifaddrs.h>
#include <net/if.h>
#include <netinet/in.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <sys/sysinfo.h>
#include <sys/types.h>
#include <sys/vfs.h>
#include <time.h>
#include <unistd.h>

/* ── Helpers ─────────────────────────────────────────────────────── */

static inline int has_prefix(const char *s, const char *prefix) { return strncmp(s, prefix, strlen(prefix)) == 0; }

/*
 * Find the default-route network interface name
 */
static int get_default_iface(char *buf, size_t buflen) {
    FILE *fp;
    char line[256], iface[64], dest[16];

    fp = fopen("/proc/net/route", "r");
    if (!fp) {
        LOG_ERROR("failed to open /proc/net/route");
        return -1;
    }
    /* Skip header line */
    if (fgets(line, sizeof(line), fp)) {
        while (fgets(line, sizeof(line), fp)) {
            if (sscanf(line, "%63s %15s", iface, dest) == 2) {
                if (strcmp(dest, "00000000") == 0) {
                    strncpy(buf, iface, buflen - 1);
                    buf[buflen - 1] = '\0';
                    fclose(fp);
                    return 0;
                }
            }
        }
    }
    fclose(fp);
    return -1;
}

/*
 * Read aggregate CPU idle and total ticks from /proc/stat
 */
static int read_cpu_stat(unsigned long long *idle, unsigned long long *total) {
    unsigned long long user, nice, system, idle_val, iowait, irq, softirq, steal;
    FILE *fp = fopen("/proc/stat", "r");
    if (!fp) {
        LOG_ERROR("failed to open /proc/stat");
        return -1;
    }
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
 * Check if a /proc/diskstats device name is a whole disk (not a partition).
 * Matches: sda, mmcblk0, nvme0n1.  Rejects: sda1, mmcblk0p1, nvme0n1p1.
 *
 * sd partitions append a digit (sda1); mmcblk/nvme partitions append 'p'
 * followed by a digit (mmcblk0p1, nvme0n1p1).
 */
static int is_whole_disk(const char *name) {
    size_t len = strlen(name);

    /* sd[a-z] — exactly 3 chars, partition would be sda1 (4+) */
    if (has_prefix(name, "sd") && len == 3) return 1;

    /* mmcblk[0-9]... — partition adds 'p', so reject if 'p' present */
    if (has_prefix(name, "mmcblk")) return strchr(name + 6, 'p') == NULL;

    /* nvme[0-9]+n[0-9]+ — partition adds 'p', same check */
    if (has_prefix(name, "nvme")) return strchr(name + 4, 'p') == NULL;

    return 0;
}

static int read_sysfs_ulong(const char *path, unsigned long *out) {
    FILE *fp = fopen(path, "r");
    if (!fp) return -1;
    if (fscanf(fp, "%lu", out) != 1) {
        fclose(fp);
        return -1;
    }
    fclose(fp);
    return 0;
}

static int read_net_counter(const char *iface, const char *counter, unsigned long long *out) {
    char path[128];
    snprintf(path, sizeof(path), "/sys/class/net/%s/statistics/%s", iface, counter);
    FILE *fp = fopen(path, "r");
    if (!fp) return -1;
    if (fscanf(fp, "%llu", out) != 1) {
        fclose(fp);
        return -1;
    }
    fclose(fp);
    return 0;
}

/*
 * Get elapsed seconds since prev_time and update it
 */
static double get_elapsed_secs(struct timespec *prev_time) {
    struct timespec now;
    clock_gettime(CLOCK_MONOTONIC, &now);

    if (prev_time->tv_sec == 0 && prev_time->tv_nsec == 0) {
        *prev_time = now;
        return 0.0;
    }

    double elapsed = (now.tv_sec - prev_time->tv_sec) + (now.tv_nsec - prev_time->tv_nsec) / 1e9;
    *prev_time = now;
    return elapsed;
}

/* ── Network ─────────────────────────────────────────────────────── */

/*
 * Get the IP address of the default-route interface.
 * Auto-detects the interface via /proc/net/route instead of
 * hardcoding eth0/wlan0, so it works on any Linux system
 * (Armbian end0, USB gadgets, etc).
 * Inspired by darkgrue/SKU_RM0004.
 */
char *get_ip_address(void) {
    static char ip_buf[INET_ADDRSTRLEN];
    char iface[64];
    int fd;
    struct ifreq ifr;

    if (get_default_iface(iface, sizeof(iface)) != 0) return "no network";

    fd = socket(AF_INET, SOCK_DGRAM, 0);
    if (fd < 0) return "no network";

    ifr.ifr_addr.sa_family = AF_INET;
    strncpy(ifr.ifr_name, iface, IFNAMSIZ - 1);
    ifr.ifr_name[IFNAMSIZ - 1] = '\0';

    if (ioctl(fd, SIOCGIFADDR, &ifr) != 0) {
        close(fd);
        return "no network";
    }
    close(fd);

    const char *ip = inet_ntoa(((struct sockaddr_in *)&ifr.ifr_addr)->sin_addr);
    snprintf(ip_buf, sizeof(ip_buf), "%s", ip);
    return ip_buf;
}

/*
 * Get the host suffix of the first global-scope IPv6 address on the
 * default-route interface.  Uses the netmask to determine prefix length
 * and returns only the host portion, prefixed with "::".
 * Returns "no IPv6" on failure.
 */
char *get_ip6_suffix(void) {
    static char buf[INET6_ADDRSTRLEN + 3];
    char iface[64];
    if (get_default_iface(iface, sizeof(iface)) != 0) return "no IPv6";

    struct ifaddrs *ifa_list, *ifa;
    if (getifaddrs(&ifa_list) != 0) return "no IPv6";

    char *result = "no IPv6";
    for (ifa = ifa_list; ifa; ifa = ifa->ifa_next) {
        if (!ifa->ifa_addr || ifa->ifa_addr->sa_family != AF_INET6) continue;
        if (strcmp(ifa->ifa_name, iface) != 0) continue;

        struct sockaddr_in6 *sa6 = (struct sockaddr_in6 *)ifa->ifa_addr;
        if (IN6_IS_ADDR_LINKLOCAL(&sa6->sin6_addr)) continue;

        /* Count prefix length from netmask */
        if (!ifa->ifa_netmask) continue;
        struct sockaddr_in6 *mask = (struct sockaddr_in6 *)ifa->ifa_netmask;
        uint8_t *m = mask->sin6_addr.s6_addr;
        int prefix = 0;
        for (int i = 0; i < 16; i++) {
            if (m[i] == 0xFF) {
                prefix += 8;
                continue;
            }
            for (int b = 7; b >= 0; b--) {
                if (m[i] & (1 << b))
                    prefix++;
                else
                    break;
            }
            break;
        }

        /* Format host groups (those after the prefix) */
        uint8_t *a = sa6->sin6_addr.s6_addr;
        int first_group = prefix / 16;
        char *p = buf;
        char *end = buf + sizeof(buf);
        p += snprintf(p, end - p, "::");
        for (int g = first_group; g < 8 && p < end; g++) {
            if (g > first_group) p += snprintf(p, end - p, ":");
            p += snprintf(p, end - p, "%x", (a[g * 2] << 8) | a[g * 2 + 1]);
        }
        result = buf;
        break;
    }

    freeifaddrs(ifa_list);
    return result;
}

/*
 * Get network bandwidth usage on the default-route interface.
 * Delta-based: first call returns zeros, subsequent calls return
 * bytes/sec. Resets if the active interface changes between calls
 * or if counters regress (driver/link reset).
 */
net_bandwidth_t get_net_bandwidth(void) {
    static unsigned long long prev_rx = 0, prev_tx = 0;
    static struct timespec prev_time = {0, 0};
    static char prev_iface[64] = "";
    net_bandwidth_t result = {0, 0};

    char iface[64];
    if (get_default_iface(iface, sizeof(iface)) != 0) return result;

    /* Reset state if the interface changed */
    if (strcmp(iface, prev_iface) != 0) {
        strncpy(prev_iface, iface, sizeof(prev_iface) - 1);
        prev_iface[sizeof(prev_iface) - 1] = '\0';
        prev_time = (struct timespec){0, 0};
    }

    unsigned long long rx, tx;
    if (read_net_counter(iface, "rx_bytes", &rx) != 0) return result;
    if (read_net_counter(iface, "tx_bytes", &tx) != 0) return result;

    double elapsed = get_elapsed_secs(&prev_time);

    if (elapsed <= 0.0 || rx < prev_rx || tx < prev_tx) {
        prev_rx = rx;
        prev_tx = tx;
        return result;
    }

    result.rx_bytes_per_sec = (uint64_t)((rx - prev_rx) / elapsed);
    result.tx_bytes_per_sec = (uint64_t)((tx - prev_tx) / elapsed);
    prev_rx = rx;
    prev_tx = tx;
    return result;
}

/* ── CPU ─────────────────────────────────────────────────────────── */

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
    uint8_t pct = (uint8_t)((100 * (diff_total - diff_idle) + diff_total / 2) / diff_total);
    return pct < 1 ? 1 : pct;
}

/*
 * Get CPU frequency (current, min, max) in MHz
 */
cpu_freq_t get_cpu_freq(void) {
    cpu_freq_t freq = {0, 0, 0};
    unsigned long khz;

    if (read_sysfs_ulong("/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq", &khz) == 0)
        freq.cur_mhz = (uint16_t)(khz / 1000);
    if (read_sysfs_ulong("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_min_freq", &khz) == 0)
        freq.min_mhz = (uint16_t)(khz / 1000);
    if (read_sysfs_ulong("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq", &khz) == 0)
        freq.max_mhz = (uint16_t)(khz / 1000);

    return freq;
}

/*
 * Get CPU throttle status as a bitmask (see THROTTLE_* in rpiInfo.h)
 */
uint32_t get_cpu_throttle_status(void) {
    int fd = open("/dev/vcio", O_RDWR);
    if (fd < 0) return 0;

    /* Mailbox property buffer for GET_THROTTLED (tag 0x00030046) */
    uint32_t buf[8] __attribute__((aligned(16))) = {sizeof(buf), /* buffer size */
                                                    0x00000000,  /* request code */
                                                    0x00030046,  /* tag: GET_THROTTLED */
                                                    4,           /* value buffer size */
                                                    0,           /* request/response indicator */
                                                    0,           /* value (filled by firmware) */
                                                    0,           /* end tag */
                                                    0};

    int ret = ioctl(fd, _IOWR(100, 0, char *), buf);
    close(fd);

    if (ret < 0 || buf[1] != 0x80000000) return 0;
    return buf[5];
}

/* ── Disk ────────────────────────────────────────────────────────── */

/*
 * Get SD card usage in MiB
 */
static void get_sd_memory(uint32_t *total_mib, uint32_t *used_mib) {
    struct statfs info;
    if (statfs("/", &info) != 0) {
        LOG_ERROR("statfs(\"/\") failed");
        *total_mib = 0;
        *used_mib = 0;
        return;
    }
    unsigned long long block = info.f_bsize;
    unsigned long long total = block * info.f_blocks;
    unsigned long long used = total - block * info.f_bfree;
    *total_mib = (uint32_t)(total >> 20);
    *used_mib = (uint32_t)(used >> 20);
}

/*
 * Get hard disk usage in MiB via /proc/mounts + statfs
 */
static void get_hard_disk_memory(uint32_t *total_mib, uint32_t *used_mib) {
    *total_mib = 0;
    *used_mib = 0;
    char line[512], device[256], mountpoint[256];
    struct statfs info;

    FILE *fp = fopen("/proc/mounts", "r");
    if (!fp) {
        LOG_ERROR("failed to open /proc/mounts");
        return;
    }

    while (fgets(line, sizeof(line), fp)) {
        if (sscanf(line, "%255s %255s", device, mountpoint) == 2) {
            if (has_prefix(device, "/dev/sda") || has_prefix(device, "/dev/nvme")) {
                if (statfs(mountpoint, &info) == 0) {
                    unsigned long long block = info.f_bsize;
                    unsigned long long total = block * info.f_blocks;
                    unsigned long long used = total - (block * info.f_bfree);
                    *total_mib += (uint32_t)(total >> 20);
                    *used_mib += (uint32_t)(used >> 20);
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
 * Get disk I/O throughput and IOPS aggregated across all disks.
 * Delta-based: first call returns zeros, subsequent calls return
 * rates per second. Resets on counter regression (device reset).
 */
disk_io_t get_disk_io(void) {
    static unsigned long long prev_sectors_read = 0, prev_sectors_written = 0;
    static unsigned long long prev_reads = 0, prev_writes = 0;
    static struct timespec prev_time = {0, 0};
    disk_io_t result = {0, 0, 0, 0};

    unsigned long long tot_reads = 0, tot_sectors_read = 0;
    unsigned long long tot_writes = 0, tot_sectors_written = 0;

    FILE *fp = fopen("/proc/diskstats", "r");
    if (!fp) {
        LOG_ERROR("failed to open /proc/diskstats");
        return result;
    }

    char line[256];
    while (fgets(line, sizeof(line), fp)) {
        unsigned int major, minor;
        char name[64];
        unsigned long long reads, reads_merged, sectors_r, ms_reading;
        unsigned long long writes, writes_merged, sectors_w;

        if (sscanf(line, " %u %u %63s %llu %llu %llu %llu %llu %llu %llu", &major, &minor, name, &reads, &reads_merged,
                   &sectors_r, &ms_reading, &writes, &writes_merged, &sectors_w) < 10)
            continue;

        if (!is_whole_disk(name)) continue;

        tot_reads += reads;
        tot_sectors_read += sectors_r;
        tot_writes += writes;
        tot_sectors_written += sectors_w;
    }
    fclose(fp);

    double elapsed = get_elapsed_secs(&prev_time);

    if (elapsed <= 0.0 || tot_sectors_read < prev_sectors_read || tot_sectors_written < prev_sectors_written ||
        tot_reads < prev_reads || tot_writes < prev_writes) {
        prev_sectors_read = tot_sectors_read;
        prev_sectors_written = tot_sectors_written;
        prev_reads = tot_reads;
        prev_writes = tot_writes;
        return result;
    }

    result.read_bytes_per_sec = (uint64_t)(((tot_sectors_read - prev_sectors_read) * 512) / elapsed);
    result.write_bytes_per_sec = (uint64_t)(((tot_sectors_written - prev_sectors_written) * 512) / elapsed);
    result.read_iops = (uint32_t)((tot_reads - prev_reads) / elapsed);
    result.write_iops = (uint32_t)((tot_writes - prev_writes) / elapsed);

    prev_sectors_read = tot_sectors_read;
    prev_sectors_written = tot_sectors_written;
    prev_reads = tot_reads;
    prev_writes = tot_writes;
    return result;
}

/* ── System ──────────────────────────────────────────────────────── */

/*
 * Get RAM usage as a percentage (0-100)
 */
uint8_t get_ram_percent(void) {
    unsigned int value = 0;
    unsigned int total = 0, avail = 0;
    char buffer[128], label[32];

    FILE *fp = fopen("/proc/meminfo", "r");
    if (!fp) {
        LOG_ERROR("failed to open /proc/meminfo");
        return 0;
    }

    while (fgets(buffer, sizeof(buffer), fp)) {
        if (sscanf(buffer, "%31s %u", label, &value) != 2) continue;
        if (strcmp(label, "MemTotal:") == 0)
            total = value;
        else if (strcmp(label, "MemAvailable:") == 0)
            avail = value;
    }
    fclose(fp);

    if (total == 0) return 0;
    uint8_t pct = (uint8_t)((uint64_t)(total - avail) * 100 / total);
    return pct < 1 ? 1 : pct;
}

/*
 * Get CPU temperature in Celsius.
 */
uint8_t get_temperature(void) {
    unsigned long millideg;
    if (read_sysfs_ulong("/sys/class/thermal/thermal_zone0/temp", &millideg) != 0) return 0;
    return (uint8_t)(millideg / 1000);
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
 * Get system uptime in seconds
 */
uint32_t get_uptime_secs(void) {
    struct sysinfo si;
    if (sysinfo(&si) != 0) return 0;
    return (uint32_t)si.uptime;
}

/* ── DietPi ──────────────────────────────────────────────────────── */

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
    if (!fp) {
        /* On DietPi, missing file means no updates available */
        if (access("/boot/dietpi/.version", F_OK) == 0) return 0;
        return -1;
    }
    if (fscanf(fp, "%d", &count) != 1) count = 0;
    fclose(fp);
    return count;
}
