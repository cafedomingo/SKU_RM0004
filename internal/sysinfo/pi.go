package sysinfo

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
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

	const base = "/sys/devices/system/cpu/cpu0/cpufreq/"
	return CPUFreq{
		Cur: read(base + "scaling_cur_freq"),
		Min: read(base + "cpuinfo_min_freq"),
		Max: read(base + "cpuinfo_max_freq"),
	}
}

// readThrottleStatus reads CPU throttle status via the VideoCore mailbox.
func readThrottleStatus() uint32 {
	f, err := os.OpenFile("/dev/vcio", os.O_RDWR, 0)
	if err != nil {
		return 0
	}
	defer f.Close()

	// Mailbox property buffer for GET_THROTTLED (tag 0x00030046).
	// Must be 16-byte aligned; [8]uint32 is 32 bytes, naturally aligned.
	buf := [8]uint32{
		32,         // buffer size
		0,          // request code
		0x00030046, // GET_THROTTLED tag
		4,          // value buffer size
		0,          // request/response indicator
		0,          // value (filled by firmware)
		0,          // end tag
		0,
	}

	// _IOWR(100, 0, *byte) on ARM64 = 0xC0046400
	const ioctlMailbox = 0xC0046400

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		f.Fd(),
		ioctlMailbox,
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if errno != 0 {
		return 0
	}
	if buf[1] != 0x80000000 {
		return 0
	}
	return buf[5]
}

// readDietPiStatus checks DietPi installation and update status.
func readDietPiStatus() DietPiStatus {
	if _, err := os.Stat("/run/dietpi"); os.IsNotExist(err) {
		return DietPiNotInstalled
	}
	if _, err := os.Stat("/run/dietpi/.update_available"); err == nil {
		return DietPiUpdateAvail
	}
	return DietPiUpToDate
}

// readAPTUpdateCount reads the APT update count from DietPi cache files.
func readAPTUpdateCount() int {
	data, err := os.ReadFile("/run/dietpi/.apt_updates")
	if err == nil {
		var count int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &count); err == nil {
			return count
		}
		return 0
	}

	// No .apt_updates file; check if this is DietPi at all.
	if _, err := os.Stat("/boot/dietpi/.version"); err == nil {
		return 0
	}
	return -1
}
