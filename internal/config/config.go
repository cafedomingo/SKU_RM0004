package config

import (
	"bufio"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ConfigPath     = "/etc/uctronics-display.conf"
	DefaultRefresh  = 5 * time.Second
	DefaultTempUnit = TempCelsius

	TempCelsius    = "C"
	TempFahrenheit = "F"

	MinRefresh = 2 * time.Second
	MaxRefresh = 30 * time.Second

	// Screen names
	ScreenDashboard  = "dashboard"
	ScreenDiagnostic = "diagnostic"
	ScreenSparkline  = "sparkline"

	// Config file keys
	keyScreen   = "screen"
	keyRefresh  = "refresh"
	keyTempUnit = "temp_unit"
)

const DefaultScreen = ScreenDashboard

var validScreens = map[string]bool{
	ScreenDashboard:  true,
	ScreenDiagnostic: true,
	ScreenSparkline:  true,
}

type Config struct {
	Screen   string
	Refresh  time.Duration
	TempUnit string
}

type Loader struct {
	path      string
	logger    *slog.Logger
	lastMtime time.Time
	cached    Config
}

func NewLoader(path string, logger *slog.Logger) *Loader {
	return &Loader{
		path:   path,
		logger: logger,
		cached: Config{Screen: DefaultScreen, Refresh: DefaultRefresh, TempUnit: DefaultTempUnit},
	}
}

func (l *Loader) Load() Config {
	info, err := os.Stat(l.path)
	if err != nil {
		// File doesn't exist or unreadable; return defaults
		return Config{Screen: DefaultScreen, Refresh: DefaultRefresh, TempUnit: DefaultTempUnit}
	}

	mtime := info.ModTime()
	if !l.lastMtime.IsZero() && !mtime.After(l.lastMtime) {
		return l.cached
	}

	cfg := Config{Screen: DefaultScreen, Refresh: DefaultRefresh, TempUnit: DefaultTempUnit}

	f, err := os.Open(l.path)
	if err != nil {
		l.logger.Warn("config: could not open file", "path", l.path, "err", err)
		return cfg
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sep := strings.IndexByte(line, '=')
		if sep < 0 {
			continue
		}
		key := strings.TrimSpace(line[:sep])
		val := strings.TrimSpace(line[sep+1:])

		switch key {
		case keyScreen:
			if validScreens[val] {
				cfg.Screen = val
			} else {
				l.logger.Warn("config: invalid screen value, using default", "value", val, "default", DefaultScreen)
				cfg.Screen = DefaultScreen
			}
		case keyRefresh:
			n, err := strconv.Atoi(val)
			if err != nil {
				l.logger.Warn("config: invalid refresh value, using default", "value", val)
				cfg.Refresh = DefaultRefresh
			} else {
				d := time.Duration(n) * time.Second
				if d < MinRefresh {
					d = MinRefresh
				} else if d > MaxRefresh {
					d = MaxRefresh
				}
				cfg.Refresh = d
			}
		case keyTempUnit:
			if val == TempCelsius || val == TempFahrenheit {
				cfg.TempUnit = val
			} else {
				l.logger.Warn("config: invalid temp_unit value, using default", "value", val, "default", DefaultTempUnit)
				cfg.TempUnit = DefaultTempUnit
			}
		}
	}

	if cfg.Screen != l.cached.Screen {
		l.logger.Info("config: screen changed", "from", l.cached.Screen, "to", cfg.Screen)
	}
	if cfg.Refresh != l.cached.Refresh {
		l.logger.Info("config: refresh changed", "from", l.cached.Refresh, "to", cfg.Refresh)
	}
	if cfg.TempUnit != l.cached.TempUnit {
		l.logger.Info("config: temp_unit changed", "from", l.cached.TempUnit, "to", cfg.TempUnit)
	}

	l.cached = cfg
	l.lastMtime = mtime
	return cfg
}
