# cwdgo v1 — Launcher Panel for Recent Folders

**Status:** ready-for-agent

## Problem Statement

用户经常需要打开最近访问过的文件夹：在资源管理器里一层层导航很慢，想用 IDE、终端打开某个文件夹也没有统一入口。需要一个常驻的、一键呼出的面板，把 Recent Folders 摆到手边。

## Solution

cwdgo 是一个 Windows 常驻托盘工具：通过 Launcher Hotkey 呼出 Launcher Panel，列出 Recent Folders，支持 fuzzy 搜索和多种 Open Action（默认资源管理器，或 Software List 中的指定应用）。搜索框可直接输入完整路径打开任意文件夹（历史为空时的自举路径）。设置窗口统一配置快捷键、软件列表、历史上限与开机自启。

## User Stories

1. As a user, I want to press the global Launcher Hotkey anywhere, so that the panel opens on top without alt-tabbing
2. As a user, I want the panel to auto-focus the search box on open, so that I can start typing immediately
3. As a user, I want Recent Folders listed newest-first by access time, so that I can find what I recently used
4. As a user, I want fuzzy search over folder name and full path (case-insensitive), so that partial input still finds the folder
5. As a user, I want to type or paste a full folder path into the search box and press Enter, so that I can open any folder even when history is empty, and it gets recorded as a Recent Folder
6. As a user, I want arrow-key navigation, so that I can select entries without the mouse
7. As a user, I want Enter to open the selected folder with the default Open Action (Explorer), so that the common case is one key away
8. As a user, I want keys `1-9` to trigger Software List actions on the selected folder, so that I can open it in a specific app directly
9. As a user, I want Esc or clicking outside to close the panel, so that it never lingers
10. As a user, I want every successful Open Action to record the folder and move it to the top, so that recency stays accurate
11. As a user, I want history capped at 50 entries with the oldest evicted, so that the list stays manageable
12. As a user, I want to configure the Launcher Hotkey (default `Alt+X`, with `Win+Q` offered as a preset option), so that it doesn't clash with my setup
13. As a user, I want to manage the Software List (name, executable path, arguments) in settings, so that I can open folders with my own tools
14. As a user, I want PowerShell preloaded in the Software List (always) and Antigravity / Trae CN preloaded when installed, so that setup is fast
15. As a user, I want to adjust the history cap in settings, so that I control how much is remembered
16. As a user, I want an auto-start-with-Windows toggle (default off), so that the tool is always available if I want it
17. As a user, I want settings persisted across restarts, so that my configuration survives
18. As a user, I want a tray icon with a menu (open panel, settings, quit), so that the app is always reachable
19. As a user, I want a friendly empty state when there is no history yet, so that first run isn't confusing

## Implementation Decisions

- **技术栈**：Go + Wails v2，单 exe 分发（ADR-0001）；托盘与全局热键基于 systray / win32 API。
- **模块划分**：Go 域逻辑包承载全部行为（RecentFoldersStore、Search、SettingsStore、OpenActions），与 Wails/systray/win32 完全解耦；Wails 层为薄绑定（方法转发 + 事件），前端为纯视图（渲染、键盘导航、调后端）。
- **存储**：`%APPDATA%\cwdgo\` 下两个 JSON 文件（历史 + 设置），无数据库。
- **Recent Folders 语义**：cwdgo 自跟踪；路径做大小写不敏感规范化后去重；任何 Open Action 成功即把条目置顶并刷新时间戳；上限 50，超出淘汰最旧。
- **搜索**：fuzzy、大小写不敏感，匹配文件夹名称 + 完整路径；输入为合法目录路径时，`Enter` 直接打开该目录并记入历史（历史为空时的自举路径，story 5）。
- **Open Action 模型**：默认动作 = 在资源管理器中打开；Software List 每个应用是一个动作，按序号对应数字键 `1-9`，超出部分可鼠标点击。执行走注入的 Launcher 接口（fake 可替换，便于测试）。
- **Launcher Hotkey**：默认 `Alt+X`；设置中可选 `Win+Q` 或自定义组合。面板失焦自动关闭，不可配置（与 Q11 共识一致）。
- **设置项**：全局快捷键、软件列表（名称/可执行路径/参数）、历史上限、开机自启（默认关，注册表实现）。
- **预置软件**：PowerShell 必预置；Antigravity、Trae CN 检测到已安装才预置。

## Testing Decisions

- **Seam**：单 seam —— Go 域逻辑包。Wails 绑定、托盘、热键注册、webview UI 均为薄胶水层，只做手工验证，不写单元测试。
- **测试内容**（只测 external behavior，不测实现细节）：
  - RecentFoldersStore：记录、去重、置顶、上限淘汰、持久化往返
  - Search：fuzzy 匹配矩阵（名称/路径、大小写、部分匹配）
  - SettingsStore：默认值、读写往返、损坏文件处理
  - OpenActions：命令构造（纯函数）、经 fake Launcher 的成功/失败路径
- **Prior art**：greenfield，仓库内无既有测试；以上为第一组。

## Out of Scope

- 全盘文件搜索（仅搜索 Recent Folders；输入完整路径直达打开除外，见 story 5）
- CLI 形态（v1 只做托盘 + 面板，ADR-0002）
- 导入系统「最近访问」记录（纯自跟踪）

## Further Notes

- 面板出现在鼠标所在显示器。
- 历史/设置文件损坏时静默重置为默认值，不阻塞启动。
- 空历史时显示空状态提示（story 18）。
