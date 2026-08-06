// Package autostart manages the Windows auto-start-with-login entry for
// cwdgo via the HKCU Run key. It is platform glue (no domain logic) and is
// not unit-tested — Enable/Disable are exercised behaviourally from the
// settings binding.
package autostart

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

// runKey is the per-user Run key Windows consults at logon.
const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// valueName is the registry value that holds cwdgo's launch command.
const valueName = "cwdgo"

// Enable writes the current executable's path (quoted, as it may contain
// spaces) to the HKCU Run key so cwdgo starts at the next logon.
func Enable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(valueName, `"`+exe+`"`)
}

// Disable removes cwdgo's Run entry. It is idempotent: a missing value is
// treated as success.
func Disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil // key gone → already disabled
		}
		return err
	}
	defer k.Close()
	if err := k.DeleteValue(valueName); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

// IsEnabled reports whether cwdgo's Run entry currently exists.
func IsEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(valueName)
	return err == nil
}
