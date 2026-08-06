// Package hotkey registers the global Launcher Hotkey. The default is
// Alt+X; a different combination can be registered via Listen (the settings
// ticket will make this user-configurable).
//
// The hotkey is registered with a NULL window handle on a dedicated OS
// thread: WM_HOTKEY is then posted to that thread's message queue, which
// runs a plain GetMessage loop. (A message-only window would be the more
// conventional receiver, but some environments reject creating one.)
package hotkey

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"

	"golang.org/x/sys/windows"

	"cwdgo/internal/applog"
	"cwdgo/internal/win32"
)

// Default modifier flags and virtual key code for Alt+X.
const (
	modAlt      = 0x0001
	modNoRepeat = 0x4000
	vkX         = 0x58
)

// hotkeyID identifies our hotkey within this process.
const hotkeyID = 1

// Hotkey is a global hotkey: a combination of modifier flags and a virtual
// key code.
type Hotkey struct {
	Modifiers uint32
	VK        uint32
}

// AltX is the default Launcher Hotkey.
var AltX = Hotkey{Modifiers: modAlt | modNoRepeat, VK: vkX}

// String returns a human-readable description for messages.
func (h Hotkey) String() string {
	if h == AltX {
		return "Alt+X"
	}
	mods := ""
	if h.Modifiers&modAlt != 0 {
		mods += "Alt+"
	}
	return mods + fmt.Sprintf("0x%X", h.VK)
}

// ErrAlreadyRegistered is returned when the combination is already taken.
var ErrAlreadyRegistered = errors.New("该组合已被占用")

// Listen registers hk and runs a message loop on a dedicated OS thread,
// invoking fn whenever the hotkey is pressed. The returned stop function
// unregisters the hotkey and terminates the loop.
func Listen(hk Hotkey, fn func()) (stop func(), err error) {
	regErr := make(chan error, 1)
	exited := make(chan struct{})
	threadID := make(chan uint32, 1)
	var once sync.Once

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		tid := windows.GetCurrentThreadId()
		threadID <- tid

		if err := win32.RegisterHotKey(0, hotkeyID, hk.Modifiers, hk.VK); err != nil {
			regErr <- fmt.Errorf("注册 %s 失败: %w", hk, describeHotkeyError(err))
			return
		}
		applog.Log("hotkey: registered %s, message loop running", hk)
		regErr <- nil // success; Listen returns from here

		var msg win32.MSG
		for {
			ok, err := win32.GetMessageW(&msg, 0, 0, 0)
			if !ok || err != nil {
				break // WM_QUIT or error
			}
			if msg.Message == win32.WM_HOTKEY && int32(msg.WParam) == hotkeyID {
				fn()
				continue
			}
			win32.TranslateMessage(&msg)
			win32.DispatchMessageW(&msg)
		}
		win32.UnregisterHotKey(0, hotkeyID)
		applog.Log("hotkey: message loop exited")
		close(exited)
	}()

	if err := <-regErr; err != nil {
		return nil, err
	}

	tid := <-threadID
	stop = func() {
		once.Do(func() {
			win32.PostThreadMessageW(tid, win32.WM_QUIT, 0, 0)
			<-exited
		})
	}
	return stop, nil
}

// ReportFailure shows a readable message box describing why the Launcher
// Hotkey could not be registered. The app keeps running — the tray menu
// still opens the panel.
func ReportFailure(err error) {
	windows.MessageBox(
		0,
		windows.StringToUTF16Ptr(fmt.Sprintf(
			"Launcher Hotkey 注册失败：%v\n\n面板仍可通过托盘菜单「打开面板」打开。", err)),
		windows.StringToUTF16Ptr("cwdgo"),
		0x00000010, // MB_ICONERROR
	)
}

// describeHotkeyError maps common Win32 errors to a readable message.
func describeHotkeyError(err error) error {
	if errno, ok := err.(syscall.Errno); ok {
		switch errno {
		case 1409: // ERROR_HOTKEY_ALREADY_REGISTERED
			return ErrAlreadyRegistered
		}
	}
	return err
}
