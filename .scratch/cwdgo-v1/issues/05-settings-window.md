# 05 — 设置窗口：上限 + 开机自启 + 持久化

**What to build:** 托盘菜单新增「设置」打开设置窗口；可修改历史上限（默认 50，即时生效于 RecentFoldersStore）和开机自启开关（默认关闭，通过注册表实现）；所有设置持久化，重启后保留。设置窗口复用 01 的窗口基础设施。

**Blocked by:** 01（骨架与托盘菜单）

**Status:** resolved

- [x] 托盘菜单「设置」打开设置窗口
- [x] 历史上限可修改，保存后立即作用于历史存储，重启后保留
- [x] 开机自启开关默认关闭；开启后写入注册表，重启 Windows 后自动启动；关闭则移除
- [x] 设置文件损坏时静默重置默认值，不阻塞启动
- [x] 全部设置重启后保留

## Answer

交付了设置窗口（历史上限 + 开机自启 + 持久化）。新增了 `domain/settings` SettingsStore seam（TDD）并将 RecentFoldersStore 的上限从硬编码常量改为可配置字段（TDD）。开机自启走注册表（win32 胶水，行为验证）。前端为双视图（面板 / 设置）即时保存——无保存按钮，改动即落盘并即时应用。

### 交付的 artifact

- `domain/settings/settings.go`（新，TDD）：`Settings{HistoryLimit, AutoStart}` + `Store`。默认值（50 / 关）、`Update`（全量替换 + Validate + 原子持久化 temp+rename）、`New` 对缺失/不可读/损坏文件静默回退默认值（spec「设置文件损坏时静默重置默认值」）。6 个测试：默认值、缺失文件、往返、损坏重置、非法上限拒绝、全量替换。
- `domain/recentfolders/recentfolders.go`（改，TDD）：上限从硬编码 `const MaxEntries` 改为可配置字段 `limit`（默认仍 `MaxEntries`）；新增 `SetLimit(n)`——即时 trim 超出条目 + 持久化、拒绝 `<1`。4 个新测试：即时 trim、持久化 trim、后续 Record 遵守新上限、拒绝 0。
- `internal/autostart/autostart.go`（新，平台胶水）：注册表 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` 的 `Enable`（写当前 exe 路径）、`Disable`（幂等删除）、`IsEnabled`。
- `internal/panel/panel.go`（改）：提取 `show(emitEvent)`，`Open`（panel-opened）与新增 `OpenSettings`（settings-opened）共用同一显示路径；`OpenSettings` 不 toggle（窗口可见时只切视图）。
- `internal/tray/tray.go`（改）：菜单加「设置」项，`Run(onOpen, onSettings, onExit)`。
- `app.go`（改）：`GetSettings` / `SaveSettings(historyLimit, autoStart)` 绑定——保存即应用：`store.SetLimit`（即时 trim 历史）+ `autostart.Enable/Disable`（同步注册表）。
- `main.go`（改）：加载 settings store、启动时 `store.SetLimit(持久化上限)` 应用上限、接线 `tray.Run(p.Open, p.OpenSettings, app.Quit)`。
- 前端 `index.html` / `main.js` / `style.css`：`#panel` / `#settings` 双视图；设置表单即时保存（历史上限 `<input type="text" inputmode="numeric">`——无原生 spinner；自启 checkbox）；`change` 事件触发 `SaveSettings`；Esc 从任一视图关闭；返回按钮切回面板。

### 证据（域层机器可复现 + 胶水层真机行为）

- **域层**：`go test ./domain/...` 四包全 PASS（settings 6 + recentfolders 新增 4 + 既有全绿）；`go vet` clean；`gofmt` clean。
- **构建**：`wails build` 成功；绑定生成 `GetSettings` / `SaveSettings` / `Settings{historyLimit,autoStart}` 模型。
- **持久化往返**：`%APPDATA%\cwdgo\settings.json` 落盘 `{"historyLimit":50,"autoStart":true}`（用户通过 UI 切换自启后写入）。
- **开机自启注册表**：`reg query HKCU\...\Run /v cwdgo` → `cwdgo REG_SZ "F:\Playground\cwdgo\build\bin\cwdgo.exe"`（开启时写入；关闭时移除）。UI 勾选/取消即时同步注册表。
- **启动应用上限**：`main.go` 启动时 `SetLimit(hist.Get().HistoryLimit)`，持久化的上限在重启后立即作用于历史存储。
- **真机行为**（用户确认）：设置窗口打开、历史上限即时生效、自启切换工作。

### 仍存在的不确定性

- 胶水层（注册表、窗口切换、webview）按架构决策不做单测，主观质量由 fresh Judge 独立评估。
- 开机自启写的是当前 exe 的绝对路径（开发构建路径）；正式分发后路径为安装位置，行为不变。注册表条目在卸载时需由卸载程序清理（v1 无安装器逻辑，超出本 ticket 范围）。
