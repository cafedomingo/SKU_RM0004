//! Raspberry Pi system metrics collection.

use std::fs;
use std::path::Path;

/// Temperature unit configuration.
pub const CELSIUS: u8 = 0;
pub const FAHRENHEIT: u8 = 1;
pub const TEMPERATURE_TYPE: u8 = CELSIUS;

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

    let mut default_iface: Option<String> = None;
    for (i, line) in route_contents.lines().enumerate() {
        if i == 0 {
            continue; // skip header
        }
        let fields: Vec<&str> = line.split_whitespace().collect();
        if fields.len() >= 2 && fields[1] == "00000000" {
            default_iface = Some(fields[0].to_string());
            break;
        }
    }

    let iface = match default_iface {
        Some(i) => i,
        None => return "no network".to_string(),
    };

    // Use ioctl SIOCGIFADDR to get the interface IP address
    let fd = unsafe { libc::socket(libc::AF_INET, libc::SOCK_DGRAM, 0) };
    if fd < 0 {
        return "no network".to_string();
    }

    let mut ifr: libc::ifreq = unsafe { std::mem::zeroed() };
    let iface_bytes = iface.as_bytes();
    let copy_len = iface_bytes.len().min(libc::IFNAMSIZ - 1);
    unsafe {
        std::ptr::copy_nonoverlapping(iface_bytes.as_ptr(), ifr.ifr_name.as_mut_ptr() as *mut u8, copy_len);
    }
    ifr.ifr_ifru = unsafe { std::mem::zeroed() };
    // Set sa_family to AF_INET
    unsafe {
        let addr_ptr = &mut ifr.ifr_ifru as *mut _ as *mut libc::sockaddr_in;
        (*addr_ptr).sin_family = libc::AF_INET as libc::sa_family_t;
    }

    let ret = unsafe { libc::ioctl(fd, libc::SIOCGIFADDR, &mut ifr) };
    unsafe {
        libc::close(fd);
    }

    if ret != 0 {
        return "no network".to_string();
    }

    unsafe {
        let addr_ptr = &ifr.ifr_ifru as *const _ as *const libc::sockaddr_in;
        let ip_addr = (*addr_ptr).sin_addr;
        let raw = u32::from_be(ip_addr.s_addr);
        format!(
            "{}.{}.{}.{}",
            (raw >> 24) & 0xFF,
            (raw >> 16) & 0xFF,
            (raw >> 8) & 0xFF,
            raw & 0xFF
        )
    }
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
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() >= 2 {
            if parts[0] == "MemTotal:" {
                total = parts[1].parse().unwrap_or(0);
            } else if parts[0] == "MemAvailable:" {
                avail = parts[1].parse().unwrap_or(0);
            }
        }
    }

    if total == 0 {
        return 0;
    }
    ((total - avail) * 100 / total) as u8
}

fn get_sd_memory() -> (u32, u32) {
    let mut info: libc::statfs = unsafe { std::mem::zeroed() };
    let path = std::ffi::CString::new("/").unwrap();
    let ret = unsafe { libc::statfs(path.as_ptr(), &mut info) };
    if ret != 0 {
        eprintln!("rpiInfo: statfs(\"/\") failed");
        return (0, 0);
    }
    let block = info.f_bsize as u64;
    let total = block * info.f_blocks;
    let used = total - block * info.f_bfree;
    ((total >> 30) as u32, (used >> 30) as u32)
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
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() >= 2 {
            let device = parts[0];
            let mountpoint = parts[1];
            if device.starts_with("/dev/sda") || device.starts_with("/dev/nvme") {
                let mp = std::ffi::CString::new(mountpoint).unwrap();
                let mut info: libc::statfs = unsafe { std::mem::zeroed() };
                let ret = unsafe { libc::statfs(mp.as_ptr(), &mut info) };
                if ret == 0 {
                    let block = info.f_bsize as u64;
                    let total = block * info.f_blocks;
                    let used = total - block * info.f_bfree;
                    total_gib += (total >> 30) as u32;
                    used_gib += (used >> 30) as u32;
                }
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
    let pct = used * 100 / total;
    if pct > 100 {
        100
    } else {
        pct as u8
    }
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
    if TEMPERATURE_TYPE == FAHRENHEIT {
        (celsius * 9 / 5 + 32) as u8
    } else {
        celsius as u8
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
    let parts: Vec<&str> = first_line.split_whitespace().collect();
    if parts.len() < 9 || parts[0] != "cpu" {
        return None;
    }

    let user: u64 = parts[1].parse().ok()?;
    let nice: u64 = parts[2].parse().ok()?;
    let system: u64 = parts[3].parse().ok()?;
    let idle_val: u64 = parts[4].parse().ok()?;
    let iowait: u64 = parts[5].parse().ok()?;
    let irq: u64 = parts[6].parse().ok()?;
    let softirq: u64 = parts[7].parse().ok()?;
    let steal: u64 = parts[8].parse().ok()?;

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
    let mut buf = [0u8; 65];
    let ret = unsafe { libc::gethostname(buf.as_mut_ptr() as *mut libc::c_char, buf.len()) };
    if ret != 0 {
        return "unknown".to_string();
    }
    // Find null terminator
    let len = buf.iter().position(|&b| b == 0).unwrap_or(buf.len());
    String::from_utf8_lossy(&buf[..len]).into_owned()
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
