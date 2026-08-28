<p align="center">
  <img src="build/appicon.png" width="120" height="120" alt="cwdgo" />
</p>

<h1 align="center">cwdgo</h1>

<p align="center">
  <strong>别再翻资源管理器了。两次按键，跳到任意文件夹。</strong><br/>
  常驻系统托盘、键盘优先的「最近文件夹」启动器。
</p>

<p align="center">
  <a href="https://github.com/QL-4/cwdgo/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/QL-4/cwdgo/actions/workflows/ci.yml/badge.svg" /></a>
  <a href="https://github.com/QL-4/cwdgo/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/QL-4/cwdgo?display_name=tag" /></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-blue.svg" /></a>
  <img alt="Platform" src="https://img.shields.io/badge/platform-Windows-0078D4.svg" />
  <img alt="Go" src="https://img.shields.io/badge/Go-1.26-00ADD8.svg" />
  <img alt="Wails" src="https://img.shields.io/badge/Wails-v2-FF6B35.svg" />
  <img alt="Single binary" src="https://img.shields.io/badge/distribution-single%20exe-success.svg" />
</p>

<p align="center">
  <b>简体中文</b> · <a href="README.en.md">English</a>
</p>

---

## 😩 痛点

你每天都在打开那么几个项目文件夹，但每次都是同一套流程：

- **打开资源管理器 → 一层层点进五级目录 → 再来一遍** —— 而这个文件夹你十分钟前才进过。
- **编辑器的「打开文件夹」？** 大多数编辑器只记得住*上一个*项目，切回第二个又得重新找。
- **右键「在终端中打开」** 确实好用，前提是那个文件夹当前正开着。
- **Win+R 粘贴路径回车** —— 够快，但你根本记不住路径，也没有任何工具帮你记。

每个工具都维护*自己*的最近列表，彼此互不相通。于是你在每个应用里，一遍遍地走同样的路径。

**没有一个全局的、键盘驱动的「跳到我刚用过的那个文件夹」按钮。** 直到现在。

---

## ✨ cwdgo 做什么

cwdgo 是一个轻量、常驻的 **Windows 系统托盘应用**，把你碰过的每个文件夹都放在一个快捷键之外 —— 在*任意*应用中，用*任意*你选定的工具打开。

> ![cwdgo 主面板](docs/screenshots/%E4%B8%BB%E9%9D%A2%E6%9D%BF.png)

### 🎯 一个快捷键，零摩擦

在任何地方按 **`Alt+X`**（或**左键点击托盘图标**）。浮动面板会贴到鼠标所在显示器上弹出，搜索框已自动聚焦。开始输入 —— **模糊搜索**会按名称*和*完整路径过滤你的最近文件夹。按 `1`–`9` 用对应的编辑器/终端打开选中项，或按 `Enter` 记录它。`Esc` 或点击面板外部立即关闭。

```
        ┌─────────────────────────────────────┐
        │  🔍 powers_                         │
        ├─────────────────────────────────────┤
        │  D:\Work\cwdgo                      │
        │  F:\Playground\cwdgo                │
        │  D:\Work\powershell-scripts    [1]  │  ← 按 1 → PowerShell
        │  D:\Work\my-app                [2]  │  ← 按 2 → Antigravity
        │  D:\Work\another               [3]  │  ← 按 3 → Trae CN
        └─────────────────────────────────────┘
```

### 🔢 用*任意*工具打开 —— 不只是资源管理器

列表中每个文件夹都带着对应**软件列表**的数字角标。按下数字键，文件夹就在那个应用中打开 —— `{folder}` 占位符会把路径作为参数传入：

> ![输入路径，按数字键，在你的工具里打开](docs/screenshots/%E6%93%8D%E4%BD%9C%20GIF.gif)

PowerShell、你的编辑器、你的终端 —— 全都共享**同一份全局最近列表**，所以你在 VS Code 里打开的文件夹，在终端里也能立刻够到。

### 📋 输入路径，它会记住

粘贴或输入一个完整路径然后按 `Enter` —— cwdgo 会**把它记录到历史顶部**（不弹资源管理器，面板保持打开），下次就是一个按键的距离。首次运行时历史为空？靠输入就能把它养起来。

### 🗂️ 智能合并搜索

- **最近文件夹**按名称与完整路径模糊过滤（不区分大小写），保持最近使用顺序。
- **文件系统补全**：输入部分路径会实时列出匹配的子文件夹，并标上 **`新`** 角标，让你一眼区分新发现与已知项。
- 历史记录与补全结果自动合并去重。

> ![带「新」角标的路径补全](docs/screenshots/%E6%90%9C%E7%B4%A2.png)

### ⚙️ 合理的默认值，完全可配置

内置设置面板（无保存按钮 —— 所有改动自动持久化）：

> ![设置页 —— 软件列表增删改查](docs/screenshots/%E8%AE%BE%E7%BD%AE%E9%A1%B5.png)

- **软件列表增删改查** —— 添加/编辑/删除启动器，可配置名称、路径与参数。
- **历史上限**（默认 50）—— 保持列表精简。
- **开机自启**开关（默认关闭）。

PowerShell 始终预置；Antigravity 与 Trae CN 在检测到安装时自动预置。

### 📌 常驻托盘

cwdgo 在后台静默运行。**左键点击托盘图标**切换面板；**右键**打开菜单 —— 打开、设置、退出。不占任务栏，无需管理窗口。

> ![托盘菜单](docs/screenshots/%E6%89%98%E7%9B%98%E8%8F%9C%E5%8D%95.png)

---

## 🚀 安装

### 方式 A —— 下载二进制文件

从[最新 Release](https://github.com/QL-4/cwdgo/releases/latest) 下载 `cwdgo.exe`，放到任意位置双击运行。就这样 —— 它会驻留在托盘里。

每个 Release 都附带 `cwdgo.exe.sha256`，如需校验下载：

```powershell
(Get-FileHash cwdgo.exe -Algorithm SHA256).Hash.ToLower()
```

> 由于二进制未做代码签名，Windows SmartScreen 首次运行时可能会警告 —— *更多信息 → 仍要运行*。

### 方式 B —— 从源码构建

**前置依赖：**[Go 1.26+](https://go.dev/dl/) 和 [Node.js](https://nodejs.org/)（用于前端）。

```bash
git clone https://github.com/QL-4/cwdgo.git
cd cwdgo

# 安装 Wails CLI（一次性；与 CI 构建所用版本一致）
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0

# 构建单文件 exe
wails build
# → build/bin/cwdgo.exe
```

然后运行 `build/bin/cwdgo.exe`。在 Windows 上也可以用仓库自带的辅助脚本：

```bash
./build.sh   # 杀掉旧实例、重新构建并启动
```

---

## ⌨️ 键盘快捷键

| 按键                | 动作                                |
| ------------------- | ----------------------------------- |
| `Alt+X`             | 在任意位置切换面板（打开 / 关闭）   |
| `1`–`9`             | 用软件列表第 N 项打开选中的文件夹   |
| `Enter`             | 将输入/选中的路径记录到历史         |
| `↑` / `↓`           | 移动选中项                          |
| `Esc` / 点击外部    | 关闭面板                            |
| **左键点击托盘**    | 切换面板                            |
| **右键点击托盘**    | 菜单：打开 · 设置 · 退出            |

> 提示：点击列表项（或其所在行）会以默认动作在资源管理器中打开。

---

## 🏗️ 工作原理

```
            ┌──────────────────────────────────────────┐
            │            Wails webview (UI)             │
            │       搜索框 · 列表 · 设置面板            │
            └───────────────────┬──────────────────────┘
                                │ Wails bindings（薄胶水层）
            ┌───────────────────┴──────────────────────┐
            │                 app.go                    │  ← 编排
            └───┬──────────┬───────────┬──────────┬─────┘
                │          │           │          │
        ┌───────▼──┐ ┌─────▼────┐ ┌───▼────┐ ┌───▼──────┐
        │  domain  │ │  domain  │ │ domain │ │ internal │
        │ recent-  │ │  search  │ │settings│ │ launcher │
        │ folders  │ │ (模糊)   │ │        │ │ (Win32)  │
        └──────────┘ └──────────┘ └────────┘ └──────────┘
              │
        ┌─────▼───────────────────────────────────────┐
        │  %APPDATA%\cwdgo\  (history.json + settings) │
        └──────────────────────────────────────────────┘
```

**设计原则：领域逻辑纯粹且完全解耦。** 所有行为都在 `domain/` 包中（`recentfolders`、`search`、`settings`、`openactions`），零 Wails / Win32 / systray 依赖 —— 纯 Go、有单元测试，UI 层只是一层薄视图。托盘、快捷键与 Win32 互操作都隔离在 `internal/` 中。

- **无数据库** —— `%APPDATA%\cwdgo\` 下两个 JSON 文件。
- **无后台索引** —— cwdgo 只跟踪*你通过它打开*的文件夹。
- **单一二进制**，原生 Win32 托盘（无 Electron，无第三方托盘库）。

<details>
<summary><b>📦 项目结构</b></summary>

```
cwdgo/
├── main.go                  # 入口：托盘 + 快捷键 + Wails 应用
├── app.go                   # Wails bindings（到 domain 的薄胶水层）
├── domain/                  # 纯粹、有测试的业务逻辑
│   ├── recentfolders/       #   历史存储（记录/去重/上限/持久化）
│   ├── search/              #   模糊搜索 + 保序过滤
│   ├── settings/            #   设置存储 + 软件列表
│   └── openactions/         #   打开动作命令 + 软件模型
├── internal/                # 平台胶水层（无测试，人工验证）
│   ├── tray/                #   原生 Win32 托盘（左键切换，右键菜单）
│   ├── hotkey/              #   基于 RegisterHotKey 的全局 Alt+X
│   ├── panel/               #   窗口定位与失焦处理
│   ├── launcher/            #   纯净环境 CreateProcessW + IDE 检测
│   ├── folderscan/          #   文件系统路径补全
│   ├── win32/               #   NOTIFYICONDATA、弹出菜单等
│   ├── icon/                #   内嵌图标 → HICON
│   └── applog/              #   发布构建的文件日志
├── frontend/                # 原生 JS 视图（渲染 + 键盘导航）
│   └── src/
├── docs/                    # ADR + agent 文档
└── .scratch/                # 设计规格 + issue 工单（已纳入版本控制）
```

</details>

---

## 🧪 测试与质量

领域层采用测试先行开发。只测纯逻辑 —— 不对 Win32 或 UI 做 mock。

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

每个 issue 都以 markdown 工单形式记录在 `.scratch/cwdgo-v1/issues/` 下，包含明确的 **Problem → Solution → Evidence** 说明；架构决策记录在 [`docs/adr/`](docs/adr)。

---

## ❓ 常见问题

**只支持 Windows 吗？**
是的。v1 是 Windows 优先的（全局快捷键、Win32 托盘、注册表自启）。领域层是可移植的，平台胶水层还不是。

**它会监视我打开了什么吗？**
不会。cwdgo 只记录你*通过 cwdgo 本身*打开的文件夹，从不扫描你的文件系统历史，也不监听其他应用。

**我的数据在哪？**
`%APPDATA%\cwdgo\` —— 两个纯文本 JSON 文件（`history.json`、`settings.json`）。人类可读，易于备份或删除。文件损坏时会静默重置为默认值。

**为什么不直接用 PowerToys Run / Listary / Everything？**
那些是*文件*搜索工具 —— 它们索引整块磁盘。cwdgo 是*时近性*工具：它呈现你每天真正在轮换使用的那几个文件夹，按最后访问时间排序，并支持一键「在应用 N 中打开」。不同的肌肉记忆，不同的问题。

---

## 🤝 参与贡献

欢迎提 Issue 和 PR。代码库刻意保持小巧且接缝清晰 —— 领域逻辑是隔离且有测试的，所以大多数行为变更都是单包内的 TDD 循环。设计规格与待办工单见 `.scratch/`。

---

## 📄 许可证

[MIT](LICENSE) © 2026 QL-4
