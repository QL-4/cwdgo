# 04 — 软件列表 + 数字键动作

**What to build:** Open Actions 的软件侧：OpenActions 执行器通过注入的 Launcher 接口启动应用（测试用 fake 替换）；Software List 预置 Windows PowerShell（必有）、Antigravity 与 Trae CN（检测到已安装才预置）；面板条目上显示带序号的软件动作，按 `1-9` 用对应应用打开选中文件夹；经软件打开同样记录访问历史。命令构造为纯函数，单独测试。

**Blocked by:** 03（面板列表与交互）

**Status:** ready-for-agent

- [ ] 预置列表：PowerShell 始终存在；Antigravity/Trae CN 仅已安装时出现
- [ ] 每个软件动作在面板条目上带序号可见
- [ ] 按 `1-9` 用对应应用打开选中文件夹，配置的参数被正确传入
- [ ] 经软件打开成功后同样记录访问（置顶+刷新时间戳）
- [ ] Launcher 可注入 fake；命令构造与成功/失败路径均有测试
- [ ] 启动失败（应用不存在/路径无效）给出可读提示且不崩溃
