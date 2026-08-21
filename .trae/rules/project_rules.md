# cwdgo 项目规则

## 构建

- 涉及运行行为验证的改动，必须重建正式产物并用它实测：

  ```powershell
  C:\Users\jerem\go\bin\wails.exe build   # 产物 build\bin\cwdgo.exe，用户运行的也是它
  ```

- 纯类型检查/单测可用 `go build ./... ; go vet ./... ; go test ./...`
