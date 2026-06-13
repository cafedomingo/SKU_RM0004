package st7735

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"os"
	"syscall"
	"time"
)

const (
	i2cBusPath   = "/dev/i2c-1"
	i2cAddress   = 0x18
	i2cSlave     = 0x0703 // I2C_SLAVE ioctl from <linux/i2c-dev.h>
	burstMaxLen  = 160    // hardware limit, do NOT increase
	burstDelayUS = 450    // empirically tuned at 400kHz
	yOffset      = 24     // controller is 160x160, our 160x80 starts at row 24

	regWriteData  = 0x00
	regBurstWrite = 0x01
	regSync       = 0x03
	regXCoord     = 0x2A
	regYCoord     = 0x2B
	regCharData   = 0x2C
)

// Display sends pixel data to the ST7735 LCD over I2C.
type Display interface {
	SendRegion(x, y, w, h int, pixels []uint16)
	SendFull(pixels []uint16)
	Close() error
}

// i2cConn talks to an I2C device through the kernel's i2c-dev interface:
// one I2C_SLAVE ioctl to latch the address, then plain write(2) for each
// transaction.
type i2cConn struct {
	f *os.File
}

func openI2C(path string, addr uint8) (*i2cConn, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), i2cSlave, uintptr(addr)); errno != 0 {
		_ = f.Close()
		return nil, fmt.Errorf("ioctl I2C_SLAVE 0x%02x: %w", addr, errno)
	}
	return &i2cConn{f: f}, nil
}

func (c *i2cConn) Write(p []byte) (int, error) {
	return c.f.Write(p)
}

func (c *i2cConn) Close() error {
	return c.f.Close()
}

type display struct {
	dev    io.WriteCloser
	logger *slog.Logger
}

// NewDisplay opens the I2C bus and returns a Display backed by the
// UCTRONICS SKU_RM0004 ST7735 controller at address 0x18.
func NewDisplay(logger *slog.Logger) (Display, error) {
	dev, err := openI2C(i2cBusPath, i2cAddress)
	if err != nil {
		return nil, fmt.Errorf("st7735: open i2c bus %s: %w", i2cBusPath, err)
	}
	return &display{dev: dev, logger: logger}, nil
}

// writeCommand sends a 3-byte I2C command: [register, high, low].
func (d *display) writeCommand(reg, hi, lo byte) {
	if _, err := d.dev.Write([]byte{reg, hi, lo}); err != nil {
		d.logger.Warn("i2c write failed", "register", reg, "error", err)
	}
}

// setAddressWindow configures the ST7735 column/row address range for the
// next pixel write, applying the yOffset for the 160x80 panel position.
func (d *display) setAddressWindow(x0, y0, x1, y1 int) {
	d.writeCommand(regXCoord, byte(x0), byte(x1))
	d.writeCommand(regYCoord, byte(y0+yOffset), byte(y1+yOffset))
	d.writeCommand(regCharData, 0x00, 0x00)
	d.writeCommand(regSync, 0x00, 0x01)
}

// burstBegin enables burst-write mode on the I2C bridge.
func (d *display) burstBegin() {
	d.writeCommand(regBurstWrite, 0x00, 0x01)
}

// burstEnd disables burst-write mode and syncs.
func (d *display) burstEnd() {
	d.writeCommand(regBurstWrite, 0x00, 0x00)
	d.writeCommand(regSync, 0x00, 0x01)
}

// burstSend writes data in chunks of burstMaxLen with inter-chunk delays.
func (d *display) burstSend(data []byte) {
	for offset := 0; offset < len(data); {
		chunk := len(data) - offset
		if chunk > burstMaxLen {
			chunk = burstMaxLen
		}
		if _, err := d.dev.Write(data[offset : offset+chunk]); err != nil {
			d.logger.Warn("burst send failed", "offset", offset, "error", err)
		}
		offset += chunk
		time.Sleep(time.Duration(burstDelayUS) * time.Microsecond)
	}
}

// pixelsToBytes converts RGB565 pixel values to big-endian bytes (MSB first)
// as expected by the ST7735 controller.
func pixelsToBytes(pixels []uint16) []byte {
	buf := make([]byte, len(pixels)*2)
	for i, px := range pixels {
		binary.BigEndian.PutUint16(buf[i*2:], px)
	}
	return buf
}

// SendRegion sends a rectangular region of pixels to the display.
// The caller provides the contiguous pixel slice for the region.
func (d *display) SendRegion(x, y, w, h int, pixels []uint16) {
	d.setAddressWindow(x, y, x+w-1, y+h-1)
	data := pixelsToBytes(pixels)
	d.burstBegin()
	d.burstSend(data)
	d.burstEnd()
}

// SendFull sends the entire 160x80 framebuffer to the display.
func (d *display) SendFull(pixels []uint16) {
	d.SendRegion(0, 0, Width, Height, pixels)
}

// Close releases the I2C bus.
func (d *display) Close() error {
	return d.dev.Close()
}
