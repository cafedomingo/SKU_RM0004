const std = @import("std");
const lcd = @import("st7735.zig");
const dashboard = @import("dashboard.zig");
const rpi = @import("rpi_info.zig");

pub const std_options = std.Options{
    .log_level = .info,
    .logFn = customLog,
};

fn customLog(
    comptime level: std.log.Level,
    comptime scope: @TypeOf(.EnumLiteral),
    comptime format: []const u8,
    args: anytype,
) void {
    const scope_prefix = if (scope == .default) "" else @tagName(scope) ++ ": ";
    const prefix = comptime level.asText() ++ ": " ++ scope_prefix;
    std.io.getStdErr().writer().print(prefix ++ format ++ "\n", args) catch {};
}

pub fn main() void {
    std.log.info("display: starting (refresh every {d}s)", .{rpi.refresh_interval_secs});

    lcd.begin() catch {
        std.log.err("display: lcd_begin failed, exiting", .{});
        return;
    };

    lcd.fillScreen(lcd.BLACK);

    while (true) {
        dashboard.displayDashboard();
        std.time.sleep(rpi.refresh_interval_secs * std.time.ns_per_s);
    }
}
