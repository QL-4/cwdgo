// genicon renders the cwdgo app icon into the files `wails build` needs:
// build/appicon.png (source icon) and build/windows/icon.ico (exe icon).
// The tray icon uses the same drawing code at runtime.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"cwdgo/internal/icon"
)

func main() {
	png, err := icon.PNG(256)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render png:", err)
		os.Exit(1)
	}
	if err := os.WriteFile("build/appicon.png", png, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write appicon.png:", err)
		os.Exit(1)
	}

	ico := icon.ICO(16, 32, 48, 256)
	if err := os.WriteFile(filepath.Join("build", "windows", "icon.ico"), ico, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write icon.ico:", err)
		os.Exit(1)
	}
	fmt.Println("wrote build/appicon.png and build/windows/icon.ico")
}
