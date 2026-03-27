package sysinfo

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	cpuFreqPath      = "/sys/devices/system/cpu/cpu0/cpufreq/"
	cpuFreqCurPath   = cpuFreqPath + "scaling_cur_freq"
	cpuFreqMinPath   = cpuFreqPath + "cpuinfo_min_freq"
	cpuFreqMaxPath   = cpuFreqPath + "cpuinfo_max_freq"
	vcioPath         = "/dev/vcio"
	tagGetThrottled  = 0x00030046
	ioctlMailbox     = 0xC0046400
	mailboxSuccess   = 0x80000000
	dietpiRunPath    = "/run/dietpi"
	dietpiUpdatePath = dietpiRunPath + "/.update_available"
	dietpiAPTPath    = dietpiRunPath + "/.apt_updates"
)

// readCPUFreq reads current/min/max CPU frequency from sysfs.
func readCPUFreq() CPUFreq {
	read := func(path string) uint16 {
		data, err := os.ReadFile(path)
		if err != nil {
			return 0
		}
		kHz, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return 0
		}
		return uint16(kHz / 1000)
	}

	return CPUFreq{
		Cur: read(cpuFreqCurPath),
		Min: read(cpuFreqMinPath),
		Max: read(cpuFreqMaxPath),
	}
}

// readThrottleStatus reads CPU throttle status via the VideoCore mailbox.
func readThrottleStatus() uint32 {
	f, err := os.OpenFile(vcioPath, os.O_RDWR, 0)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()

	// Mailbox property buffer for GET_THROTTLED (tag 0x00030046).
	// Must be 16-byte aligned; [8]uint32 is 32 bytes, naturally aligned.
	buf := [8]uint32{
		32,              // buffer size
		0,               // request code
		tagGetThrottled, // GET_THROTTLED
		4,               // value buffer size
		0,               // request/response indicator
		0,               // value (filled by firmware)
		0,               // end tag
		0,
	}

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		ioctlMailbox,
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if errno != 0 {
		return 0
	}
	if buf[1] != mailboxSuccess {
		return 0
	}
	return buf[5]
}

// isDietPi reports whether the system is running DietPi.
// Checked once and cached — can't change during program lifetime.
var isDietPi = sync.OnceValue(func() bool {
	_, err := os.Stat(dietpiRunPath)
	return err == nil
})

// readDietPiStatus checks DietPi installation and update status.
func readDietPiStatus() DietPiStatus {
	if !isDietPi() {
		return DietPiNotInstalled
	}
	if _, err := os.Stat(dietpiUpdatePath); err == nil {
		return DietPiUpdateAvail
	}
	return DietPiUpToDate
}

// readAPTUpdateCount reads the APT update count from DietPi cache files.
// Returns -1 if not running DietPi.
func readAPTUpdateCount() int {
	if !isDietPi() {
		return -1
	}
	data, err := os.ReadFile(dietpiAPTPath)
	if err != nil {
		return 0
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return count
}
