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
#include <sys/ioctl.h>
#include <linux/i2c.h>
#include <linux/i2c-dev.h>
#include <fcntl.h>
#include "st7735.h"
#include <stdlib.h>

/*
* Get the IP address — tries eth0 first, falls back to wlan0
*/
char* get_ip_address(void)
{
    int fd;
    struct ifreq ifr;
    int symbol=0;

    fd = socket(AF_INET, SOCK_DGRAM, 0);
    /* I want to get an IPv4 IP address */
    ifr.ifr_addr.sa_family = AF_INET;
    /* I want IP address attached to "eth0" */
    strncpy(ifr.ifr_name, "eth0", IFNAMSIZ-1);
    symbol=ioctl(fd, SIOCGIFADDR, &ifr);
    close(fd);
    if(symbol==0)
    {
      return inet_ntoa(((struct sockaddr_in *)&ifr.ifr_addr)->sin_addr);
    }
    else
    {
      fd = socket(AF_INET, SOCK_DGRAM, 0);
      /* I want to get an IPv4 IP address */
      ifr.ifr_addr.sa_family = AF_INET;
      /* I want IP address attached to "wlan0" */
      strncpy(ifr.ifr_name, "wlan0", IFNAMSIZ-1);
      symbol=ioctl(fd, SIOCGIFADDR, &ifr);
      close(fd);    
      if(symbol==0)
      {
        return inet_ntoa(((struct sockaddr_in *)&ifr.ifr_addr)->sin_addr);   
      }
      else
      {
        char* buffer="xxx.xxx.xxx.xxx";
        return buffer;
      }
    }
}



/*
* get ram memory
*/
void get_cpu_memory(float *Totalram,float *availram)
{
  struct sysinfo s_info;

  unsigned int value=0;
  char buffer[100]={0};
  char famer[100]={0};
    if(sysinfo(&s_info)==0)            //Get memory information
    {
        FILE* fp=fopen("/proc/meminfo","r");
        if(fp==NULL)
        {
            return ;
        }
        while(fgets(buffer,sizeof(buffer),fp))
        {
            if(sscanf(buffer,"%s%u",famer,&value)!=2)
            {
            continue;
            }
            if(strcmp(famer,"MemTotal:")==0)
            {
             *Totalram=value/1000.0/1000.0;
            }
            else if(strcmp(famer,"MemAvailable:")==0)
            {
              *availram=value/1000.0/1000.0;
            }
        }
        fclose(fp);    
    }   
}

/*
* get sd memory
*/
void get_sd_memory(uint32_t *MemSize, uint32_t *freesize)
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
uint8_t get_hard_disk_memory(uint16_t *diskMemSize, uint16_t *useMemSize)
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
* get temperature
*/

uint8_t get_temperature(void)
{
    FILE *fd;
    unsigned int temp;
    char buff[10] = {0};
    fd = fopen("/sys/class/thermal/thermal_zone0/temp","r");
    if (!fd) return 0;
    fgets(buff,sizeof(buff),fd);
    sscanf(buff, "%d", &temp);
    fclose(fd);
    return TEMPERATURE_TYPE == FAHRENHEIT ? temp/1000*1.8+32 : temp/1000;    
}

/*
* Get cpu usage via /proc/stat delta
*/
uint8_t get_cpu_message(void)
{
    static unsigned long long prev_idle = 0, prev_total = 0;
    static int initialized = 0;
    unsigned long long user, nice, system, idle_val, iowait, irq, softirq, steal;
    unsigned long long idle_sum, total, diff_idle, diff_total;
    FILE *fp;

    if (!initialized) {
        fp = fopen("/proc/stat", "r");
        if (!fp) return 0;
        fscanf(fp, "cpu %llu %llu %llu %llu %llu %llu %llu %llu",
               &user, &nice, &system, &idle_val, &iowait, &irq, &softirq, &steal);
        fclose(fp);
        prev_idle = idle_val + iowait;
        prev_total = user + nice + system + idle_val + iowait + irq + softirq + steal;
        usleep(100000);
        initialized = 1;
    }

    fp = fopen("/proc/stat", "r");
    if (!fp) return 0;
    fscanf(fp, "cpu %llu %llu %llu %llu %llu %llu %llu %llu",
           &user, &nice, &system, &idle_val, &iowait, &irq, &softirq, &steal);
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
    fscanf(fp, "%d", &count);
    fclose(fp);
    return count;
}