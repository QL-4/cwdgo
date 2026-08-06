// Package launcher provides the real Launcher implementation that starts
// Open Action programs on Windows. It uses ShellExecuteW rather than
// os/exec because cwdgo is a GUI-subsystem process (no console): a child
// console app launched via os/exec inherits null/EOF standard handles, so
// an interactive host like PowerShell sees a non-interactive session and
// exits immediately regardless of -NoExit. ShellExecute launches the
// program the way the shell does, decoupled from the parent's stdio and
// with its own console, so the app runs as if the user double-clicked it.
package launcher

import (
	"fmt"
	"strings"
	"unsafe"

	"cwdgo/domain/openactions"

	"golang.org/x/sys/windows"
)

var (
	shell32           = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

// OSLauncher implements openactions.Launcher by handing the program to the
// Windows shell via ShellExecute. It is platform glue and is not unit-tested;
// the command construction and launch/record flow it feeds are tested in
// domain/openactions.
type OSLauncher struct{}

// Show-window command used by ShellExecute.
const swShowNormal = 1

// Launch opens name with args as if via the shell. The "open" verb runs the
// program normally (it also resolves bare names through PATH/App Paths and
// absolute paths directly). args are joined into a single command line with
// correct Windows quoting so folder paths containing spaces survive.
func (OSLauncher) Launch(name string, args []string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	file, _ := windows.UTF16PtrFromString(name)
	params, _ := windows.UTF16PtrFromString(buildParams(args))

	hinst, _, _ := procShellExecuteW.Call(0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		0,
		uintptr(swShowNormal),
	)
	// ShellExecute returns an HINSTANCE result; values <= 32 are error codes.
	if hinst <= 32 {
		return fmt.Errorf("launcher: %s", shellExecuteError(hinst))
	}
	return nil
}

// buildParams joins args into a single Windows command-line string, quoting
// any arg that contains whitespace or quotes per the CommandLineToArgvW
// rules. An empty arg slice yields "".
func buildParams(args []string) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = quoteArg(a)
	}
	return strings.Join(parts, " ")
}

// needsQuoting reports whether arg must be wrapped in quotes on a Windows
// command line.
func needsQuoting(s string) bool {
	if s == "" {
		return true
	}
	return strings.ContainsAny(s, " \t\n\r\v\"")
}

// quoteArg wraps s in quotes when it needs quoting, escaping backslashes
// that precede an embedded or the closing quote (the standard algorithm).
func quoteArg(s string) string {
	if !needsQuoting(s) {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			slashes++
		case '"':
			// Double the run of backslashes preceding the quote, then escape
			// the quote itself.
			for j := 0; j < slashes*2; j++ {
				b.WriteByte('\\')
			}
			b.WriteString(`\"`)
			slashes = 0
		default:
			for j := 0; j < slashes; j++ {
				b.WriteByte('\\')
			}
			slashes = 0
			b.WriteByte(s[i])
		}
	}
	// Double trailing backslashes so they are not taken as escaping the
	// closing quote.
	for j := 0; j < slashes*2; j++ {
		b.WriteByte('\\')
	}
	b.WriteByte('"')
	return b.String()
}

// shellExecuteError maps a ShellExecute HINSTANCE error code (<=32) to a
// readable message. See ShellExecute documentation.
func shellExecuteError(hinst uintptr) string {
	switch hinst {
	case 0:
		return "out of memory"
	case 2:
		return "file not found"
	case 3:
		return "path not found"
	case 5:
		return "access denied"
	case 8, 26:
		return "not enough memory / share error"
	case 27:
		return "incomplete file association"
	case 28, 29, 30:
		return "DDE failure"
	case 31:
		return "no application associated with this file"
	case 32:
		return "DLL not found"
	default:
		return fmt.Sprintf("ShellExecute error %d", hinst)
	}
}

// Compile-time check that OSLauncher satisfies the domain Launcher seam.
var _ openactions.Launcher = OSLauncher{}
