# Software 条目的 Name 只作显示，能力由命令模板表达

Software List 的 Name 是纯显示字段，任何行为判断不得匹配名字。SSH 项目的打开能力不设独立布尔标记：条目上的 `remoteArgs` 命令模板（与 `args` 同构的占位符机制，新增 `{sshUri}` 占位符承载 `SSHFolderURI` 生成的 `vscode-remote://` URI）**存在即能力** —— Go 的 `Open`/`OpenWith` 与前端 SSH 行的按键过滤都以"有无 `remoteArgs`"为准。预置检测为 Trae CN 播种模板 `["--folder-uri", "{sshUri}"]`（VS Code 系 CLI 通用）；设置界面编辑条目时模板原样保留，重命名不影响行为。

任何能写成命令行的远程打开方式（其他 `--folder-uri` 兼容编辑器、自定义包装脚本等）都可以通过配置模板表达，无需改代码——这是选模板而非枚举 kind 的原因。

历史：最初以 `EqualFold(name, "Trae CN")` 匹配（Go 两处 + 前端一处），用户重命名即破坏功能；中间一度改为布尔 `remote` 标记，因"能力声明与打开命令是两个字段、存在失同步"在发布前重构为本方案。
