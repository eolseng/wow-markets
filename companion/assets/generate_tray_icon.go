//go:build ignore

// Generates assets/tray.ico, the Windows tray icon: a white rendition of the
// bar-chart status icon drawn natively for macOS in status_item_darwin.go.
// The ICO uses uncompressed 32-bit DIB entries because the systray library
// loads it with LoadImageW, which cannot read PNG-compressed entries.
//
// Run from companion/: go run assets/generate_tray_icon.go
package main

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"log"
	"math"
	"os"

	"image"
	"image/color"
)

// Design space matches the 18x18 y-up canvas in status_item_darwin.go.
const canvas = 18.0

type roundedRect struct {
	x, y, w, h, radius float64
}

// sdf returns the signed distance from point (u, v) to the rectangle edge.
func (r roundedRect) sdf(u, v float64) float64 {
	cx, cy := r.x+r.w/2, r.y+r.h/2
	qx := math.Abs(u-cx) - (r.w/2 - r.radius)
	qy := math.Abs(v-cy) - (r.h/2 - r.radius)
	ax, ay := math.Max(qx, 0), math.Max(qy, 0)
	return math.Hypot(ax, ay) + math.Min(math.Max(qx, qy), 0) - r.radius
}

var (
	frame       = roundedRect{2.5, 2.5, 13, 13, 3}
	frameStroke = 1.6
	bars        = []roundedRect{
		{5, 5, 2, 5, 1},
		{8, 5, 2, 8, 1},
		{11, 5, 2, 6.5, 1},
	}
)

func covered(u, v float64) bool {
	if math.Abs(frame.sdf(u, v)) <= frameStroke/2 {
		return true
	}
	for _, bar := range bars {
		if bar.sdf(u, v) <= 0 {
			return true
		}
	}
	return false
}

// render rasterizes the design at size px with 8x8 supersampling. The design
// space is y-up; rasters are y-down, so v is flipped.
func render(size int) *image.NRGBA {
	const sub = 8
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := canvas / float64(size)
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			hits := 0
			for sy := 0; sy < sub; sy++ {
				for sx := 0; sx < sub; sx++ {
					u := (float64(px) + (float64(sx)+0.5)/sub) * scale
					v := canvas - (float64(py)+(float64(sy)+0.5)/sub)*scale
					if covered(u, v) {
						hits++
					}
				}
			}
			alpha := uint8(hits * 255 / (sub * sub))
			img.SetNRGBA(px, py, color.NRGBA{R: 255, G: 255, B: 255, A: alpha})
		}
	}
	return img
}

// dibEntry encodes an image as a 32-bit BGRA DIB with an AND mask, the classic
// ICO payload LoadImageW understands.
func dibEntry(img *image.NRGBA) []byte {
	size := img.Bounds().Dx()
	var buf bytes.Buffer
	write := func(v any) { _ = binary.Write(&buf, binary.LittleEndian, v) }

	write(uint32(40))            // biSize
	write(int32(size))           // biWidth
	write(int32(size * 2))       // biHeight: XOR + AND
	write(uint16(1))             // biPlanes
	write(uint16(32))            // biBitCount
	write(uint32(0))             // biCompression
	write(uint32(0))             // biSizeImage
	write([4]uint32{0, 0, 0, 0}) // resolution, colors

	for y := size - 1; y >= 0; y-- { // bottom-up BGRA
		for x := 0; x < size; x++ {
			c := img.NRGBAAt(x, y)
			buf.Write([]byte{c.B, c.G, c.R, c.A})
		}
	}

	maskStride := ((size + 31) / 32) * 4
	for y := size - 1; y >= 0; y-- { // bottom-up 1bpp AND mask
		row := make([]byte, maskStride)
		for x := 0; x < size; x++ {
			if img.NRGBAAt(x, y).A == 0 {
				row[x/8] |= 0x80 >> (x % 8)
			}
		}
		buf.Write(row)
	}
	return buf.Bytes()
}

func main() {
	sizes := []int{16, 20, 24, 32, 48, 64}
	entries := make([][]byte, len(sizes))
	for i, size := range sizes {
		entries[i] = dibEntry(render(size))
	}

	var ico bytes.Buffer
	write := func(v any) { _ = binary.Write(&ico, binary.LittleEndian, v) }
	write(uint16(0)) // reserved
	write(uint16(1)) // type: icon
	write(uint16(len(sizes)))

	offset := 6 + 16*len(sizes)
	for i, size := range sizes {
		write(uint8(size)) // 0 would mean 256
		write(uint8(size))
		write(uint8(0))  // palette
		write(uint8(0))  // reserved
		write(uint16(1)) // planes
		write(uint16(32))
		write(uint32(len(entries[i])))
		write(uint32(offset))
		offset += len(entries[i])
	}
	for _, entry := range entries {
		ico.Write(entry)
	}

	if err := os.WriteFile("assets/tray.ico", ico.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}

	if preview := os.Getenv("TRAY_ICON_PREVIEW"); preview != "" {
		var buf bytes.Buffer
		if err := png.Encode(&buf, render(64)); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(preview, buf.Bytes(), 0o644); err != nil {
			log.Fatal(err)
		}
	}
}
