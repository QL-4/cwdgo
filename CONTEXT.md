# cwdgo

Windows 桌面小工具：常驻系统托盘，通过全局快捷键弹出启动面板，列出最近访问过的文件夹，支持搜索与多种打开方式。

## Language

**Launcher Panel（启动面板）**:
快捷键弹出的临时 UI，列出最近文件夹并支持搜索，失焦自动关闭。
_Avoid_: 面板、弹窗、窗口

**Recent Folders（最近文件夹）**:
按访问时间倒序排列的文件夹列表，是启动面板的核心内容。由 cwdgo 自行跟踪：任何一次打开动作都会把该条目置顶并刷新时间戳，重复项按路径去重。
_Avoid_: 最近打开、历史记录

**Open Action（打开动作）**:
对某个 Recent Folders 条目执行的操作。默认动作是在资源管理器中打开该文件夹；软件列表中的每个应用也是一个可选动作，用数字键触发。
_Avoid_: 打开方式

**Software List（软件列表）**:
用户可配置的应用集合（预置 Windows PowerShell、检测到安装时的 Antigravity / Trae CN），用于以指定应用打开文件夹。
_Avoid_: 应用列表、程序

**Launcher Hotkey（启动快捷键）**:
全局快捷键，默认 `Alt+X`，可在设置中改为 `Win+Q` 或自定义。
_Avoid_: 热键、快捷方式
