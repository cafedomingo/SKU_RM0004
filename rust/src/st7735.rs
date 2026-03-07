//! ST7735 LCD I2C driver for 160x80 TFT display.

use crate::fonts::FontDef;
use std::fs::OpenOptions;
use std::os::unix::io::{AsRawFd, RawFd};
use std::thread;
use std::time::Duration;

// Display dimensions
pub const WIDTH: u16 = 160;
pub const HEIGHT: u16 = 80;

// RGB565 color constants
pub const BLACK: u16 = 0x0000;
pub const BLUE: u16 = 0x001F;
pub const CYAN: u16 = 0x07FF;
pub const GRAY: u16 = 0x8410;
pub const GREEN: u16 = 0x07E0;
#[allow(dead_code)]
pub const MAGENTA: u16 = 0xF81F;
pub const ORANGE: u16 = 0xFD20;
pub const RED: u16 = 0xF800;
pub const VIOLET: u16 = 0xB41F;
pub const WHITE: u16 = 0xFFFF;
pub const YELLOW: u16 = 0xFFE0;

// I2C constants
const I2C_ADDRESS: u64 = 0x18;
const I2C_SLAVE_FORCE: u64 = 0x0706;
const BURST_MAX_LENGTH: usize = 160;

// Register addresses
const X_COORDINATE_REG: u8 = 0x2A;
const Y_COORDINATE_REG: u8 = 0x2B;
const CHAR_DATA_REG: u8 = 0x2C;
#[allow(dead_code)]
const WRITE_DATA_REG: u8 = 0x00;
const BURST_WRITE_REG: u8 = 0x01;
const SYNC_REG: u8 = 0x03;

// Display offsets
const X_START: u8 = 0;
const Y_START: u8 = 24;

/// Constructs an RGB565 color value from 8-bit R, G, B components.
#[allow(dead_code)]
pub const fn color565(r: u8, g: u8, b: u8) -> u16 {
    ((r as u16 & 0xF8) << 8) | ((g as u16 & 0xFC) << 3) | ((b as u16 & 0xF8) >> 3)
}

/// ST7735 LCD display driver communicating over I2C.
pub struct Lcd {
    fd: RawFd,
    /// Keep the file alive so the fd stays valid.
    _file: std::fs::File,
}

impl Lcd {
    /// Initialize the I2C connection to the LCD display.
    pub fn begin() -> Result<Self, ()> {
        let file = OpenOptions::new()
            .read(true)
            .write(true)
            .open("/dev/i2c-1")
            .map_err(|_| {
                eprintln!("Device I2C-1 failed to initialize");
            })?;

        let fd = file.as_raw_fd();

        // SAFETY: ioctl to set I2C slave address. The fd is valid and open.
        let ret = unsafe { libc::ioctl(fd, I2C_SLAVE_FORCE, I2C_ADDRESS) };
        if ret < 0 {
            eprintln!("st7735: ioctl I2C_SLAVE_FORCE failed");
            return Err(());
        }

        Ok(Lcd { fd, _file: file })
    }

    #[allow(dead_code)]
    fn i2c_write_data(&self, high: u8, low: u8) {
        let msg: [u8; 3] = [WRITE_DATA_REG, high, low];
        let written = unsafe { libc::write(self.fd, msg.as_ptr() as *const libc::c_void, 3) };
        if written != 3 {
            eprintln!("st7735: i2c_write_data failed");
        }
        thread::sleep(Duration::from_micros(10));
    }

    fn i2c_write_command(&self, command: u8, high: u8, low: u8) {
        let msg: [u8; 3] = [command, high, low];
        let written = unsafe { libc::write(self.fd, msg.as_ptr() as *const libc::c_void, 3) };
        if written != 3 {
            eprintln!("st7735: i2c_write_command failed");
        }
        thread::sleep(Duration::from_micros(10));
    }

    fn i2c_burst_transfer(&self, buff: &[u8]) {
        let length = buff.len();
        let mut count: usize = 0;

        self.i2c_write_command(BURST_WRITE_REG, 0x00, 0x01);

        while length > count {
            let chunk = if (length - count) > BURST_MAX_LENGTH {
                BURST_MAX_LENGTH
            } else {
                length - count
            };

            let written = unsafe { libc::write(self.fd, buff[count..].as_ptr() as *const libc::c_void, chunk) };

            if written < 0 {
                eprintln!("st7735: burst write failed at offset {}", count);
                break;
            }
            count += written as usize;
            thread::sleep(Duration::from_micros(700));
        }

        self.i2c_write_command(BURST_WRITE_REG, 0x00, 0x00);
        self.i2c_write_command(SYNC_REG, 0x00, 0x01);
    }

    fn set_address_window(&self, x0: u8, y0: u8, x1: u8, y1: u8) {
        self.i2c_write_command(X_COORDINATE_REG, x0.wrapping_add(X_START), x1.wrapping_add(X_START));
        self.i2c_write_command(Y_COORDINATE_REG, y0.wrapping_add(Y_START), y1.wrapping_add(Y_START));
        self.i2c_write_command(CHAR_DATA_REG, 0x00, 0x00);
        self.i2c_write_command(SYNC_REG, 0x00, 0x01);
    }

    fn draw_image(&self, x: u16, y: u16, w: u16, h: u16, data: &[u8]) {
        self.set_address_window(x as u8, y as u8, (x + w - 1) as u8, (y + h - 1) as u8);
        self.i2c_burst_transfer(&data[..((w * h * 2) as usize).min(data.len())]);
    }

    /// Fill a rectangle with a solid color.
    pub fn fill_rectangle(&self, x: u16, y: u16, mut w: u16, mut h: u16, color: u16) {
        if x >= WIDTH || y >= HEIGHT {
            return;
        }
        if x + w >= WIDTH {
            w = WIDTH - x;
        }
        if y + h >= HEIGHT {
            h = HEIGHT - y;
        }

        self.set_address_window(x as u8, y as u8, (x + w - 1) as u8, (y + h - 1) as u8);

        let mut buff = [0u8; 320];
        for i in 0..w as usize {
            buff[i * 2] = (color >> 8) as u8;
            buff[i * 2 + 1] = (color & 0xFF) as u8;
        }

        let row_bytes = (w as usize) * 2;
        for _ in 0..h {
            self.i2c_burst_transfer(&buff[..row_bytes]);
        }
    }

    /// Fill the entire screen with a solid color.
    pub fn fill_screen(&self, color: u16) {
        self.fill_rectangle(0, 0, WIDTH, HEIGHT, color);
        self.i2c_write_command(SYNC_REG, 0x00, 0x01);
    }

    /// Draw a progress bar with a filled portion and gray remainder.
    pub fn draw_bar(&self, x: u16, y: u16, w: u16, h: u16, val: u8, color: u16) {
        let mut filled = val as u16 * w / 100;
        if filled > w {
            filled = w;
        }
        if filled > 0 {
            self.fill_rectangle(x, y, filled, h, color);
        }
        if filled < w {
            self.fill_rectangle(x + filled, y, w - filled, h, GRAY);
        }
    }

    fn write_char(&self, x: u16, y: u16, ch: char, font: FontDef, color: u16, bgcolor: u16) {
        let ch_idx = ch as usize;
        if !(32..=126).contains(&ch_idx) {
            return;
        }
        let char_offset = (ch_idx - 32) * font.height as usize;
        let buf_size = font.width as usize * font.height as usize * 2;
        let mut buff = vec![0u8; buf_size];

        for i in 0..font.height as usize {
            let b = font.data[char_offset + i];
            for j in 0..font.width as usize {
                let idx = (i * font.width as usize + j) * 2;
                let c = if (b << j) & 0x8000 != 0 { color } else { bgcolor };
                buff[idx] = (c >> 8) as u8;
                buff[idx + 1] = (c & 0xFF) as u8;
            }
        }

        self.draw_image(x, y, font.width as u16, font.height as u16, &buff);
    }

    /// Write a string to the display at the given position.
    pub fn write_string(&self, mut x: u16, mut y: u16, s: &str, font: FontDef, color: u16, bgcolor: u16) {
        for ch in s.chars() {
            if x + font.width as u16 >= WIDTH {
                x = 0;
                y += font.height as u16;
                if y + font.height as u16 >= HEIGHT {
                    break;
                }
                if ch == ' ' {
                    continue;
                }
            }
            self.write_char(x, y, ch, font, color, bgcolor);
            x += font.width as u16;
        }
    }
}
