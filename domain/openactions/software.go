package openactions

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	pathpkg "path"
	"strings"
)

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

// SSHFolderURI converts a host:/remote/path target into the URI accepted by
// Trae CN's --folder-uri CLI option. hostName is the SSH config alias used by
// Remote SSH (for example "beishida"), while target may display its IP.
func SSHFolderURI(target, hostName string) (string, error) {
	colon := strings.Index(target, ":/")
	if colon <= 1 || strings.TrimSpace(hostName) == "" {
		return "", fmt.Errorf("openactions: invalid SSH target: %s", target)
	}
	remotePath := pathpkg.Clean(target[colon+1:])
	authorityJSON, err := json.Marshal(struct {
		HostName   string `json:"hostName"`
		FromConfig bool   `json:"fromConfig"`
	}{HostName: hostName, FromConfig: true})
	if err != nil {
		return "", err
	}
	authority := "ssh-remote+" + hex.EncodeToString(authorityJSON)
	return "vscode-remote://" + url.QueryEscape(authority) + remotePath, nil
}

// RemoteCommand returns the Trae CN command for an SSH target, or falls back
// to the normal positional folder command for a local directory.
func (s Software) RemoteCommand(folder, hostName string) (name string, args []string, err error) {
	if !strings.Contains(folder, ":/") {
		name, args = s.Command(folder)
		return name, args, nil
	}
	uri, err := SSHFolderURI(folder, hostName)
	if err != nil {
		return "", nil, err
	}
	return s.Exe, append(append([]string(nil), s.Args...), "--folder-uri", uri), nil
}

// OpenSoftware opens a local folder with sw and records it on success.
func OpenSoftware(folder string, sw Software, ln Launcher, rec Recorder) error {
	name, args := sw.Command(folder)
	return launchAndRecord(folder, name, args, ln, rec)
}

// OpenSSHSoftware opens an SSH project in software that supports Trae CN's
// --folder-uri argument, then records the SSH target on success.
func OpenSSHSoftware(folder, hostName string, sw Software, ln Launcher, rec Recorder) error {
	if !IsSSHFolder(folder) {
		return fmt.Errorf("openactions: not an SSH folder: %s", folder)
	}
	name, args, err := sw.RemoteCommand(folder, hostName)
	if err != nil {
		return err
	}
	if err := ln.Launch(name, args); err != nil {
		return fmt.Errorf("openactions: launch %s: %w", name, err)
	}
	if err := rec.Record(folder); err != nil {
		return fmt.Errorf("openactions: record %s: %w", folder, err)
	}
	return nil
}
