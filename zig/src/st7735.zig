const std = @import("std");
const posix = std.posix;
const fonts = @import("fonts.zig");

const log = std.log.scoped(.st7735);

pub const WIDTH: u16 = 160;
pub const HEIGHT: u16 = 80;

pub fn color565(r: u8, g: u8, b: u8) u16 {
    return (@as(u16, r & 0xF8) << 8) | (@as(u16, g & 0xFC) << 3) | (@as(u16, b & 0xF8) >> 3);
}

pub const BLACK: u16 = 0x0000;
pub const BLUE: u16 = 0x001F;
pub const CYAN: u16 = 0x07FF;
pub const GRAY: u16 = 0x8410;
pub const GREEN: u16 = 0x07E0;
pub const MAGENTA: u16 = 0xF81F;
pub const ORANGE: u16 = 0xFD20;
pub const RED: u16 = 0xF800;
pub const VIOLET: u16 = 0xB41F;
pub const WHITE: u16 = 0xFFFF;
pub const YELLOW: u16 = 0xFFE0;

const I2C_ADDRESS: u16 = 0x18;
const I2C_SLAVE_FORCE: u16 = 0x0706;
const BURST_MAX_LENGTH: u32 = 160;

const X_COORDINATE_REG: u8 = 0x2A;
const Y_COORDINATE_REG: u8 = 0x2B;
const CHAR_DATA_REG: u8 = 0x2C;
const WRITE_DATA_REG: u8 = 0x00;
const BURST_WRITE_REG: u8 = 0x01;
const SYNC_REG: u8 = 0x03;

const XSTART: u8 = 0;
const YSTART: u8 = 24;

var i2c_fd: posix.fd_t = -1;

pub const LcdError = error{
    OpenFailed,
    IoctlFailed,
    WriteFailed,
};

pub fn begin() LcdError!void {
    i2c_fd = posix.open("/dev/i2c-1", .{ .ACCMODE = .RDWR }, 0) catch {
        log.err("Device I2C-1 failed to initialize", .{});
        return LcdError.OpenFailed;
    };

    const result = std.os.linux.ioctl(@bitCast(i2c_fd), I2C_SLAVE_FORCE, I2C_ADDRESS);
    if (@as(isize, @bitCast(result)) < 0) {
        log.err("ioctl I2C_SLAVE_FORCE failed", .{});
        return LcdError.IoctlFailed;
    }
}

fn i2cWrite(buf: []const u8) void {
    _ = posix.write(i2c_fd, buf) catch {
        log.err("i2c write failed", .{});
        return;
    };
    std.time.sleep(10 * std.time.ns_per_us);
}

fn i2cWriteData(high: u8, low: u8) void {
    const msg = [3]u8{ WRITE_DATA_REG, high, low };
    i2cWrite(&msg);
}

fn i2cWriteCommand(command: u8, high: u8, low: u8) void {
    const msg = [3]u8{ command, high, low };
    i2cWrite(&msg);
}

fn i2cBurstTransfer(buff: []const u8) void {
    var count: u32 = 0;
    const length: u32 = @intCast(buff.len);
    i2cWriteCommand(BURST_WRITE_REG, 0x00, 0x01);
    while (length > count) {
        const remaining = length - count;
        const chunk: u32 = if (remaining > BURST_MAX_LENGTH) BURST_MAX_LENGTH else remaining;
        const written = posix.write(i2c_fd, buff[count .. count + chunk]) catch |err| {
            log.err("burst write failed at offset {}: {}", .{ count, err });
            break;
        };
        count += @intCast(written);
        std.time.sleep(700 * std.time.ns_per_us);
    }
    i2cWriteCommand(BURST_WRITE_REG, 0x00, 0x00);
    i2cWriteCommand(SYNC_REG, 0x00, 0x01);
}

fn setAddressWindow(x0: u8, y0: u8, x1: u8, y1: u8) void {
    i2cWriteCommand(X_COORDINATE_REG, x0 +% XSTART, x1 +% XSTART);
    i2cWriteCommand(Y_COORDINATE_REG, y0 +% YSTART, y1 +% YSTART);
    i2cWriteCommand(CHAR_DATA_REG, 0x00, 0x00);
    i2cWriteCommand(SYNC_REG, 0x00, 0x01);
}

fn drawImage(x: u16, y: u16, w: u16, h: u16, data: []const u8) void {
    setAddressWindow(
        @intCast(x),
        @intCast(y),
        @intCast(x + w - 1),
        @intCast(y + h - 1),
    );
    i2cBurstTransfer(data);
}

pub fn fillRectangle(x: u16, y: u16, w_in: u16, h_in: u16, color_val: u16) void {
    var w = w_in;
    var h = h_in;
    if (x >= WIDTH or y >= HEIGHT) return;
    if (x + w >= WIDTH) w = WIDTH - x;
    if (y + h >= HEIGHT) h = HEIGHT - y;

    setAddressWindow(
        @intCast(x),
        @intCast(y),
        @intCast(x + w - 1),
        @intCast(y + h - 1),
    );

    var buff: [320]u8 = undefined;
    var i: u16 = 0;
    while (i < w) : (i += 1) {
        buff[i * 2] = @intCast(color_val >> 8);
        buff[i * 2 + 1] = @intCast(color_val & 0xFF);
    }
    const slice = buff[0 .. @as(usize, w) * 2];
    var row: u16 = h;
    while (row > 0) : (row -= 1) {
        i2cBurstTransfer(slice);
    }
}

pub fn fillScreen(color_val: u16) void {
    fillRectangle(0, 0, WIDTH, HEIGHT, color_val);
    i2cWriteCommand(SYNC_REG, 0x00, 0x01);
}

pub fn drawBar(x: u16, y: u16, w: u16, h: u16, val: u8, color_val: u16) void {
    var filled: u16 = @as(u16, val) * w / 100;
    if (filled > w) filled = w;
    if (filled > 0) fillRectangle(x, y, filled, h, color_val);
    if (filled < w) fillRectangle(x + filled, y, w - filled, h, GRAY);
}

fn writeChar(x: u16, y: u16, ch: u8, font: fonts.FontDef, color_val: u16, bgcolor: u16) void {
    const max_buf_size = 16 * 26 * 2;
    var buff: [max_buf_size]u8 = undefined;

    const char_idx: usize = if (ch >= 32) ch - 32 else 0;

    var i: usize = 0;
    while (i < font.height) : (i += 1) {
        const b: u16 = font.data[char_idx * font.height + i];
        var j: usize = 0;
        while (j < font.width) : (j += 1) {
            const idx = (i * font.width + j) * 2;
            const shifted: u16 = b << @intCast(j);
            const c: u16 = if ((shifted & 0x8000) != 0) color_val else bgcolor;
            buff[idx] = @intCast(c >> 8);
            buff[idx + 1] = @intCast(c & 0xFF);
        }
    }

    const total_bytes: usize = @as(usize, font.width) * @as(usize, font.height) * 2;
    drawImage(x, y, font.width, font.height, buff[0..total_bytes]);
}

pub fn writeString(x_in: u16, y_in: u16, str: []const u8, font: fonts.FontDef, color_val: u16, bgcolor: u16) void {
    var x = x_in;
    var y = y_in;
    for (str) |ch| {
        if (x + font.width >= WIDTH) {
            x = 0;
            y += font.height;
            if (y + font.height >= HEIGHT) break;
            if (ch == ' ') continue;
        }
        writeChar(x, y, ch, font, color_val, bgcolor);
        x += font.width;
    }
}
