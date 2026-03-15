# UCTRONICS LCD Display Driver

Display driver for the [UCTRONICS SKU_RM0004](https://github.com/UCTRONICS/SKU_RM0004) 160x80 ST7735 TFT LCD on Raspberry Pi 4/5. Forked from the original and simplified to focus on the all-in-one status view.

![Display preview](display.svg)

- Hostname and auto-detected IP address
- CPU, RAM, temperature, and disk usage with color-coded bars
- DietPi update indicator (◆) and APT upgrade count (^N)
- Refreshes every 5 seconds

## Install / Update

```bash
curl -sL https://github.com/cafedomingo/SKU_RM0004/releases/latest/download/install.sh | sudo bash
```

The script is idempotent — it handles both first install and updates. On first run it configures I2C, GPIO, and installs a systemd service. On subsequent runs it downloads the latest binary and restarts the service.

To update multiple Pis:

```bash
for host in pi1 pi2 pi3 pi4; do
  ssh "$host" 'curl -sL https://github.com/cafedomingo/SKU_RM0004/releases/latest/download/install.sh | sudo bash'
done
```

## Development install

To run a locally built binary instead of the release:

```bash
git clone https://github.com/cafedomingo/SKU_RM0004.git
cd SKU_RM0004
make
sudo ./install.sh
```

When a `./display` binary exists in the current directory, `install.sh` uses it instead of downloading from GitHub.

## Uninstall

```bash
sudo systemctl disable uctronics-display.service
sudo rm /etc/systemd/system/uctronics-display.service
sudo rm -rf /opt/uctronics-lcd
sudo systemctl daemon-reload
```

## Configuration

Runtime settings are in `/etc/uctronics-display.conf` (created on first install):

```ini
# Screen to display: "dashboard" or "diagnostic"
screen=dashboard

# Refresh interval in seconds (1-30)
refresh=5
```

Changes take effect on the next refresh cycle — no restart required.

The diagnostic screen shows detailed system metrics across two pages that alternate each refresh cycle: system overview (hostname, IPs, CPU, temperature, RAM, throttle) and I/O (disk, network, IOPS, update status).

Compile-time settings are in `hardware/rpiInfo/rpiInfo.h`. Rebuild after changing.

```c
#define TEMPERATURE_TYPE  CELSIUS    // or FAHRENHEIT
#define REFRESH_INTERVAL_SECS  5     // default refresh interval
```
