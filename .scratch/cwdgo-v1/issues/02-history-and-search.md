# 02 — 历史存储 + 搜索（domain seam）

**What to build:** Go 域逻辑包中的 RecentFoldersStore 与 Search：自跟踪的访问记录，路径大小写不敏感规范化去重、成功打开后置顶并刷新时间戳、上限 50 淘汰最旧、JSON 持久化（文件损坏时静默重置）；fuzzy 搜索匹配名称与完整路径、大小写不敏感。本 ticket 是约定的单 seam，全部以 TDD 行为级测试交付，不涉及 UI。

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

- [ ] 记录访问：新条目追加、已有条目按规范化路径去重置顶并刷新时间戳
- [ ] 上限 50：超出时淘汰最旧的条目
- [ ] 持久化往返：写盘后重读内容一致；损坏文件静默重置为默认值
- [ ] 搜索：fuzzy、大小写不敏感，能命中名称或完整路径的部分输入，匹配度高的排前面
- [ ] 全部通过行为级测试（external behavior，不测实现细节）
