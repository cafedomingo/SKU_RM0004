#include "rpiInfo.h"

static char mock_hostname[] = "raspberrypi";
static char mock_ip[] = "192.168.1.42";
static char mock_ip6[] = "::a8f3";

char *get_hostname(void) { return mock_hostname; }
char *get_ip_address(void) { return mock_ip; }
char *get_ip6_suffix(void) { return mock_ip6; }

uint8_t get_cpu_percent(void) { return 47; }
uint8_t get_ram_percent(void) { return 63; }
uint8_t get_temperature(void) { return 52; }
uint8_t get_disk_percent(void) { return 34; }
uint32_t get_uptime_secs(void) { return 3 * 86400 + 12 * 3600; }

cpu_freq_t get_cpu_freq(void) { return (cpu_freq_t){.cur_mhz = 1800, .min_mhz = 600, .max_mhz = 1800}; }

uint32_t get_cpu_throttle_status(void) { return 0; }

net_bandwidth_t get_net_bandwidth(void) {
    return (net_bandwidth_t){.rx_bytes_per_sec = 251904, .tx_bytes_per_sec = 12288};
}

disk_io_t get_disk_io(void) {
    return (disk_io_t){
        .read_bytes_per_sec = 1258291,
        .write_bytes_per_sec = 262144,
        .read_iops = 142,
        .write_iops = 38,
    };
}

int get_dietpi_update_status(void) { return DIETPI_UPDATE_AVAIL; }
int get_apt_update_count(void) { return 5; }
