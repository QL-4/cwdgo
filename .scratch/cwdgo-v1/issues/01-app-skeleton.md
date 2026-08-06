# 01 — 骨架：托盘 + 热键 + 面板窗口

**What to build:** 应用运行后常驻系统托盘（菜单含「打开面板」「退出」）；按下默认 Launcher Hotkey `Alt+X`，在鼠标所在显示器弹出 Launcher Panel，搜索框自动聚焦；`Esc` 或点击面板外部自动关闭；无历史时显示空状态提示。这是贯穿 Wails / systray / 热键 / 窗口的 vertical spine。

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

- [ ] 启动后出现托盘图标，菜单含「打开面板」和「退出」，退出后进程结束
- [ ] 按 `Alt+X` 面板弹出在鼠标所在显示器，搜索框自动聚焦
- [ ] `Esc` 或点击面板外部关闭面板
- [ ] 无历史数据时显示空状态提示
- [ ] 热键注册失败时给出可读的错误提示而不是崩溃
