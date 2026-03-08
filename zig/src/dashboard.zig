const std = @import("std");
const lcd = @import("st7735.zig");
const fonts = @import("fonts.zig");
const rpi = @import("rpi_info.zig");

const METRIC_BAR_WIDTH: u16 = 65;
const METRIC_BAR_HEIGHT: u16 = 6;

fn thresholdColor(val: u8) u16 {
    if (val < 60) return lcd.GREEN;
    if (val < 80) return lcd.YELLOW;
    if (val < 90) return lcd.ORANGE;
    return lcd.RED;
}

fn tempThresholdColor(celsius: u8) u16 {
    if (celsius < 40) return lcd.CYAN;
    if (celsius < 50) return lcd.GREEN;
    if (celsius < 60) return lcd.YELLOW;
    if (celsius < 70) return lcd.ORANGE;
    return lcd.RED;
}

fn drawMetric(x: u16, y: u16, label: []const u8, value: []const u8, bar_pct: u8, color: u16) void {
    const val_x = x + METRIC_BAR_WIDTH - @as(u16, @intCast(value.len)) * fonts.Font_7x10.width;
    lcd.writeString(x, y, label, fonts.Font_7x10, lcd.WHITE, lcd.BLACK);
    lcd.writeString(val_x, y, value, fonts.Font_7x10, color, lcd.BLACK);
    lcd.drawBar(x, y + 12, METRIC_BAR_WIDTH, METRIC_BAR_HEIGHT, bar_pct, color);
}

pub fn displayDashboard() void {
    var buf: [24]u8 = undefined;

    const cpu_percent = rpi.getCpuPercent();
    const ram_percent = rpi.getRamPercent();
    const temp = rpi.getTemperature();
    const disk_percent = rpi.getDiskPercent();
    const hostname = rpi.getHostname();
    const ip = rpi.getIpAddress();
    const dietpi_status = rpi.getDietpiUpdateStatus();
    const apt_count = rpi.getAptUpdateCount();

    // Hostname (truncated to 16 chars)
    lcd.writeString(2, 0, hostname[0..@min(hostname.len, 16)], fonts.Font_8x16, lcd.WHITE, lcd.BLACK);

    // IP address
    lcd.writeString(2, 18, ip, fonts.Font_7x10, lcd.VIOLET, lcd.BLACK);

    // Separator line
    lcd.fillRectangle(0, 30, lcd.WIDTH, 1, lcd.BLUE);

    // DietPi update indicator (diamond shape)
    if (dietpi_status == 2) {
        lcd.fillRectangle(154, 5, 2, 1, lcd.RED);
        lcd.fillRectangle(153, 6, 4, 1, lcd.RED);
        lcd.fillRectangle(152, 7, 6, 1, lcd.RED);
        lcd.fillRectangle(152, 8, 6, 1, lcd.RED);
        lcd.fillRectangle(153, 9, 4, 1, lcd.RED);
        lcd.fillRectangle(154, 10, 2, 1, lcd.RED);
    }

    // APT update badge
    lcd.fillRectangle(124, 18, 36, 10, lcd.BLACK);
    if (apt_count > 0) {
        var badge_buf: [5]u8 = undefined;
        const capped: u8 = if (apt_count > 99) 99 else @intCast(apt_count);
        const badge = std.fmt.bufPrint(&badge_buf, "^{d}", .{capped}) catch "^?";
        const badge_color: u16 = if (apt_count >= 10) lcd.RED else lcd.YELLOW;
        const bx: u16 = lcd.WIDTH - @as(u16, @intCast(badge.len)) * fonts.Font_7x10.width - 2;
        lcd.writeString(bx, 19, badge, fonts.Font_7x10, badge_color, lcd.BLACK);
    }

    // CPU metric
    {
        const color = thresholdColor(cpu_percent);
        const val = std.fmt.bufPrint(&buf, "{d:>3}%", .{cpu_percent}) catch "???%";
        drawMetric(2, 34, "CPU:", val, cpu_percent, color);
    }

    // RAM metric
    {
        const color = thresholdColor(ram_percent);
        const val = std.fmt.bufPrint(&buf, "{d:>3}%", .{ram_percent}) catch "???%";
        drawMetric(2, 56, "RAM:", val, ram_percent, color);
    }

    // Temperature metric
    {
        const color = tempThresholdColor(temp);
        const display_temp: u32 = if (rpi.temperature_type == .fahrenheit) @as(u32, temp) * 9 / 5 + 32 else temp;
        const suffix: u8 = if (rpi.temperature_type == .fahrenheit) 'F' else 'C';
        const val = std.fmt.bufPrint(&buf, "{d:>3}{c}", .{ display_temp, suffix }) catch "???C";
        const bar_val: u8 = if (temp > 100) 100 else temp;
        drawMetric(84, 34, "TEMP:", val, bar_val, color);
    }

    // Disk metric
    {
        const color = thresholdColor(disk_percent);
        const val = std.fmt.bufPrint(&buf, "{d:>3}%", .{disk_percent}) catch "???%";
        drawMetric(84, 56, "DISK:", val, disk_percent, color);
    }
}
