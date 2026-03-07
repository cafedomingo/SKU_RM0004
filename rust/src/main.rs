mod dashboard;
#[allow(dead_code)]
mod fonts;
mod rpi_info;
mod st7735;

use rpi_info::REFRESH_INTERVAL_SECS;
use std::thread;
use std::time::Duration;

fn main() {
    eprintln!("display: starting (refresh every {}s)", REFRESH_INTERVAL_SECS);

    let lcd = match st7735::Lcd::begin() {
        Ok(lcd) => lcd,
        Err(()) => {
            eprintln!("display: lcd_begin failed, exiting");
            std::process::exit(1);
        }
    };

    lcd.fill_screen(st7735::BLACK);

    let mut cpu_tracker = rpi_info::CpuTracker::new();

    loop {
        dashboard::display_dashboard(&lcd, &mut cpu_tracker);
        thread::sleep(Duration::from_secs(REFRESH_INTERVAL_SECS));
    }
}
