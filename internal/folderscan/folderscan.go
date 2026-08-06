// Package folderscan provides filesystem path completion for the launcher
// panel: given a typed query it resolves a parent directory and a name
// prefix, and lists the immediate child folders matching that prefix. This
// is the "real search" that lets the user reach folders not yet in Recent
// Folders, complementing the in-memory fuzzy search over history.
//
// This is platform I/O glue (os.ReadDir) and lives outside the domain; the
// pure split logic (Split) is unit-tested, and Scan is exercised against a
// temporary directory.
package folderscan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MaxResults caps how many filesystem matches are returned, to keep the
// panel snappy on directories with many entries.
const MaxResults = 30

// Split breaks a typed query into a parent directory to list and a name
// prefix to filter by. The split happens at the last path separator (either
// "\" or "/") so the user can type progressively:
//
//	"F:\Play\"        -> dir="F:\Play",  prefix=""   (list all children)
//	"F:\Play\cw"      -> dir="F:\Play",  prefix="cw" (narrow to "cw*")
//	"F:/Play/cw"      -> dir="F:/Play",  prefix="cw" (forward slashes too)
//	"cwdgo"           -> dir="",         prefix="cwdgo" (no dir to list)
//
// A query with no separator yields an empty dir, signalling the caller to
// fall back to history-only search.
//
// A bare Windows drive letter ("F:") is normalized to its root ("F:\"),
// because os.ReadDir("F:") reads the drive's current directory, not its
// root — so typing "F:\" must list the drive root, and "F:\P" must list
// the root filtered to "P*".
func Split(query string) (dir, prefix string) {
	idx := strings.LastIndexAny(query, `/\`)
	if idx < 0 {
		return "", query
	}
	dir = query[:idx]
	prefix = query[idx+1:]
	if isBareDrive(dir) {
		dir += `\`
	}
	return dir, prefix
}

// isBareDrive reports whether s is a single drive letter with a colon (e.g.
// "F:", "c:"), i.e. a root reference missing its separator.
func isBareDrive(s string) bool {
	if len(s) != 2 || s[1] != ':' {
		return false
	}
	c := s[0]
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// Scan lists immediate child directories of dir whose names start with prefix
// (case-insensitive). It returns cleaned paths (filepath.Join of dir + name),
// sorted by name. If dir does not exist or cannot be read, it returns nil.
// Results are capped at MaxResults.
func Scan(dir, prefix string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	lower := strings.ToLower(prefix)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(e.Name()), lower) {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil
	}
	if len(names) > MaxResults {
		names = names[:MaxResults]
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, filepath.Join(dir, n))
	}
	return out
}
