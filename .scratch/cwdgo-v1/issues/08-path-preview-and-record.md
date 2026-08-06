# 08 — 输入路径补全搜索 + 回车记录而非打开

**What to build:**
1. 输入框输入完整绝对路径（或部分路径）时，搜索结果列表应有内容——去文件系统实时搜索匹配的子文件夹，并和历史记录区分（新发现的标「新」）。
2. 回车时不要打开资源管理器，而是把解析到的路径作为最新记录加入历史顶部，并且面板不要关闭（保持打开，列表刷新，路径在顶部）。打开文件夹改为通过点击列表项或按数字键。

**Blocked by:** 03（面板数据）

**Status:** resolved

- [x] 输入部分路径 → 文件系统补全（列出父目录下匹配 prefix 的子文件夹），与历史模糊匹配合并去重
- [x] 裸盘符 `F:\` 正确解析为盘根（不是盘当前目录）
- [x] 文件系统新发现、尚未记录的项标「新」，已记录项保持历史顺序（不被模糊分数重排）
- [x] 新增 `Record(path)` 绑定：只记录到历史顶部，不打开，不关闭面板
- [x] 回车改为记录 + 刷新 + 保持打开；点击/数字键仍打开

## Answer

这一轮做了三组关联改动：路径补全搜索、回车记录语义、托盘左键 toggle（顺带替换了 getlantern/systray 三方库）。

### 路径补全搜索（需求 1+2 升级）

- 新增 `internal/folderscan`（I/O glue）：`Split(query)→(dir,prefix)` 把输入在最后一个路径分隔符处切开；`Scan(dir,prefix)` 列出父目录下匹配前缀的子文件夹（大小写不敏感、按名排序、上限 30）。裸盘符 `F:` 归一化为盘根 `F:\`（关键修复：`os.ReadDir("F:")` 读的是盘当前目录而非根目录）。
- 域层 `domain/search` 新增 `Filter`（保序模糊过滤，TDD 4 个新测试 GREEN），与 `Search`（按分数排序）并存。面板用 `Filter`：已记录项始终保持历史顺序，不被模糊分数打乱；文件系统新发现项追加在后面。
- `app.Search` 合并两源：历史模糊过滤（标记 Recorded）+ 文件系统补全（已在历史的标 Recorded，否则标未记录），按规范化路径去重。
- `Folder` 视图模型加 `Recorded bool` 字段；前端 `render` 对未记录项加 `.unrecorded` 类 + 黄色「新」标签（文字颜色不灰，只靠标签区分）。

### 回车记录语义

- 新增 `app.Record(path)` 绑定：仅 `store.Record`（加到历史顶部），不打开，不关闭面板。
- 前端回车（`openDefault`）改为 `Record` + 清空输入 + `load` 刷新 + 保持打开 + 聚焦输入框。打开文件夹的入口收窄为点击列表项 / 数字键。
- 空状态文案更新（回车是「记录到历史」而非「打开」）。

### 托盘左键 toggle（用户追加需求）

getlantern/systray v1.2.2 把左键和右键单击都硬编码为弹菜单，无法区分、无法注入单击回调。为支持「左键单击打开/关闭面板（toggle）、右键弹菜单」，自实现了极简 win32 托盘替换该三方库：

- `internal/win32` 扩展：`NOTIFYICONDATA` 结构体、`Shell_NotifyIconW`（**shell32.dll**，非 user32）、`CreateIconFromResource`（4 参数基础版）、`CreatePopupMenu`/`AppendMenuW`/`TrackPopupMenu`、`RegisterWindowMessageW`、窗口类注册与创建函数。
- `internal/tray` 重写：`Shell_NotifyIconW` 注册图标（tooltip=`CwdGo`），自定义回调消息区分 `WM_LBUTTONUP`（toggle）/`WM_RBUTTONUP`（弹菜单，用微软文档要求的 SetForegroundWindow→TrackPopupMenu→PostMessage(WM_NULL) 时序修正菜单不消失 bug），`TaskbarCreated` 消息处理 explorer 重启。
- `internal/icon` 加 `RawImageData(size)` 导出单图标的 RT_ICON 资源字节（`CreateIconFromResource` 要的是这个，不是 ICO 文件包装）。
- **deactivation 竞态修复**：左键点托盘会偷走面板前台焦点 → deactivation hook 异步 `hide()` → `Open` 读到不可见 → 又打开，面板永远关不掉。新增 `panel.ToggleFromTray()`：记录 `lastHiddenAt`，若 toggle 时面板刚在 250ms 内被自动隐藏（失焦关闭），视为「关闭」意图不重开。热键路径 `Open()` 不受影响（全局热键不改变前台）。

### 关键 bug 排查过程

1. 托盘图标「不见了」→ 日志 `tray: fatal: operation completed successfully`（errno 0）→ 加逐步诊断发现 `CreateIconFromResource` 用 7 参数版且传入整个 ICO 文件（非 RT_ICON 字节）→ 改 4 参数版 + `icon.RawImageData`。
2. 仍 fatal → `Shell_NotifyIconW` 声明在 user32.dll（**错误**，它在 shell32.dll）→ panic 被 GUI 子系统静默吞掉（无 stderr）→ 加 `wails.Run` 错误日志 + tray recover → 定位 → 改 shell32.dll。
3. 左键 toggle 关不掉 → deactivation 竞态 → `lastHiddenAt` 宽限期方案。

### 证据

- **域层 + folderscan**：`go test ./domain/... ./internal/folderscan/` 五包全 PASS（含 `search.Filter` 保序、folderscan Split/Scan/裸盘符/上限测试）；`go vet` clean；`gofmt` clean。
- **构建**：`wails build` 成功；绑定生成 `Record`。
- **真机行为**（日志验证）：托盘 `tray: ready` + `panel: activation hook installed`；toggle 循环日志显示稳定的「打开 / suppressed（just deactivated）」交替模式，证明 toggle 正确。
- **getlantern/systray 依赖移除**：`go.mod` 不再含该依赖（`go mod tidy` 清理）。

### 不确定性

- `deactivationGrace = 250ms` 是经验值（日志观察到的异步 hide 延迟在亚毫秒级），留有充分余量；极端慢机器上理论可能误判，但目前未观察到。
- folderscan 是平台 I/O glue 不做单测，纯函数 Split + 临时目录 Scan 已覆盖；主观质量待 fresh Judge 评估。
