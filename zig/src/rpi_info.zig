const std = @import("std");
const posix = std.posix;
const linux = std.os.linux;

const log = std.log.scoped(.rpi_info);

pub const TemperatureType = enum { celsius, fahrenheit };
pub const temperature_type: TemperatureType = .celsius;
pub const refresh_interval_secs: u64 = 5;

fn hasPrefix(s: []const u8, prefix: []const u8) bool {
    return std.mem.startsWith(u8, s, prefix);
}

pub fn getIpAddress() []const u8 {
    const StaticBuf = struct {
        var iface_buf: [64]u8 = undefined;
        var ip_buf: [64]u8 = undefined;
    };

    const file = std.fs.openFileAbsolute("/proc/net/route", .{}) catch {
        log.err("failed to open /proc/net/route", .{});
        return "no network";
    };
    defer file.close();

    var buf: [4096]u8 = undefined;
    var reader = file.reader();

    // Skip header line
    _ = reader.readUntilDelimiter(&buf, '\n') catch return "no network";

    var default_iface: ?[]const u8 = null;
    while (reader.readUntilDelimiter(&buf, '\n') catch null) |line| {
        var iter = std.mem.tokenizeAny(u8, line, " \t");
        const iface = iter.next() orelse continue;
        const dest = iter.next() orelse continue;
        if (std.mem.eql(u8, dest, "00000000")) {
            const len = @min(iface.len, StaticBuf.iface_buf.len);
            @memcpy(StaticBuf.iface_buf[0..len], iface[0..len]);
            default_iface = StaticBuf.iface_buf[0..len];
            break;
        }
    }

    const iface_name = default_iface orelse return "no network";

    // Use ioctl to get the IP address via SIOCGIFADDR
    const sock = posix.socket(posix.AF.INET, posix.SOCK.DGRAM, 0) catch return "no network";
    defer posix.close(sock);

    var ifr: linux.ifreq = std.mem.zeroes(linux.ifreq);
    const name_len = @min(iface_name.len, ifr.ifrn.name.len - 1);
    @memcpy(ifr.ifrn.name[0..name_len], iface_name[0..name_len]);
    ifr.ifrn.name[name_len] = 0;

    const SIOCGIFADDR: u32 = 0x8915;
    const result = linux.ioctl(@bitCast(sock), SIOCGIFADDR, @intFromPtr(&ifr));
    if (@as(isize, @bitCast(result)) != 0) {
        return "no network";
    }

    // Extract the IPv4 address from sockaddr_in
    const addr_bytes = ifr.ifru.addr.data[2..6];
    const formatted = std.fmt.bufPrint(&StaticBuf.ip_buf, "{}.{}.{}.{}", .{
        addr_bytes[0], addr_bytes[1], addr_bytes[2], addr_bytes[3],
    }) catch return "no network";
    return formatted;
}

pub fn getRamPercent() u8 {
    const file = std.fs.openFileAbsolute("/proc/meminfo", .{}) catch {
        log.err("failed to open /proc/meminfo", .{});
        return 0;
    };
    defer file.close();

    var buf: [4096]u8 = undefined;
    var reader = file.reader();
    var total: u64 = 0;
    var avail: u64 = 0;

    while (reader.readUntilDelimiter(&buf, '\n') catch null) |line| {
        var iter = std.mem.tokenizeAny(u8, line, " \t");
        const label = iter.next() orelse continue;
        const value_str = iter.next() orelse continue;
        const value = std.fmt.parseInt(u64, value_str, 10) catch continue;

        if (std.mem.eql(u8, label, "MemTotal:")) {
            total = value;
        } else if (std.mem.eql(u8, label, "MemAvailable:")) {
            avail = value;
        }
    }

    if (total == 0) return 0;
    return @intCast((total - avail) * 100 / total);
}

const StatfsResult = struct {
    total_gib: u32,
    used_gib: u32,
};

// Linux statfs64 struct for aarch64
const Statfs = extern struct {
    f_type: i64,
    f_bsize: i64,
    f_blocks: u64,
    f_bfree: u64,
    f_bavail: u64,
    f_files: u64,
    f_ffree: u64,
    f_fsid: [2]i32,
    f_namelen: i64,
    f_frsize: i64,
    f_flags: i64,
    f_spare: [4]i64,
};

fn doStatfs(path: [*:0]const u8) ?StatfsResult {
    var info: Statfs = undefined;
    // Use raw syscall: SYS_statfs on aarch64 is 43
    const ret = linux.syscall2(.statfs, @intFromPtr(path), @intFromPtr(&info));
    if (@as(isize, @bitCast(ret)) != 0) return null;

    const block: u64 = @intCast(info.f_bsize);
    const total_bytes: u64 = block * info.f_blocks;
    const free_bytes: u64 = block * info.f_bfree;
    const used_bytes: u64 = total_bytes - free_bytes;
    return StatfsResult{
        .total_gib = @intCast(total_bytes >> 30),
        .used_gib = @intCast(used_bytes >> 30),
    };
}

fn getSdMemory() StatfsResult {
    return doStatfs("/") orelse blk: {
        log.err("statfs(\"/\") failed", .{});
        break :blk StatfsResult{ .total_gib = 0, .used_gib = 0 };
    };
}

fn getHardDiskMemory() StatfsResult {
    var result = StatfsResult{ .total_gib = 0, .used_gib = 0 };

    const file = std.fs.openFileAbsolute("/proc/mounts", .{}) catch {
        log.err("failed to open /proc/mounts", .{});
        return result;
    };
    defer file.close();

    var buf: [4096]u8 = undefined;
    var reader = file.reader();

    while (reader.readUntilDelimiter(&buf, '\n') catch null) |line| {
        var iter = std.mem.tokenizeAny(u8, line, " \t");
        const device = iter.next() orelse continue;
        const mountpoint = iter.next() orelse continue;

        if (hasPrefix(device, "/dev/sda") or hasPrefix(device, "/dev/nvme")) {
            // Need null-terminated string for statfs syscall
            var mp_buf: [256]u8 = undefined;
            if (mountpoint.len >= mp_buf.len) continue;
            @memcpy(mp_buf[0..mountpoint.len], mountpoint);
            mp_buf[mountpoint.len] = 0;
            const mp_z: [*:0]const u8 = @ptrCast(mp_buf[0..mountpoint.len :0]);

            if (doStatfs(mp_z)) |info| {
                result.total_gib += info.total_gib;
                result.used_gib += info.used_gib;
            }
        }
    }

    return result;
}

pub fn getDiskPercent() u8 {
    const sd = getSdMemory();
    const disk = getHardDiskMemory();

    const total = sd.total_gib + disk.total_gib;
    const used = sd.used_gib + disk.used_gib;

    if (total == 0) return 0;
    const pct = used * 100 / total;
    return @intCast(if (pct > 100) 100 else pct);
}

pub fn getTemperature() u8 {
    const file = std.fs.openFileAbsolute("/sys/class/thermal/thermal_zone0/temp", .{}) catch {
        log.err("failed to open thermal_zone0", .{});
        return 0;
    };
    defer file.close();

    var buf: [16]u8 = undefined;
    const bytes_read = file.read(&buf) catch return 0;
    if (bytes_read == 0) return 0;

    const trimmed = std.mem.trimRight(u8, buf[0..bytes_read], " \t\n\r");
    const millideg = std.fmt.parseInt(u32, trimmed, 10) catch return 0;

    const celsius: u8 = @intCast(millideg / 1000);
    if (temperature_type == .fahrenheit) {
        return @intCast(@as(u32, celsius) * 9 / 5 + 32);
    }
    return celsius;
}

const CpuState = struct {
    prev_idle: u64 = 0,
    prev_total: u64 = 0,
    initialized: bool = false,
};

var cpu_state = CpuState{};

fn readCpuStat() ?struct { idle: u64, total: u64 } {
    const file = std.fs.openFileAbsolute("/proc/stat", .{}) catch {
        log.err("failed to open /proc/stat", .{});
        return null;
    };
    defer file.close();

    var buf: [512]u8 = undefined;
    const bytes_read = file.read(&buf) catch return null;
    if (bytes_read == 0) return null;

    const line_end = std.mem.indexOfScalar(u8, buf[0..bytes_read], '\n') orelse bytes_read;
    var iter = std.mem.tokenizeAny(u8, buf[0..line_end], " \t");

    // Skip "cpu" label
    _ = iter.next() orelse return null;

    var values: [8]u64 = undefined;
    for (&values) |*v| {
        const tok = iter.next() orelse return null;
        v.* = std.fmt.parseInt(u64, tok, 10) catch return null;
    }

    const user = values[0];
    const nice = values[1];
    const system = values[2];
    const idle_val = values[3];
    const iowait = values[4];
    const irq = values[5];
    const softirq = values[6];
    const steal = values[7];

    return .{
        .idle = idle_val + iowait,
        .total = user + nice + system + idle_val + iowait + irq + softirq + steal,
    };
}

pub fn getCpuPercent() u8 {
    if (!cpu_state.initialized) {
        const stats = readCpuStat() orelse return 0;
        cpu_state.prev_idle = stats.idle;
        cpu_state.prev_total = stats.total;
        std.time.sleep(100 * std.time.ns_per_ms);
        cpu_state.initialized = true;
    }

    const stats = readCpuStat() orelse return 0;

    const diff_idle = stats.idle - cpu_state.prev_idle;
    const diff_total = stats.total - cpu_state.prev_total;
    cpu_state.prev_idle = stats.idle;
    cpu_state.prev_total = stats.total;

    if (diff_total == 0) return 0;
    return @intCast((100 * (diff_total - diff_idle) + diff_total / 2) / diff_total);
}

pub fn getHostname() []const u8 {
    const HostBuf = struct {
        var buf: [65]u8 = undefined;
    };
    var uts: linux.utsname = undefined;
    const ret = linux.uname(&uts);
    if (@as(isize, @bitCast(ret)) != 0) {
        return "unknown";
    }
    const nodename: [*:0]const u8 = @ptrCast(&uts.nodename);
    const len = std.mem.len(nodename);
    const copy_len = @min(len, HostBuf.buf.len - 1);
    @memcpy(HostBuf.buf[0..copy_len], nodename[0..copy_len]);
    HostBuf.buf[copy_len] = 0;
    return HostBuf.buf[0..copy_len];
}

pub fn getDietpiUpdateStatus() i32 {
    std.fs.accessAbsolute("/run/dietpi", .{}) catch return 0;
    std.fs.accessAbsolute("/run/dietpi/.update_available", .{}) catch return 1;
    return 2;
}

pub fn getAptUpdateCount() i32 {
    const file = std.fs.openFileAbsolute("/run/dietpi/.apt_updates", .{}) catch return -1;
    defer file.close();

    var buf: [16]u8 = undefined;
    const bytes_read = file.read(&buf) catch return 0;
    if (bytes_read == 0) return 0;

    const trimmed = std.mem.trimRight(u8, buf[0..bytes_read], " \t\n\r");
    return std.fmt.parseInt(i32, trimmed, 10) catch 0;
}
