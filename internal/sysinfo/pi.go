package sysinfo

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	cpuFreqBase      = "/sys/devices/system/cpu/cpu0/cpufreq/"
	vcioPath         = "/dev/vcio"
	tagGetThrottled  = 0x00030046
	ioctlMailbox     = 0xC0046400
	mailboxSuccess   = 0x80000000
	dietpiRunDir     = "/run/dietpi"
	dietpiUpdateFlag = "/run/dietpi/.update_available"
	dietpiAPTCache   = "/run/dietpi/.apt_updates"
	dietpiVersionFile = "/boot/dietpi/.version"
)

// readCPUFreq reads current/min/max CPU frequency from sysfs.
func readCPUFreq() CPUFreq {
	read := func(path string) uint16 {
		data, err := os.ReadFile(path)
		if err != nil {
			return 0
		}
		var kHz int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &kHz); err != nil {
			return 0
		}
		return uint16(kHz / 1000)
	}

	const base = cpuFreqBase
	return CPUFreq{
		Cur: read(base + "scaling_cur_freq"),
		Min: read(base + "cpuinfo_min_freq"),
		Max: read(base + "cpuinfo_max_freq"),
	}
}

// readThrottleStatus reads CPU throttle status via the VideoCore mailbox.
func readThrottleStatus() uint32 {
	f, err := os.OpenFile(vcioPath, os.O_RDWR, 0)
	if err != nil {
		return 0
	}
	defer f.Close()

	// Mailbox property buffer for GET_THROTTLED (tag 0x00030046).
	// Must be 16-byte aligned; [8]uint32 is 32 bytes, naturally aligned.
	buf := [8]uint32{
		32,         // buffer size
		0,          // request code
		tagGetThrottled, // GET_THROTTLED
		4,          // value buffer size
		0,          // request/response indicator
		0,          // value (filled by firmware)
		0,          // end tag
		0,
	}

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
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

// readDietPiStatus checks DietPi installation and update status.
func readDietPiStatus() DietPiStatus {
	if _, err := os.Stat(dietpiRunDir); os.IsNotExist(err) {
		return DietPiNotInstalled
	}
	if _, err := os.Stat(dietpiUpdateFlag); err == nil {
		return DietPiUpdateAvail
	}
	return DietPiUpToDate
}

// readAPTUpdateCount reads the APT update count from DietPi cache files.
func readAPTUpdateCount() int {
	data, err := os.ReadFile(dietpiAPTCache)
	if err == nil {
		var count int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &count); err == nil {
			return count
		}
		return 0
	}

	// No .apt_updates file; check if this is DietPi at all.
	if _, err := os.Stat(dietpiVersionFile); err == nil {
		return 0
	}
	return -1
}
