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

const diagRowsPerPage = 6

type diagRow struct {
	label string
	value string
	color uint16
}

type diagnosticScreen struct {
	disp      st7735.Display
	collector sysinfo.Collector
	back      st7735.Framebuffer
	page      int
	rows      []diagRow
}

func (d *diagnosticScreen) Buffer() *st7735.Framebuffer { return &d.back }

func (d *diagnosticScreen) Update(cfg config.Config) {
	d.back.Fill(theme.ColorBG)

	f := font.Spleen6x12

	d.rows = collectDiagData(d.collector)

	// Determine which rows to render
	start := d.page * diagRowsPerPage
	end := start + diagRowsPerPage
	if end > len(d.rows) {
		end = len(d.rows)
	}

	// Render rows
	for i, row := range d.rows[start:end] {
		y := i * f.Height
		if row.label == "" {
			d.back.String(0, y, row.value, f, row.color)
		} else {
			d.back.String(0, y, row.label, f, theme.ColorMuted)
			vw := format.StringWidth(row.value, f)
			vx := st7735.Width - vw
			if vx < 0 {
				vx = 0
			}
			d.back.String(vx, y, row.value, f, row.color)
		}
	}

	// Advance page
	numPages := (len(d.rows) + diagRowsPerPage - 1) / diagRowsPerPage
	d.page = (d.page + 1) % numPages
}

func (d *diagnosticScreen) Draw() {
	drawAll(d.disp, &d.back)
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
		value: c.IPv4Address(),
		color: theme.ColorIdentity,
	})

	// Row 2: IPv6 suffix (header, no label)
	rows = append(rows, diagRow{
		value: c.IPv6Suffix(),
		color: theme.ColorIdentity,
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
		color: theme.CPUColor(c.CPUPercent()),
	})

	// Row 5: Temperature — both C and F
	tempC := c.Temperature()
	tempVal := fmt.Sprintf("%s / %s", format.Temp(tempC, false), format.Temp(tempC, true))
	rows = append(rows, diagRow{
		label: "tmp",
		value: tempVal,
		color: theme.TempColor(tempC),
	})

	// Row 6: RAM%
	rows = append(rows, diagRow{
		label: "ram",
		value: format.Pct(c.RAMPercent()),
		color: theme.RAMColor(c.RAMPercent()),
	})

	// Row 7: Throttle status
	throttle := c.ThrottleStatus()
	var throttleVal string
	var throttleColor uint16
	switch {
	case throttle&sysinfo.ThrottleCurrentMask != 0:
		throttleVal = "ACTIVE"
		throttleColor = theme.ColorCrit
	case throttle&sysinfo.ThrottlePastMask != 0:
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
		value: format.Pct(c.DiskPercent()),
		color: theme.DiskColor(c.DiskPercent()),
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
		aptColor = theme.APTColor(apt)
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
