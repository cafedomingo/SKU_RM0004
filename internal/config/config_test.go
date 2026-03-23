package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"log/slog"
)

func newTestLoader(t *testing.T, path string) *Loader {
	t.Helper()
	return NewLoader(path, slog.Default())
}

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "test.conf")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeConfig: %v", err)
	}
	return path
}

func TestDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.conf")
	l := newTestLoader(t, path)
	cfg := l.Load()
	if cfg.Screen != DefaultScreen {
		t.Errorf("Screen = %q, want %q", cfg.Screen, DefaultScreen)
	}
	if cfg.Refresh != DefaultRefresh {
		t.Errorf("Refresh = %v, want %v", cfg.Refresh, DefaultRefresh)
	}
	if cfg.TempUnit != DefaultTempUnit {
		t.Errorf("TempUnit = %q, want %q", cfg.TempUnit, DefaultTempUnit)
	}
}

func TestFullConfig(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, "screen=sparkline\nrefresh=10\ntemp_unit=F\n")
	l := newTestLoader(t, path)
	cfg := l.Load()
	if cfg.Screen != "sparkline" {
		t.Errorf("Screen = %q, want %q", cfg.Screen, "sparkline")
	}
	if cfg.Refresh != 10*time.Second {
		t.Errorf("Refresh = %v, want %v", cfg.Refresh, 10*time.Second)
	}
	if cfg.TempUnit != "F" {
		t.Errorf("TempUnit = %q, want %q", cfg.TempUnit, "F")
	}
}

func TestPartialConfig(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, "screen=sparkline\n")
	l := newTestLoader(t, path)
	cfg := l.Load()
	if cfg.Screen != "sparkline" {
		t.Errorf("Screen = %q, want %q", cfg.Screen, "sparkline")
	}
	if cfg.Refresh != DefaultRefresh {
		t.Errorf("Refresh = %v, want %v", cfg.Refresh, DefaultRefresh)
	}
	if cfg.TempUnit != DefaultTempUnit {
		t.Errorf("TempUnit = %q, want %q", cfg.TempUnit, DefaultTempUnit)
	}
}

func TestInvalidScreen(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, "screen=invalid\n")
	l := newTestLoader(t, path)
	cfg := l.Load()
	if cfg.Screen != DefaultScreen {
		t.Errorf("Screen = %q, want %q", cfg.Screen, DefaultScreen)
	}
}

func TestRefreshBounds(t *testing.T) {
	dir := t.TempDir()

	// Below min
	path := writeConfig(t, dir, "refresh=1\n")
	l := newTestLoader(t, path)
	cfg := l.Load()
	if cfg.Refresh != MinRefresh {
		t.Errorf("refresh=1: Refresh = %v, want %v", cfg.Refresh, MinRefresh)
	}

	// Above max
	if err := os.WriteFile(path, []byte("refresh=50\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// Force mtime change by touching with a future time
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	cfg = l.Load()
	if cfg.Refresh != MaxRefresh {
		t.Errorf("refresh=50: Refresh = %v, want %v", cfg.Refresh, MaxRefresh)
	}
}

func TestComments(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, "# this is a comment\nscreen=diagnostic\n# another comment\n")
	l := newTestLoader(t, path)
	cfg := l.Load()
	if cfg.Screen != "diagnostic" {
		t.Errorf("Screen = %q, want %q", cfg.Screen, "diagnostic")
	}
	if cfg.Refresh != DefaultRefresh {
		t.Errorf("Refresh = %v, want %v", cfg.Refresh, DefaultRefresh)
	}
}

func TestEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, "")
	l := newTestLoader(t, path)
	cfg := l.Load()
	if cfg.Screen != DefaultScreen {
		t.Errorf("Screen = %q, want %q", cfg.Screen, DefaultScreen)
	}
	if cfg.Refresh != DefaultRefresh {
		t.Errorf("Refresh = %v, want %v", cfg.Refresh, DefaultRefresh)
	}
	if cfg.TempUnit != DefaultTempUnit {
		t.Errorf("TempUnit = %q, want %q", cfg.TempUnit, DefaultTempUnit)
	}
}

func TestTempUnit(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, "temp_unit=F\n")
	l := newTestLoader(t, path)
	cfg := l.Load()
	if cfg.TempUnit != "F" {
		t.Errorf("TempUnit = %q, want %q", cfg.TempUnit, "F")
	}
}

func TestMtimeCaching(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, "screen=diagnostic\nrefresh=8\ntemp_unit=F\n")
	l := newTestLoader(t, path)

	cfg1 := l.Load()
	cfg2 := l.Load()

	if cfg1.Screen != cfg2.Screen {
		t.Errorf("Screen changed between loads: %q vs %q", cfg1.Screen, cfg2.Screen)
	}
	if cfg1.Refresh != cfg2.Refresh {
		t.Errorf("Refresh changed between loads: %v vs %v", cfg1.Refresh, cfg2.Refresh)
	}
	if cfg1.TempUnit != cfg2.TempUnit {
		t.Errorf("TempUnit changed between loads: %q vs %q", cfg1.TempUnit, cfg2.TempUnit)
	}
	// Verify values are what we set
	if cfg2.Screen != "diagnostic" {
		t.Errorf("Screen = %q, want %q", cfg2.Screen, "diagnostic")
	}
}

func TestMtimeReload(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, "screen=dashboard\n")
	l := newTestLoader(t, path)

	cfg1 := l.Load()
	if cfg1.Screen != "dashboard" {
		t.Errorf("first load Screen = %q, want %q", cfg1.Screen, "dashboard")
	}

	// Write new content and advance mtime
	if err := os.WriteFile(path, []byte("screen=sparkline\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	cfg2 := l.Load()
	if cfg2.Screen != "sparkline" {
		t.Errorf("second load Screen = %q, want %q", cfg2.Screen, "sparkline")
	}
}
