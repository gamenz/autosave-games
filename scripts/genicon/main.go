// Command genicon writes internal/tray/icon.ico: a 32x32 floppy-disk glyph
// stored as a PNG inside an ICO container. Run: go run ./scripts/genicon
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
)

const size = 32

func fill(img *image.RGBA, r image.Rectangle, c color.Color) {
	draw.Draw(img, r, &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func main() {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	body := color.RGBA{R: 0x1f, G: 0x6f, B: 0xd1, A: 0xff}
	fill(img, image.Rect(3, 3, 29, 29), body)
	fill(img, image.Rect(9, 3, 23, 12), color.RGBA{R: 0xd8, G: 0xe2, B: 0xee, A: 0xff})
	fill(img, image.Rect(18, 5, 21, 10), body)
	fill(img, image.Rect(7, 16, 25, 29), color.White)
	fill(img, image.Rect(10, 20, 22, 21), body)
	fill(img, image.Rect(10, 24, 22, 25), body)

	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		log.Fatal(err)
	}
	var ico bytes.Buffer
	header := []uint16{0, 1, 1} // reserved, type icon, image count
	binary.Write(&ico, binary.LittleEndian, header)
	ico.Write([]byte{size, size, 0, 0})                            // width, height, colors, reserved
	binary.Write(&ico, binary.LittleEndian, []uint16{1, 32})       // planes, bits per pixel
	binary.Write(&ico, binary.LittleEndian, uint32(pngData.Len())) // image size
	binary.Write(&ico, binary.LittleEndian, uint32(6+16))          // image offset
	ico.Write(pngData.Bytes())

	if err := os.WriteFile("internal/tray/icon.ico", ico.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
}
