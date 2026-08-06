# 01 — 骨架：托盘 + 热键 + 面板窗口

**What to build:** 应用运行后常驻系统托盘（菜单含「打开面板」「退出」）；按下默认 Launcher Hotkey `Alt+X`，在鼠标所在显示器弹出 Launcher Panel，搜索框自动聚焦；`Esc` 或点击面板外部自动关闭；无历史时显示空状态提示。这是贯穿 Wails / systray / 热键 / 窗口的 vertical spine。

**Blocked by:** None — can start immediately

**Status:** resolved

- [x] 启动后出现托盘图标，菜单含「打开面板」和「退出」，退出后进程结束
- [x] 按 `Alt+X` 面板弹出在鼠标所在显示器，搜索框自动聚焦
- [x] `Esc` 或点击面板外部关闭面板
- [x] 无历史数据时显示空状态提示
- [x] 热键注册失败时给出可读的错误提示而不是崩溃

## Answer

已交付垂直骨架（Wails v2 + systray + 全局热键 + 面板窗口）。本 ticket 是约定不做单测的薄胶水层，全部以行为级客观证据验证（见下「证据」）。

### 交付的 artifact

- `main.go` / `app.go`（package main）：Wails 入口 + 薄绑定层。`App.GetRecentFolders()` 把 `domain/recentfolders` 接到前端；窗口选项：frameless / always-on-top / 720×440 / `StartHidden`。Recent Folders 持久化到 `%APPDATA%\cwdgo\history.json`（spec 共识）。
- `internal/tray/tray.go`：getlantern/systray 托盘，独立 OS 线程跑消息循环；菜单「打开面板」「退出」；退出走 `systray.Quit() → onExit → App.Quit() → runtime.Quit`。
- `internal/hotkey/hotkey.go`：默认 `Alt+X`（`MOD_ALT | MOD_NOREPEAT` + VK_X），专用 OS 线程的 `GetMessage` 循环接收 `WM_HOTKEY`（`RegisterHotKey(hWnd=0)`，WM_HOTKEY 进线程队列）。`Hotkey{Mods,VK}` 结构已为 ticket 06 的可配置热键预留。注册失败返回可读 error。
- `internal/panel/panel.go`：`Open()` 在鼠标所在显示器居中（`GetCursorPos → MonitorFromPoint → GetMonitorInfo`，物理像素，按工作区裁剪）、`WindowShow`、`AttachThreadInput` 绕过前台锁、必要时合成 Alt 键解锁 `SetForegroundWindow`、emit `panel-opened` 让前端聚焦搜索框。**点击外部关闭**：子类化 Wails 窗口（`SetWindowLongPtrW(GWLP_WNDPROC)` + `CallWindowProcW` 链回原 proc），在 `WM_ACTIVATE/WA_INACTIVE` 与 `WM_ACTIVATEAPP` 失活时 `WindowHide`（打开瞬间用 `suppressDeactivation` 抑制自身的 WA_ACTIVE）。
- `internal/win32/win32.go`：x/sys/windows 未导出的 user32 函数薄包装（FindWindow / RegisterHotKey / GetMessage / Monitor* / AttachThreadInput / SetForegroundWindow / keybd_event 等）。
- `internal/icon/icon.go`：纯 Go 绘制 cwdgo 图标（深色圆角块 + 文件夹 glyph），同一份 `Draw()` 既产出托盘 ICO（BMP-in-ICO），又供 `tools/genicon` 生成 `build/appicon.png` + `build/windows/icon.ico`。无二进制资源入库。
- `internal/applog/applog.go`：把胶水层的日志（无 release console）写到 `%APPDATA%\cwdgo\cwdgo.log`；同时作为 Wails `Logger`，把前端 `LogDebug` 路由进同一文件。
- `frontend/`（vanilla + Vite）：`index.html`（搜索框 + 空状态 div + 结果 ul）+ `main.js`（`EventsOn('panel-opened')` 聚焦搜索框、`Esc`→`WindowHide`、`GetRecentFolders` 渲染空状态或朴素列表）+ `style.css`（深色面板）。空状态文案「还没有历史记录 / 打开过的文件夹会出现在这里」。
- `tools/genicon`、`tools/screenshot`（手工验证用的纯 Go 小工具，后者按窗口类截图）。

### 证据（行为级，机器可复现）

在真实 Windows 会话里跑 `build/bin/cwdgo.exe`，配合 `tools/screenshot` + `SendInput`（PowerShell）采样：

- **启动**：`cwdgo.log` 顺序出现 `cwdgo starting → hotkey: registered Alt+X → tray: ready → panel: activation hook installed`；进程常驻（多线程：托盘 / 热键 / Wails UI）。
- **托盘**：systray 的 `onReady` 只在 `Shell_NotifyIcon` 成功后回调 → `tray: ready` 即图标创建成功；菜单项与回调已注册（`打开面板`→`p.Open`，`退出`→`systray.Quit`→`App.Quit`）。
- **Alt+X**：`SendInput(Alt+X)` 后 20ms 内窗口 `visible=true`、`GetForegroundWindow==面板`、矩形居中于鼠标所在显示器（单屏 1920×1080 下为 (600,320)-(1320,760)）。截图像素分析：暗背景 `#1e222e`(~287k px) + 搜索框 `#262b3a`(~24k px) + 聚焦边框 accent(~1.4k px) + 空状态 muted 文字 `#8b93a7`(~0.8k px) —— UI 渲染正确，空状态可见。
- **Esc**：`SendInput(Esc)` 后窗口 `visible=false`。
- **点击外部**：面板打开后 `Start-Process notepad` 抢前台 → `WM_ACTIVATE/WA_INACTIVE` 触发 → 窗口 `visible=false`。
- **热键注册失败**：第二个实例启动时 `Alt+X` 已被首个实例占用 → `RegisterHotKey` 返回 `ERROR_HOTKEY_ALREADY_REGISTERED` → 日志记 `注册 Alt+X 失败: 该组合已被占用`，应用**不崩溃**、继续起托盘与 Wails。

`go vet ./...` clean、`gofmt -l` 无输出、`go test ./domain/...` 24/24 PASS（ticket 02 的域逻辑）。

### 仍存在的不确定性

- **托盘菜单「退出」本身**：在此自动化会话里无法可靠地模拟点击系统通知区的托盘按钮（跨进程读 `ToolbarWindow32` 按钮需 `ReadProcessMemory`，UIAutomation 也找不到该环境的图标——见下），所以「点退出→进程结束」由代码构造保证（标准 systray + `runtime.Quit` 收尾），留给真人桌面手工确认。`tray: 打开面板/退出 clicked` 日志已埋点，点击即落盘可复核。
- **环境限制**：当前会话**无法创建 message-only 窗口**（`CreateWindowEx(HWND_MESSAGE)` 返回 `ERROR_INVALID_WINDOW_HANDLE`，连系统 STATIC 类也复现），也**无法弹出 `MessageBox`**（不创建可见窗口）。所以热键走 `RegisterHotKey(hWnd=0)` 线程队列方案而非消息专用窗口；`MessageBox` 在正常用户桌面会正常显示，但**在本环境无法目视确认**，仅以日志可读文案为证。这是会话/窗口站的特性，不影响正常桌面用户。
- **聚焦**：搜索框聚焦依赖 `EventsEmit('panel-opened')` + JS `input.focus()`；前台锁定场景下用合成 Alt 键解锁，已在单屏验证通过，多屏/高 DPI 的像素级居中只验证了 100% 缩放单屏（`WindowGetSize` 取物理像素 + 按工作区裁剪，逻辑应自适应）。
- WebView2 Runtime 依赖：本机已装 Evergreen Runtime（`C:\Program Files (x86)\Microsoft\EdgeWebView\Application` 150.x）。纯净 Windows 需自带/安装 WebView2。

按 spec 约定，胶水层不做单测、不自称跨过 `BAR.md`；主观质量由 fresh Judge 独立评估。未提交 branch。
