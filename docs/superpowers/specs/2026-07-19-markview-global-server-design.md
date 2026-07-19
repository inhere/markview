# MarkView Global Server 设计

| 日期 | 修订人 | 变更 |
| --- | --- | --- |
| 2026-07-19 | Codex | 初版设计：单个 global server 原生服务所有已登记项目。 |

相关文档：

- [TODO 需求](../../TODO.md)
- [实施计划](../plans/2026-07-19-markview-global-server.md)
- [UI/UX 修复设计](2026-07-19-markview-ui-ux-fixes-design.md)

## 背景

MarkView 已通过全局 `markview-projects.json` 保存项目路径、名称、端口和添加时间，也支持 `--projects` 管理命令及 `--project/-P` 启动指定项目。现有 HTTP server 仍是单项目模型：启动阶段把一个目录写入全局 `config.Cfg.TargetDir`，页面、文件树、搜索、watcher 和 SSE 都读取该全局状态。

FEA002 要求新增 global 模式：用户不需要从某个项目目录启动，访问首页即可查看所有已登记项目；进入任一项目后，文档、目录、搜索和实时刷新都在该项目范围内工作，并可通过 topbar 返回项目主页。

## 目标

1. `markview --global` 启动一个 HTTP server，原生服务 registry 中的全部有效项目。
2. `/` 渲染项目卡片主页。
3. `/p/{project-id}/...` 提供项目文档、文件树、搜索、raw、preview 和 SSE。
4. 项目 runtime 首次访问时懒加载，后续复用到 global server 退出。
5. 多项目并发请求、watcher 和 SSE 完全隔离。
6. 单项目启动继续复用同一套项目 handler，不维护两套内容服务实现。
7. global 模式默认仅本机访问；显式公开时输出多项目暴露警告。
8. 项目根目录和符号链接使用统一的真实路径边界检查。

## 非目标

1. 不把 global server 做成已有独立项目 server 的反向代理门户。
2. 不新增登录、账号、token 或公网部署系统。
3. 不在第一版热重载项目 `.env` 或 `markview.json`。
4. 不实现 runtime 空闲淘汰、LRU、watcher 数量上限或持久化。
5. 不自动 prune、remove 或修改项目 registry。
6. 不修改 `markview-projects.json` 格式。
7. 不在本设计中实现 FEA001 单页分享。

## 总体架构

采用请求级 `ProjectServer` 和进程级 `ProjectManager`：

```text
GlobalServer
  ├─ ProjectManager
  │    └─ map[projectID]*ProjectRuntime
  │          ├─ immutable Config
  │          ├─ ProjectServer
  │          ├─ Watcher
  │          └─ EventHub
  └─ HTTP mux
       ├─ /static/
       ├─ /
       └─ /p/{id}/... → ProjectRuntime.ProjectServer
```

不采用以下方案：

- 只做项目导航页：项目独立 server 未运行时链接无效，不能满足原生多项目服务。
- 仅为每个项目挂载现有子 mux：现有 handler 仍读取全局 `config.Cfg`，并发时会串项目。
- 反向代理内部子 server：会产生多个监听端口、额外 URL 重写和生命周期管理，不是单 server 原生模型。

核心对象：

```go
type ProjectRuntime struct {
    ID      string
    Path    string
    Config  config.Config
    Server  *handlers.ProjectServer
    Watcher *handlers.Watcher
    Events  *handlers.EventHub
}

type ProjectManager struct {
    mu       sync.Mutex
    runtimes map[string]*ProjectRuntime
}
```

`ProjectRuntime.Config` 创建后不可变。handler、watcher 和 SSE 不得再通过修改包级 `config.Cfg` 切换项目。

## Project ID 与 registry 索引

### 稳定 ID

`project-id` 使用规范化项目绝对路径 SHA-256 的前 12 个小写十六进制字符：

```text
D:\work\docs\project-a
    → normalize
    → sha256
    → 7f3a92c84d10
```

规范化过程复用现有 `projects.ProjectKey()` 的 `filepath.Abs` 和 `filepath.Clean` 结果；Windows 下在计算哈希前统一路径大小写，避免同一路径仅因盘符或目录名大小写不同而生成多个 ID。哈希输入为规范化路径的 UTF-8 字节，不解析 symlink，也不写回 registry。

选择原因：

- 同一路径在重启后保持稳定。
- 不在 URL 暴露完整本地路径。
- 不依赖项目名称唯一。
- 只使用 Go 标准库。
- 不需要修改 registry JSON。

新增能力：

```go
func StableID(targetDir string) (string, error)
func BuildIndex(registry Registry) (ProjectIndex, error)
```

`BuildIndex` 必须检测短哈希碰撞。检测到碰撞时返回错误，不能把两个项目静默映射到同一 URL。

### Registry 刷新

global server 不持有永久 registry 快照：

- 请求 `/` 时重新读取 registry。
- 请求 `/p/{id}/...` 时也通过当前 registry 解析 ID。
- 外部执行 `--projects remove/prune` 后，刷新页面即可生效。
- 被删除项目即使已有缓存 runtime，也立即从路由上不可访问。
- 已创建但被删除的 runtime 第一版保留到进程退出时关闭。
- registry 读取失败时不使用过期快照。

registry 文件预计很小；第一版不增加 registry watcher 或缓存失效逻辑。

## Global 监听配置

global server 和项目 runtime 的配置作用域严格分开：

```text
Global server:
  port
  private/listen address
  no-browser

Project runtime:
  entry
  watch
  watch_dir
  watch_skip_dir
  include_dir
  preview_exts
  iframe_hosts
  layout
```

CLI 规则：

- `--global` 与 `--project/-P` 互斥。
- `--global` 与 positional directory/entry 互斥。
- `--global` 与 `--projects list/show/remove/prune` 管理命令互斥。
- 未指定端口时优先使用 6100；占用时寻找下一个可用端口。
- global listener 端口不写回任何项目 registry 记录。
- `--global` 默认等同于 private，只监听 `127.0.0.1`。
- 只有显式 `--private=false` 才公开监听。
- 公开监听时输出醒目警告，说明所有已登记项目可能被访问。

单项目启动的现有端口记忆和默认监听行为保持不变。

## 项目配置加载

配置解析从“写入全局变量”改为“返回独立配置值”：

```go
type ProjectLoadOptions struct {
    GlobalMode bool
}

func LoadProjectRuntimeConfig(
    targetDir string,
    options ProjectLoadOptions,
) (config.Config, error)
```

项目内容配置继续按现有顺序合并：

```text
CLI 项目内容选项
> 项目 .env
> 项目 markview.json
> 全局 markview.json
> 内置默认值
```

global 模式下：

- 项目 `.env` 和 `markview.json` 仍参与 entry、watch、目录过滤和 UI 配置。
- 项目 `MKVIEW_PORT`、`server.port` 和 `server.private` 不影响共享 listener。
- 被忽略的网络配置通过 debug 日志说明。
- 单个项目配置错误只导致该 runtime 创建失败，不影响主页和其他项目。
- runtime 配置创建后保持不可变。
- 配置文件修改后需要重启 global server 才重新加载该项目 runtime。

## Runtime 懒加载与并发

项目 runtime 首次访问时创建，并复用到进程退出：

```text
未加载 → 加载中 → 已就绪
```

同一项目的并发首次请求必须共享一次初始化：

```go
type runtimeSlot struct {
    ready   chan struct{}
    runtime *ProjectRuntime
    err     error
}
```

流程：

1. 第一个请求在 manager 中创建 slot。
2. 配置读取、路径解析和 watcher 初始化在 mutex 外执行。
3. 后续同项目请求等待 `ready`。
4. 初始化完成后写入 runtime 或 error，并关闭 `ready`。
5. 初始化成功后持续复用。
6. 初始化失败不永久缓存；下一次请求允许重试。
7. 不同项目可并发初始化，不能被一个慢目录串行阻塞。

`ProjectManager.Close()` 阻止新 runtime 创建，并幂等关闭全部已创建 runtime。

## ProjectServer

单项目和 global 模式统一使用：

```go
type ProjectServer struct {
    Config Config
    Root   ProjectRoot
    Events *EventHub
    Assets fs.FS
}

func (s *ProjectServer) ServeHTTP(w http.ResponseWriter, r *http.Request)
func (s *ProjectServer) Close() error
```

项目内路由继续使用现有相对结构：

```text
/
/api/search
/api/file-tree
/sse
/docs/guide.md
```

global mux 去掉 `/p/{id}` 前缀后交给 `ProjectServer`，减少页面、搜索和文件树的重复实现。

## Watcher 与 EventHub

每个 runtime 拥有独立 watcher 和 SSE hub：

```text
ProjectRuntime
  ├─ Watcher → EventHub
  └─ /sse    → EventHub subscription
```

规则：

- watcher 只监听该项目配置允许的目录。
- watcher 只向该项目 EventHub 发布事件。
- SSE 消息中的文件路径保持项目内相对路径，不暴露绝对路径。
- `watch=false` 时不创建 watcher；SSE 可以连接并保持 idle。
- watcher 初始化失败视为 runtime 创建失败，不静默降级。
- watcher、EventHub 和 runtime 的关闭操作必须幂等。
- global server shutdown 时统一关闭全部 runtime。

SSE 消息格式保持不变：

```json
{
  "type": "reload",
  "files": ["docs/guide.md"],
  "action": "create"
}
```

## HTTP 路由

global mux 暴露：

```text
/static/...                 全局静态资源
/                           项目主页
/p/{id}/                    项目入口
/p/{id}/path/to/doc.md      项目文档
/p/{id}/api/search          项目搜索
/p/{id}/api/file-tree       项目文件树
/p/{id}/sse                 项目 SSE
```

路由约束：

- `{id}` 必须是 12 位小写十六进制字符。
- `/p/{id}` 使用 308 跳转到 `/p/{id}/`。
- 非法 ID 直接返回 404，不读取项目路径。
- 不允许通过 query、header 或 body 指定本地项目路径。
- SSE 建连前完成项目解析；失败时返回普通 HTTP 错误。

## 项目根目录安全

所有页面、raw、静态文件、搜索和文件树复用一个真实路径解析入口：

```go
type ProjectRoot struct {
    DisplayPath string
    RealPath    string
}

func (root ProjectRoot) Resolve(urlPath string) (string, error)
```

处理顺序：

1. URL path 只解码一次。
2. 拒绝 NUL、无效编码和平台非法路径。
3. 使用 `filepath.Clean` 与 `filepath.Join` 构造候选路径。
4. 解析项目根和候选路径的 symlink、junction/reparse point 实际路径。
5. 使用 `filepath.Rel` 验证目标仍位于真实项目根内。
6. 越界统一返回 `ErrPathOutsideProject`。
7. 不存在目标验证最近存在父目录真实路径后返回 404。
8. 搜索和文件树跳过指向根目录外的 symlink，不遍历、不索引。

允许真实目标仍位于项目根内的 symlink；拒绝指向 registry 项目根之外的 symlink。Windows 路径比较必须使用平台适配的规范化，不能用简单字符串前缀。

## Frontend basePath

项目页面注入：

```json
{
  "basePath": "/p/7f3a92c84d10",
  "previewExts": [".md", ".json"],
  "iframeHosts": [],
  "layout": "compact"
}
```

单项目模式注入空 `basePath`。

前端提供统一构造函数：

```ts
export function projectURL(path: string, basePath: string): string
```

示例：

```text
projectURL("/")              → /p/{id}/
projectURL("/api/search")    → /p/{id}/api/search
projectURL("/api/file-tree") → /p/{id}/api/file-tree
projectURL("/sse")           → /p/{id}/sse
projectURL("/docs/a.md")     → /p/{id}/docs/a.md
```

以下 URL 必须统一迁移：

- 文件树和 file-tree API。
- 内容搜索 API 和结果跳转。
- SSE。
- Markdown 内部链接。
- split preview。
- raw/source 链接。
- 文件变动 Toast。
- history push/replace/popstate。

`/static/`、favicon、JS 和 CSS 继续使用 server 全局路径。

## Global 主页

使用独立服务端模板 `web/template-projects.html`，不加载完整文档阅读器应用。

项目卡片显示：

- 项目名称。
- 简化后的路径。
- global 项目 URL。
- added 时间。
- 可用或目录不存在状态。
- 可用项目的进入链接。
- 无效项目的禁用入口和 prune/remove 提示。

主页不展示旧端口，不探测独立 server，不自动 prune。

空 registry 显示 CLI 引导：

```text
尚无已保存项目
先在项目目录运行 markview，或使用 markview --projects list 查看记录
```

## 项目 Topbar

global 项目页面在阅读布局顶部显示：

```text
← Projects | 项目名称 | 简化路径
```

规则：

- 返回链接固定为 `/`。
- 名称和路径由服务端数据注入。
- Markdown 和目录列表显示 topbar。
- raw、API、SSE 和静态响应不显示。
- 单项目模式不显示。
- topbar 作为布局行参与高度计算，不覆盖正文；sidebar 和正文使用其下方的剩余高度。
- toc-middle、toc-right 仍相对内容区底部定位，保持现有 16px 下边距；preview 开启时继续按现有规则横向避让。
- 已完成的 TOC 左下、右下控制按钮行为保持不变。

## 无效项目和错误响应

无效项目保留在主页：

- 目录不存在时显示错误状态并禁用进入。
- 不自动删除 registry 记录。
- 不把旧端口当作在线状态。
- 卡片排序复用 `projects.List()`。

HTML 页面错误：

- registry 读取失败：500 global 错误页。
- 项目不存在或已移除：404，提供返回主页链接。
- 项目目录不存在：404，提示 prune/remove。
- 项目配置无效：500，显示配置文件路径和安全处理后的错误。
- 路径越界：403，不显示真实目标路径。

API 错误统一返回 JSON：

```json
{
  "error": {
    "code": "project_not_found",
    "message": "Project is not available"
  }
}
```

## Shutdown

关闭顺序：

1. HTTP server 停止接受新请求。
2. `ProjectManager.Close()` 阻止创建新 runtime。
3. 等待或取消正在进行的 runtime 初始化。
4. 关闭全部 watcher。
5. 关闭全部 EventHub 订阅。
6. 汇总并返回 shutdown 错误。

关闭方法必须幂等，正常 shutdown 和初始化失败清理可以安全重复执行。

## 模块改动范围

### `internal/projects`

- `StableID`
- `BuildIndex`
- project-id 解析、目录状态和碰撞检测

### `internal/config`

- `ProjectLoadOptions`
- `LoadProjectRuntimeConfig`
- 去除 handler 对可变全局项目配置的依赖

### `internal/handlers`

```text
project_server.go
project_root.go
event_hub.go
watch_handler.go
page_handler.go
search_handler.go
sse_handler.go
```

### `internal/bootstrap`

```text
global_server.go
project_manager.go
bootstrap.go
```

### `web`

```text
src/app-config.ts
src/util.ts
src/app.ts
src/components/sidebar.ts
src/components/content-search.ts
src/components/link-preview.ts
src/components/live-status.ts
template.html
template-projects.html
src/style/layout.css
```

文件清单是责任边界，不要求为了清单机械创建抽象；实施时优先复用现有文件和函数。

## 实施阶段

### 阶段一：项目标识与路径安全

- StableID 和 registry index。
- 碰撞检测。
- `ProjectRoot.Resolve`。
- symlink/junction 边界测试。
- 不改变页面行为。

### 阶段二：实例化项目 handler

- 配置加载返回独立值。
- `ProjectServer`、EventHub 和可关闭 watcher。
- 单项目模式迁移到 ProjectServer。
- 现有单项目行为必须通过完整测试。

### 阶段三：Global manager 和路由

- `--global` CLI 规则。
- registry 请求时刷新。
- runtime 懒加载和并发初始化。
- `/p/{id}/...`。
- shutdown。
- global 主页和错误页。

### 阶段四：BasePath 与 topbar

- app config 注入 basePath。
- 迁移所有项目级前端 URL。
- topbar。
- sidebar、TOC 和 preview 布局适配。

### 阶段五：集成验证与文档

- 多项目并发验证。
- 搜索、文件树、watcher 和 SSE 隔离。
- 公开监听警告。
- README、配置文档和 TODO。
- desktop/mobile 浏览器 QA。

每个阶段完成后更新实施计划 checkbox，并形成独立提交。

## 测试矩阵

### Registry 与 ID

- 同一路径生成稳定 ID。
- 规范化等价路径使用同一 ID。
- 不同路径生成不同 ID。
- 人工碰撞返回错误。
- registry 增删后下一请求生效。
- 缺失目录显示但不能进入。

### 路径安全

- 正常项目内文件允许。
- `../` 和编码 traversal 被拒绝。
- 相似前缀目录不能越界。
- 根内 symlink 允许。
- 根外 symlink/junction 拒绝。
- 不存在文件验证父目录后返回 404。
- 页面、raw、搜索和文件树边界一致。

### Runtime manager

- 同项目并发首次访问只创建一次。
- 不同项目可并发初始化。
- 初始化失败后可重试。
- registry 删除后缓存 runtime 不可访问。
- Close 幂等。
- 关闭后不能创建 runtime。

### 项目隔离

使用至少两个临时项目验证：

- entry 和 UI 配置独立。
- 同名文件返回各自内容。
- A 搜索不返回 B。
- A 文件树不包含 B。
- A 文件变动只发送给 A 的 SSE。
- 并发请求不串 Config、Root 或页面数据。

### HTTP 与前端

- canonical 308。
- 非法、未知和失效 ID。
- 单项目空 basePath 行为不变。
- API、SSE、文件树、搜索、preview、raw、Toast 和 history 保留项目前缀。
- topbar 返回主页。
- 有 topbar 时 TOC 底部控制按钮仍正确。
- mobile 不发生 topbar/sidebar/preview 重叠。

### 质量门禁

```bash
go test ./...
cd web && bun test
bun run build
```

再启动包含至少两个项目的 global server，通过真实浏览器验证项目切换、搜索、文件变动和返回主页。

## 迁移与兼容

- 不修改 registry JSON。
- 不改变 `markview [directory]` 和 `--project/-P` 的用户行为。
- 不改变项目配置文件格式。
- global 模式明确忽略项目 port/private。
- `basePath` 默认空，单项目 URL 保持现状。
- global URL 是新增入口，不迁移旧书签。
- 不持久化 runtime 状态。

## 实施前置条件

当前 `go test ./...` 存在 4 个 dotenv/bootstrap 基线失败，表现为项目 `.env` 中的 port、debug 和 entry 未生效。FEA002 阶段二会重构同一配置路径，因此实施前必须先单独诊断并修复该回归，形成独立提交；否则不能可靠判断 global 配置改造是否引入新问题。
