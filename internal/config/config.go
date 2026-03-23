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
	DefaultPath     = "/etc/uctronics-display.conf"
	DefaultScreen   = "dashboard"
	DefaultRefresh  = 5 * time.Second
	DefaultTempUnit = "C"

	MinRefresh = 2 * time.Second
	MaxRefresh = 30 * time.Second
)

var validScreens = map[string]bool{
	"dashboard":  true,
	"diagnostic": true,
	"sparkline":  true,
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
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		switch key {
		case "screen":
			if validScreens[val] {
				cfg.Screen = val
			} else {
				l.logger.Warn("config: invalid screen value, using default", "value", val, "default", DefaultScreen)
				cfg.Screen = DefaultScreen
			}
		case "refresh":
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
		case "temp_unit":
			if val == "C" || val == "F" {
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
