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
)

const (
	i2cClockFreqPath = "/proc/device-tree/soc/i2c@7e804000/clock-frequency"
	i2cExpectedHz    = 400000
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("starting")

	checkI2CSpeed(logger)

	disp, err := st7735.NewDisplay(logger)
	if err != nil {
		logger.Error("failed to open display", "error", err)
		os.Exit(1)
	}
	defer func() { _ = disp.Close() }()

	collector := sysinfo.NewCollector(logger)
	cfgLoader := config.NewLoader(config.ConfigPath, logger)
	var activeScreen screen.Screen
	lastScreenName := ""

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	for {
		start := time.Now()
		cfg := cfgLoader.Load()

		if cfg.Screen != lastScreenName {
			activeScreen = screen.New(cfg.Screen, disp, collector)
			lastScreenName = cfg.Screen
		}

		collector.Refresh()
		activeScreen.Update(cfg)
		activeScreen.Draw()

		// Sleep until next refresh, or exit on shutdown signal
		select {
		case <-ctx.Done():
			logger.Info("shutting down")
			return
		case <-time.After(cfg.Refresh - time.Since(start)):
		}
	}
}

func checkI2CSpeed(logger *slog.Logger) {
	data, err := os.ReadFile(i2cClockFreqPath)
	if err != nil {
		logger.Warn("could not read I2C clock frequency", "error", err)
		return
	}
	if len(data) < 4 {
		logger.Warn("I2C clock frequency file too short", "path", i2cClockFreqPath, "len", len(data))
		return
	}
	freq := binary.BigEndian.Uint32(data[:4])
	if freq == i2cExpectedHz {
		logger.Info("I2C bus speed", "hz", freq)
	} else {
		logger.Warn("I2C bus speed unexpected", "hz", freq, "expected", i2cExpectedHz)
	}
}
