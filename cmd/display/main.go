package main

import (
	"context"
	"encoding/binary"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/screen"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
	"github.com/cafedomingo/SKU_RM0004/internal/theme"
)

const (
	i2cBus           = "/dev/i2c-1"
	i2cClockFreqPath = "/proc/device-tree/soc/i2c@7e804000/clock-frequency"
	i2cExpectedHz    = 400000
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("starting")

	checkI2CSpeed(logger)

	disp, err := st7735.NewDisplay(i2cBus, logger)
	if err != nil {
		logger.Error("failed to open display", "error", err)
		os.Exit(1)
	}
	defer disp.Close()

	collector := sysinfo.NewCollector(logger)
	cfgLoader := config.NewLoader(config.ConfigPath, logger)

	var front, back st7735.Framebuffer
	var current screen.Screen
	prevScreen := ""

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	for {
		start := time.Now()
		cfg := cfgLoader.Load()

		if cfg.Screen != prevScreen {
			current = screen.New(cfg.Screen)
			clearScreen(&back, &front, disp)
			prevScreen = cfg.Screen
		}

		if current.NeedsRefresh() {
			collector.Refresh()
		}

		back.Fill(theme.ColorBG)
		current.Render(&back, collector, cfg)

		if current.FullRedraw() {
			disp.SendFull(back.Pixels[:])
			front = back
		} else {
			sendDirty(disp, &front, &back)
		}

		if !sleepOrExit(ctx, cfg.Refresh-time.Since(start)) {
			logger.Info("shutting down")
			return
		}
	}
}

func clearScreen(back, front *st7735.Framebuffer, disp st7735.Display) {
	back.Fill(theme.ColorBG)
	disp.SendFull(back.Pixels[:])
	*front = *back
}

func sendDirty(disp st7735.Display, front, back *st7735.Framebuffer) {
	for _, r := range st7735.DiffRegions(front, back) {
		disp.SendRegion(0, r.Y, st7735.Width, r.H,
			back.Pixels[r.Y*st7735.Width:(r.Y+r.H)*st7735.Width])
	}
	*front = *back
}

func sleepOrExit(ctx context.Context, remaining time.Duration) bool {
	if remaining <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(remaining):
		return true
	}
}

func checkI2CSpeed(logger *slog.Logger) {
	data, err := os.ReadFile(i2cClockFreqPath)
	if err != nil {
		logger.Warn("could not read I2C clock frequency", "error", err)
		return
	}
	if len(data) < 4 {
		logger.Warn("I2C clock frequency file too short")
		return
	}
	freq := binary.BigEndian.Uint32(data[:4])
	if freq == i2cExpectedHz {
		logger.Info("I2C bus speed", "hz", freq)
	} else {
		logger.Warn("I2C bus speed unexpected", "hz", freq, "expected", i2cExpectedHz)
	}
}
