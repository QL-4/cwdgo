# 07 — 软件列表管理 UI

**What to build:** 设置窗口中管理 Software List：新增、编辑、删除应用（名称、可执行路径、启动参数）；变更持久化；面板打开时动作列表反映最新配置（序号随之更新）。

**Blocked by:** 05（设置窗口）

**Status:** resolved

- [x] 设置中可新增软件（名称/路径/参数），必填校验（名称、路径非空）
- [x] 可编辑已有条目、可删除条目
- [x] 变更保存后持久化，重启后保留
- [x] 面板中的动作序号与最新配置一致（删除后序号顺延、无空洞）
- [x] 预置条目（PowerShell 等）与用户新增条目可区分，预置条目也可删除

## Answer

交付了软件列表的完整 CRUD：设置视图内新增/编辑/删除，即时持久化，面板按最新配置渲染序号。软件列表从原先的启动期探测（只读）升级为持久化、用户全权管理；首次启动一次性播种探测到的预置，之后完全用户所有。

### 关键设计决策（经 llm-call 二选一）

软件列表的持久化模型放哪——**settings 独立定义 `Software` 类型（选 B）**，而非复用 `openactions.Software`。理由：配置层与执行层演化速率不同，独立类型避免一方改动破坏另一方的存储契约；settings 包不反向依赖 openactions，可独立测试；app.go 薄绑定层做显式转换（`toActionSoftware`），几行代码换 domain 包各自闭合。

### 交付的 artifact

**域层（TDD seam，全绿）**
- `domain/settings/settings.go`（改）：`Settings` 加 `Software []Software` 字段；新增 `settings.Software{Name,Exe,Args}` 类型；`Validate` 增加软件校验（名称/路径 TrimSpace 后非空）。
- 测试（`settings_test.go` 改）：software 字段往返、空名/空路径拒绝、无 args 接受、默认空、全量替换（不清空软件）、`assertSettingsEqual` helper（slice 不可直接比较，逐字段断言）。

**胶水层（手工验证）**
- `app.go`（改）：software 列表改为从 settings store **动态读取**（不再静态 slice）；`GetSoftwareList`/`OpenWith` 读 settings 转换；`SaveSettings` 改 merge（保留 software，不再清空——这是 05 引入 software 字段后必须的修复）；新增 `AddSoftware`/`UpdateSoftware`/`DeleteSoftware` 绑定 + `mutateSoftware` helper（读-改-写 + 复制 slice 防 alias）+ `toActionSoftware` 转换；`defaultSoftware()` 返回 `[]settings.Software`。
- `main.go`（改）：`seedSoftware()` 首次启动播种（settings 无 software 时一次性写入探测结果，之后用户全权管理，不再因卸载重置）；`NewApp` 签名简化（去掉静态 software 参数）。
- 前端 `index.html` / `main.js` / `style.css`：设置视图加软件列表区域——序号 kbd（1-9）、名称/路径展示、编辑/删除；「+ 新建」按钮触发表单（默认隐藏）；编辑表单 2 行（名称+路径 / 参数+保存+取消）；CRUD 即时持久化 + 重绘；空校验 + 错误提示。

### 设置视图 UI 优化（用户验收时迭代）
- 历史上限 label+input 横排放左、开机自启放右，同一行（`space-between`）。
- 自定义细滚动条（8px、透明轨道、border 色圆角滑块、hover 变 muted），替代原生粗滚动条，应用到 `#results` 与 `#sw-list`。

### 证据（域层机器可复现 + 胶水层真机行为）

- **域层**：`go test ./domain/...` 四包全 PASS（settings 含 software CRUD 校验/持久化测试）；`go vet` clean；`gofmt` clean。
- **构建**：`wails build` 成功；绑定生成 `AddSoftware` / `UpdateSoftware` / `DeleteSoftware`。
- **首次播种**：`%APPDATA%\cwdgo\settings.json` 落盘 `software: [PowerShell(-NoExit ...), Antigravity, Trae CN]`（探测到的预置，args 正确序列化，null args 处理正确）。
- **持久化往返**：用户编辑/增删后 settings.json 反映最新列表，重启 cwdgo 后保留。
- **真机行为**（用户确认「完成了可以收尾」）：软件列表 CRUD、序号顺延、UI 布局、滚动条均符合预期。

### 仍存在的不确定性

- 胶水层（绑定转换、webview CRUD 交互）按架构决策不做单测，主观质量由 fresh Judge 独立评估。
- 参数输入框按空格分词成 `[]string`（`{folder}` 占位符保留字面量）；含空格的单个参数无法表达（v1 无引号语法）。多数 IDE 接受位置参数或单 token 参数，够用；复杂参数需后续支持引号解析。
- 「打开设置后关闭再开面板偶发无反应」用户无法稳定复现；已加诊断日志捕获过一次但未复现，疑为偶发时序问题（未阻塞，留待观察）。
