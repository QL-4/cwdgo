// screenshot captures a window (matched by class name or title substring,
// or the foreground window) to a PNG file. It is a manual-verification
// helper for the launcher panel: usage:
//
//	screenshot -out panel.png [-class wailsWindow] [-title cwdgo] [-foreground]
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"cwdgo/internal/win32"
)

var (
	out        = flag.String("out", "shot.png", "output PNG file")
	class      = flag.String("class", "", "window class name to match")
	title      = flag.String("title", "", "window title substring to match")
	foreground = flag.Bool("foreground", false, "capture the foreground window")
	state      = flag.Bool("state", false, "print window state instead of capturing")
)

func main() {
	flag.Parse()

	var hwnd windows.HWND
	switch {
	case *foreground:
		hwnd = windows.GetForegroundWindow()
	case *class != "" || *title != "":
		hwnd = findWindow()
	default:
		fmt.Fprintln(os.Stderr, "specify -class, -title or -foreground")
		os.Exit(2)
	}
	if hwnd == 0 {
		fmt.Fprintln(os.Stderr, "no matching window")
		os.Exit(1)
	}

	if *state {
		var rect windows.Rect
		win32.GetWindowRect(hwnd, &rect)
		pt, _ := win32.GetCursorPos()
		visible := windows.IsWindowVisible(hwnd)
		fg := windows.GetForegroundWindow()
		fmt.Printf("hwnd=0x%X visible=%v rect=(%d,%d)-(%d,%d) fgIsPanel=%v mouse=(%d,%d)\n",
			hwnd, visible, rect.Left, rect.Top, rect.Right, rect.Bottom,
			fg == hwnd, pt.X, pt.Y)
		return
	}

	var rect windows.Rect
	if err := win32.GetWindowRect(hwnd, &rect); err != nil {
		fmt.Fprintln(os.Stderr, "GetWindowRect:", err)
		os.Exit(1)
	}
	w, h := int(rect.Right-rect.Left), int(rect.Bottom-rect.Top)
	if w <= 0 || h <= 0 {
		fmt.Fprintln(os.Stderr, "window has zero size")
		os.Exit(1)
	}

	dc, err := win32.GetDC(0) // screen DC
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetDC:", err)
		os.Exit(1)
	}
	defer win32.ReleaseDC(0, dc)

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// BitBlt the window region from the screen into the bitmap.
	hdcMem := createCompatibleDC(dc)
	defer deleteDC(hdcMem)
	hbm := createCompatibleBitmap(dc, w, h)
	defer deleteObject(hbm)
	selectObject(hdcMem, hbm)
	bitBlt(hdcMem, 0, 0, w, h, dc, int(rect.Left), int(rect.Top), 0x00CC0020) // SRCCOPY

	// Read the bitmap bits back into the Go image (bottom-up BGRA).
	info := BITMAPINFO{
		Header: BITMAPINFOHEADER{
			Size:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			Width:       int32(w),
			Height:      -int32(h), // negative: top-down rows
			Planes:      1,
			BitCount:    32,
			Compression: 0,
		},
	}
	var size uint32
	if err := getDIBits(hdcMem, hbm, 0, uint32(h), nil, &info, 0); err != nil {
		fmt.Fprintln(os.Stderr, "GetDIBits size:", err)
		os.Exit(1)
	}
	size = info.Header.SizeImage
	if size == 0 {
		size = uint32(w * h * 4)
	}
	buf := make([]byte, size)
	if err := getDIBits(hdcMem, hbm, 0, uint32(h), &buf[0], &info, 0); err != nil {
		fmt.Fprintln(os.Stderr, "GetDIBits:", err)
		os.Exit(1)
	}
	for y := 0; y < h; y++ {
		row := buf[y*4*w : (y+1)*4*w]
		for x := 0; x < w; x++ {
			b, g, r, a := row[4*x], row[4*x+1], row[4*x+2], row[4*x+3]
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
	fmt.Printf("saved %s (%dx%d)\n", *out, w, h)
}

// findWindow scans top-level windows for a class/title match.
func findWindow() windows.HWND {
	var found windows.HWND
	cb := syscall.NewCallback(func(hwnd windows.HWND, lparam uintptr) uintptr {
		if *class != "" {
			var buf [256]uint16
			if n, err := windows.GetClassName(hwnd, &buf[0], int32(len(buf))); err == nil && n > 0 {
				if windows.UTF16ToString(buf[:n]) == *class {
					found = hwnd
					return 0 // stop
				}
			}
		}
		if *title != "" {
			if t := win32.GetWindowText(hwnd); t != "" && strings.Contains(t, *title) {
				found = hwnd
				return 0
			}
		}
		return 1
	})
	windows.EnumWindows(cb, nil)
	return found
}

type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type BITMAPINFO struct {
	Header BITMAPINFOHEADER
	Colors [1]uint32
}

var (
	gdi32                      = windows.NewLazySystemDLL("gdi32.dll")
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procGetDIBits              = gdi32.NewProc("GetDIBits")
)

// GDI handle types (aliases of windows.Handle; x/sys does not export
// HDC/HBITMAP/HGDIOBJ).
type HDC = windows.Handle
type HBITMAP = windows.Handle
type HGDIOBJ = windows.Handle

func createCompatibleDC(dc HDC) HDC {
	r, _, _ := procCreateCompatibleDC.Call(uintptr(dc))
	return HDC(r)
}

func createCompatibleBitmap(dc HDC, w, h int) HBITMAP {
	r, _, _ := procCreateCompatibleBitmap.Call(uintptr(dc), uintptr(w), uintptr(h))
	return HBITMAP(r)
}

func selectObject(dc HDC, obj HGDIOBJ) HGDIOBJ {
	r, _, _ := procSelectObject.Call(uintptr(dc), uintptr(obj))
	return HGDIOBJ(r)
}

func bitBlt(dst HDC, x, y, w, h int, src HDC, sx, sy int, rop uint32) {
	procBitBlt.Call(uintptr(dst), uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(src), uintptr(sx), uintptr(sy), uintptr(rop))
}

func deleteDC(dc HDC) { procDeleteDC.Call(uintptr(dc)) }
func deleteObject(obj HGDIOBJ) {
	procDeleteObject.Call(uintptr(obj))
}

func getDIBits(dc HDC, bmp HBITMAP, start, lines uint32,
	buf *byte, info *BITMAPINFO, usage uint32) error {
	r0, _, e1 := procGetDIBits.Call(
		uintptr(dc), uintptr(bmp), uintptr(start), uintptr(lines),
		uintptr(unsafe.Pointer(buf)), uintptr(unsafe.Pointer(info)), uintptr(usage))
	if r0 == 0 {
		return e1
	}
	return nil
}
