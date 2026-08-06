# 04 — 软件列表 + 数字键动作

**What to build:** Open Actions 的软件侧：OpenActions 执行器通过注入的 Launcher 接口启动应用（测试用 fake 替换）；Software List 预置 Windows PowerShell（必有）、Antigravity 与 Trae CN（检测到已安装才预置）；面板条目上显示带序号的软件动作，按 `1-9` 用对应应用打开选中文件夹；经软件打开同样记录访问历史。命令构造为纯函数，单独测试。

**Blocked by:** 03（面板列表与交互）

**Status:** resolved

- [x] 预置列表：PowerShell 始终存在；Antigravity/Trae CN 仅已安装时出现
- [x] 每个软件动作在面板条目上带序号可见
- [x] 按 `1-9` 用对应应用打开选中文件夹，配置的参数被正确传入
- [x] 经软件打开成功后同样记录访问（置顶+刷新时间戳）
- [x] Launcher 可注入 fake；命令构造与成功/失败路径均有测试
- [x] 启动失败（应用不存在/路径无效）给出可读提示且不崩溃

## Answer

交付了 Open Actions 的软件侧，并修掉过程中暴露的三个 GUI 层 bug（其中两个是 03 遗留、被软件路径首次触发）。域层 Open Action 以 TDD 完成，胶水层以生产构建 + 真机行为验证。

### 交付的 artifact

- `domain/openactions/software.go`（新增，TDD）：`Software{Name,Exe,Args}` + 纯函数 `Command(folder)`——参数含 `{folder}` 占位符则替换，否则把 folder 追加为末参（一条规则同时表达 `pwsh ... Set-Location '{folder}'` 与 `code <folder>`）；`OpenSoftware(folder, sw, ln, rec)` 复用从 `Open` 抽出的 `launchAndRecord`，Explorer 与 Software 共享同一条「验证目录→启动→成功才记录」路径。`software_test.go` 11 个测试红→绿（占位替换/末参追加/多重占位 + 成功/各种失败路径）。
- `internal/launcher/`（新增，平台胶水）：`OSLauncher` 改用 **ShellExecuteW** 启动（见下方根因），带正确的 Windows 命令行参数 quoting（含空格路径正确传递）+ ShellExecute 错误码→可读消息；`detect.go` 解析开始菜单快捷方式探测 Antigravity/Trae CN 安装位置。domain 只留 `Launcher` 接口 + 纯逻辑，平台 syscalls 全在 internal（符合 spec「domain 与 win32 解耦」）。
- `app.go`（薄绑定）：`GetSoftwareList()`、`OpenWith(folder, index)`（越界报错）；`defaultSoftware()` 探测——PowerShell 走 PATH（Windows 必有），Antigravity/Trae CN 走快捷方式解析（任意安装盘符/目录）。
- `frontend/src/main.js`：每条目右侧渲染软件徽章（`<kbd>序号</kbd> 名称`，1-9）；数字键→用对应应用打开「解析目标」（搜索框合法路径优先，否则选中项，空历史也能自举）；点徽章→该应用，点其余→资源管理器；成功打开→刷新置顶→关面板，失败→留面板显示原因。

### 修复的三个 bug（过程中暴露，03 遗留 + GUI 路径首次触发）

1. **PowerShell 弹出秒退**（根因经 web-research + llm-call 双源确认）：cwdgo 是 GUI 子系统进程（无控制台），`os/exec` 把子进程 stdin 接 `NUL`(EOF)，PowerShell 交互宿主检测到非控制台 stdio → 无视 `-NoExit` 立即退出。控制台父进程下能存活（诊断时被计数 bug 误导走偏，已修正）。**解法**：`OSLauncher` 改用 ShellExecuteW，已在 GUI 子系统干净复现验证保活（计数 5s 稳定为 1）。Antigravity/Trae CN 是 Electron GUI 应用无此问题。
2. **Antigravity/Trae CN 检测不到**：实际装在 `D:\Programs\`（用户自定义盘符），原硬编码 `%LOCALAPPDATA%\Programs\<Name>\<Name>.exe` 错误且无 App Paths 注册表项。**解法**：解析开始菜单 Programs 快捷方式（WScript.Shell 取 TargetPath），独立验证返回正确路径。
3. **关闭面板后键盘仍生效**（焦点泄漏）：webview 在 host 窗口隐藏后仍短暂处理 window 级 keydown。**解法**：前端 `visible` 标志门控全部 keydown；Go `hide()` 发 `panel-hidden` 事件；Escape/打开后本地立即 disarm + blur。

### 顺带交付的增强（非 ticket 要求但验证中发现并修）

- **Alt+X 切换开关**：再按一次关闭面板（之前只能开）。
- **关闭后焦点还原**：`Open()` 最开头抓 `GetForegroundWindow`，`hide()` 后 `SetForegroundWindow` 还回（`AttachThreadInput` 跨进程保证可靠）。客观证据：日志 `prevForeground` 抓取+还原后前台变化，用户确认「可以了」。
- `build.sh`：构建前自动杀旧进程，避免旧进程占热键导致「该组合已被占用」弹窗（build 调试流程问题，非产品 bug）。

### 证据（域层机器可复现 + 胶水层真机行为）

- **域层**：`go test -count=1 ./domain/...` 全 PASS（openactions 含 11 个新增软件测试 + recentfolders + search）；`go vet` clean；`gofmt` clean（真实代码）。
- **构建**：`wails build` 成功产出 exe（绑定生成→前端编译→资源嵌入→Go 编译全绿）。
- **真机行为**（真实 Windows 会话，用户确认「是对的」「可以了」）：
  - 三个按键 `[1]PowerShell [2]Antigravity [3]Trae CN` 正确显示并各自打开选中文件夹。
  - PowerShell 弹出后**保持打开**（ShellExecute 修复后）。
  - `%APPDATA%\cwdgo\history.json` 落盘，PowerShell 打开置顶+刷新时间戳（软件打开走与已验证的 Open 相同的 launchAndRecord 路径，域测试 `TestOpenSoftwareLaunchesCommandAndRecords` 覆盖记录断言）。
  - 焦点还原：日志 `captured prevForeground hwnd=X` → `restoreForeground target=X` 还原后前台变化，用户确认焦点回到原窗口。
  - 关闭后按 `1` 不再触发任何窗口。

### 仍存在的不确定性

- 与 03 一致：胶水层（Wails 绑定、webview 视图、win32 调用）按架构决策不做单测，主观质量由 fresh Judge 独立评估；未提交 branch。
- Antigravity/Trae CN 作为 Electron 应用，打开文件夹依赖各自命令行参数约定（当前无 args，folder 作为末参传入）；若某 IDE 不认位置参数或需 `--folder` 之类，需在 Software List 配置里加参数（spec 的设置窗口 ticket 处理，本 ticket 仅做预置 + 默认行为）。
