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
	WM_SYSKEYUP    = 0x0105
	WM_SYSCOMMAND  = 0x0112

	waInactive     = 0
	vkMenu         = 0x12   // VK_MENU (Alt)
	scKeyMenu      = 0xF100 // SC_KEYMENU
	sysCommandMask = 0xFFF0
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
	// lastHiddenAt records when the panel was last hidden, so the tray
	// left-click toggle can tell a deliberate close from a deactivation
	// close (clicking the tray steals focus and deactivates the panel).
	lastHiddenAt atomic.Int64 // unix millis
	// skipNextRestore suppresses restoreForeground on the next hide. It is
	// set when the panel is closed because the user just launched an app
	// (Open/OpenWith): in that case we want the freshly launched process to
	// keep the foreground, not the window that was foreground before the
	// panel opened.
	skipNextRestore atomic.Bool
	// prevForeground is the window that was foreground just before the panel
	// opened. It is restored on hide so dismissing the panel returns keyboard
	// focus to whatever the user was working in, instead of leaving focus
	// dangling (SW_HIDE does not reliably restore the previous foreground).
	prevForeground windows.HWND
}

// New creates a Controller with no context yet.
func New() *Controller { return &Controller{} }

// SetContext wires the Wails runtime context (call from OnStartup) and
// installs the activation subclass that closes the panel on deactivation.
func (c *Controller) SetContext(ctx context.Context) {
	c.ctx = ctx
	c.installActivationHook()
}

// Open is the Launcher Hotkey / tray action. If the panel is already
// visible it is hidden (toggle); otherwise it is shown (centred on the
// monitor under the mouse) and the «panel-opened» event is emitted so the
// Open is the Launcher Hotkey / tray toggle. When the panel is visible it
// hides; when hidden it shows. Safe to call from any goroutine.
//
// From the tray left-click there is a deactivation race: clicking the tray
// icon steals focus from the panel, so WM_ACTIVATE/WM_ACTIVATEAPP fires and
// the activation hook hides the panel asynchronously — before this call
// observes IsWindowVisible the panel may already be hidden, making the
// toggle re-open it every time. ToggleFromTray guards against that.
func (c *Controller) Open() {
	c.toggle(false)
}

// ToggleFromTray is the tray left-click entry point. It behaves like Open
// except that, when the panel was hidden very recently (within
// deactivationGrace), it treats that hide as the toggle's «close» half and
// does not re-open — so clicking the tray to close actually closes.
func (c *Controller) ToggleFromTray() {
	c.toggle(true)
}

// HideAfterOpen hides the panel but skips the foreground restore, so the
// process just launched by Open/OpenWith keeps the foreground instead of
// being pushed behind the pre-panel window. The skip flag is consumed by
// the next hide (the deactivation triggered by our own WindowHide).
func (c *Controller) HideAfterOpen() {
	c.skipNextRestore.Store(true)
	c.hide()
}

// deactivationGrace is how long after an auto-hide a tray click is still
// considered the «close» half of the toggle. Picked comfortably above the
// asynchronous hide latency observed in logs (sub-millisecond) while short
// enough not to swallow a genuine later re-open.
const deactivationGrace = 250 * time.Millisecond

func (c *Controller) toggle(fromTray bool) {
	if c.ctx == nil {
		return
	}
	hwnd := findPanelWindow()
	if hwnd == 0 {
		return
	}
	if windows.IsWindowVisible(hwnd) {
		c.hide()
		return
	}
	// Tray click path: if the panel was just auto-hidden by deactivation
	// (clicking the tray stole focus), treat it as an intentional close.
	if fromTray {
		if t := c.lastHiddenAt.Load(); t > 0 && time.Since(time.UnixMilli(t)) < deactivationGrace {
			return
		}
	}
	c.show("panel-opened")
}

// OpenSettings is the tray «设置» action. If the window is already visible
// it just emits «settings-opened» to switch the frontend view in place;
// otherwise it shows the window (reusing the panel's show path) then emits
// the event. It never toggles closed — the settings window is opened
// deliberately from the tray, unlike the hotkey-driven panel.
func (c *Controller) OpenSettings() {
	if c.ctx == nil {
		return
	}
	hwnd := findPanelWindow()
	if hwnd == 0 {
		return
	}
	if windows.IsWindowVisible(hwnd) {
		runtime.EventsEmit(c.ctx, "settings-opened")
		return
	}
	c.show("settings-opened")
}

// show brings the hidden window up centred on the monitor under the mouse,
// takes the foreground and emits emitEvent so the frontend can switch its
// view (panel vs settings) and focus the right control. It captures the
// previous foreground first so hide() can return focus to it.
func (c *Controller) show(emitEvent string) {
	// Capture the current foreground window BEFORE anything else: any later
	// call (findPanelWindow, AttachThreadInput, showing our window) can
	// perturb the foreground, so the restore target must be grabbed first.
	prev := windows.GetForegroundWindow()

	hwnd := findPanelWindow()
	if hwnd == 0 {
		return
	}

	// Remember the window we are about to cover (skipping ourselves), so
	// hide() can give focus back to it.
	if prev != 0 && prev != hwnd {
		c.prevForeground = prev
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

	runtime.EventsEmit(c.ctx, emitEvent)
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
		case WM_SYSKEYUP:
			// Alt+X is registered globally. The panel can become foreground
			// before the Alt key-up arrives; DefWindowProc then puts this
			// frameless window into Alt menu mode, so a later arrow key opens
			// its system menu. Alt has no panel-specific meaning, so consume
			// that trailing key-up instead.
			if wparam == vkMenu {
				return 0
			}
		case WM_SYSCOMMAND:
			// Belt-and-braces: do not let the system activate a menu via Alt
			// or F10. The panel is frameless and has no menu bar to use.
			if wparam&sysCommandMask == scKeyMenu {
				return 0
			}
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
	if c.ctx == nil {
		return
	}
	c.lastHiddenAt.Store(time.Now().UnixMilli())
	hwnd := findPanelWindow()
	runtime.WindowHide(c.ctx)
	// Tell the frontend the panel is no longer user-visible so it stops
	// reacting to keyboard input. The deactivation that triggers this hide
	// is Go-side, so without this event the webview would keep its key
	// handlers armed and a stray keystroke could still open a folder.
	runtime.EventsEmit(c.ctx, "panel-hidden")
	// Return keyboard focus to whatever was foreground before the panel
	// opened. SW_HIDE does not do this reliably; restoring explicitly avoids
	// leaving focus dangling after Escape / a toggle-close. Skipped when the
	// close follows a launch (Open/OpenWith): the new process should inherit
	// the foreground, not the pre-panel window.
	if c.skipNextRestore.CompareAndSwap(true, false) {
		c.prevForeground = 0
		return
	}
	c.restoreForeground(hwnd)
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

// restoreForeground returns keyboard focus to the window that was foreground
// before the panel opened (captured in Open). It is a no-op when there was no
// previous window or it no longer exists. SetForegroundWindow is guarded with
// AttachThreadInput across the panel and target threads: we are now hidden so
// we may have lost the foreground lock, and attaching lets the restore take
// effect reliably regardless of which process owns the target window.
func (c *Controller) restoreForeground(panel windows.HWND) {
	target := c.prevForeground
	c.prevForeground = 0
	if target == 0 || !windows.IsWindow(target) || target == panel {
		return
	}

	curFg := windows.GetForegroundWindow()
	panelThread, _ := windows.GetWindowThreadProcessId(panel, nil)
	targetThread, _ := windows.GetWindowThreadProcessId(target, nil)
	fgThread := uint32(0)
	if curFg != 0 {
		fgThread, _ = windows.GetWindowThreadProcessId(curFg, nil)
	}

	attached1 := fgThread != 0 && fgThread != panelThread &&
		win32.AttachThreadInput(panelThread, fgThread, true) == nil
	attached2 := targetThread != panelThread && targetThread != fgThread &&
		win32.AttachThreadInput(panelThread, targetThread, true) == nil

	win32.BringWindowToTop(target)
	win32.SetForegroundWindow(target)

	if attached2 {
		win32.AttachThreadInput(panelThread, targetThread, false)
	}
	if attached1 {
		win32.AttachThreadInput(panelThread, fgThread, false)
	}
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
