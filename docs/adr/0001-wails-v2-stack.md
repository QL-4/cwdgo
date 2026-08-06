# Go + Wails v2 技术选型

cwdgo 使用 Go 后端 + Wails v2（WebView2 前端）实现托盘、全局热键和启动面板。考虑过 Fyne（纯 Go 原生 UI），但启动面板需要高质量的列表渲染、搜索高亮和键盘导航，Web 技术栈的开发成本和观感都显著更优；仓库定位为 Go 项目，Wails 保持 Go 单语言后端。
