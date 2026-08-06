// Package icon draws the cwdgo app icon: a dark rounded tile with a folder
// glyph. The same drawing code feeds the PNG app icon (used by `wails
// build`) and the tray ICO bytes, so both always stay in sync. Pure Go, no
// assets required.
package icon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Tile is the dark rounded background of the icon.
var Tile = color.RGBA{R: 0x1E, G: 0x22, B: 0x2E, A: 0xFF}

// FolderBack is the rear flap of the folder glyph.
var FolderBack = color.RGBA{R: 0xE2, G: 0x9E, B: 0x33, A: 0xFF}

// FolderFront is the front panel of the folder glyph.
var FolderFront = color.RGBA{R: 0xF4, G: 0xC2, B: 0x5C, A: 0xFF}

// Draw renders the app icon at the given size in pixels.
func Draw(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	f := float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			if inRoundedRect(px, py, 0, 0, f, f, 0.16*f) {
				img.SetRGBA(x, y, Tile)
			}
			if inRoundedRect(px, py, 0.10*f, 0.26*f, 0.90*f, 0.86*f, 0.05*f) {
				img.SetRGBA(x, y, FolderBack)
			}
			if inRoundedRect(px, py, 0.18*f, 0.38*f, 0.90*f, 0.86*f, 0.05*f) {
				img.SetRGBA(x, y, FolderFront)
			}
		}
	}
	return img
}

// inRoundedRect reports whether (x, y) lies inside the rectangle with
// rounded corners of the given radius.
func inRoundedRect(x, y, x0, y0, x1, y1, r float64) bool {
	if x < x0 || x > x1 || y < y0 || y > y1 {
		return false
	}
	// Distance to the nearest corner center; inside if within the radius.
	cx := math.Min(math.Max(x, x0+r), x1-r)
	cy := math.Min(math.Max(y, y0+r), y1-r)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}

// TrayICO returns a multi-resolution ICO file (16 and 32 px, 32bpp) for the
// system tray icon.
func TrayICO() []byte {
	return ICO(16, 32)
}

// RawImageData returns the raw RT_ICON resource data for one size: a
// BITMAPINFOHEADER + bottom-up BGRA pixels + AND mask. This is exactly what
// Win32 CreateIconFromResource expects (NOT the ICO file wrapper).
func RawImageData(size int) []byte {
	return icoImage(size)
}

// ICO encodes the given sizes of Draw as a 32bpp BMP-in-ICO container.
func ICO(sizes ...int) []byte {
	var buf bytes.Buffer
	// ICONDIR
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // type: icon
	binary.Write(&buf, binary.LittleEndian, uint16(len(sizes)))

	type entry struct {
		offset uint32
		data   []byte
	}
	entries := make([]entry, 0, len(sizes))
	offset := uint32(6 + 16*len(sizes))
	for _, s := range sizes {
		data := icoImage(s)
		entries = append(entries, entry{offset, data})
		offset += uint32(len(data))
	}
	for i, s := range sizes {
		e := entries[i]
		if s >= 256 {
			binary.Write(&buf, binary.LittleEndian, byte(0))
		} else {
			binary.Write(&buf, binary.LittleEndian, byte(s))
		}
		binary.Write(&buf, binary.LittleEndian, byte(s))    // height
		binary.Write(&buf, binary.LittleEndian, byte(0))    // palette
		binary.Write(&buf, binary.LittleEndian, byte(0))    // reserved
		binary.Write(&buf, binary.LittleEndian, uint16(1))  // planes
		binary.Write(&buf, binary.LittleEndian, uint16(32)) // bpp
		binary.Write(&buf, binary.LittleEndian, uint32(len(e.data)))
		binary.Write(&buf, binary.LittleEndian, e.offset)
	}
	for _, e := range entries {
		buf.Write(e.data)
	}
	return buf.Bytes()
}

// icoImage encodes one RGBA image as BITMAPINFOHEADER + bottom-up BGRA
// rows + an empty 1bpp AND mask, the classic 32bpp icon layout.
func icoImage(size int) []byte {
	img := Draw(size)
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(40)) // biSize
	binary.Write(&buf, binary.LittleEndian, int32(size))
	binary.Write(&buf, binary.LittleEndian, int32(size*2)) // XOR + AND
	binary.Write(&buf, binary.LittleEndian, uint16(1))     // planes
	binary.Write(&buf, binary.LittleEndian, uint16(32))    // bitcount
	binary.Write(&buf, binary.LittleEndian, uint32(0))     // BI_RGB
	binary.Write(&buf, binary.LittleEndian, uint32(0))     // size image (0 ok for BI_RGB)
	binary.Write(&buf, binary.LittleEndian, int32(0))      // x ppm
	binary.Write(&buf, binary.LittleEndian, int32(0))      // y ppm
	binary.Write(&buf, binary.LittleEndian, uint32(0))     // colors used
	binary.Write(&buf, binary.LittleEndian, uint32(0))     // important colors
	// XOR bitmap: bottom-up BGRA rows.
	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			c := img.RGBAAt(x, y)
			buf.WriteByte(c.B)
			buf.WriteByte(c.G)
			buf.WriteByte(c.R)
			buf.WriteByte(c.A)
		}
	}
	// AND mask: 1bpp, rows padded to 32-bit, all opaque (0 = show pixels).
	rowBytes := (size + 31) / 32 * 4
	mask := make([]byte, rowBytes*size)
	buf.Write(mask)
	return buf.Bytes()
}

// PNG renders the icon as a PNG file (used as the app icon for `wails
// build`).
func PNG(size int) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, Draw(size)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
