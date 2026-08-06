# 03 — 面板接上真实数据：列表 + 搜索 UX + 默认动作

**What to build:** Launcher Panel 显示 Recent Folders（按访问时间倒序）；输入即时 fuzzy 过滤；`↑↓` 键盘导航；`Enter` 对选中条目执行默认 Open Action（在资源管理器中打开）；打开成功后条目置顶并刷新列表。搜索框输入或粘贴完整路径且为合法目录时，`Enter` 直接打开该目录并记入历史（空历史时的自举路径，spec story 5）。把 01 的窗口骨架与 02 的域逻辑通过 Wails 绑定接通。

**Blocked by:** 01（骨架）、02（存储+搜索）

**Status:** resolved

- [x] 面板列出 Recent Folders，最新访问的排最上面
- [x] 搜索框输入即时过滤，无匹配时有明确提示
- [x] 输入/粘贴完整目录路径（如 `D:\projects\foo`），`Enter` 直接打开并记入历史（空历史时可用）
- [x] `↑↓` 移动选中项，高亮清晰可见
- [x] `Enter` 在资源管理器中打开选中文件夹
- [x] 打开成功后该条目置顶、列表即时刷新
- [x] 面板内交互全键盘可达，无鼠标也可完成

## Answer

已交付，把 01 的窗口骨架与 02 的域逻辑通过 Wails 绑定接通。域层 Open Action（可测 seam）以 TDD 完成，胶水层以生产构建 + 真机行为验证。

### 交付的 artifact

- `domain/openactions/`（新增，TDD seam）：`ExplorerCommand`（纯函数，构造 `explorer <folder>` 命令）；`Launcher` 接口 + `OSLauncher`（真实 `exec.Command.Start` + `Process.Release`，因 Explorer 是单例进程，转发后即退出，用 Start 而非 Run）；`Open(folder, launcher, recorder)` —— 验证是已存在目录 → 启动 → 成功才 Record（任何失败都不记录）；`IsExistingDir(path)`。`openactions` 故意不 import `recentfolders`，两者解耦。
- `app.go`（薄绑定）：新增 `Search(query)`、`IsDirectory(path)`、`Open(folder)`；引入 `Folder{Name,Path}` 视图模型——顺带修掉 01 遗留的隐 bug：`recentfolders.Entry.Name()` 是 Go 方法、JSON 不序列化，旧前端 `entry.Name` 会是 `undefined`（01 只测了空状态所以没暴露）。`GetRecentFolders` 现在返回 `[]Folder`。
- `main.go`：把 `openactions.OSLauncher{}` 经 `NewApp` 注入（spec 要求的注入式 Launcher 接口）。
- `frontend/src/main.js`（纯视图，重写）：即时 fuzzy 过滤（每次按键调 Go `Search`）；`↑↓` 键盘导航 + `selected` 高亮 + `scrollIntoView`；`Enter` 优先解析搜索框内容为合法目录路径（`IsDirectory`）直开（空历史自举，spec story 5），否则开选中项；成功打开 → `load()` 置顶刷新 → `WindowHide`；点击项也可开；两种空状态（无历史 / 无匹配，文案分别给出 bootstrap 与完整路径提示）。
- `frontend/src/style.css`：清晰的 `.selected` 高亮（金色背景 + 左侧 inset accent 竖条）+ hover。
- `frontend/index.html`：动态空状态元素、占位符改为「搜索或输入完整路径…」。
- `frontend/wailsjs/...`：`wails generate module` 重新生成（`Folder` 模型 + `Search`/`IsDirectory`/`Open` 绑定）。

### 证据（域层机器可复现 + 胶水层真机行为）

- **域层**：`go test -count=1 ./domain/...` 全 PASS（openactions 7/7 TDD red→green + recentfolders + search）；`go vet ./...` clean；`gofmt -l .` 无输出。
- **构建**：`wails build` 成功产出 `build/bin/cwdgo.exe`（绑定生成 → 前端编译 → 资源嵌入 → Go 编译全绿），证明 domain → 绑定 → 前端嵌入 端到端连通。
- **真机行为**（用户在真实 Windows 会话验证）：启动日志 `cwdgo starting → hotkey: registered Alt+X → tray: ready → panel: activation hook installed`。自举打开两次后 `%APPDATA%\cwdgo\history.json` 实际落盘两条，且置顶语义正确——先打开 `F:\Playground\cwdgo`(19:06:12) 再打开父目录 `F:\Playground`(19:06:16)，后者被置顶并刷新时间戳：
  ```json
  { "entries": [
    {"path":"F:\\Playground","lastUsed":"...19:06:16..."},
    {"path":"F:\\Playground\\cwdgo","lastUsed":"...19:06:12..."}
  ]}
  ```
  同时印证：自举打开生效、成功打开置顶+刷新、持久化往返。
- UI 交互（列表渲染、即时过滤、↑↓ 高亮、Enter 打开、Esc 关闭、两种空状态）经真机确认「是对的」。

### 仍存在的不确定性

- **Open 后的 UX 取舍**：当前成功打开后先 `load()` 刷新再 `WindowHide`（策略是「绝不残留」）。Explorer 抢前台时失焦钩子也会关面板，属双重保险；若希望置顶刷新结果在 Explorer 弹出前可见一拍，去掉那行 `WindowHide` 即可，一行改动。
- 与 01 一致：胶水层（Wails 绑定、webview 视图）按架构决策不做单测、不自称跨过 `BAR.md`；主观质量由 fresh Judge 独立评估。未提交 branch。
