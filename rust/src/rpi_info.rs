//! Raspberry Pi system metrics collection.

use std::fs;
use std::path::Path;

/// Temperature unit configuration.
#[derive(Clone, Copy, PartialEq)]
pub enum TemperatureType { Celsius, Fahrenheit }
pub const TEMPERATURE_TYPE: TemperatureType = TemperatureType::Celsius;

/// Dashboard refresh interval in seconds.
pub const REFRESH_INTERVAL_SECS: u64 = 5;

/// Get the IP address of the default network interface.
pub fn get_ip_address() -> String {
    let route_contents = match fs::read_to_string("/proc/net/route") {
        Ok(c) => c,
        Err(_) => {
            eprintln!("rpiInfo: failed to open /proc/net/route");
            return "no network".to_string();
        }
    };

    let iface_name = route_contents
        .lines()
        .skip(1)
        .find_map(|line| {
            let mut fields = line.split_whitespace();
            let name = fields.next()?;
            if fields.next() == Some("00000000") { Some(name) } else { None }
        });

    let iface_name = match iface_name {
        Some(name) => name,
        None => return "no network".to_string(),
    };

    // Look up the interface's IPv4 address using getifaddrs
    let addrs = match nix::ifaddrs::getifaddrs() {
        Ok(a) => a,
        Err(_) => return "no network".to_string(),
    };

    for ifaddr in addrs {
        if ifaddr.interface_name == iface_name {
            if let Some(sin) = ifaddr.address.as_ref().and_then(|a| a.as_sockaddr_in()) {
                return std::net::Ipv4Addr::from(sin.ip()).to_string();
            }
        }
    }

    "no network".to_string()
}

/// Get RAM usage as a percentage (0-100).
pub fn get_ram_percent() -> u8 {
    let contents = match fs::read_to_string("/proc/meminfo") {
        Ok(c) => c,
        Err(_) => {
            eprintln!("rpiInfo: failed to open /proc/meminfo");
            return 0;
        }
    };

    let mut total: u64 = 0;
    let mut avail: u64 = 0;

    for line in contents.lines() {
        let mut parts = line.split_whitespace();
        let label = parts.next();
        let value = parts.next();
        match (label, value) {
            (Some("MemTotal:"), Some(v)) => total = v.parse().unwrap_or(0),
            (Some("MemAvailable:"), Some(v)) => avail = v.parse().unwrap_or(0),
            _ => {}
        }
    }

    if total == 0 {
        return 0;
    }
    ((total - avail) * 100 / total) as u8
}

fn get_sd_memory() -> (u32, u32) {
    match nix::sys::statfs::statfs("/") {
        Ok(info) => {
            let block = info.block_size() as u64;
            let total = block * info.blocks() as u64;
            let used = total - block * info.blocks_free() as u64;
            ((total >> 30) as u32, (used >> 30) as u32)
        }
        Err(_) => {
            eprintln!("rpiInfo: statfs(\"/\") failed");
            (0, 0)
        }
    }
}

fn get_hard_disk_memory() -> (u32, u32) {
    let mut total_gib: u32 = 0;
    let mut used_gib: u32 = 0;

    let contents = match fs::read_to_string("/proc/mounts") {
        Ok(c) => c,
        Err(_) => {
            eprintln!("rpiInfo: failed to open /proc/mounts");
            return (0, 0);
        }
    };

    for line in contents.lines() {
        let mut parts = line.split_whitespace();
        let (device, mountpoint) = match (parts.next(), parts.next()) {
            (Some(d), Some(m)) => (d, m),
            _ => continue,
        };
        if device.starts_with("/dev/sda") || device.starts_with("/dev/nvme") {
            if let Ok(info) = nix::sys::statfs::statfs(mountpoint) {
                let block = info.block_size() as u64;
                let total = block * info.blocks() as u64;
                let used = total - block * info.blocks_free() as u64;
                total_gib += (total >> 30) as u32;
                used_gib += (used >> 30) as u32;
            }
        }
    }

    (total_gib, used_gib)
}

/// Get total disk usage (SD + hard disks) as a percentage (0-100).
pub fn get_disk_percent() -> u8 {
    let (sd_total, sd_used) = get_sd_memory();
    let (disk_total, disk_used) = get_hard_disk_memory();

    let total = sd_total + disk_total;
    let used = sd_used + disk_used;

    if total == 0 {
        return 0;
    }
    (used * 100 / total).min(100) as u8
}

/// Get CPU temperature in the configured unit (Celsius or Fahrenheit).
pub fn get_temperature() -> u8 {
    let contents = match fs::read_to_string("/sys/class/thermal/thermal_zone0/temp") {
        Ok(c) => c,
        Err(_) => {
            eprintln!("rpiInfo: failed to open thermal_zone0");
            return 0;
        }
    };

    let millideg: u32 = match contents.trim().parse() {
        Ok(v) => v,
        Err(_) => return 0,
    };

    let celsius = millideg / 1000;
    match TEMPERATURE_TYPE {
        TemperatureType::Fahrenheit => (celsius * 9 / 5 + 32) as u8,
        TemperatureType::Celsius => celsius as u8,
    }
}

fn read_cpu_stat() -> Option<(u64, u64)> {
    let contents = match fs::read_to_string("/proc/stat") {
        Ok(c) => c,
        Err(_) => {
            eprintln!("rpiInfo: failed to open /proc/stat");
            return None;
        }
    };

    let first_line = contents.lines().next()?;
    let mut parts = first_line.split_whitespace();
    if parts.next() != Some("cpu") {
        return None;
    }

    let mut next_val = || -> Option<u64> { parts.next()?.parse().ok() };
    let user = next_val()?;
    let nice = next_val()?;
    let system = next_val()?;
    let idle_val = next_val()?;
    let iowait = next_val()?;
    let irq = next_val()?;
    let softirq = next_val()?;
    let steal = next_val()?;

    let idle = idle_val + iowait;
    let total = user + nice + system + idle_val + iowait + irq + softirq + steal;
    Some((idle, total))
}

/// CPU usage tracker that maintains state between calls.
pub struct CpuTracker {
    prev_idle: u64,
    prev_total: u64,
    initialized: bool,
}

impl CpuTracker {
    pub fn new() -> Self {
        CpuTracker {
            prev_idle: 0,
            prev_total: 0,
            initialized: false,
        }
    }

    /// Get current CPU usage as a percentage (0-100).
    pub fn get_cpu_percent(&mut self) -> u8 {
        if !self.initialized {
            if let Some((idle, total)) = read_cpu_stat() {
                self.prev_idle = idle;
                self.prev_total = total;
            } else {
                return 0;
            }
            std::thread::sleep(std::time::Duration::from_millis(100));
            self.initialized = true;
        }

        let (idle, total) = match read_cpu_stat() {
            Some(v) => v,
            None => return 0,
        };

        let diff_idle = idle - self.prev_idle;
        let diff_total = total - self.prev_total;
        self.prev_idle = idle;
        self.prev_total = total;

        if diff_total == 0 {
            return 0;
        }
        ((100 * (diff_total - diff_idle) + diff_total / 2) / diff_total) as u8
    }
}

/// Get the system hostname.
pub fn get_hostname() -> String {
    match nix::sys::utsname::uname() {
        Ok(info) => info.nodename().to_string_lossy().into_owned(),
        Err(_) => "unknown".to_string(),
    }
}

/// Check DietPi update status.
/// Returns: 0 = not DietPi, 1 = up to date, 2 = update available.
pub fn get_dietpi_update_status() -> i32 {
    if !Path::new("/run/dietpi").exists() {
        return 0;
    }
    if Path::new("/run/dietpi/.update_available").exists() {
        return 2;
    }
    1
}

/// Get the number of pending APT updates.
/// Returns: -1 on error, 0+ = count.
pub fn get_apt_update_count() -> i32 {
    match fs::read_to_string("/run/dietpi/.apt_updates") {
        Ok(contents) => contents.trim().parse().unwrap_or(0),
        Err(_) => -1,
    }
}
