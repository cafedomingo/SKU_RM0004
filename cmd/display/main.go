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
	i2cBus          = "/dev/i2c-1"
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
	cfgLoader := config.NewLoader(config.DefaultPath, logger)

	var front, back st7735.Framebuffer
	var diagState screen.DiagState
	var sparkState screen.SparklineState
	prevScreen := ""

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Fill screen black
	back.Fill(theme.ColorBG)
	disp.SendFull(back.Pixels[:])
	front = back

	for {
		start := time.Now()
		cfg := cfgLoader.Load()

		// Screen change: reset state, full black
		if cfg.Screen != prevScreen {
			diagState = screen.DiagState{}
			sparkState = screen.SparklineState{}
			back.Fill(theme.ColorBG)
			disp.SendFull(back.Pixels[:])
			front = back
			prevScreen = cfg.Screen
		}

		back.Fill(theme.ColorBG)

		switch cfg.Screen {
		case config.ScreenDiagnostic:
			if diagState.Page == 0 {
				collector.Refresh()
			}
			screen.RenderDiagnostic(&back, collector, cfg, &diagState)
			disp.SendFull(back.Pixels[:]) // diagnostic always full redraw
			front = back

		case config.ScreenSparkline:
			collector.Refresh()
			screen.RenderSparkline(&back, collector, cfg, &sparkState)
			for _, r := range st7735.DiffRegions(&front, &back) {
				disp.SendRegion(0, r.Y, st7735.Width, r.H,
					back.Pixels[r.Y*st7735.Width:(r.Y+r.H)*st7735.Width])
			}
			front = back

		default: // dashboard
			collector.Refresh()
			screen.RenderDashboard(&back, collector, cfg)
			for _, r := range st7735.DiffRegions(&front, &back) {
				disp.SendRegion(0, r.Y, st7735.Width, r.H,
					back.Pixels[r.Y*st7735.Width:(r.Y+r.H)*st7735.Width])
			}
			front = back
		}

		// Sleep remaining time, or exit on signal
		elapsed := time.Since(start)
		if remaining := cfg.Refresh - elapsed; remaining > 0 {
			select {
			case <-ctx.Done():
				logger.Info("shutting down")
				return
			case <-time.After(remaining):
			}
		}

		select {
		case <-ctx.Done():
			logger.Info("shutting down")
			return
		default:
		}
	}
}

func checkI2CSpeed(logger *slog.Logger) {
	// Read /proc/device-tree/soc/i2c@7e804000/clock-frequency
	// It's a big-endian uint32 in a binary file
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
