# MarkView Global Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现单个 `markview --global` HTTP server，原生、隔离且安全地服务 registry 中的全部项目，同时保持现有单项目行为。

**Architecture:** 先消除 dotenv 对进程环境的副作用，再建立稳定项目索引、原子 registry 和统一真实路径边界；随后把现有 handler、SSE 和 watcher 实例化为 `ProjectRuntime`，让单项目与 global 模式复用同一个 `ProjectServer`。Global 模式由 `ProjectManager` 按项目懒加载 runtime，并通过 `/p/{id}/...` 路由；前端只增加一个 `basePath` URL helper 和 global topbar。

**Tech Stack:** Go 1.25+ 标准库、现有 `gookit/goutil`、`fsnotify`、TypeScript、CSS、Bun test、JSDOM。

## Global Constraints

- 不新增依赖；优先复用现有 `config.MergeRuntimeConfig`、`projects.ProjectKey`、handler 和前端站内导航。
- `--global` 默认绑定 `127.0.0.1`；只有命令行显式 `--private=false` 才允许公开监听。
- 项目 `.env` 只能解析成局部数据，runtime 初始化禁止修改进程环境。
- 页面、raw、搜索、文件树和 watcher 必须使用同一个真实项目根边界。
- URL 只由 `net/http` 解码一次；应用层不得再次 `PathUnescape`。
- `ProjectRuntime.Close()` 是 watcher 和 EventHub 的唯一生命周期 owner；所有 Close 必须幂等。
- 每个任务遵循 RED → GREEN → 聚焦测试 → 相关完整测试 → 更新本计划 checkbox → 独立提交。
- 用户已有的 `docs/TODO.md` 修改必须保留；只有 FEA002 全部验证完成后才更新其 checkbox。
- 当前 4 个 dotenv/bootstrap 基线失败必须在 Task 1 独立修复，之后才能开始 runtime 重构。

## 修订记录

| 日期 | 修订人 | 变更 |
| --- | --- | --- |
| 2026-07-19 | Codex | 初版实施计划：单个 global server 原生服务所有 registry 项目。 |

相关文档：

- [设计文档](../specs/2026-07-19-markview-global-server-design.md)
- [TODO 需求](../../TODO.md)

---

## 文件职责

| 文件 | 职责 |
| --- | --- |
| `internal/bootstrap/bootstrap.go` | 保留 CLI 启动编排；移出项目级可变 handler 状态 |
| `internal/config/project_runtime.go` | 纯局部 dotenv 解析及独立项目 runtime config 构建 |
| `internal/bootstrap/project_manager.go` | runtime slot、懒加载、失败重试和幂等关闭 |
| `internal/bootstrap/global_server.go` | `--global` 参数、listener、registry 刷新、global mux 和主页 |
| `internal/projects/registry.go` | StableID、索引和原子 registry 快照保存 |
| `internal/handlers/project_root.go` | URL 子路径到真实项目文件的统一安全解析 |
| `internal/handlers/project_server.go` | 可挂载、无生命周期所有权的项目 HTTP handler |
| `internal/handlers/event_hub.go` | 每项目 SSE 订阅、发布和关闭 |
| `internal/handlers/watch_handler.go` | 每项目可关闭 watcher 与真实路径边界 |
| `web/src/project-url.ts` | 所有项目级 URL 的唯一 basePath helper |
| `web/template-projects.html` | 不加载阅读器 bundle 的 global 项目主页；由 `main.go` 嵌入 |
| `web/src/style/layout.css` | global topbar 和现有布局行适配 |

---

### Task 1: 修复 dotenv 基线并移除进程环境副作用

**Files:**
- Modify: `internal/bootstrap/bootstrap.go`
- Modify: `internal/bootstrap/bootstrap_test.go`

**Interfaces:**
- Produces: `loadProjectDotenv(targetDir string) (map[string]string, error)`，通过 `envutil.SplitText2map` 纯解析，不修改 `os.Environ`。
- Produces: `buildRuntimeConfig(targetDir string, dotenv map[string]string) (config.Config, error)`，保持现有单项目合并顺序。

- [x] **Step 1: 写出失败的无副作用与并发隔离测试**

在 `internal/bootstrap/bootstrap_test.go` 保留现有 4 个失败用例，并增加 barrier 并发用例：

```go
func TestLoadProjectDotenvConcurrentDoesNotMutateEnvironment(t *testing.T) {
	t.Setenv(config.EnvEntry, "process.md")
	a, b := t.TempDir(), t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(a, ".env"), []byte("MKVIEW_ENTRY=a.md\n"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(b, ".env"), []byte("MKVIEW_ENTRY=b.md\n"), 0644))

	var wg sync.WaitGroup
	type result struct { value string; err error }
	results := make(chan result, 2)
	for _, dir := range []string{a, b} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			env, err := loadProjectDotenv(dir)
			results <- result{value: env[config.EnvEntry], err: err}
		}()
	}
	wg.Wait()
	close(results)
	got := []string{}
	for result := range results {
		assert.NoErr(t, result.err)
		got = append(got, result.value)
	}
	slices.Sort(got)
	assert.Eq(t, []string{"a.md", "b.md"}, got)
	assert.Eq(t, "process.md", os.Getenv(config.EnvEntry))
}
```

- [x] **Step 2: 运行现有失败用例，确认 RED**

Run:

```bash
go test ./internal/bootstrap -run 'TestPrepareLoadsDotenvFromTargetDir|TestPrepareDotenvDoesNotLeakBetweenProjects|TestPrepareDotenvDebugEnablesDebugWithoutLeakingEnv|TestPrepareEnvEntryAppliesWithoutPositionalEntry|TestLoadProjectDotenvConcurrent' -count=1
```

Expected: 现有基线用例至少一个 FAIL；并发测试在旧 `LoadAndInit` 实现下可能失败或在 race 门禁中报告环境竞态。

- [x] **Step 3: 用现有纯解析 helper 做最小局部解析**

直接在 `internal/bootstrap/bootstrap.go` 替换有副作用的实现：

```go
func loadProjectDotenv(targetDir string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(targetDir, ".env"))
	if errors.Is(err, fs.ErrNotExist) { return map[string]string{}, nil }
	if err != nil { return nil, err }
	return envutil.SplitText2map(string(data)), nil
}
```

删除 `environMap`、`restoreEnv` 以及所有只为恢复进程环境存在的代码；`envValue` 保持“项目 dotenv 优先，否则只读进程环境”。

- [x] **Step 4: 验证基线和 race**

Run:

```bash
go test ./internal/bootstrap -count=1
go test -race ./internal/bootstrap -run 'Dotenv|EnvEntry' -count=1
go test ./... -count=1
```

Expected: dotenv/bootstrap 用例 PASS；`go test ./...` 不再出现原有 4 个失败；race 命令无 data race。

执行记录：本机为 Windows/amd64 且 `CGO_ENABLED=0`，`go test -race` 无法启动；已通过两个项目并发加载测试替代本机 race 验证，最终门禁仍需在支持 race 的环境补跑。

- [x] **Step 5: 更新计划并提交基线修复**

```bash
git add internal/bootstrap/bootstrap.go internal/bootstrap/bootstrap_test.go docs/superpowers/plans/2026-07-19-markview-global-server.md
git commit -m "fix: isolate project dotenv loading"
```

---

### Task 2: 建立稳定项目 ID 和原子 registry 快照

**Files:**
- Modify: `internal/projects/registry.go`
- Modify: `internal/projects/registry_test.go`

**Interfaces:**
- Produces: `StableID(targetDir string) (string, error)`。
- Produces: `IndexedProject`、`ProjectIndex`、`BuildIndex(registry Registry) (ProjectIndex, error)`。
- Preserves: `Load`、`Save`、`Upsert`、`List`、`Resolve` 的现有外部行为。

- [ ] **Step 1: 写 StableID、碰撞和原子保存失败测试**

```go
func TestStableIDUsesNormalizedProjectKey(t *testing.T) {
	dir := t.TempDir()
	id1, err := StableID(dir)
	assert.NoErr(t, err)
	id2, err := StableID(filepath.Join(dir, "."))
	assert.NoErr(t, err)
	assert.True(t, regexp.MustCompile(`^[0-9a-f]{12}$`).MatchString(id1))
	assert.Eq(t, id1, id2)
}

func TestBuildIndexRejectsDuplicateID(t *testing.T) {
	original := stableID
	t.Cleanup(func() { stableID = original })
	stableID = func(string) (string, error) { return "aaaaaaaaaaaa", nil }
	_, err := BuildIndex(Registry{
		filepath.Join(t.TempDir(), "a"): {Name: "A"},
		filepath.Join(t.TempDir(), "b"): {Name: "B"},
	})
	assert.Err(t, err)
}
```

原子保存测试通过注入窄小的包级 `renameFile = os.Rename` seam，让 rename 返回错误并断言旧文件内容未改变；不要暴露可配置 ID 长度。

- [ ] **Step 2: 运行项目包测试，确认 RED**

```bash
go test ./internal/projects -run 'StableID|BuildIndex|Save' -count=1
```

Expected: FAIL，缺少 StableID/BuildIndex，旧 Save 不满足 rename 失败保留旧快照测试。

- [ ] **Step 3: 实现最小标准库 ID 与索引**

```go
type IndexedProject struct {
	ID     string
	Path   string
	Entry  ProjectEntry
	Exists bool
}

type ProjectIndex map[string]IndexedProject

func StableID(targetDir string) (string, error) {
	key, err := ProjectKey(targetDir)
	if err != nil { return "", err }
	key = filepath.ToSlash(key)
	if runtime.GOOS == "windows" { key = strings.ToLower(key) }
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:12], nil
}
```

`BuildIndex` 遍历 registry map，计算 ID、检测已占用 ID，并用 `fsutil.IsDir(path)` 塡充 `Exists`。

- [ ] **Step 4: 把 Save 改为同目录原子替换**

```go
func Save(path string, registry Registry) (err error) {
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil { return err }
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil { return err }
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".markview-projects-*")
	if err != nil { return err }
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err = tmp.Chmod(0644); err == nil { _, err = tmp.Write(data) }
	if err == nil { err = tmp.Sync() }
	closeErr := tmp.Close()
	if err == nil { err = closeErr }
	if err != nil { return err }
	return renameFile(tmpPath, path)
}
```

- [ ] **Step 5: 验证项目包并提交**

```bash
go test ./internal/projects -count=1
go test ./... -count=1
git add internal/projects/registry.go internal/projects/registry_test.go docs/superpowers/plans/2026-07-19-markview-global-server.md
git commit -m "feat: index global projects safely"
```

---

### Task 3: 统一项目真实路径与 URL 安全边界

**Files:**
- Create: `internal/handlers/project_root.go`
- Create: `internal/handlers/project_root_test.go`

**Interfaces:**
- Produces: `NewProjectRoot(path string) (ProjectRoot, error)`。
- Produces: `(ProjectRoot).Resolve(urlPath string) (string, error)`。
- Produces: `ErrPathOutsideProject`。

- [ ] **Step 1: 写项目内、根外 symlink/junction 和 traversal 测试**

使用 `t.Run()` 覆盖 `/docs/a.md`、`/../secret.md`、`/%2e%2e/secret.md` 字面量、`/..foo/a.md`、反斜杠、NUL、相似前缀目录和不存在文件。Windows junction 用 `cmd /c mklink /J` 仅在具备权限时运行，否则 `t.Skip`；Unix 使用 `os.Symlink`。

```go
func TestProjectRootResolveRejectsTraversal(t *testing.T) {
	root, err := NewProjectRoot(t.TempDir())
	assert.NoErr(t, err)
	for _, path := range []string{"/../secret.md", "/a/../../secret.md", "/a\\..\\secret.md", "/x\x00.md"} {
		t.Run(path, func(t *testing.T) {
			_, err := root.Resolve(path)
			assert.True(t, errors.Is(err, ErrPathOutsideProject))
		})
	}
}
```

- [ ] **Step 2: 确认 RED**

```bash
go test ./internal/handlers -run ProjectRoot -count=1
```

Expected: FAIL，类型和方法尚不存在。

- [ ] **Step 3: 实现 ProjectRoot**

```go
var ErrPathOutsideProject = errors.New("path outside project")

type ProjectRoot struct {
	DisplayPath string
	RealPath    string
}

func NewProjectRoot(path string) (ProjectRoot, error) {
	display, err := filepath.Abs(path)
	if err != nil { return ProjectRoot{}, err }
	real, err := filepath.EvalSymlinks(display)
	if err != nil { return ProjectRoot{}, err }
	return ProjectRoot{DisplayPath: filepath.Clean(display), RealPath: filepath.Clean(real)}, nil
}
```

`Resolve` 接收已经由 `net/http` 解码一次的 `/...` 路径，不调用 `PathUnescape`；先拒绝 NUL、反斜杠和 `.`/`..` segment，再 Join、EvalSymlinks，并使用 `filepath.Rel` 的 segment 语义判界。目标不存在时验证最近存在父目录仍在 root 内，再返回包装 `fs.ErrNotExist` 的错误；所有消费者只使用 Resolve 返回的 real path。

- [ ] **Step 4: 验证安全测试和 race**

```bash
go test ./internal/handlers -run ProjectRoot -count=1
go test -race ./internal/handlers -run ProjectRoot -count=1
```

Expected: PASS，无 data race。

- [ ] **Step 5: 提交路径安全边界**

```bash
git add internal/handlers/project_root.go internal/handlers/project_root_test.go docs/superpowers/plans/2026-07-19-markview-global-server.md
git commit -m "feat: constrain project filesystem paths"
```

---

### Task 4: 实例化 ProjectServer 并保持单项目行为

**Files:**
- Create: `internal/config/project_runtime.go`
- Create: `internal/config/project_runtime_test.go`
- Create: `internal/handlers/event_hub.go`
- Create: `internal/handlers/event_hub_test.go`
- Create: `internal/handlers/project_server.go`
- Create: `internal/handlers/project_server_test.go`
- Modify: `internal/handlers/page_handler.go`
- Modify: `internal/handlers/search_handler.go`
- Modify: `internal/handlers/sse_handler.go`
- Modify: `internal/handlers/handlers_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/bootstrap/bootstrap.go`
- Modify: `internal/bootstrap/bootstrap_test.go`

**Interfaces:**
- Produces: `config.ProjectLoadOptions{GlobalMode bool}` 和 `config.LoadProjectRuntimeConfig(targetDir string, options ProjectLoadOptions) (config.Config, error)`。
- Produces: `NewEventHub()`、`Subscribe() (<-chan string, func())`、`Publish(string) bool`、`Close() error`。
- Produces: `NewProjectServer(cfg config.Config, root ProjectRoot, events *EventHub, content fs.FS) *ProjectServer`。
- Produces: `(ProjectServer).ServeHTTP(http.ResponseWriter, *http.Request)`。
- Produces: `config.AppConfig.BasePath string`，单项目默认空。
- Consumes: Task 3 的 `ProjectRoot.Resolve`。

- [ ] **Step 1: 写两个 server 不串项目的失败测试**

```go
func TestProjectServersKeepRootsAndConfigIndependent(t *testing.T) {
	a, b := newTestProject(t, "# A"), newTestProject(t, "# B")
	sa := newTestProjectServer(t, a, config.Config{EntryFile: "README.md", UILayout: config.UILayoutCompact})
	sb := newTestProjectServer(t, b, config.Config{EntryFile: "README.md", UILayout: config.UILayoutTOCRight})

	for _, tc := range []struct{ server http.Handler; want string }{{sa, "# A"}, {sb, "# B"}} {
		rr := httptest.NewRecorder()
		tc.server.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/?q=raw", nil))
		assert.Eq(t, http.StatusOK, rr.Code)
		assert.StrContains(t, rr.Body.String(), tc.want)
	}
}
```

增加单项目协议测试：search/file-tree 10 秒 timeout wrapper 保留、SSE 不经过 TimeoutHandler、静态缓存头不变、HTML no-store、`?q=main` 与 `?q=raw` 行为不变。

- [ ] **Step 2: 确认 RED**

```bash
go test ./internal/handlers -run 'ProjectServer|ProjectServersKeepRoots' -count=1
```

Expected: FAIL，ProjectServer 尚不存在。

- [ ] **Step 3: 提取可并发的项目 runtime 配置加载**

把 Task 1 已验证的局部 dotenv 解析和纯 merge 逻辑从 `internal/bootstrap/bootstrap.go` 移动到 `internal/config/project_runtime.go`。GlobalMode 忽略项目 port/private 和 registry port，保留 entry/watch/watch_dir/watch_skip/include/preview_exts/iframe_hosts/layout；函数只返回 Config，不写 `config.Cfg` 或 debug 包级变量。

```go
type ProjectLoadOptions struct {
	GlobalMode bool
}

func LoadProjectRuntimeConfig(targetDir string, options ProjectLoadOptions) (Config, error) {
	dotenv, err := loadProjectDotenv(targetDir)
	if err != nil { return Config{}, err }
	globalCfg, _, err := LoadGlobalFileConfig()
	if err != nil { return Config{}, err }
	projectCfg, _, err := LoadProjectFileConfig(targetDir)
	if err != nil { return Config{}, err }
	envCfg, err := runtimeEnvConfig(dotenv)
	if err != nil { return Config{}, err }
	if options.GlobalMode {
		projectCfg.Server.Port = nil
		projectCfg.Server.Private = nil
		envCfg.Port = nil
	}
	return MergeRuntimeConfig(MergeInput{Global: globalCfg, Project: projectCfg, Env: envCfg})
}
```

测试两个 goroutine 同时加载不同项目，断言配置值隔离、进程环境不变，并验证 global 模式忽略 project/env port/private。

- [ ] **Step 4: 实现项目级 EventHub 和 mux，不复制 handler**

EventHub 使用一个 mutex、`map[chan string]struct{}` 和 closed bool。`Publish` 持锁执行非阻塞发送，确保不会与 unsubscribe 关闭 channel 竞争；`Close` 只关闭一次：

```go
func (h *EventHub) Subscribe() (<-chan string, func()) {
	ch := make(chan string, 8)
	h.mu.Lock()
	if h.closed { close(ch) } else { h.clients[ch] = struct{}{} }
	h.mu.Unlock()
	return ch, sync.OnceFunc(func() { h.unsubscribe(ch) })
}

func (h *EventHub) Publish(message string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed { return false }
	for ch := range h.clients {
		select { case ch <- message: default: }
	}
	return true
}
```

`event_hub_test.go` 验证两个 hub 不串消息、取消订阅安全、Publish/Close 并发无 panic、Close 幂等。

建立项目级 mux：

```go
type ProjectServer struct {
	Config  config.Config
	Root    ProjectRoot
	Events  *EventHub
	content fs.FS
	mux     *http.ServeMux
}

func NewProjectServer(cfg config.Config, root ProjectRoot, events *EventHub, content fs.FS) *ProjectServer {
	s := &ProjectServer{Config: cfg, Root: root, Events: events, content: content}
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", s.handleSSE)
	mux.Handle("/api/search", http.TimeoutHandler(http.HandlerFunc(s.handleSearch), 10*time.Second, "request timeout"))
	mux.Handle("/api/file-tree", http.TimeoutHandler(http.HandlerFunc(s.handleFileTree), 10*time.Second, "request timeout"))
	mux.HandleFunc("/", s.handlePage)
	s.mux = mux
	return s
}

func (s *ProjectServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
```

将现有 handler 主流程改成方法或显式接收 `Config`、`ProjectRoot`、`EventHub`；删除读取 `config.Cfg.TargetDir`、包级 SSE clients 和 `IfsReader` 的项目级路径。渲染模板所需的嵌入资源通过 `ProjectServer.content` 读取。

- [ ] **Step 5: 单项目 bootstrap 也构造 ProjectServer**

`prepare` 返回独立 config 和 root，`run` 创建 EventHub 与 ProjectServer；global 功能尚未接入。`newServerMux` 只挂载 `/static/` 和项目 server：

```go
func newServerMux(content fs.FS, project http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/static/", newStaticHandler(content))
	mux.Handle("/", project)
	return mux
}
```

- [ ] **Step 6: 验证全部单项目行为**

```bash
go test ./internal/config ./internal/handlers ./internal/bootstrap -count=1
go test ./... -count=1
```

Expected: PASS；现有单项目 URL、缓存头、SSE timeout 回归测试均不变。

- [ ] **Step 7: 提交 ProjectServer 迁移**

```bash
git add internal/config internal/handlers internal/bootstrap docs/superpowers/plans/2026-07-19-markview-global-server.md
git commit -m "refactor: instance project HTTP server"
```

---

### Task 5: 实例化安全可关闭 Watcher

**Files:**
- Modify: `internal/handlers/event_hub.go`
- Modify: `internal/handlers/event_hub_test.go`
- Modify: `internal/handlers/watch_handler.go`
- Create: `internal/handlers/watch_handler_test.go`
- Modify: `internal/handlers/project_server.go`

**Interfaces:**
- Produces: `NewWatcher(root ProjectRoot, cfg config.Config, events *EventHub) (*Watcher, error)`、`Run(context.Context) error`、`Close() error`。
- Consumes: Task 3 的 `ProjectRoot.Resolve` 和 Task 4 的 `EventHub`。

- [ ] **Step 1: 写 watcher 根外动态目录测试**

创建项目根和外部目录，启动 Watcher 后在项目内创建指向外部目录的 symlink/junction，写入 Markdown 文件，并断言 EventHub 在短超时内没有消息。另测项目内 Markdown 修改只发布项目内 slash 相对路径。

- [ ] **Step 2: 确认 RED**

```bash
go test ./internal/handlers -run 'EventHub|Watcher' -count=1
```

Expected: FAIL，实例类型不存在或旧 watcher 将根外动态目录加入监听。

- [ ] **Step 3: 实现 Watcher 生命周期和边界**

Watcher 持有自己的 `fsnotify.Watcher`、root、config、hub、`sync.Once` 和 done channel。初始 WalkDir、Create 目录和事件发布都调用 root boundary helper；越界目录 `SkipDir`，越界事件记录 debug 后丢弃。删除包级 `watcher`、`stopChan`、`watchedDir` 和共享 debounce 状态。

- [ ] **Step 4: 验证 handler 与 race**

```bash
go test ./internal/handlers -count=1
go test -race ./internal/handlers -run 'EventHub|Watcher|SSE' -count=1
go test ./... -count=1
```

Expected: PASS，无跨 hub 消息、根外监听或 data race。

- [ ] **Step 5: 提交 runtime 事件隔离**

```bash
git add internal/handlers docs/superpowers/plans/2026-07-19-markview-global-server.md
git commit -m "refactor: isolate project events and watcher"
```

---

### Task 6: 实现 ProjectManager、Global 路由和项目主页

**Files:**
- Create: `internal/bootstrap/project_manager.go`
- Create: `internal/bootstrap/project_manager_test.go`
- Create: `internal/bootstrap/global_server.go`
- Create: `internal/bootstrap/global_server_test.go`
- Modify: `internal/bootstrap/bootstrap.go`
- Modify: `internal/bootstrap/bootstrap_test.go`
- Modify: `main.go`
- Create: `web/template-projects.html`

**Interfaces:**
- Produces: `NewProjectManager(content fs.FS) *ProjectManager`；registry path 由 global request handler 每次读取，不进入 manager 状态。
- Produces: `(ProjectManager).Runtime(context.Context, projects.IndexedProject) (*ProjectRuntime, error)`、`Close() error`。
- Produces: `newGlobalMux(manager *ProjectManager, content fs.FS) http.Handler`。
- Consumes: Tasks 2–5 的 ProjectIndex、ProjectRoot、ProjectServer、Watcher 和 EventHub。

- [ ] **Step 1: 写 runtime slot 并发、失败重试和 Close 竞态测试**

使用可阻塞 factory seam：100 个 goroutine 请求同一项目只调用一次 factory；A/B 两个项目能同时进入 factory；首次返回错误后第二次重新调用；初始化阻塞时调用 Close，释放 factory 后 runtime 被立即关闭且不发布。

```go
func TestProjectManagerInitializesProjectOnce(t *testing.T) {
	manager, calls := newCountingManager(t)
	project := testIndexedProject(t)
	var wg sync.WaitGroup
	errs := make(chan error, 100)
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.Runtime(context.Background(), project)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs { assert.NoErr(t, err) }
	assert.Eq(t, 1, calls.Load())
}
```

- [ ] **Step 2: 写 global CLI 和真实路由测试**

覆盖：`--global` 与 `--project`/positional/项目内容 flags 互斥；无 private flag 和配置 private=false 均使用 `127.0.0.1`；显式 `--private=false` 才使用公开地址；port 为 CLI > MKVIEW_PORT > global config > 6100。

HTTP 用例覆盖 `/`、`/static/...`、`/p/{id}` 308、`/p/{id}/`、非法/未知/已删除 ID、registry 刷新、缺失目录卡片、API JSON 项目解析错误。

- [ ] **Step 3: 确认 RED**

```bash
go test ./internal/bootstrap -run 'ProjectManager|Global' -count=1
```

Expected: FAIL，manager、global flag 和路由尚不存在。

- [ ] **Step 4: 实现 runtime slot 和关闭交接**

```go
type runtimeSlot struct {
	ready   chan struct{}
	runtime *ProjectRuntime
	err     error
}

type ProjectManager struct {
	mu       sync.Mutex
	closed   bool
	runtimes map[string]*ProjectRuntime
	loading  map[string]*runtimeSlot
	inFlight sync.WaitGroup
	// factory 仅作为包内测试 seam，生产默认 buildProjectRuntime。
	factory func(context.Context, projects.IndexedProject) (*ProjectRuntime, error)
}
```

失败时给当前 slot 写 error、关闭 ready、从 loading 删除；成功发布前在锁内检查 closed。Close 先置 closed、等待 inFlight，再取出并关闭全部 runtime；并发 Close 通过 `sync.Once` 返回同一完成结果。

生产 factory 按固定顺序创建 runtime；任一步失败都逆序关闭已经创建的资源：

```go
func buildProjectRuntime(ctx context.Context, project projects.IndexedProject, content fs.FS) (*ProjectRuntime, error) {
	cfg, err := config.LoadProjectRuntimeConfig(project.Path, config.ProjectLoadOptions{GlobalMode: true})
	if err != nil { return nil, err }
	if err = cfg.Init(project.Path, ""); err != nil { return nil, err }
	root, err := handlers.NewProjectRoot(project.Path)
	if err != nil { return nil, err }
	events := handlers.NewEventHub()
	server := handlers.NewProjectServer(cfg, root, events, content)
	runtime := &ProjectRuntime{ID: project.ID, Path: project.Path, Config: cfg, Server: server, Events: events}
	if cfg.EnableWatch {
		runtime.Watcher, err = handlers.NewWatcher(root, cfg, events)
		if err != nil { _ = runtime.Close(); return nil, err }
	}
	return runtime, nil
}
```

- [ ] **Step 5: 实现 global URL 契约和动态 registry**

global handler 每个 `/` 和 `/p/...` 请求调用 `projects.Load` 与 `BuildIndex`。检查 `EscapedPath()` 中大小写不敏感的 `%2f`、`%5c`、`%00`；从已解码 `URL.Path` 取 ID 和 subpath，clone request/URL，设置新 Path 并清空 RawPath 后交给 ProjectServer。被 registry 删除的 ID 在 runtime 缓存存在时仍返回 404。

- [ ] **Step 6: 实现最小服务端项目主页**

`web/template-projects.html` 只渲染名称、简化路径、global URL、added、状态和进入链接；无效目录禁用入口并显示 prune/remove 提示。不加载阅读器 JS/CSS bundle，不探测旧端口。同步把新模板加入 `main.go` 的 `//go:embed` 列表。

```go
type projectsPageData struct {
	Projects []projects.IndexedProject
}
```

- [ ] **Step 7: 验证 global manager、router、shutdown 和 race**

```bash
go test ./internal/bootstrap ./internal/projects ./internal/handlers -count=1
go test -race ./internal/bootstrap ./internal/projects ./internal/handlers -count=1
go test ./... -count=1
```

Expected: PASS；初始化/Close 无泄漏和 race，registry 更新在下一请求生效。

- [ ] **Step 8: 提交 global server 主链路**

```bash
git add internal/bootstrap main.go web/template-projects.html docs/superpowers/plans/2026-07-19-markview-global-server.md
git commit -m "feat: serve registered projects globally"
```

---

### Task 7: 统一前端 basePath URL

**Files:**
- Create: `web/src/project-url.ts`
- Create: `web/src/project-url.test.ts`
- Modify: `web/src/app-config.ts`
- Modify: `web/src/app-config.test.ts`
- Modify: `web/src/app.ts`
- Modify: `web/src/util.ts`
- Modify: `web/src/components/sidebar.ts`
- Modify: `web/src/components/content-search.ts`
- Modify: `web/src/components/link-preview.ts`
- Modify: `web/src/components/live-status.ts`
- Modify: `web/template.html`
- Modify: `internal/config/config.go`
- Modify: `internal/handlers/page_handler.go`
- Modify: `internal/handlers/handlers_test.go`

**Interfaces:**
- Produces: `AppConfig.basePath: string`，单项目为 `""`，global 项目为 `/p/{id}`。
- Produces: `projectURL(path: string, basePath?: string): string`。

- [ ] **Step 1: 写 projectURL 精确行为测试**

```ts
test('prefixes project URLs once and preserves query/hash', () => {
  expect(projectURL('/docs/a.md?q=main#h', '/p/aaaaaaaaaaaa'))
    .toBe('/p/aaaaaaaaaaaa/docs/a.md?q=main#h')
  expect(projectURL('/p/aaaaaaaaaaaa/docs/a.md', '/p/aaaaaaaaaaaa'))
    .toBe('/p/aaaaaaaaaaaa/docs/a.md')
})

test('leaves global and external URLs unchanged', () => {
  expect(projectURL('/static/app.js', '/p/aaaaaaaaaaaa')).toBe('/static/app.js')
  expect(projectURL('https://example.com/a', '/p/aaaaaaaaaaaa')).toBe('https://example.com/a')
  expect(projectURL('#section', '/p/aaaaaaaaaaaa')).toBe('#section')
})
```

增加单项目空 basePath、相对路径、图片、download、raw、SSE、search、file-tree、Toast 和 popstate 用例。

- [ ] **Step 2: 确认 RED**

```bash
cd web && bun test src/project-url.test.ts src/app-config.test.ts
```

Expected: FAIL，helper/basePath 尚不存在。

- [ ] **Step 3: 实现唯一 URL helper**

```ts
export function projectURL(path: string, basePath = readAppConfig().basePath): string {
  if (!basePath || /^(?:[a-z][a-z0-9+.-]*:|#)/i.test(path) || path.startsWith('/static/')) return path
  const queryAt = path.indexOf('?')
  const hashAt = path.indexOf('#')
  const suffixAt = [queryAt, hashAt].filter(index => index >= 0).reduce((a, b) => Math.min(a, b), path.length)
  const pathname = path.slice(0, suffixAt)
  const suffix = path.slice(suffixAt)
  if (pathname === basePath || pathname.startsWith(`${basePath}/`)) return path
  const joined = `${basePath}/${pathname.replace(/^\/+/, '')}`
  return joined + suffix
}
```

`normalizeAppConfig` 对 basePath 只接受空字符串或 `/p/[0-9a-f]{12}`；无效注入回退为空。

- [ ] **Step 4: 迁移全部项目级 URL**

所有 `/api/search`、`/api/file-tree`、`/sse`、搜索结果、文件树、目录列表、raw、preview、Markdown 相对资源、Toast 和 history 调用统一经过 `projectURL`。`buildContentBaseURL` 使用项目 URL，不再固定从 origin 根构造；模板由服务端 helper 输出带 basePath 的 raw/download/目录链接。

- [ ] **Step 5: 验证前后端 URL 行为**

```bash
cd web && bun test
bun run build
cd .. && go test ./internal/config ./internal/handlers -count=1
```

Expected: Bun 全部 PASS、build 成功、Go 模板测试 PASS。

- [ ] **Step 6: 提交 basePath 迁移**

```bash
git add web/src web/template.html internal/config internal/handlers docs/superpowers/plans/2026-07-19-markview-global-server.md
git commit -m "feat: scope reader URLs by project"
```

---

### Task 8: 增加 Global topbar 并适配现有布局

**Files:**
- Modify: `web/template.html`
- Modify: `web/src/style/layout.css`
- Modify: `web/src/layout-css.test.ts`
- Modify: `internal/handlers/page_handler.go`
- Modify: `internal/handlers/handlers_test.go`

**Interfaces:**
- Consumes: 项目页面数据中的 `GlobalMode`、`ProjectName`、`ProjectPath`。
- Preserves: toc-middle/toc-right 内容区底部 16px、preview 横向避让和单项目布局。

- [ ] **Step 1: 写 topbar 渲染与布局失败测试**

Go 测试断言 global Markdown/目录页包含返回 `/`、名称和安全简化路径；raw/API/SSE 与单项目 HTML 不包含 topbar。CSS 测试断言 topbar 为独立 grid row，内容区使用剩余高度，TOC 仍保持 `bottom: 16px`。

- [ ] **Step 2: 确认 RED**

```bash
go test ./internal/handlers -run GlobalTopbar -count=1
cd web && bun test src/layout-css.test.ts
```

Expected: FAIL，topbar markup 和布局规则尚不存在。

- [ ] **Step 3: 实现最小语义化 topbar**

```html
{{if .GlobalMode}}
<nav class="global-topbar" aria-label="Project navigation">
  <a href="/">← Projects</a>
  <span class="global-topbar-name">{{.ProjectName}}</span>
  <span class="global-topbar-path">{{.ProjectPath}}</span>
</nav>
{{end}}
```

路径由服务端简化并通过 `html/template` 转义。CSS 只增加 topbar grid row；正文/sidebar 使用剩余高度，TOC 的 bottom 和 preview 避让规则不改。

- [ ] **Step 4: 验证 Go、CSS 和前端**

```bash
go test ./internal/handlers -count=1
cd web && bun test
bun run build
```

Expected: PASS；单项目模板快照不新增 topbar。

- [ ] **Step 5: 提交 topbar**

```bash
git add web/template.html web/src/style/layout.css web/src/layout-css.test.ts internal/handlers docs/superpowers/plans/2026-07-19-markview-global-server.md
git commit -m "feat: add global project navigation"
```

---

### Task 9: 集成验证、文档和 FEA002 完成标记

**Files:**
- Modify: `README.md`
- Create: `docs/global-server.md`
- Modify: `docs/TODO.md`
- Modify: `docs/superpowers/plans/2026-07-19-markview-global-server.md`

**Interfaces:**
- Documents: `markview --global`、安全默认、global URL、项目配置作用域和 registry 刷新语义。
- Completes: FEA002 checkbox。

- [ ] **Step 1: 运行完整自动化质量门禁**

```bash
go test ./... -count=1
go test -race ./internal/bootstrap/... ./internal/projects/... ./internal/handlers/... -count=1
cd web && bun test
bun run build
```

Expected: Go 全部 PASS、race 无报告、Bun 全部 PASS、build 成功。若当前平台不支持 race，记录准确错误，并在支持的平台补跑后才能完成本任务。

- [ ] **Step 2: 真实启动两个项目做 HTTP 验证**

使用两个临时项目和临时 registry 启动 `markview --global --no-browser`，确认实际 listener 为 `127.0.0.1`；验证主页、两个项目同名文件、search、file-tree、raw、SSE 隔离、registry remove 后下一请求 404、无效目录卡片和 `/p/{id}` 308。

- [ ] **Step 3: 用真实浏览器做 UI/UX 验证**

验证项目切换、返回 Projects、Markdown 相对图片、搜索结果、preview、raw、Toast、history back/forward；分别检查 compact、toc-middle、toc-right 和 preview 开启状态。确认 topbar 不覆盖正文，toc-middle/toc-right 控制按钮仍位于内容区左下/右下并保持 16px 下边距。

- [ ] **Step 4: 更新用户文档**

README 只增加快速入口并链接 `docs/global-server.md`；新文档详细说明：

```text
markview --global
markview --global --port 6200
markview --global --private=false
```

明确默认 loopback、公开警告、项目级 port/private 被忽略、项目 runtime 懒加载、registry 更新在下一请求生效，以及不存在项目不会自动 prune。

- [ ] **Step 5: 更新 checkbox 并提交文档**

只对 `docs/TODO.md` 中 FEA002 对应行使用精确 cached patch 标记完成，保留其他用户修改；勾选本计划全部任务。

```bash
git add README.md docs/global-server.md docs/superpowers/plans/2026-07-19-markview-global-server.md
git diff -- docs/TODO.md
git add -p docs/TODO.md
git commit -m "docs: complete global server feature"
```

- [ ] **Step 6: 最终核验提交与推送状态**

```bash
git status --short --branch
git log --oneline --decorate -12
git pull --rebase
git push
git status --short --branch
```

Expected: 所有计划提交已推送，分支显示与 `origin/main` up to date；`docs/TODO.md` 仅保留用户未纳入 FEA002 提交的其他修改。
