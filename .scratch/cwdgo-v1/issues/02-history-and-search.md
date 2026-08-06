# 02 — 历史存储 + 搜索（domain seam）

**What to build:** Go 域逻辑包中的 RecentFoldersStore 与 Search：自跟踪的访问记录，路径大小写不敏感规范化去重、成功打开后置顶并刷新时间戳、上限 50 淘汰最旧、JSON 持久化（文件损坏时静默重置）；fuzzy 搜索匹配名称与完整路径、大小写不敏感。本 ticket 是约定的单 seam，全部以 TDD 行为级测试交付，不涉及 UI。

**Blocked by:** None — can start immediately

**Status:** resolved

- [x] 记录访问：新条目追加、已有条目按规范化路径去重置顶并刷新时间戳
- [x] 上限 50：超出时淘汰最旧的条目
- [x] 持久化往返：写盘后重读内容一致；损坏文件静默重置为默认值
- [x] 搜索：fuzzy、大小写不敏感，能命中名称或完整路径的部分输入，匹配度高的排前面
- [x] 全部通过行为级测试（external behavior，不测实现细节）

## Answer

已交付，TDD（red → green）完成，24/24 行为级测试通过。

- `domain/recentfolders/`：`Store`（`New` / `Record` / `All`）+ `Entry`（`Name()`）；`MaxEntries = 50`。路径 clean 后大小写不敏感去重（保留首次出现的 casing）、置顶并刷新时间戳；超限淘汰最旧；`Record` 自动原子写盘（temp + rename），损坏/缺失/不可读文件静默重置为空；加载时同样裁剪超限文件；持久化失败返回 error 但内存状态保留。
- `domain/search/`：`Search(entries, query)` — rune 级 fuzzy 子序列匹配、大小写不敏感，匹配名称 + 完整路径；排序：名称匹配 > 仅路径匹配 → exact > prefix > substring > fuzzy → 更早 start → 更少 gaps → 更短目标；同分保持输入顺序（stable）；空 query 返回全部；不修改输入。
- 证据：`go test -count=1 ./...` PASS（24 tests）、`go vet ./...` clean、`gofmt -l` 无输出；磁盘 JSON 格式人工核验过（casing 保留、CJK 路径正常）。
- 备注：本机无 gcc，`-race` 无法运行（cgo）；模块 `cwdgo`（go 1.26，零外部依赖），Go 1.26.5 经 winget 安装。未提交 branch。
