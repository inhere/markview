# Project Port Registry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-project port memory backed by `~/.config/markview/markview-projects.json` for automatic port selection modes.

**Architecture:** A new `internal/projects` package owns registry persistence and project record semantics. `main.go` owns CLI/ENV port-source detection and listener selection, using the registry only when CLI `-p -1` is used or no port was configured through CLI/ENV.

**Tech Stack:** Go `net`, `os`, `encoding/json`, `path/filepath`, `github.com/gookit/goutil/cflag`, existing `github.com/gookit/goutil/x/assert`.

---

## File Structure

- Create: `internal/projects/registry.go`
  - Registry path resolution, JSON load/save, project key normalization, lookup, upsert.
- Create: `internal/projects/registry_test.go`
  - Unit tests for registry behavior.
- Modify: `internal/config/config.go`
  - Add port-source state and use it during `Init`.
- Modify: `main.go`
  - Detect whether `--port/-p` was explicitly set, select listener through registry mode when needed, save bound ports.
- Modify: `main_test.go`
  - Unit tests for port source detection and listener selection helpers.
- Modify: `docs/TODO.md`
  - Mark random port persistence done after implementation passes.

## Task 1: Registry Package

**Files:**
- Create: `internal/projects/registry.go`
- Create: `internal/projects/registry_test.go`

- [ ] **Step 1: Write failing registry tests**

Cover:

- `RegistryPath` returns `<HOME>/.config/markview/markview-projects.json`
- missing file loads empty registry
- valid JSON loads records
- `Upsert` creates record with default name and timestamp
- `Upsert` updates only `port` for existing records

Run:

```bash
go test ./internal/projects
```

Expected: fail because package/functions do not exist.

- [ ] **Step 2: Implement minimal registry package**

Implement:

```go
type ProjectRecord struct {
    Port  int    `json:"port"`
    Name  string `json:"name"`
    Added string `json:"added"`
}

type Registry map[string]ProjectRecord
```

Functions:

```go
func RegistryPath() (string, error)
func ProjectKey(targetDir string) (string, error)
func Load(path string) (Registry, error)
func Save(path string, registry Registry) error
func LookupPort(registry Registry, targetDir string) (int, bool)
func Upsert(registry Registry, targetDir string, port int, now time.Time) error
```

Run:

```bash
go test ./internal/projects
```

Expected: pass.

## Task 2: Port Source Detection

**Files:**
- Modify: `internal/config/config.go`
- Modify: `main.go`
- Modify: `main_test.go`

- [ ] **Step 1: Write failing tests for registry activation**

Add tests that verify:

- no CLI/ENV port activates registry mode
- CLI `-p -1` activates registry mode
- CLI fixed port does not activate registry mode
- `MKVIEW_PORT` does not activate registry mode

Run:

```bash
go test .
```

Expected: fail because helpers/source state are missing.

- [ ] **Step 2: Add port source state**

Add a small source enum in `internal/config`:

```go
type PortSource string

const (
    PortSourceUnset PortSource = "unset"
    PortSourceCLI   PortSource = "cli"
    PortSourceEnv   PortSource = "env"
)
```

Set `config.Cfg.PortSource` before or during `prepare()`:

- use `c.Visit` to detect the CLI `port` flag
- use `os.LookupEnv(config.EnvPort)` to detect env

Run:

```bash
go test .
```

Expected: port-source tests pass.

## Task 3: Registry-Aware Listener Selection

**Files:**
- Modify: `main.go`
- Modify: `main_test.go`

- [ ] **Step 1: Write failing tests for listener selection**

Cover:

- saved port is used when available
- occupied saved/default port falls through to the next available port
- no record with CLI `-p -1` uses OS random port

Run:

```bash
go test .
```

Expected: fail because listener selection helpers are missing.

- [ ] **Step 2: Implement listener helpers**

Add helpers near current server startup logic:

```go
func shouldUseProjectPortRegistry() bool
func listenProjectPort(addrHost, targetDir string, preferDefault bool) (net.Listener, int)
func listenNextAvailable(host string, startPort int, limit int) (net.Listener, int, error)
```

Keep comments on the decision points where registry mode differs from fixed port mode.

Run:

```bash
go test .
```

Expected: pass.

## Task 4: Integrate Save-on-Start

**Files:**
- Modify: `main.go`
- Modify: `docs/TODO.md`

- [ ] **Step 1: Wire registry mode into `run()`**

When registry mode is active:

- load registry
- choose listener according to source mode
- update `config.Cfg.SetPort(actualPort)`
- call `projects.Upsert`
- save registry
- continue startup even if load/save fails, logging warnings

- [ ] **Step 2: Update TODO**

Mark:

```markdown
- [x] 随机端口时，按项目路径自动保存获取到的端口号（详细说明见下面的章节）
```

Run:

```bash
go test ./...
```

Expected: pass.

## Task 5: Verification

**Files:**
- All modified files

- [ ] **Step 1: Run Go tests**

```bash
go test ./...
```

Expected: pass.

- [ ] **Step 2: Run frontend tests only if web files changed**

No frontend files should change for this feature. Skip unless implementation touches `web/`.

- [ ] **Step 3: Review diff**

```bash
git diff --stat
git diff -- internal/projects main.go internal/config docs/TODO.md
```

Expected: changes are limited to project registry behavior and TODO status.
