package screen

import (
	"fmt"

	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/font"
	"github.com/cafedomingo/SKU_RM0004/internal/format"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
	"github.com/cafedomingo/SKU_RM0004/internal/theme"
)

const diagRowsPerPage = 5

// DiagState holds the current page and cached row data for the diagnostic screen.
type DiagState struct {
	Page int
	rows []diagRow
}

type diagRow struct {
	label string
	value string
	color uint16
}

// RenderDiagnostic draws the diagnostic detail screen onto the framebuffer.
// It cycles through 3 pages of 5 rows each on successive calls.
// Data is refreshed only when returning to page 0.
func RenderDiagnostic(fb *st7735.Framebuffer, c sysinfo.Collector, cfg config.Config, state *DiagState) {
	f := font.Spleen8x16

	// Refresh data only on page 0
	if state.Page == 0 {
		state.rows = collectDiagData(c)
	}

	// Determine which rows to render
	start := state.Page * diagRowsPerPage
	end := start + diagRowsPerPage
	if end > len(state.rows) {
		end = len(state.rows)
	}

	// Render rows
	for i, row := range state.rows[start:end] {
		y := i * f.Height
		if row.label == "" {
			// Header row: value left-aligned
			fb.String(0, y, row.value, f, row.color)
		} else {
			// Label left-aligned in muted
			fb.String(0, y, row.label, f, theme.ColorMuted)
			// Value right-aligned
			vw := len([]rune(row.value)) * f.Width
			vx := st7735.Width - vw
			if vx < 0 {
				vx = 0
			}
			fb.String(vx, y, row.value, f, row.color)
		}
	}

	// Advance page
	numPages := (len(state.rows) + diagRowsPerPage - 1) / diagRowsPerPage
	state.Page = (state.Page + 1) % numPages
}

func collectDiagData(c sysinfo.Collector) []diagRow {
	rows := make([]diagRow, 0, 15)

	// Row 0: hostname (header, no label)
	rows = append(rows, diagRow{
		value: c.Hostname(),
		color: theme.ColorFG,
	})

	// Row 1: IPv4 (header, no label)
	rows = append(rows, diagRow{
		value: c.IPAddress(),
		color: theme.ColorIP,
	})

	// Row 2: IPv6 suffix (header, no label)
	rows = append(rows, diagRow{
		value: c.IPv6Suffix(),
		color: theme.ColorIP,
	})

	// Row 3: Uptime
	rows = append(rows, diagRow{
		label: "up",
		value: format.Uptime(c.Uptime()),
		color: theme.ColorFG,
	})

	// Row 4: CPU% + freq
	freq := c.CPUFreq()
	cpuVal := fmt.Sprintf("%d%% %s", int(c.CPUPercent()), format.Freq(freq.Cur))
	rows = append(rows, diagRow{
		label: "cpu",
		value: cpuVal,
		color: theme.ThresholdColor(c.CPUPercent(), theme.CPUWarn, theme.CPUCrit),
	})

	// Row 5: Temperature — always both °C and °F
	tempC := c.Temperature()
	tempVal := fmt.Sprintf("%s / %s", format.Temp(tempC, "C"), format.Temp(tempC, "F"))
	rows = append(rows, diagRow{
		label: "tmp",
		value: tempVal,
		color: theme.TempRampColor(tempC),
	})

	// Row 6: RAM%
	rows = append(rows, diagRow{
		label: "ram",
		value: fmt.Sprintf("%d%%", int(c.RAMPercent())),
		color: theme.ThresholdColor(c.RAMPercent(), theme.RAMWarn, theme.RAMCrit),
	})

	// Row 7: Throttle status
	throttle := c.ThrottleStatus()
	const (
		throttleCurrent = uint32(0x0000000F)
		throttlePast    = uint32(0x000F0000)
	)
	var throttleVal string
	var throttleColor uint16
	switch {
	case throttle&throttleCurrent != 0:
		throttleVal = "ACTIVE"
		throttleColor = theme.ColorCrit
	case throttle&throttlePast != 0:
		throttleVal = "past"
		throttleColor = theme.ColorWarn
	default:
		throttleVal = "OK"
		throttleColor = theme.ColorOK
	}
	rows = append(rows, diagRow{
		label: "thr",
		value: throttleVal,
		color: throttleColor,
	})

	// Row 8: Disk%
	rows = append(rows, diagRow{
		label: "dsk",
		value: fmt.Sprintf("%d%%", int(c.DiskPercent())),
		color: theme.ThresholdColor(c.DiskPercent(), theme.DiskWarn, theme.DiskCrit),
	})

	// Row 9: Net RX rate
	net := c.NetBandwidth()
	rows = append(rows, diagRow{
		label: "rx",
		value: format.Rate(net.RxBytesPerSec),
		color: theme.ColorFG,
	})

	// Row 10: Net TX rate
	rows = append(rows, diagRow{
		label: "tx",
		value: format.Rate(net.TxBytesPerSec),
		color: theme.ColorFG,
	})

	// Row 11: IO R/W bytes
	dio := c.DiskIO()
	rows = append(rows, diagRow{
		label: "io",
		value: fmt.Sprintf("%s/%s", format.Rate(dio.ReadBytesPerSec), format.Rate(dio.WriteBytesPerSec)),
		color: theme.ColorFG,
	})

	// Row 12: IOPS R/W
	rows = append(rows, diagRow{
		label: "iops",
		value: fmt.Sprintf("%d/%d", dio.ReadIOPS, dio.WriteIOPS),
		color: theme.ColorFG,
	})

	// Row 13: DietPi status
	var dietpiVal string
	var dietpiColor uint16
	switch c.DietPiStatus() {
	case sysinfo.DietPiUpdateAvail:
		dietpiVal = "update!"
		dietpiColor = theme.ColorAlert
	case sysinfo.DietPiUpToDate:
		dietpiVal = "OK"
		dietpiColor = theme.ColorOK
	default:
		dietpiVal = "N/A"
		dietpiColor = theme.ColorMuted
	}
	rows = append(rows, diagRow{
		label: "dpi",
		value: dietpiVal,
		color: dietpiColor,
	})

	// Row 14: APT update count
	apt := c.APTUpdateCount()
	var aptVal string
	var aptColor uint16
	switch {
	case apt > 0:
		aptVal = fmt.Sprintf("%d updates", apt)
		aptColor = theme.ColorWarn
	case apt == 0:
		aptVal = "up to date"
		aptColor = theme.ColorOK
	default:
		aptVal = "N/A"
		aptColor = theme.ColorMuted
	}
	rows = append(rows, diagRow{
		label: "apt",
		value: aptVal,
		color: aptColor,
	})

	return rows
}
