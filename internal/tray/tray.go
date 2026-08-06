// Package tray runs the system tray icon and its menu, using a minimal
// hand-rolled Win32 Shell_NotifyIcon implementation rather than a third
// party library. This is the only way to distinguish left vs. right click
// on the tray icon: a left click opens the panel, a right click shows the
// popup menu. It runs on its own OS thread.
package tray

import (
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"cwdgo/internal/applog"
	"cwdgo/internal/icon"
	"cwdgo/internal/win32"
)

// Tray application name shown as the icon tooltip.
const AppName = "CwdGo"

// trayCallbackMessage is the custom message Windows posts to our window
// when a mouse event happens over the tray icon.
const trayCallbackMessage = 0x8000 // WM_APP

// Callbacks is the set of actions the tray can trigger.
type Callbacks struct {
	OnLeftClick func() // single left click on the icon
	OnOpen      func() // «打开面板» menu item
	OnSettings  func() // «设置» menu item
	OnExit      func() // «退出» menu item
}

// Menu item command IDs (sent in WM_COMMAND wParam).
const (
	cmdOpen = iota + 1
	cmdSettings
	cmdExit
)

// Run starts the tray icon on a dedicated OS thread and blocks until the
// process exits. cb provides the action callbacks. Call it from main before
// the Wails main loop; it spawns its own goroutine internally.
func Run(cb Callbacks) {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := run(cb); err != nil {
			applog.Log("tray: fatal: %v", err)
		}
	}()
}

// run is the message-loop body, kept separate so the LockOSThread goroutine
// is tidy.
func run(cb Callbacks) error {
	defer func() {
		if r := recover(); r != nil {
			applog.Log("tray: PANIC: %v", r)
		}
	}()
	className, _ := windows.UTF16PtrFromString("CwdGoTray")
	var hinst windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &hinst); err != nil {
		return err
	}
	cursor, _ := loadArrowCursor()

	wc := win32.WNDCLASSEXW{
		Size: uint32(unsafe.Sizeof(win32.WNDCLASSEXW{})),
		WndProc: windows.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
			return wndProc(hwnd, msg, wParam, lParam, cb)
		}),
		Instance:  hinst,
		Cursor:    cursor,
		ClassName: className,
	}
	if _, err := win32.RegisterClassExW(&wc); err != nil {
		return err
	}

	windowName, _ := windows.UTF16PtrFromString("CwdGo Tray Window")
	hwndRaw, err := win32.CreateWindowExW(
		0, className, windowName, 0,
		0, 0, 0, 0,
		windows.HWND(win32.HWNDMessage), 0, hinst, 0)
	if err != nil {
		return err
	}
	hwnd := windows.HWND(hwndRaw)

	hicon, err := win32.CreateIconFromResource(icon.RawImageData(32))
	if err != nil {
		return err
	}

	nid := win32.NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(win32.NOTIFYICONDATA{})),
		HWnd:             windows.HWND(hwnd),
		UID:              1,
		UFlags:           win32.NIF_MESSAGE | win32.NIF_ICON | win32.NIF_TIP,
		UCallbackMessage: trayCallbackMessage,
		HIcon:            hicon,
	}
	copy(nid.SzTip[:], windows.StringToUTF16(AppName))
	if err := win32.Shell_NotifyIconW(win32.NIM_ADD, &nid); err != nil {
		return err
	}

	taskbarCreated := win32.RegisterWindowMessageW(syscall.StringToUTF16Ptr("TaskbarCreated"))
	applog.Log("tray: ready")

	var msg win32.MSG
	for {
		ok, err := win32.GetMessageW(&msg, 0, 0, 0)
		if !ok || err != nil {
			break
		}
		if msg.Message == taskbarCreated {
			// Explorer restarted: re-add the icon.
			win32.Shell_NotifyIconW(win32.NIM_ADD, &nid)
			continue
		}
		win32.TranslateMessage(&msg)
		win32.DispatchMessageW(&msg)
	}

	win32.Shell_NotifyIconW(win32.NIM_DELETE, &nid)
	return nil
}

// wndProc is the window procedure for the hidden tray window. It routes
// tray mouse events (left click -> open, right click -> menu) and menu
// command selections to the callbacks.
func wndProc(hwnd, msg, wParam, lParam uintptr, cb Callbacks) uintptr {
	switch msg {
	case uintptr(win32.WM_COMMAND):
		switch wParam & 0xffff {
		case cmdOpen:
			if cb.OnOpen != nil {
				cb.OnOpen()
			}
		case cmdSettings:
			if cb.OnSettings != nil {
				cb.OnSettings()
			}
		case cmdExit:
			if cb.OnExit != nil {
				cb.OnExit()
			}
		}
		return 0
	case uintptr(trayCallbackMessage):
		switch lParam & 0xffff {
		case uintptr(win32.WM_LBUTTONUP):
			if cb.OnLeftClick != nil {
				cb.OnLeftClick()
			}
		case win32.WM_RBUTTONUP:
			showMenu(windows.HWND(hwnd))
		}
		return 0
	case uintptr(win32.WM_CLOSE):
		win32.DestroyWindow(windows.HWND(hwnd))
		return 0
	case uintptr(win32.WM_DESTROY):
		win32.PostQuitMessage(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

// showMenu builds and tracks the popup menu at the cursor. The menu items
// are fixed (打开面板 / 设置 / 分隔符 / 退出).
func showMenu(hwnd windows.HWND) {
	menu := win32.CreatePopupMenu()
	openLabel, _ := windows.UTF16PtrFromString("打开面板")
	settingsLabel, _ := windows.UTF16PtrFromString("设置")
	exitLabel, _ := windows.UTF16PtrFromString("退出")
	win32.AppendMenuW(menu, win32.MF_STRING, cmdOpen, openLabel)
	win32.AppendMenuW(menu, win32.MF_STRING, cmdSettings, settingsLabel)
	win32.AppendMenuW(menu, win32.MF_SEPARATOR, 0, nil)
	win32.AppendMenuW(menu, win32.MF_STRING, cmdExit, exitLabel)

	pt, err := win32.GetCursorPos()
	if err != nil {
		pt = win32.POINT{}
	}

	// SetForegroundWindow + PostMessage(WM_NULL) is the documented workaround
	// for the tray menu not auto-dismissing when the user clicks elsewhere.
	procSetForegroundWindow.Call(uintptr(hwnd))
	win32.TrackPopupMenu(menu, win32.TPM_LEFTALIGN|win32.TPM_BOTTOMALIGN, pt.X, pt.Y, hwnd)
	procPostMessageW.Call(uintptr(hwnd), uintptr(win32.WM_NULL), 0, 0)

	win32.DestroyMenu(menu)
}

// loadArrowCursor loads the standard arrow cursor (IDC_ARROW).
func loadArrowCursor() (windows.Handle, error) {
	const IDC_ARROW = 32512
	r0, _, e1 := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
	if r0 == 0 {
		return 0, e1
	}
	return windows.Handle(r0), nil
}

// Proc references used directly by tray (window creation / message helpers
// not yet in win32.go). Declared once at package level.
var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procLoadCursorW         = user32.NewProc("LoadCursorW")
)
