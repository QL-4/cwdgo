package openactions

import "strings"

// Software is one entry in the Software List: an external app that can open
// a folder. The panel renders Name with its ordinal (the 1-9 key); Exe and
// Args are used in Go to build the launch command.
type Software struct {
	Name string   `json:"name"`
	Exe  string   `json:"exe"`
	Args []string `json:"args"`
}

// FolderPlaceholder, when present in any arg, is replaced with the folder
// path. When no arg contains it the folder is appended as the final
// argument, so an app that takes the folder positionally (most IDEs) needs
// no args at all. This lets both `pwsh ... Set-Location '{folder}'` and
// `code <folder>` be expressed by one rule.
const FolderPlaceholder = "{folder}"

// Command returns the program and arguments that launch folder with sw. It
// is a pure function: same input always yields the same command, with no
// side effects.
func (s Software) Command(folder string) (name string, args []string) {
	out := make([]string, 0, len(s.Args)+1)
	placed := false
	for _, a := range s.Args {
		if strings.Contains(a, FolderPlaceholder) {
			out = append(out, strings.ReplaceAll(a, FolderPlaceholder, folder))
			placed = true
			continue
		}
		out = append(out, a)
	}
	if !placed {
		out = append(out, folder)
	}
	return s.Exe, out
}

// OpenSoftware opens folder with sw via ln and, on a successful launch,
// records it with rec so it moves to the top of Recent Folders — exactly as
// the default Explorer action does. Software and Explorer share the same
// launch/record path; only the command differs.
//
// It returns an error — and records nothing — if folder is not an existing
// directory or the launcher fails. A recorder failure is returned too, but
// only after the folder has already been launched.
func OpenSoftware(folder string, sw Software, ln Launcher, rec Recorder) error {
	name, args := sw.Command(folder)
	return launchAndRecord(folder, name, args, ln, rec)
}
