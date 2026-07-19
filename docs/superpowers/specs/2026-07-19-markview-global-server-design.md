# MarkView Global Server 设计

| 日期 | 修订人 | 变更 |
| --- | --- | --- |
| 2026-07-19 | Codex | 初版设计：单个 global server 原生服务所有已登记项目。 |
| 2026-07-19 | Codex | 根据独立审核补齐并发配置、URL 解码、watcher 边界、registry 原子写和 shutdown 契约，并收紧非必要范围。 |

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

采用项目级 HTTP handler `ProjectServer` 和进程级 `ProjectManager`：

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

规范化过程先复用现有 `projects.ProjectKey()` 的 `filepath.Abs` 和 `filepath.Clean`，再使用 `filepath.ToSlash` 统一分隔符；Windows 下对整个结果执行 `strings.ToLower`，形成明确且可测试的词法 key。哈希输入为该 key 的 UTF-8 字节，不解析 symlink，也不尝试建立完整的 Windows 文件身份；短哈希碰撞继续由索引检测。ID 不写回 registry。

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

registry 写入必须提供完整快照：在目标文件同目录创建临时文件，写入完整 JSON 后执行 `Sync`、`Close`，再通过同卷 `os.Rename` 替换目标文件；任一步失败都保留旧 registry。请求只接受可完整解析的快照，不能把截断 JSON 当空 registry。该规则同时覆盖 global server 与外部 `--projects add/remove/prune` 进程；不能用仅限当前进程的 mutex 代替原子文件替换。第一版只保证 reader 看到完整快照，不为同时运行的多个 registry 管理命令增加跨进程锁；并发 writer 仍按现有 CLI 使用约束串行执行。

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
- global 端口优先级为显式 `--port`、进程 `MKVIEW_PORT`、全局 `markview.json` 的 `server.port`、6100；候选端口占用时寻找下一个可用端口。
- global listener 端口不写回任何项目 registry 记录。
- `--global` 默认只监听 `127.0.0.1`；`--private=true` 结果相同。
- 只有命令行显式访问 `--private=false` 才公开监听；全局配置、环境变量和项目配置都不能把 global server 自动改成公开监听。
- 公开监听时输出醒目警告，说明所有已登记项目可能被访问。
- `--no-browser` 只控制 global 进程是否打开一次项目主页，不进入项目 runtime 配置。
- global 模式禁止显式使用 entry、watch、watch-dir、watch-skip-dir、include-dir、preview-exts、iframe-hosts 和 layout 等项目内容 CLI flags；每个项目只使用自己的 `.env`、项目配置、全局配置和内置默认值。

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

项目内容配置继续按以下顺序合并：

```text
项目 .env
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
- 项目 `.env` 必须解析到调用局部的 `map[string]string` 并直接参与 merge；整个 runtime 初始化链路禁止调用 `os.Setenv`、`os.Unsetenv` 或任何会把 dotenv 写入进程环境的 API。
- 不同项目的配置加载必须可以安全并发；进程启动时已有的环境变量只读，并按既有环境配置语义参与每个项目的合并。

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
4. 初始化完成后在锁内检查 manager 是否已关闭；未关闭才发布 runtime。
5. 初始化成功后写入 runtime 并关闭 `ready`，随后持续复用。
6. 初始化失败时写入 `slot.err`、关闭 `ready`，再从 map 删除 slot；当前等待者继续通过持有的 slot 指针读取同一错误，后续请求创建新 slot 重试。
7. manager 已关闭时，初始化方立即关闭刚创建或部分创建的资源，不发布 runtime；等待者收到统一的 manager closed 错误。
8. 不同项目可并发初始化，不能被一个慢目录串行阻塞。

`ProjectManager.Close()` 阻止新 runtime 创建，并幂等关闭全部已创建 runtime。

## ProjectServer

单项目和 global 模式统一使用：

```go
type ProjectServer struct {
    Config Config
    Root   ProjectRoot
    Events *EventHub
}

func (s *ProjectServer) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

`ProjectServer` 只处理项目 HTTP 请求，不拥有生命周期。`ProjectRuntime.Close()` 是 watcher 和 EventHub 的唯一关闭入口，并负责部分初始化失败时的清理，避免重复关闭和所有权交叉。

项目内路由继续使用现有相对结构：

```text
/
/api/search
/api/file-tree
/sse
/docs/guide.md
```

global mux 按下文 URL 契约去掉 `/p/{id}` 前缀后交给 `ProjectServer`，减少页面、搜索和文件树的重复实现。

## Watcher 与 EventHub

每个 runtime 拥有独立 watcher 和 SSE hub：

```text
ProjectRuntime
  ├─ Watcher → EventHub
  └─ /sse    → EventHub subscription
```

规则：

- watcher 只监听该项目配置允许的目录。
- watcher 的初始根目录、`watch_dir` 和遍历得到的子目录都必须经过 `ProjectRoot` 真实路径边界验证；根外 symlink、junction 和 reparse point 不得加入 watcher。
- 运行期间收到新目录 Create 事件时，在 `watcher.Add` 前重新解析真实路径并验证边界。
- 文件事件发布到 EventHub 前再次验证目标仍位于项目根内，再转换成项目内相对路径；越界事件直接丢弃并记录 debug 日志。
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

URL 解析采用唯一契约：

1. URL 百分号解码只由 Go `net/http` 完成一次；应用层统一读取 `r.URL.Path`，不得再次调用 `url.PathUnescape`。
2. global mux 从已解码的 `URL.Path` 校验第一个 segment 为 `p`、第二个 segment 为 12 位 ID，余下部分作为项目子路径。
3. 路由前检查 `r.URL.EscapedPath()`；拒绝大小写不敏感的 encoded slash `%2f`、encoded backslash `%5c` 和 encoded NUL `%00`，避免不同层对 segment 边界产生不同解释。无效 `%` 由 `net/http` 在进入 handler 前返回 400。
4. 去掉 `/p/{id}` 后克隆 request 和 URL，只重建已解码的 `URL.Path`，并清空 `RawPath`；`ProjectServer` 接收以 `/` 开头、已解码一次的项目路径。
5. `%252e%252e` 在应用层保持字面量 `%2e%2e`，不会再次变成 `..`；已经解码成 `..` 的 segment 由 `ProjectRoot.Resolve` 拒绝。
6. query 和 fragment 不参与文件路径解析；query 原样保留，fragment 不会发送到 server。

## 项目根目录安全

所有页面、raw、项目资源、搜索、文件树和 watcher 复用一个真实路径解析入口：

```go
type ProjectRoot struct {
    DisplayPath string
    RealPath    string
}

func (root ProjectRoot) Resolve(urlPath string) (string, error)
```

处理顺序：

1. 输入必须是 URL 层已经解码一次、以 `/` 开头的项目路径；`Resolve` 不再执行 URL 解码。
2. 拒绝 NUL、反斜杠 traversal、`.`/`..` traversal segment 和平台非法路径。
3. 使用 `filepath.Clean` 与 `filepath.Join` 构造 display candidate。
4. 解析项目根和 candidate 的 symlink、junction/reparse point 实际路径。
5. 使用 `filepath.Rel` 按 segment 判界：只有 `rel == ".."` 或以 `".." + separator` 开头才算越界，合法的 `..foo` 不受影响。
6. 越界统一返回 `ErrPathOutsideProject`。
7. 不存在目标验证最近存在父目录的真实路径仍在根内后返回 404。
8. `Resolve` 返回经过边界验证的真实目标路径；`Stat`、`ReadFile`、`ServeFile` 等消费者只能使用返回值，不能检查 real path 后再打开 display candidate。
9. 搜索、文件树和 watcher 跳过指向根目录外的 symlink，不遍历、不索引、不监听。

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

前端提供统一构造函数，并从已注入配置读取默认 basePath：

```ts
export function projectURL(path: string, basePath = appConfig.basePath): string
```

示例：

```text
projectURL("/")              → /p/{id}/
projectURL("/api/search")    → /p/{id}/api/search
projectURL("/api/file-tree") → /p/{id}/api/file-tree
projectURL("/sse")           → /p/{id}/sse
projectURL("/docs/a.md")     → /p/{id}/docs/a.md
```

构造规则：

- `path` 可以包含 query 和 hash，构造后必须完整保留。
- 已带当前 `basePath` 的 URL 不重复添加前缀。
- `http:`、`https:`、`data:`、`mailto:` 和 `#fragment` 等外部或非项目 URL 保持原样。
- `/static/`、favicon、JS 和 CSS 是 server 全局资源，不添加项目 basePath。
- 服务端模板使用同一语义的 Go helper 生成项目链接；不能在模板中手工拼接 origin 根路径。

以下 URL 必须统一迁移：

- 文件树和 file-tree API。
- 内容搜索 API 和结果跳转。
- SSE。
- Markdown 内部链接。
- Markdown 相对图片、音视频和其他 `src` 资源。
- 目录列表、download 和服务端模板生成的链接。
- split preview。
- preview/iframe 内继续解析的项目相对资源。
- raw/source 链接。
- 文件变动 Toast。
- history push/replace/popstate。

根相对 Markdown 链接按项目根解释并添加 basePath；外部绝对 URL 保持不变。

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

新增的 global 项目解析错误在 `/p/{id}/api/...` 路径返回 JSON：

```json
{
  "error": {
    "code": "project_not_found",
    "message": "Project is not available"
  }
}
```

现有 search、file-tree 等 endpoint 的成功和错误协议保持不变；本功能不统一改造单项目 API 错误格式。

## Shutdown

关闭顺序：

1. HTTP server 停止接受新请求。
2. `ProjectManager.Close()` 阻止创建新 runtime。
3. 等待全部 in-flight slot 完成；初始化方发现 manager 已关闭时自行关闭部分或完整 runtime，且不得发布到 map。
4. 等待者收到统一的 manager closed 错误，不启动重试。
5. 由各 `ProjectRuntime.Close()` 依次关闭 watcher 和 EventHub。
6. 汇总并返回 shutdown 错误。

manager 使用 closed 状态和 in-flight 计数协调初始化与关闭。`ProjectManager.Close()` 和 `ProjectRuntime.Close()` 必须幂等，并发调用 Close、正常 shutdown 和初始化失败清理都可以安全重复执行。

## 模块改动范围

### `internal/projects`

- `StableID`
- `BuildIndex`
- registry 原子快照保存
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
- registry 原子快照写入和并发读取。
- 唯一 URL 解码与项目子路径契约。
- `ProjectRoot.Resolve`。
- symlink/junction 边界测试。
- 不改变页面行为。

### 阶段二：实例化项目 handler

- 先修复 dotenv/bootstrap 基线，项目 `.env` 解析不得修改进程环境。
- 配置加载返回独立值，并通过多项目并发与 race 测试。
- `ProjectServer`、EventHub 和可关闭 watcher。
- watcher 初始遍历、动态目录和事件发布统一执行真实路径边界检查。
- 单项目模式迁移到 ProjectServer。
- 现有单项目行为必须通过完整测试。

### 阶段三：Global manager 和路由

- `--global` CLI 规则。
- registry 请求时刷新。
- runtime 懒加载和并发初始化。
- `/p/{id}/...`。
- shutdown 与 in-flight 初始化交接。
- global 主页和错误页。

### 阶段四：BasePath 与 topbar

- app config 注入 basePath。
- 迁移所有项目级前端 URL。
- topbar。
- sidebar、TOC 和 preview 布局适配。

### 阶段五：集成验证与文档

- 多项目并发验证。
- 搜索、文件树、watcher 和 SSE 隔离。
- private 默认真实监听地址和公开监听警告。
- README、配置文档和 TODO。
- desktop/mobile 浏览器 QA。

每个阶段完成后更新实施计划 checkbox，并形成独立提交。

## 测试矩阵

### Registry 与 ID

- 同一路径生成稳定 ID。
- 规范化等价路径使用同一 ID。
- Windows 条件测试覆盖盘符大小写、目录名大小写、分隔符和尾随分隔符；UNC 与扩展路径按已定义的词法 key 产生确定结果。
- 不同路径生成不同 ID。
- 人工碰撞返回错误。
- registry 增删后下一请求生效。
- 外部进程原子替换 registry 时，请求只能读取旧或新完整快照，不能读到空文件或截断 JSON。
- 原子替换失败时旧 registry 保持可读；Windows 条件测试覆盖同目录替换。
- 缺失目录显示但不能进入。

### 路径安全

- 正常项目内文件允许。
- `../`、`%2e%2e`、大小写混合编码和反斜杠 traversal 被拒绝。
- `%252e%252e` 不发生二次解码。
- encoded slash、encoded backslash、NUL、无效 `%` 和多余斜杠按 URL 契约处理。
- 相似前缀目录不能越界。
- 根内 symlink 允许。
- 根外 symlink/junction 拒绝。
- 不存在文件验证父目录后返回 404。
- 页面、raw、搜索、文件树和 watcher 边界一致。
- 初始和运行期创建的根外 symlink/junction 都不加入 watcher，也不产生 SSE。

### Runtime manager

- 同项目并发首次访问只创建一次。
- 不同项目可并发初始化。
- 初始化失败后可重试。
- 失败 slot 的现有等待者收到同一错误，之后的新请求创建新 slot。
- registry 删除后缓存 runtime 不可访问。
- Close 幂等。
- 初始化中 Close、多个并发 Close、部分初始化失败和 Close/失败重试竞争都不泄漏 runtime、watcher 或 goroutine。
- 关闭后不能创建 runtime。

### 项目隔离

使用至少两个临时项目验证：

- entry 和 UI 配置独立。
- 两个项目通过 barrier 并发加载不同 `.env`，配置不串项目且进程环境不变。
- 同名文件返回各自内容。
- A 搜索不返回 B。
- A 文件树不包含 B。
- A 文件变动只发送给 A 的 SSE。
- 并发请求不串 Config、Root 或页面数据。

### HTTP 与前端

- canonical 308。
- 非法、未知和失效 ID。
- 无 `--private`、`--private=true`、显式 `--private=false` 和全局配置 `private=false` 分别断言真实 listener 地址；只有显式 false 允许非 loopback。
- global port 按 CLI、`MKVIEW_PORT`、全局配置和 6100 的顺序选择；项目内容 CLI flags 在 global 模式返回明确错误。
- 单项目空 basePath 行为不变。
- API、SSE、文件树、搜索、目录列表、Markdown 图片、download、preview、raw、Toast 和 history 保留项目前缀及 query/hash。
- 外部 URL 和 `/static/` 不添加 basePath，已带 basePath 的 URL 不重复添加。
- topbar 返回主页。
- 有 topbar 时 TOC 底部控制按钮仍正确。
- mobile 不发生 topbar/sidebar/preview 重叠。

### 质量门禁

```bash
go test ./...
go test -race ./internal/bootstrap/... ./internal/projects/... ./internal/handlers/...
cd web && bun test
bun run build
```

Go race 检查在支持的平台执行；Windows 至少运行相同并发行为测试。再启动包含至少两个项目的 global server，通过真实 listener 地址断言默认绑定 loopback，并用真实浏览器验证项目切换、搜索、相对资源、文件变动和返回主页。单项目回归必须覆盖 search/file-tree timeout、SSE 长连接、静态缓存头、页面 no-store 和现有站内导航。

## 迁移与兼容

- 不修改 registry JSON。
- 不改变 `markview [directory]` 和 `--project/-P` 的用户行为。
- 不改变项目配置文件格式。
- global 模式明确忽略项目 port/private。
- `basePath` 默认空，单项目 URL 保持现状。
- global URL 是新增入口，不迁移旧书签。
- 不持久化 runtime 状态。

## 实施前置条件

当前 `go test ./...` 存在 4 个 dotenv/bootstrap 基线失败，表现为项目 `.env` 中的 port、debug 和 entry 未生效。FEA002 阶段二会重构同一配置路径，因此实施前必须先单独诊断并修复该回归，形成独立提交；修复方向必须是把 `.env` 解析为局部数据并直接参与配置合并，禁止通过 `LoadAndInit`、`os.Setenv` 或 `os.Unsetenv` 修改进程环境。基线未恢复前，不能可靠判断 global 配置改造是否引入新问题。
