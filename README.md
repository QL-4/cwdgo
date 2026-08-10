<p align="center">
  <img src="build/appicon.png" width="120" height="120" alt="cwdgo" />
</p>

<h1 align="center">cwdgo</h1>

<p align="center">
  <strong>Stop digging through Explorer. Jump to any folder in two keystrokes.</strong><br/>
  A keyboard-first recent-folders launcher that lives in your system tray.
</p>

<p align="center">
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-blue.svg" /></a>
  <img alt="Platform" src="https://img.shields.io/badge/platform-Windows-0078D4.svg" />
  <img alt="Go" src="https://img.shields.io/badge/Go-1.26-00ADD8.svg" />
  <img alt="Wails" src="https://img.shields.io/badge/Wails-v2-FF6B35.svg" />
  <img alt="Single binary" src="https://img.shields.io/badge/distribution-single%20exe-success.svg" />
</p>

---

## 😩 The Pain

You open the same handful of project folders every single day. Yet every time it's the same ritual:

- **Click Explorer → navigate five levels deep → repeat** — for a folder you were in ten minutes ago.
- **"Open Folder in IDE"?** Most editors only remember the *last* project. Switch back to a second one and you're navigating again.
- **Right-click → "Open in Terminal"** works, but good luck doing it for a folder that isn't currently open in anything.
- **Win+R, paste a path, Enter** — fast, except you can never remember the path, and nothing remembers it for you.

Every tool tracks *its own* recent list. None of them talks to each other. So you end up navigating the same paths over and over, in every app.

**There is no single, global, keyboard-driven "jump to that folder I just used" button.** Until now.

---

## ✨ What cwdgo Does

cwdgo is a tiny, always-on **Windows system-tray app** that puts every folder you touch one hotkey away — from *any* app, with *any* tool you choose.

> ![cwdgo panel](docs/screenshots/%E4%B8%BB%E9%9D%A2%E6%9D%BF.png)

### 🎯 One hotkey, zero friction

Press **`Alt+X`** anywhere (or **left-click the tray icon**). A floating panel snaps to your cursor's monitor, search box already focused. Start typing — **fuzzy search** filters your Recent Folders by name *and* full path. Press `1`–`9` to open the selection in your favorite editor/terminal, or `Enter` to record it. `Esc` or click-away dismisses it instantly.

```
        ┌─────────────────────────────────────┐
        │  🔍 powers_                         │
        ├─────────────────────────────────────┤
        │  D:\Work\cwdgo                      │
        │  F:\Playground\cwdgo                │
        │  D:\Work\powershell-scripts    [1]  │  ← press 1 → PowerShell
        │  D:\Work\my-app                [2]  │  ← press 2 → Antigravity
        │  D:\Work\another                [3]  │  ← press 3 → Trae CN
        └─────────────────────────────────────┘
```

### 🔢 Open with *any* tool — not just Explorer

Each folder in the list carries numbered badges for your **Software List**. Hit a digit and the folder opens in that exact app — `{folder}` placeholder passes the path as an argument:

> ![Type a path, press a number, it opens in your tool](docs/screenshots/%E6%93%8D%E4%BD%9C%20GIF.gif)

PowerShell, your editor, your terminal — all share the **same global recent-list**, so the folder you opened in VS Code is instantly reachable from your terminal too.

### 📋 Type a path, it remembers

Paste or type a full path and press `Enter` — cwdgo **records it to the top of your history** (no Explorer popup, panel stays open) so it's one keystroke away next time. Empty history on first run? You just bootstrap it by typing.

### 🗂️ Smart, merged search

- **Recent Folders** are fuzzy-filtered by name and full path (case-insensitive), kept in recency order.
- **Filesystem completion**: typing a partial path lists matching subfolders live, flagged with a **`新` (new)** badge so you can tell a fresh discovery from a known one.
- History and completions merge and de-duplicate automatically.

> ![Path completion with «新» badges](docs/screenshots/%E6%90%9C%E7%B4%A2.png)

### ⚙️ Sensible defaults, fully configurable

A built-in settings panel (no save button — everything auto-persists):

> ![Settings — software list CRUD](docs/screenshots/%E8%AE%BE%E7%BD%AE%E9%A1%B5.png)

- **Software List CRUD** — add/edit/remove launchers with name, path, and args.
- **History cap** (default 50) — keeps the list lean.
- **Start with Windows** toggle (off by default).

PowerShell is preloaded always; Antigravity & Trae CN are auto-detected and preloaded when installed.

### 📌 Lives in your tray

cwdgo runs silently in the background. **Left-click the tray icon** to toggle the panel; **right-click** for a menu — Open, Settings, Quit. No taskbar clutter, no window to manage.

> ![Tray menu](docs/screenshots/%E6%89%98%E7%9B%98%E8%8F%9C%E5%8D%95.png)

---

## 🚀 Install

### Option A — Download the binary *(coming once releases are published)*

Grab `cwdgo.exe` from [Releases](../../releases), drop it anywhere, double-click. That's it — it lives in your tray.

### Option B — Build from source

**Prerequisites:** [Go 1.26+](https://go.dev/dl/) and [Node.js](https://nodejs.org/) (for the frontend).

```bash
git clone https://github.com/QL-4/cwdgo.git
cd cwdgo

# Install the Wails CLI (one-time)
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Build the single .exe
wails build
# → build/bin/cwdgo.exe
```

Then run `build/bin/cwdgo.exe`. On Windows you can also use the included helper:

```bash
./build.sh   # kills any old instance, rebuilds, and launches
```

---

## ⌨️ Keyboard Shortcuts

| Key                 | Action                                            |
| ------------------- | ------------------------------------------------- |
| `Alt+X`             | Toggle the panel (open / close), anywhere         |
| `1`–`9`             | Open the selected folder in Software List slot N  |
| `Enter`             | Record the typed/selected path to history         |
| `↑` / `↓`           | Move selection                                    |
| `Esc` / click-away  | Close the panel                                   |
| **Left-click tray** | Toggle the panel                                  |
| **Right-click tray**| Menu: Open · Settings · Quit                      |

> Tip: click a list item (or its row) to open it in Explorer with the default action.

---

## 🏗️ How It Works

```
            ┌──────────────────────────────────────────┐
            │            Wails webview (UI)             │
            │   search box · list · settings panel      │
            └───────────────────┬──────────────────────┘
                                │ Wails bindings (thin glue)
            ┌───────────────────┴──────────────────────┐
            │                 app.go                    │  ← orchestration
            └───┬──────────┬───────────┬──────────┬─────┘
                │          │           │          │
        ┌───────▼──┐ ┌─────▼────┐ ┌───▼────┐ ┌───▼──────┐
        │  domain  │ │  domain  │ │ domain │ │ internal │
        │ recent-  │ │  search  │ │settings│ │ launcher │
        │ folders  │ │ (fuzzy)  │ │        │ │ (Win32)  │
        └──────────┘ └──────────┘ └────────┘ └──────────┘
              │
        ┌─────▼───────────────────────────────────────┐
        │  %APPDATA%\cwdgo\  (history.json + settings) │
        └──────────────────────────────────────────────┘
```

**Design principle: domain logic is pure and fully decoupled.** All behavior lives in the `domain/` packages (`recentfolders`, `search`, `settings`, `openactions`) with zero Wails / Win32 / systray dependencies — they're plain Go, unit-tested, and the UI layer is just a thin view. The tray, hotkey, and Win32 interop are isolated in `internal/`.

- **No database** — two JSON files under `%APPDATA%\cwdgo\`.
- **No background indexing** — cwdgo self-tracks only folders *you* open through it.
- **Single binary**, native Win32 tray (no Electron, no third-party tray library).

<details>
<summary><b>📦 Project layout</b></summary>

```
cwdgo/
├── main.go                  # entry: tray + hotkey + Wails app
├── app.go                   # Wails bindings (thin glue to domain)
├── domain/                  # pure, tested business logic
│   ├── recentfolders/       #   history store (record/dedupe/cap/persist)
│   ├── search/              #   fuzzy search + order-preserving filter
│   ├── settings/            #   settings store + software list
│   └── openactions/         #   open-action commands + software model
├── internal/                # platform glue (untested, manually verified)
│   ├── tray/                #   native Win32 tray (left=toggle, right=menu)
│   ├── hotkey/              #   global Alt+X via RegisterHotKey
│   ├── panel/               #   window positioning + deactivation handling
│   ├── launcher/            #   fresh-environment CreateProcessW + IDE detection
│   ├── folderscan/          #   filesystem path completion
│   ├── win32/               #   NOTIFYICONDATA, popup menu, etc.
│   ├── icon/                #   embedded icon → HICON
│   └── applog/              #   file logger for release builds
├── frontend/                # vanilla JS view (render + keyboard nav)
│   └── src/
├── docs/                    # ADRs + agent docs
└── .scratch/                # design spec + issue tickets (tracked)
```

</details>

---

## 🧪 Testing & Quality

The domain layer is developed test-first. Pure logic only — no mocks of Win32 or the UI.

```bash
go test ./domain/... ./internal/folderscan/
```

```
ok  cwdgo/domain/openactions
ok  cwdgo/domain/recentfolders
ok  cwdgo/domain/search
ok  cwdgo/domain/settings
ok  cwdgo/internal/folderscan
```

Every issue is tracked as a markdown ticket under `.scratch/cwdgo-v1/issues/` with an explicit **Problem → Solution → Evidence** write-up, and architectural decisions are recorded in [`docs/adr/`](docs/adr).

---

## ❓ FAQ

**Windows only?**
Yes. v1 is Windows-first (global hotkey, Win32 tray, registry autostart). The domain layer is portable; the platform glue is not yet.

**Does it spy on what I open?**
No. cwdgo only records folders you open *through cwdgo itself*. It never scans your filesystem history or watches other apps.

**Where's my data?**
`%APPDATA%\cwdgo\` — two plain JSON files (`history.json`, `settings.json`). Human-readable, easy to back up or nuke. A corrupted file silently resets to defaults.

**Why not just use PowerToys Run / Listary / Everything?**
Those are *file* search tools — they index your whole disk. cwdgo is a *recency* tool: it surfaces the handful of folders you actually cycle through daily, ordered by when you last touched them, with one-keystroke "open in app N". Different muscle memory, different problem.

---

## 🤝 Contributing

Issues and PRs welcome. The codebase is intentionally small and well-seamed — domain logic is isolated and tested, so most behavior changes are a one-package TDD loop. See `.scratch/` for the design spec and open tickets.

---

## 📄 License

[MIT](LICENSE) © 2026 QL-4
