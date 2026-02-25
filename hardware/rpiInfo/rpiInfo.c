#include "rpiInfo.h"
#include <stdio.h>
#include <string.h>
#include <sys/sysinfo.h>
#include <sys/vfs.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/ioctl.h>
#include <netinet/in.h>
#include <net/if.h>
#include <unistd.h>
#include <arpa/inet.h>

/*
* Get the IP address of the default-route interface.
* Auto-detects the interface via /proc/net/route instead of
* hardcoding eth0/wlan0, so it works on any Linux system
* (Armbian end0, USB gadgets, etc).
* Inspired by darkgrue/SKU_RM0004.
*/
char* get_ip_address(void)
{
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

    if (!default_iface)
        return "no route";

    fd = socket(AF_INET, SOCK_DGRAM, 0);
    if (fd < 0)
        return "no socket";

    ifr.ifr_addr.sa_family = AF_INET;
    strncpy(ifr.ifr_name, default_iface, IFNAMSIZ - 1);
    ifr.ifr_name[IFNAMSIZ - 1] = '\0';

    if (ioctl(fd, SIOCGIFADDR, &ifr) != 0) {
        close(fd);
        return "no addr";
    }
    close(fd);

    return inet_ntoa(((struct sockaddr_in *)&ifr.ifr_addr)->sin_addr);
}



/*
* Get RAM usage as a percentage (0-100)
*/
uint8_t get_ram_percent(void)
{
    struct sysinfo s_info;
    unsigned int value = 0;
    char buffer[100] = {0};
    char label[100] = {0};
    float total = 0.0, avail = 0.0;

    if (sysinfo(&s_info) != 0)
        return 0;

    FILE *fp = fopen("/proc/meminfo", "r");
    if (!fp)
        return 0;

    while (fgets(buffer, sizeof(buffer), fp)) {
        if (sscanf(buffer, "%s%u", label, &value) != 2)
            continue;
        if (strcmp(label, "MemTotal:") == 0)
            total = value / 1000.0 / 1000.0;
        else if (strcmp(label, "MemAvailable:") == 0)
            avail = value / 1000.0 / 1000.0;
    }
    fclose(fp);

    if (total <= 0)
        return 0;
    return (uint8_t)((total - avail) / total * 100);
}

/*
* get sd memory
*/
static void get_sd_memory(uint32_t *MemSize, uint32_t *freesize)
{
    struct statfs diskInfo;
    statfs("/",&diskInfo);
    unsigned long long blocksize = diskInfo.f_bsize;// The number of bytes per block
    unsigned long long totalsize = blocksize*diskInfo.f_blocks;//Total number of bytes	
    *MemSize=(unsigned int)(totalsize>>30);


    unsigned long long size = blocksize*diskInfo.f_bfree; //Now let's figure out how much space we have left
    *freesize=size>>30;
    *freesize=*MemSize-*freesize;
}


/*
* get hard disk memory via /proc/mounts + statfs
*/
static uint8_t get_hard_disk_memory(uint16_t *diskMemSize, uint16_t *useMemSize)
{
  *diskMemSize = 0;
  *useMemSize = 0;
  char line[512], device[256], mountpoint[256];
  struct statfs disk_info;

  FILE *fp = fopen("/proc/mounts", "r");
  if (!fp) return 1;

  while (fgets(line, sizeof(line), fp)) {
      if (sscanf(line, "%255s %255s", device, mountpoint) == 2) {
          if (strncmp(device, "/dev/sda", 8) == 0 ||
              strncmp(device, "/dev/nvme", 9) == 0) {
              if (statfs(mountpoint, &disk_info) == 0) {
                  unsigned long long block = disk_info.f_bsize;
                  unsigned long long total = block * disk_info.f_blocks;
                  unsigned long long used = total - (block * disk_info.f_bfree);
                  *diskMemSize += (uint16_t)(total >> 30);
                  *useMemSize += (uint16_t)(used >> 30);
              }
          }
      }
  }
  fclose(fp);
  return 0;
}

/*
* Get combined disk usage (SD + hard disk) as a percentage (0-100)
*/
uint8_t get_disk_percent(void)
{
    uint32_t sdMemSize = 0, sdUseMemSize = 0;
    uint16_t diskMemSize = 0, diskUseMemSize = 0;

    get_sd_memory(&sdMemSize, &sdUseMemSize);
    get_hard_disk_memory(&diskMemSize, &diskUseMemSize);

    uint32_t total = sdMemSize + diskMemSize;
    uint32_t used = sdUseMemSize + diskUseMemSize;

    if (total == 0)
        return 0;
    uint32_t pct = used * 100 / total;
    return (uint8_t)(pct > 100 ? 100 : pct);
}

/*
* get temperature
*/

uint8_t get_temperature(void)
{
    FILE *fp;
    unsigned int temp;
    char buff[10] = {0};
    fp = fopen("/sys/class/thermal/thermal_zone0/temp", "r");
    if (!fp) return 0;
    if (!fgets(buff, sizeof(buff), fp)) { fclose(fp); return 0; }
    fclose(fp);
    if (sscanf(buff, "%u", &temp) != 1) return 0;
    return TEMPERATURE_TYPE == FAHRENHEIT ? temp/1000*1.8+32 : temp/1000;
}

/*
* Get CPU usage as a percentage (0-100) via /proc/stat delta
*/
uint8_t get_cpu_percent(void)
{
    static unsigned long long prev_idle = 0, prev_total = 0;
    static int initialized = 0;
    unsigned long long user, nice, system, idle_val, iowait, irq, softirq, steal;
    unsigned long long idle_sum, total, diff_idle, diff_total;
    FILE *fp;

    if (!initialized) {
        fp = fopen("/proc/stat", "r");
        if (!fp) return 0;
        if (fscanf(fp, "cpu %llu %llu %llu %llu %llu %llu %llu %llu",
               &user, &nice, &system, &idle_val, &iowait, &irq, &softirq, &steal) != 8)
        { fclose(fp); return 0; }
        fclose(fp);
        prev_idle = idle_val + iowait;
        prev_total = user + nice + system + idle_val + iowait + irq + softirq + steal;
        usleep(100000);
        initialized = 1;
    }

    fp = fopen("/proc/stat", "r");
    if (!fp) return 0;
    if (fscanf(fp, "cpu %llu %llu %llu %llu %llu %llu %llu %llu",
           &user, &nice, &system, &idle_val, &iowait, &irq, &softirq, &steal) != 8)
    { fclose(fp); return 0; }
    fclose(fp);

    idle_sum = idle_val + iowait;
    total = user + nice + system + idle_val + iowait + irq + softirq + steal;
    diff_idle = idle_sum - prev_idle;
    diff_total = total - prev_total;
    prev_idle = idle_sum;
    prev_total = total;

    if (diff_total == 0) return 0;
    return (uint8_t)(100 * (diff_total - diff_idle) / diff_total);
}

/*
* Get hostname
*/
char* get_hostname(void)
{
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
int get_dietpi_update_status(void)
{
    if (access("/run/dietpi", F_OK) != 0)
        return 0;
    if (access("/run/dietpi/.update_available", F_OK) == 0)
        return 2;
    return 1;
}

/*
* Get APT upgradable package count from DietPi cache
* Returns: -1 if file missing, otherwise the count
*/
int get_apt_update_count(void)
{
    int count = 0;
    FILE *fp = fopen("/run/dietpi/.apt_updates", "r");
    if (!fp) return -1;
    if (fscanf(fp, "%d", &count) != 1) count = 0;
    fclose(fp);
    return count;
}