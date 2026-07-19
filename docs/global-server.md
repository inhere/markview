# MarkView Global Server

## 修订记录

| 日期 | 修订人 | 变更 |
| --- | --- | --- |
| 2026-07-19 | Codex | 初版：单个 global server 服务全部已保存项目。 |

相关文档：

- [设计文档](superpowers/specs/2026-07-19-markview-global-server-design.md)
- [实施计划](superpowers/plans/2026-07-19-markview-global-server.md)
- [项目管理 CLI 设计](superpowers/specs/2026-05-14-projects-cli-design.md)

## 用途

`markview --global` 启动一个 HTTP server，并在主页列出 `markview-projects.json` 中保存的全部项目。访问项目卡片时，MarkView 按需初始化该项目的配置、文件边界、搜索、SSE 和 watcher；不需要为每个项目单独启动进程。

```text
http://127.0.0.1:6100/                 项目列表
http://127.0.0.1:6100/p/{id}/          项目首页
http://127.0.0.1:6100/p/{id}/docs/a.md 项目文档
```

项目页面顶部会显示 `← Projects`、项目名称和简化路径。项目内的文件树、搜索、raw 链接、相对图片、预览、SSE 和浏览历史都保持在对应的 `/p/{id}` 路径下。

## 启动方式

使用默认端口和安全监听地址：

```bash
markview --global
```

指定端口：

```bash
markview --global --port 6200
```

不自动打开浏览器：

```bash
markview --global --no-browser
```

只有明确需要让其他设备访问时，才公开监听：

```bash
markview --global --private=false
```

> `--private=false` 会监听所有网络接口。请只在可信网络中使用，并由防火墙或反向代理限制访问；global 模式本身不提供身份认证。

## 安全默认值与参数规则

- 默认只监听 `127.0.0.1`。即使全局配置文件包含 `"private": false`，global 模式也不会自动公开服务。
- 只有命令行显式传入 `--private=false` 才监听所有接口。
- 端口优先级为：命令行 `--port` > `MKVIEW_PORT` > 全局 `markview.json` > `6100`。
- `--global` 不能与目录、entry file、`--project/-P` 或 `--projects` 同时使用。
- 项目内的 `server.port`、`server.private` 和对应 `.env` 值在 global 模式下被忽略，避免项目改变共享 listener。
- 项目的 entry、watch、watch_dir、watch_skip、include_dir、preview_exts、iframe_hosts 和 layout 仍按项目分别加载。

## Registry 行为

项目 registry 默认位于：

```text
~/.config/markview/markview-projects.json
```

可以继续使用现有项目管理命令维护它：

```bash
markview --projects list
markview --projects show <name-or-path>
markview --projects remove <name-or-path>
markview --projects prune
```

Global server 会在每次请求主页或项目路由时重新读取 registry，因此：

- 新增项目后，刷新主页即可看到项目卡片；
- 删除项目后，该项目的下一次请求立即返回 `404`，即使其 runtime 曾经加载过；
- 项目目录不存在时，主页保留卡片并禁用入口，不会自动修改或 prune registry；
- 项目 runtime 只在首次访问时懒加载，同一项目的并发首次请求共享一次初始化。

## 项目隔离

每个项目拥有独立的 runtime config、真实文件根、HTTP handler、SSE EventHub 和 watcher。文件请求会校验真实路径边界，拒绝 traversal、根外 symlink/junction 和编码路径分隔符。项目 `.env` 仅解析为该 runtime 的局部数据，不修改 server 进程环境。

静态资源仍使用共享的 `/static/` 路径；项目内容、API 和事件路径使用：

```text
/p/{id}/api/search
/p/{id}/api/file-tree
/p/{id}/sse
/p/{id}/{document-path}?q=raw
```

## 常见问题

### 项目卡片显示不可用

先确认路径仍存在。若项目已删除或移动，使用 `markview --projects remove` 或 `markview --projects prune` 清理 registry；MarkView 不会自动删除记录。

### 项目配置的端口没有生效

这是 global 模式的预期行为。所有项目共享同一个 listener，使用 `markview --global --port <port>`、`MKVIEW_PORT` 或全局 `markview.json` 设置端口。

### 局域网设备无法访问

默认 listener 仅限本机。确认访问风险后显式使用 `markview --global --private=false`，并检查操作系统防火墙规则。
