package folderscan

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestSplit(t *testing.T) {
	cases := []struct {
		in          string
		dir, prefix string
	}{
		{`F:\Playground\cw`, `F:\Playground`, `cw`},
		{`F:\Playground\`, `F:\Playground`, ``},
		{`F:/Playground/cw`, `F:/Playground`, `cw`},
		{`cwdgo`, ``, `cwdgo`},
		{``, ``, ``},
		{`\`, ``, ``},
		// Bare drive letter resolves to the drive root, not its CWD.
		{`F:\`, `F:\`, ``},
		{`F:\P`, `F:\`, `P`},
		{`F:/`, `F:\`, ``},
	}
	for _, c := range cases {
		dir, prefix := Split(c.in)
		if dir != c.dir || prefix != c.prefix {
			t.Errorf("Split(%q) = (%q, %q); want (%q, %q)", c.in, dir, prefix, c.dir, c.prefix)
		}
	}
}

func TestScan(t *testing.T) {
	dir := t.TempDir()
	// Build: alpha/, beta/, gamma/ + a file (must be ignored).
	for _, name := range []string{"alpha", "beta", "gamma", "AlphaCap"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "notafolder.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	basenames := func(paths []string) []string {
		out := make([]string, 0, len(paths))
		for _, p := range paths {
			out = append(out, filepath.Base(p))
		}
		return out
	}

	// Empty prefix -> all children, sorted, file excluded.
	got := basenames(Scan(dir, ""))
	want := []string{"alpha", "AlphaCap", "beta", "gamma"} // sort is case-sensitive byte order
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("Scan(empty) = %v; want %v", got, want)
	}

	// Prefix "a" -> case-insensitive alpha + AlphaCap.
	got = basenames(Scan(dir, "a"))
	if len(got) != 2 {
		t.Fatalf("Scan(a) = %v; want 2 entries", got)
	}

	// Prefix "z" -> none.
	if Scan(dir, "z") != nil {
		t.Errorf("Scan(z) = %v; want nil", Scan(dir, "z"))
	}

	// Non-existent dir -> nil.
	if Scan(filepath.Join(dir, "does-not-exist"), "") != nil {
		t.Errorf("Scan(missing) should be nil")
	}
}

func TestScanCap(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < MaxResults+5; i++ {
		name := string(rune('a'+(i%26))) + fmtIndex(i)
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got := Scan(dir, ""); len(got) != MaxResults {
		t.Errorf("Scan returned %d; want cap %d", len(got), MaxResults)
	}
}

// fmtIndex keeps names unique across the 26-letter cycle without importing fmt.
func fmtIndex(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{digits[i%10]}, b...)
		i /= 10
	}
	return string(b)
}
