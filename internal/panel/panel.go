// Package panel opens the Launcher Panel on the monitor under the mouse,
// brings it to the foreground and notifies the frontend so it can focus the
// search box.
//
// Click-outside-to-close is handled by subclassing the Wails window and
// reacting to WM_ACTIVATE/WM_ACTIVATEAPP deactivation. We deliberately do
// NOT use the webview's window.blur event: WebView2 bounces focus between
// the host window and its own Chromium helper windows, which fires spurious
// blur events and would close the panel the instant it opens.
package panel

import (
	"context"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"

	"cwdgo/internal/applog"
	"cwdgo/internal/win32"
)

// wailsWindowClass is the window class Wails v2 registers for its main
// window on Windows.
const wailsWindowClass = "wailsWindow"

// gwlpWndProc is the index for SetWindowLongPtrW: GWLP_WNDPROC = -4.
// uintptr has no negative literals, so -4 is written as ^uintptr(3).
const gwlpWndProc = ^uintptr(3)

// Win32 message/flag constants the subclass cares about.
const (
	WM_ACTIVATE    = 0x0006
	WM_ACTIVATEAPP = 0x001C
	waInactive     = 0
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procSetWindowLongPtr = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProc   = user32.NewProc("CallWindowProcW")
)

// Controller opens the launcher panel. It needs the Wails runtime context,
// wired up at startup via SetContext.
type Controller struct {
	ctx context.Context
	// suppressDeactivation is set briefly while we are activating the panel
	// ourselves, so our own WM_ACTIVATE does not immediately re-hide it.
	suppressDeactivation atomic.Bool
}

// New creates a Controller with no context yet.
func New() *Controller { return &Controller{} }

// SetContext wires the Wails runtime context (call from OnStartup) and
// installs the activation subclass that closes the panel on deactivation.
func (c *Controller) SetContext(ctx context.Context) {
	c.ctx = ctx
	c.installActivationHook()
}

// Open positions the panel centred on the monitor under the mouse cursor,
// shows it, brings it to the foreground and emits «panel-opened» so the
// frontend focuses the search box. Safe to call from any goroutine.
func (c *Controller) Open() {
	if c.ctx == nil {
		return
	}
	hwnd := findPanelWindow()
	if hwnd == 0 {
		return
	}

	// Allow our SetForegroundWindow: attach the panel window's thread to
	// the current foreground thread's input queue.
	attached := attachToForeground(hwnd)

	// Move before showing so there is no flicker at the old position.
	x, y := c.centeredOnMouseMonitor()
	runtime.WindowSetPosition(c.ctx, x, y)

	// Suppress deactivation around our own activation so the subclass does
	// not hide the window on the WA_ACTIVE we are about to cause.
	c.suppressDeactivation.Store(true)
	runtime.WindowShow(c.ctx)
	c.waitAndForceForeground(hwnd)
	c.suppressDeactivation.Store(false)

	if attached {
		detachFromForeground(hwnd)
	}

	runtime.EventsEmit(c.ctx, "panel-opened")
}

// findPanelWindow returns the Wails main window handle, or 0 if it does not
// exist yet.
func findPanelWindow() windows.HWND {
	hwnd, err := win32.FindWindowW(windows.StringToUTF16Ptr(wailsWindowClass), nil)
	if err != nil || hwnd == 0 {
		return 0
	}
	return hwnd
}

// installActivationHook subclasses the Wails window. When the panel is
// deactivated (the user clicked another application) it hides the panel.
// Spurious deactivations while we are opening the panel are suppressed.
func (c *Controller) installActivationHook() {
	hwnd := findPanelWindow()
	if hwnd == 0 {
		applog.Log("panel: window not found at startup; activation hook not installed")
		return
	}
	orig, _, _ := procSetWindowLongPtr.Call(uintptr(hwnd), gwlpWndProc, 0)
	if orig == 0 {
		applog.Log("panel: could not read original window proc")
		return
	}
	cb := syscall.NewCallback(func(h, msg, wparam, lparam uintptr) uintptr {
		switch msg {
		case WM_ACTIVATE:
			if int16(wparam) == waInactive && !c.suppressDeactivation.Load() {
				go c.hide()
			}
		case WM_ACTIVATEAPP:
			if wparam == 0 && !c.suppressDeactivation.Load() {
				go c.hide()
			}
		}
		r, _, _ := procCallWindowProc.Call(orig, h, msg, wparam, lparam)
		return r
	})
	prev, _, _ := procSetWindowLongPtr.Call(uintptr(hwnd), gwlpWndProc, cb)
	if prev == 0 {
		applog.Log("panel: install activation hook failed")
		return
	}
	applog.Log("panel: activation hook installed")
}

func (c *Controller) hide() {
	if c.ctx != nil {
		runtime.WindowHide(c.ctx)
	}
}

// attachToForeground attaches the panel window's thread input queue to the
// current foreground thread's queue, so SetForegroundWindow calls made on
// the panel's thread are not blocked by the foreground lock.
func attachToForeground(hwnd windows.HWND) bool {
	uiThread, _ := windows.GetWindowThreadProcessId(hwnd, nil)
	foreground := windows.GetForegroundWindow()
	if foreground == 0 {
		return false
	}
	fgThread, _ := windows.GetWindowThreadProcessId(foreground, nil)
	if fgThread == uiThread {
		return false
	}
	if err := win32.AttachThreadInput(uiThread, fgThread, true); err != nil {
		return false
	}
	return true
}

func detachFromForeground(hwnd windows.HWND) {
	uiThread, _ := windows.GetWindowThreadProcessId(hwnd, nil)
	foreground := windows.GetForegroundWindow()
	if foreground == 0 {
		return
	}
	fgThread, _ := windows.GetWindowThreadProcessId(foreground, nil)
	win32.AttachThreadInput(uiThread, fgThread, false)
}

// centeredOnMouseMonitor returns the top-left position (physical pixels)
// that centres the panel on the monitor under the mouse cursor, clamped to
// the monitor's work area.
func (c *Controller) centeredOnMouseMonitor() (int, int) {
	pt, err := win32.GetCursorPos()
	if err != nil {
		pt = win32.POINT{X: 0, Y: 0}
	}
	monitor := win32.MonitorFromPoint(pt, win32.MONITOR_DEFAULTTONEAREST)
	var info win32.MONITORINFO
	if err := win32.GetMonitorInfo(monitor, &info); err != nil {
		info.RCWork = windows.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}
	}

	w, h := runtime.WindowGetSize(c.ctx) // physical pixels
	work := info.RCWork
	width := int(work.Right - work.Left)
	height := int(work.Bottom - work.Top)

	x := int(work.Left) + (width-w)/2
	y := int(work.Top) + (height-h)/2
	if x < int(work.Left) {
		x = int(work.Left)
	}
	if y < int(work.Top) {
		y = int(work.Top)
	}
	if x+w > int(work.Right) {
		x = int(work.Right) - w
	}
	if y+h > int(work.Bottom) {
		y = int(work.Bottom) - h
	}
	return x, y
}

// waitAndForceForeground waits for the panel to become visible and
// foreground (Wails posts the show work to its UI thread asynchronously).
// If activation is rejected by the foreground lock, a momentary synthetic
// Alt tap grants this process the right to set the foreground window.
func (c *Controller) waitAndForceForeground(hwnd windows.HWND) {
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if windows.IsWindowVisible(hwnd) && windows.GetForegroundWindow() == hwnd {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	win32.KeybdEvent(0x12, 0, 0, 0) // VK_MENU down — unlocks SetForegroundWindow
	win32.KeybdEvent(0x12, 0, win32.KEYEVENTF_KEYUP, 0)
	win32.SetForegroundWindow(hwnd)
	win32.BringWindowToTop(hwnd)
	time.Sleep(50 * time.Millisecond)
}
