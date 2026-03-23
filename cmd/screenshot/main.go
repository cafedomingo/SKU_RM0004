package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"

	"github.com/cafedomingo/SKU_RM0004/internal/config"
	"github.com/cafedomingo/SKU_RM0004/internal/screen"
	"github.com/cafedomingo/SKU_RM0004/internal/st7735"
	"github.com/cafedomingo/SKU_RM0004/internal/sysinfo"
)

const scale = 5

func main() {
	mock := &sysinfo.MockCollector{
		CPU:  47, RAM: 63, Disk: 42, Temp: 52,
		Host: "raspberrypi",
		IPv4: "192.168.1.100",
		IPv6: "::a8f1:23bc:4567",
		Freq: sysinfo.CPUFreq{Cur: 1800, Min: 600, Max: 2400},
		Net:  sysinfo.NetBandwidth{RxBytesPerSec: 15360, TxBytesPerSec: 2048},
		DIO:  sysinfo.DiskIO{ReadBytesPerSec: 1048576, WriteBytesPerSec: 524288, ReadIOPS: 150, WriteIOPS: 80},
		Up:   3*24*time.Hour + 2*time.Hour,
		DietPi: sysinfo.DietPiUpdateAvail,
		APT:     3,
		LinkSpeed: 1000,
	}
	cfg := config.Config{Screen: "dashboard", Refresh: 5 * time.Second, TempUnit: "C"}

	fmt.Println("Rendering screenshots...")

	// Dashboard
	dash := screen.New(config.ScreenDashboard, nil, mock)
	dash.Update(cfg)
	writePNG("docs/dashboard.png", dash.Buffer())

	// Sparkline — render 13 frames to fill history with varying values
	spark := screen.New(config.ScreenSparkline, nil, mock)
	cpuSamples := []float64{22, 35, 28, 45, 52, 38, 61, 73, 55, 42, 68, 50, 47}
	ramSamples := []float64{40, 42, 45, 48, 50, 53, 55, 57, 58, 60, 61, 62, 63}
	for i := range cpuSamples {
		mock.CPU = cpuSamples[i]
		mock.RAM = ramSamples[i]
		spark.Update(cfg)
	}
	writePNG("docs/sparkline.png", spark.Buffer())

	fmt.Println("Done.")
}

func writePNG(path string, fb *st7735.Framebuffer) {
	w := st7735.Width * scale
	h := st7735.Height * scale

	palette := collectPalette(fb)
	img := image.NewPaletted(image.Rect(0, 0, w, h), palette)

	for y := 0; y < st7735.Height; y++ {
		for x := 0; x < st7735.Width; x++ {
			c := rgb565ToRGBA(fb.Pixels[y*st7735.Width+x])
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.Set(x*scale+dx, y*scale+dy, c)
				}
			}
		}
	}

	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating %s: %v\n", path, err)
		return
	}
	defer f.Close()

	encoder := &png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(f, img); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding %s: %v\n", path, err)
		return
	}
	fmt.Printf("  %s (%dx%d)\n", path, w, h)
}

func rgb565ToRGBA(c uint16) color.RGBA {
	r5 := (c >> 11) & 0x1F
	g6 := (c >> 5) & 0x3F
	b5 := c & 0x1F
	return color.RGBA{
		R: uint8(r5<<3 | r5>>2),
		G: uint8(g6<<2 | g6>>4),
		B: uint8(b5<<3 | b5>>2),
		A: 255,
	}
}

func collectPalette(fb *st7735.Framebuffer) color.Palette {
	seen := map[uint16]bool{}
	for _, px := range fb.Pixels {
		seen[px] = true
	}
	p := make(color.Palette, 0, len(seen))
	for c := range seen {
		p = append(p, rgb565ToRGBA(c))
	}
	return p
}
