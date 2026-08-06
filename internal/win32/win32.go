// Package win32 declares the small set of user32 functions the app needs
// beyond what golang.org/x/sys/windows already exposes. All calls are pure
// Win32 via syscall, no cgo beyond what Wails already pulls in.
package win32

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                     = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW            = user32.NewProc("FindWindowW")
	procGetCursorPos           = user32.NewProc("GetCursorPos")
	procMonitorFromPoint       = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW        = user32.NewProc("GetMonitorInfoW")
	procRegisterHotKey         = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey       = user32.NewProc("UnregisterHotKey")
	procGetMessageW            = user32.NewProc("GetMessageW")
	procTranslateMessage       = user32.NewProc("TranslateMessage")
	procDispatchMessageW       = user32.NewProc("DispatchMessageW")
	procPostMessageW           = user32.NewProc("PostMessageW")
	procPostThreadMessageW     = user32.NewProc("PostThreadMessageW")
	procAttachThreadInput      = user32.NewProc("AttachThreadInput")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procSetFocus               = user32.NewProc("SetFocus")
	procBringWindowToTop       = user32.NewProc("BringWindowToTop")
	procShowWindow             = user32.NewProc("ShowWindow")
	procKeybdEvent             = user32.NewProc("keybd_event")
	procGetWindowRect          = user32.NewProc("GetWindowRect")
	procGetWindowTextW         = user32.NewProc("GetWindowTextW")
	procGetDC                  = user32.NewProc("GetDC")
	procReleaseDC              = user32.NewProc("ReleaseDC")
	procDestroyWindow          = user32.NewProc("DestroyWindow")
	procCreateIconFromResource = user32.NewProc("CreateIconFromResource")
	procCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	procAppendMenuW            = user32.NewProc("AppendMenuW")
	procDestroyMenu            = user32.NewProc("DestroyMenu")
	procTrackPopupMenu         = user32.NewProc("TrackPopupMenu")
	procRegisterWindowMessageW = user32.NewProc("RegisterWindowMessageW")
	procCreateWindowExW        = user32.NewProc("CreateWindowExW")
	procRegisterClassExW       = user32.NewProc("RegisterClassExW")
	procDefWindowProcW         = user32.NewProc("DefWindowProcW")
	procPostQuitMessage        = user32.NewProc("PostQuitMessage")
	procLoadCursorW            = user32.NewProc("LoadCursorW")
)

// Window messages used by this package.
const (
	WM_HOTKEY = 0x0312
	WM_QUIT   = 0x0012
	WM_NULL   = 0x0000
)

// Window message constants for tray mouse events.
const (
	WM_LBUTTONUP = 0x0202
	WM_RBUTTONUP = 0x0205
	WM_COMMAND   = 0x0111
	WM_CLOSE     = 0x0010
	WM_DESTROY   = 0x0002
)

// Shell_NotifyIcon operations and flags.
const (
	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
)

// Menu flags for AppendMenuW / TrackPopupMenu.
const (
	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800

	TPM_LEFTALIGN   = 0x0000
	TPM_BOTTOMALIGN = 0x0020
	TPM_RETURNCMD   = 0x0100
)

// Modifier flags for RegisterHotKey.
const (
	MOD_ALT      = 0x0001
	MOD_NOREPEAT = 0x4000
)

// ShowWindow commands.
const (
	SW_SHOW = 5
)

// MonitorFromPoint flags.
const (
	MONITOR_DEFAULTTONEAREST = 2
)

// KeybdEvent flags.
const (
	KEYEVENTF_KEYUP = 0x0002
)

var (
	shell32              = windows.NewLazySystemDLL("shell32.dll")
	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
)

// POINT is a screen coordinate (Win32 POINT).
type POINT struct {
	X int32
	Y int32
}

// HWNDMessage is the predefined message-only window handle parent.
const HWNDMessage = ^uintptr(2) // HWND_MESSAGE = (HWND)-3

// WNDCLASSEXW is the Win32 window class structure for RegisterClassExW.
type WNDCLASSEXW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

// RegisterClassExW registers a window class.
func RegisterClassExW(wc *WNDCLASSEXW) (uint16, error) {
	r0, _, e1 := procRegisterClassExW.Call(uintptr(unsafe.Pointer(wc)))
	if r0 == 0 {
		return 0, e1
	}
	return uint16(r0), nil
}

// CreateWindowExW creates a window.
func CreateWindowExW(exStyle uint32, className, windowName *uint16, style uint32, x, y, width, height int32, parent windows.HWND, menu windows.Handle, instance windows.Handle, param uintptr) (uintptr, error) {
	r0, _, e1 := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		uintptr(style),
		uintptr(x), uintptr(y), uintptr(width), uintptr(height),
		uintptr(parent), uintptr(menu), uintptr(instance), param)
	if r0 == 0 {
		return 0, e1
	}
	return r0, nil
}

// DestroyWindow destroys a window created by CreateWindowExW.
func DestroyWindow(hwnd windows.HWND) error {
	r0, _, e1 := procDestroyWindow.Call(uintptr(hwnd))
	if r0 == 0 {
		return e1
	}
	return nil
}

// PostQuitMessage posts WM_QUIT to the thread's message queue.
func PostQuitMessage(exitCode int32) {
	procPostQuitMessage.Call(uintptr(exitCode))
}

// MSG is the Win32 message structure.
type MSG struct {
	Hwnd    windows.HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

// MONITORINFO describes a display monitor.
type MONITORINFO struct {
	Size      uint32
	RCMonitor windows.Rect
	RCWork    windows.Rect
	DwFlags   uint32
}

// FindWindowW returns the top-level window with the given class name and
// window name, or 0 when no such window exists.
func FindWindowW(className, windowName *uint16) (windows.HWND, error) {
	r0, _, e1 := procFindWindowW.Call(
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)))
	if r0 == 0 {
		if e1 != syscall.Errno(0) {
			return 0, e1
		}
		return 0, nil // not found
	}
	return windows.HWND(r0), nil
}

// GetCursorPos returns the screen position of the mouse cursor.
func GetCursorPos() (POINT, error) {
	var pt POINT
	r0, _, e1 := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if r0 == 0 {
		return pt, e1
	}
	return pt, nil
}

// MonitorFromPoint returns the monitor containing the given screen point.
func MonitorFromPoint(pt POINT, flags uint32) windows.Handle {
	r0, _, _ := procMonitorFromPoint.Call(
		uintptr(pt.X), uintptr(pt.Y), uintptr(flags))
	return windows.Handle(r0)
}

// GetMonitorInfo fills info for the given monitor.
func GetMonitorInfo(hmonitor windows.Handle, info *MONITORINFO) error {
	info.Size = uint32(unsafe.Sizeof(*info))
	r0, _, e1 := procGetMonitorInfoW.Call(
		uintptr(hmonitor), uintptr(unsafe.Pointer(info)))
	if r0 == 0 {
		return e1
	}
	return nil
}

// RegisterHotKey registers a system-wide hotkey for the calling thread.
func RegisterHotKey(hwnd windows.HWND, id int32, modifiers, vk uint32) error {
	r0, _, e1 := procRegisterHotKey.Call(
		uintptr(hwnd), uintptr(id), uintptr(modifiers), uintptr(vk))
	if r0 == 0 {
		return e1
	}
	return nil
}

// UnregisterHotKey releases a previously registered hotkey.
func UnregisterHotKey(hwnd windows.HWND, id int32) error {
	r0, _, e1 := procUnregisterHotKey.Call(uintptr(hwnd), uintptr(id))
	if r0 == 0 {
		return e1
	}
	return nil
}

// GetMessageW retrieves the next message from the calling thread's queue.
// It blocks until a message is available and returns false on WM_QUIT.
func GetMessageW(msg *MSG, hwnd windows.HWND, min, max uint32) (bool, error) {
	r0, _, e1 := procGetMessageW.Call(
		uintptr(unsafe.Pointer(msg)), uintptr(hwnd), uintptr(min), uintptr(max))
	if r0 == 0 {
		return false, nil // WM_QUIT
	}
	if r0 == ^uintptr(0) {
		return false, e1 // error
	}
	return true, nil
}

// TranslateMessage translates virtual-key messages into character messages.
func TranslateMessage(msg *MSG) bool {
	r0, _, _ := procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
	return r0 != 0
}

// DispatchMessageW dispatches a message to the window procedure.
func DispatchMessageW(msg *MSG) {
	procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
}

// PostMessageW posts a message to a window's message queue.
func PostMessageW(hwnd windows.HWND, msg uint32, wparam, lparam uintptr) error {
	r0, _, e1 := procPostMessageW.Call(
		uintptr(hwnd), uintptr(msg), wparam, lparam)
	if r0 == 0 {
		return e1
	}
	return nil
}

// PostThreadMessageW posts a message to another thread's message queue.
func PostThreadMessageW(threadID uint32, msg uint32, wparam, lparam uintptr) error {
	r0, _, e1 := procPostThreadMessageW.Call(
		uintptr(threadID), uintptr(msg), wparam, lparam)
	if r0 == 0 {
		return e1
	}
	return nil
}

// AttachThreadInput attaches or detaches the input queues of two threads.
func AttachThreadInput(idAttach, idAttachTo uint32, attach bool) error {
	r0, _, e1 := procAttachThreadInput.Call(
		uintptr(idAttach), uintptr(idAttachTo), boolToUintptr(attach))
	if r0 == 0 {
		return e1
	}
	return nil
}

// SetForegroundWindow brings the window to the foreground.
func SetForegroundWindow(hwnd windows.HWND) error {
	r0, _, e1 := procSetForegroundWindow.Call(uintptr(hwnd))
	if r0 == 0 {
		return e1
	}
	return nil
}

// SetFocus sets the keyboard focus to the window (same-thread only).
func SetFocus(hwnd windows.HWND) {
	procSetFocus.Call(uintptr(hwnd))
}

// BringWindowToTop brings the window to the top of the Z order.
func BringWindowToTop(hwnd windows.HWND) error {
	r0, _, e1 := procBringWindowToTop.Call(uintptr(hwnd))
	if r0 == 0 {
		return e1
	}
	return nil
}

// ShowWindow sets the window's visibility state.
func ShowWindow(hwnd windows.HWND, cmdShow int32) bool {
	r0, _, _ := procShowWindow.Call(uintptr(hwnd), uintptr(cmdShow))
	return r0 != 0
}

// KeybdEvent synthesizes a keyboard input event.
func KeybdEvent(vk byte, scan byte, flags uint32, extra uintptr) {
	procKeybdEvent.Call(
		uintptr(vk), uintptr(scan), uintptr(flags), extra)
}

func boolToUintptr(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

// GetWindowRect returns the bounding rectangle of a window in screen
// coordinates.
func GetWindowRect(hwnd windows.HWND, rect *windows.Rect) error {
	r0, _, e1 := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)))
	if r0 == 0 {
		return e1
	}
	return nil
}

// GetWindowText returns the window title.
func GetWindowText(hwnd windows.HWND) string {
	var buf [512]uint16
	n, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf[:n])
}

// GetDC returns the device context of the given window (0 = whole screen).
func GetDC(hwnd windows.HWND) (windows.Handle, error) {
	r0, _, e1 := procGetDC.Call(uintptr(hwnd))
	if r0 == 0 {
		return 0, e1
	}
	return windows.Handle(r0), nil
}

// ReleaseDC releases a device context obtained from GetDC.
func ReleaseDC(hwnd windows.HWND, dc windows.Handle) error {
	r0, _, e1 := procReleaseDC.Call(uintptr(hwnd), uintptr(dc))
	if r0 == 0 {
		return e1
	}
	return nil
}

// NOTIFYICONDATA is the Win32 structure for Shell_NotifyIconW. Only the
// fields up to SzTip are needed (icon + tooltip + callback message); the
// struct size matches NOTIFYICONDATA_V2 on both amd64 and 386.
type NOTIFYICONDATA struct {
	CbSize           uint32
	HWnd             windows.HWND
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            windows.Handle
	SzTip            [128]uint16
}

// Shell_NotifyIconW sends a taskbar icon operation (add/modify/delete).
func Shell_NotifyIconW(dwMessage uint32, nid *NOTIFYICONDATA) error {
	r0, _, e1 := procShellNotifyIconW.Call(uintptr(dwMessage), uintptr(unsafe.Pointer(nid)))
	if r0 == 0 {
		return e1
	}
	return nil
}

// CreateIconFromResource creates an HICON from raw RT_ICON resource
// bytes (BITMAPINFOHEADER + BGRA + AND mask, NOT the ICO file wrapper).
// icon.RawImageData provides such bytes.
func CreateIconFromResource(iconData []byte) (windows.Handle, error) {
	// Base version: CreateIconFromResource(presbits, dwResSize, fIcon, dwVer).
	// dwVer 0x00030000 is the version constant for icons.
	r0, _, e1 := procCreateIconFromResource.Call(
		uintptr(unsafe.Pointer(&iconData[0])),
		uintptr(len(iconData)),
		uintptr(1),          // fIcon = TRUE (icon, not cursor)
		uintptr(0x00030000)) // dwVer: version for icons
	if r0 == 0 {
		return 0, e1
	}
	return windows.Handle(r0), nil
}

// CreatePopupMenu creates an empty popup menu.
func CreatePopupMenu() windows.Handle {
	r0, _, _ := procCreatePopupMenu.Call()
	return windows.Handle(r0)
}

// AppendMenuW appends a string item or separator to a menu.
func AppendMenuW(menu windows.Handle, flags uint32, id uintptr, text *uint16) error {
	r0, _, e1 := procAppendMenuW.Call(
		uintptr(menu), uintptr(flags), id, uintptr(unsafe.Pointer(text)))
	if r0 == 0 {
		return e1
	}
	return nil
}

// DestroyMenu destroys a menu created by CreatePopupMenu.
func DestroyMenu(menu windows.Handle) error {
	r0, _, e1 := procDestroyMenu.Call(uintptr(menu))
	if r0 == 0 {
		return e1
	}
	return nil
}

// TrackPopupMenuEx wraps TrackPopupMenu (the classic API is sufficient for
// our simple menu). It blocks until the menu closes.
func TrackPopupMenu(menu windows.Handle, flags uint32, x, y int32, hwnd windows.HWND) uintptr {
	r0, _, _ := procTrackPopupMenu.Call(
		uintptr(menu), uintptr(flags),
		uintptr(x), uintptr(y),
		uintptr(0), uintptr(hwnd), uintptr(0))
	return r0
}

// RegisterWindowMessageW returns the message id for a system-wide string
// message; used to detect TaskbarCreated (explorer.exe restart) so the tray
// icon can be re-added.
func RegisterWindowMessageW(name *uint16) uint32 {
	r0, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(name)))
	return uint32(r0)
}
