//! Dashboard rendering for the ST7735 LCD display.

use crate::fonts::{FONT_7X10, FONT_8X16};
use crate::rpi_info::{self, CpuTracker, FAHRENHEIT, TEMPERATURE_TYPE};
use crate::st7735::{self, Lcd};

const METRIC_BAR_WIDTH: u16 = 65;
const METRIC_BAR_HEIGHT: u16 = 6;

fn threshold_color(val: u8) -> u16 {
    if val < 60 {
        st7735::GREEN
    } else if val < 80 {
        st7735::YELLOW
    } else if val < 90 {
        st7735::ORANGE
    } else {
        st7735::RED
    }
}

fn temp_threshold_color(celsius: u8) -> u16 {
    if celsius < 40 {
        st7735::CYAN
    } else if celsius < 50 {
        st7735::GREEN
    } else if celsius < 60 {
        st7735::YELLOW
    } else if celsius < 70 {
        st7735::ORANGE
    } else {
        st7735::RED
    }
}

fn draw_metric(lcd: &Lcd, x: u16, y: u16, label: &str, value: &str, bar_pct: u8, color: u16) {
    let val_x = x + METRIC_BAR_WIDTH - (value.len() as u16) * FONT_7X10.width as u16;
    lcd.write_string(x, y, label, FONT_7X10, st7735::WHITE, st7735::BLACK);
    lcd.write_string(val_x, y, value, FONT_7X10, color, st7735::BLACK);
    lcd.draw_bar(x, y + 12, METRIC_BAR_WIDTH, METRIC_BAR_HEIGHT, bar_pct, color);
}

/// Render the full dashboard to the LCD display.
pub fn display_dashboard(lcd: &Lcd, cpu_tracker: &mut CpuTracker) {
    let cpu_percent = cpu_tracker.get_cpu_percent();
    let ram_percent = rpi_info::get_ram_percent();
    let temp = rpi_info::get_temperature();
    let disk_percent = rpi_info::get_disk_percent();
    let hostname = rpi_info::get_hostname();
    let ip = rpi_info::get_ip_address();
    let dietpi_status = rpi_info::get_dietpi_update_status();
    let apt_count = rpi_info::get_apt_update_count();

    // Hostname (truncated to 16 chars)
    let host_display: String = hostname.chars().take(16).collect();
    lcd.write_string(2, 0, &host_display, FONT_8X16, st7735::WHITE, st7735::BLACK);

    // IP address
    lcd.write_string(2, 18, &ip, FONT_7X10, st7735::VIOLET, st7735::BLACK);

    // Horizontal divider
    lcd.fill_rectangle(0, 30, st7735::WIDTH, 1, st7735::BLUE);

    // DietPi update indicator (diamond shape)
    if dietpi_status == 2 {
        lcd.fill_rectangle(154, 5, 2, 1, st7735::RED);
        lcd.fill_rectangle(153, 6, 4, 1, st7735::RED);
        lcd.fill_rectangle(152, 7, 6, 1, st7735::RED);
        lcd.fill_rectangle(152, 8, 6, 1, st7735::RED);
        lcd.fill_rectangle(153, 9, 4, 1, st7735::RED);
        lcd.fill_rectangle(154, 10, 2, 1, st7735::RED);
    }

    // APT update badge
    lcd.fill_rectangle(124, 18, 36, 10, st7735::BLACK);
    if apt_count > 0 {
        let capped = if apt_count > 99 { 99 } else { apt_count };
        let badge = format!("^{}", capped);
        let color = if apt_count >= 10 { st7735::RED } else { st7735::YELLOW };
        let bx = st7735::WIDTH - (badge.len() as u16) * FONT_7X10.width as u16 - 2;
        lcd.write_string(bx, 19, &badge, FONT_7X10, color, st7735::BLACK);
    }

    // CPU metric
    let color = threshold_color(cpu_percent);
    let buf = format!("{:3}%", cpu_percent);
    draw_metric(lcd, 2, 34, "CPU:", &buf, cpu_percent, color);

    // RAM metric
    let color = threshold_color(ram_percent);
    let buf = format!("{:3}%", ram_percent);
    draw_metric(lcd, 2, 56, "RAM:", &buf, ram_percent, color);

    // Temperature metric
    let temp_for_bar = if TEMPERATURE_TYPE == FAHRENHEIT {
        ((temp as f32 - 32.0) / 1.8) as u8
    } else {
        temp
    };
    let color = temp_threshold_color(temp_for_bar);
    let unit_char = if TEMPERATURE_TYPE == FAHRENHEIT { 'F' } else { 'C' };
    let buf = format!("{:3}{}", temp, unit_char);
    let bar_val = if temp_for_bar > 100 { 100 } else { temp_for_bar };
    draw_metric(lcd, 84, 34, "TEMP:", &buf, bar_val, color);

    // Disk metric
    let color = threshold_color(disk_percent);
    let buf = format!("{:3}%", disk_percent);
    draw_metric(lcd, 84, 56, "DISK:", &buf, disk_percent, color);
}
