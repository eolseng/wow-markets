//go:build ignore

// Generates the WoW addon-list icon from the companion's three-bar identity.
// Run from the repository root with: go run scripts/generate-addon-icon.go
package main

import (
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

const (
	iconSize = 64
	samples  = 4
)

var (
	gold = color.NRGBA{R: 0xF2, G: 0xBD, B: 0x55, A: 0xFF}
	navy = color.NRGBA{R: 0x0D, G: 0x16, B: 0x26, A: 0xFF}
)

func insideRoundedRect(x, y, left, top, right, bottom, radius float64) bool {
	cx := math.Max(left+radius, math.Min(x, right-radius))
	cy := math.Max(top+radius, math.Min(y, bottom-radius))
	return math.Hypot(x-cx, y-cy) <= radius
}

func sample(x, y float64) color.NRGBA {
	if math.Hypot(x-32, y-32) > 28 {
		return color.NRGBA{}
	}

	for _, bar := range [][4]float64{
		{19, 31, 25, 45},
		{29, 20, 35, 45},
		{39, 27, 45, 45},
	} {
		if insideRoundedRect(x, y, bar[0], bar[1], bar[2], bar[3], 3) {
			return navy
		}
	}

	return gold
}

func render() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, iconSize, iconSize))
	for py := 0; py < iconSize; py++ {
		for px := 0; px < iconSize; px++ {
			var red, green, blue, alpha int
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					x := float64(px) + (float64(sx)+0.5)/samples
					y := float64(py) + (float64(sy)+0.5)/samples
					pixel := sample(x, y)
					red += int(pixel.R)
					green += int(pixel.G)
					blue += int(pixel.B)
					alpha += int(pixel.A)
				}
			}
			count := samples * samples
			img.SetNRGBA(px, py, color.NRGBA{
				R: uint8(red / count), G: uint8(green / count),
				B: uint8(blue / count), A: uint8(alpha / count),
			})
		}
	}
	return img
}

func writeTGA(path string, img *image.NRGBA) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	header := make([]byte, 18)
	header[2] = 2 // uncompressed true-color image
	binary.LittleEndian.PutUint16(header[12:14], uint16(img.Bounds().Dx()))
	binary.LittleEndian.PutUint16(header[14:16], uint16(img.Bounds().Dy()))
	header[16] = 32
	header[17] = 0x28 // top-left origin with 8 alpha bits
	if _, err := file.Write(header); err != nil {
		return err
	}

	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			pixel := img.NRGBAAt(x, y)
			if _, err := file.Write([]byte{pixel.B, pixel.G, pixel.R, pixel.A}); err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	img := render()
	if err := writeTGA("addon/WoWMarkets/Icon.tga", img); err != nil {
		log.Fatal(err)
	}
	if preview := os.Getenv("ADDON_ICON_PREVIEW"); preview != "" {
		file, err := os.Create(preview)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		if err := png.Encode(file, img); err != nil {
			log.Fatal(err)
		}
	}
}
