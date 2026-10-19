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
// Args are used in Go to build the launch command. Name is display-only —
// whether an entry can open SSH projects is expressed by RemoteArgs, never
// by the name.
type Software struct {
	Name string   `json:"name"`
	Exe  string   `json:"exe"`
	Args []string `json:"args"`
	// RemoteArgs is the command template for opening SSH projects
	// (host:/remote/path targets). Its presence IS the capability: entries
	// without one cannot open SSH projects (see CanOpenSSH). It uses the
	// same placeholder mechanism as Args, with SSHURIPlaceholder carrying
	// the vscode-remote:// URI built by SSHFolderURI.
	RemoteArgs []string `json:"remoteArgs,omitempty"`
}

// FolderPlaceholder, when present in any arg, is replaced with the folder
// path. When no arg contains it the folder is appended as the final
// argument, so an app that takes the folder positionally (most IDEs) needs
// no args at all. This lets both `pwsh ... Set-Location '{folder}'` and
// `code <folder>` be expressed by one rule.
const FolderPlaceholder = "{folder}"

// SSHURIPlaceholder is RemoteArgs' counterpart of FolderPlaceholder: any
// element containing it is replaced with the vscode-remote:// URI
// SSHFolderURI builds for the target, and when no element carries it the
// URI is appended as the final argument — so ["--folder-uri"] alone is
// already a complete template.
const SSHURIPlaceholder = "{sshUri}"

// CanOpenSSH reports whether sw can open SSH projects: it has a remote
// command template. The display Name never decides this.
func (s Software) CanOpenSSH() bool { return len(s.RemoteArgs) > 0 }

// Command returns the program and arguments that launch folder with sw. It
// is a pure function: same input always yields the same command, with no
// side effects.
func (s Software) Command(folder string) (name string, args []string) {
	return s.Exe, applyPlaceholder(s.Args, FolderPlaceholder, folder)
}

// applyPlaceholder returns template with every element's placeholder
// replaced by value; when no element carried the placeholder, value is
// appended as the final argument so positional-argument apps need no
// placeholder at all.
func applyPlaceholder(template []string, placeholder, value string) []string {
	out := make([]string, 0, len(template)+1)
	placed := false
	for _, a := range template {
		if strings.Contains(a, placeholder) {
			out = append(out, strings.ReplaceAll(a, placeholder, value))
			placed = true
			continue
		}
		out = append(out, a)
	}
	if !placed {
		out = append(out, value)
	}
	return out
}

// SSHFolderURI converts a host:/remote/path target into the URI accepted by
// Trae CN's --folder-uri CLI option. hostName is the SSH config alias used by
// Remote SSH (for example "server-a"), while target may display its IP.
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

// RemoteCommand returns the command that opens folder with sw. A local
// folder goes through Command; an SSH target is rendered into the
// RemoteArgs template via SSHURIPlaceholder. An SSH target with no
// RemoteArgs is an error — callers gate on CanOpenSSH, and the pure
// function stays total for any input.
func (s Software) RemoteCommand(folder, hostName string) (name string, args []string, err error) {
	if !strings.Contains(folder, ":/") {
		name, args = s.Command(folder)
		return name, args, nil
	}
	if len(s.RemoteArgs) == 0 {
		return "", nil, fmt.Errorf("openactions: software %q has no remote command", s.Name)
	}
	uri, err := SSHFolderURI(folder, hostName)
	if err != nil {
		return "", nil, err
	}
	return s.Exe, applyPlaceholder(s.RemoteArgs, SSHURIPlaceholder, uri), nil
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
