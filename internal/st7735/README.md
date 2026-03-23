# ST7735 TFT LCD Driver

## Display

- UCTRONICS SKU_RM0004: 160x80 pixel ST7735 TFT LCD
- Color format: RGB565, 2 bytes per pixel
- Full framebuffer: 25,600 bytes (160 × 80 × 2)
- I2C address: `0x18`
- Y-offset: 24 pixels (`ST7735_YSTART`) — the controller addresses a 160x160 panel; our 160x80 window starts at row 24

## I2C Interface

- Bus: `/dev/i2c-1` at 400kHz
- Config: `dtparam=i2c_arm=on,i2c_arm_baudrate=400000` in `/boot/firmware/config.txt`
  - **Gotcha:** bare `i2c_arm_baudrate=X` lines (without `dtparam=` prefix) are silently ignored
- Burst chunk size: 160 bytes max (`BURST_MAX_LENGTH`) — this is a hardware limit, do not increase
- Inter-chunk delay (`BURST_DELAY_US`) — tuned empirically at 400kHz:
  - 300μs: garbled output
  - 400μs: intermittent color errors
  - **450μs: stable (current value)**
  - 500μs: stable but unnecessarily slow
- Full-screen transfer: 160 chunks × (160 bytes + 450μs delay) ≈ 720ms
- These burst parameters should not be changed without hardware testing
