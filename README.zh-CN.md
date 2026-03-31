# MarkView

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/inhere/markview?style=flat-square)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/inhere/markview)](https://github.com/inhere/markview)
[![Unit-Tests](https://github.com/inhere/markview/actions/workflows/go.yml/badge.svg)](https://github.com/inhere/markview)

---

[English](./README.md) | [简体中文](./README.zh-CN.md)

MarkView 是一个零配置的 Markdown 预览服务器，使用 Go 提供后端，Bun 打包前端资源。

它专注于本地文档阅读体验：快速启动、实时刷新、清晰的侧边导航，以及对 Mermaid / 代码高亮等常见文档内容的良好支持。

## 功能特性

- **🚀 Zero Config**：在任意目录直接启动，默认打开 `README.md`
- **⚡ 单文件服务**：Go 二进制内嵌 `frontend/dist` 和模板，无需额外静态资源部署
- **🔄 Live Reload**：监听 Markdown 变更，通过 SSE 局部刷新页面
- **🧭 双侧边栏导航**：
  - `Files` 文件树，支持目录展开、当前文件高亮
  - `On This Page` 目录，支持滚动高亮
- **🔁 无刷新导航**：
  - 文件树点击不整页刷新
  - 文档正文中的站内 Markdown 链接支持无刷新切换
  - 支持浏览器前进 / 后退
- **📖 分屏预览**：hover 站内 Markdown 链接显示预览按钮，点击后右侧 40% 面板预览目标文档
- **🎨 丰富渲染能力**：
  - GFM（GitHub Flavored Markdown）
  - `highlight.js` 代码高亮
  - `mermaid.js` 图表渲染、源码展开、全屏查看
- **⚙️ 阅读设置**：
  - 页面宽度切换
  - 字体大小调整与重置
  - 设置持久化到 `localStorage`
- **📱 响应式布局**：桌面端侧边栏阅读，移动端自动收敛为单栏

## 安装

```bash
go install github.com/inhere/markview@latest
```

## 使用

### 运行可执行文件

下载并运行 `markview`：

```powershell
# 预览当前目录(可选指定port)
markview [-p PORT]

# 预览指定目录
markview "path/to/docs"

# 预览指定目录，并设置默认入口文件
markview "path/to/docs" "intro.md"
```

默认会启动在 `http://localhost:6100`。

示例文档见 [example/](example/)。

### 配置

可通过环境变量/选项调整端口和默认入口：

```powershell
markview -p 6543
markview . "guide.md"
```

## 开发

### 前提

- **Go** 1.22+
- **Bun** 1.0+

### 项目结构

```text
markview/
├── frontend/           # 前端源码与模板
│   ├── src/
│   │   ├── app.ts              # 页面生命周期、导航、渲染编排
│   │   ├── sidebar.ts          # 文件树与 TOC 逻辑
│   │   ├── mermaid.ts          # Mermaid 增强与全屏交互
│   │   ├── link-preview.ts     # 站内链接分屏预览
│   │   ├── preferences.ts      # 阅读设置持久化
│   │   └── live-status.ts      # SSE 状态显示
│   ├── template.html           # 页面模板与主要样式
│   ├── dist/                   # Bun 打包产物（Go embed）
│   └── package.json
├── main.go                     # Go server 入口
├── handlers.go                 # 缓存相关响应头与 handler 辅助逻辑
├── example/                    # 示例 Markdown 文档
└── README.md
```

### 从源代码构建

1. 安装前端依赖并构建：

```bash
cd frontend
bun install
bun run build
```

这会生成 `frontend/dist/`，并自动复制：
- `highlight.css`
- `logo.svg`
- `favicon.svg`

2. 构建后端：

```bash
cd ..
go build --ldflags "-w -s" -o markview.exe

# 或安装到 GOPATH/bin
go install -ldflags "-s -w" .
```

也可以直接使用 `Makefile`：

```bash
make frontend
make build
make run
```

### 验证

常用验证命令：

```bash
go test ./...
cd frontend && bun test ./src/*.test.ts
cd frontend && bun run build
```

> `go:embed` 会将 `frontend/template.html` 与 `frontend/dist/` 一起打包进最终二进制。

## License

MIT
