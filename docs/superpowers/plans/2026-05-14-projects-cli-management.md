# Projects CLI 管理命令实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 新增 `markview --projects list/show/remove/prune`，用于管理已保存项目 registry，并且这些命令不启动预览 server。

**架构：** 扩展 `internal/projects`，加入排序列表、selector 解析、删除和 prune 等纯数据 helpers。新增一个轻量 CLI 处理层，在 `--projects` 被设置时先执行管理命令并提前返回，避免进入普通 server 启动流程。

**技术栈：** Go、`github.com/gookit/goutil/cflag`、`github.com/gookit/cliui`、`encoding/json`、`os`、现有 `github.com/gookit/goutil/testutil/assert`。

---

## 文件结构

- 修改：`internal/projects/registry.go`
  - 新增项目 entry view、排序列表、selector 解析、删除、prune helpers。
- 修改：`internal/projects/registry_test.go`
  - 新增 list/resolve/remove/prune 单元测试。
- 新增：`projects_cli.go`
  - 解析和执行 `--projects` actions；使用 `github.com/gookit/cliui` 渲染 list 表格和 show 信息；测试中可注入 output writer。
- 新增：`projects_cli_test.go`
  - 测试管理命令输出，以及命令不会启动 server 的早返回行为。
- 修改：`main.go`
  - 注册 `--projects` flag，并在普通 `prepare()` / server 启动前调用管理命令处理逻辑。
- 修改：`go.mod`
  - 新增 `github.com/gookit/cliui` 依赖。
- 修改：`go.sum`
  - 记录 `github.com/gookit/cliui` 及其间接依赖校验值。
- 修改：`docs/TODO.md`
  - 阶段一完成后暂不勾选整个 `--projects` TODO；等阶段二快速启动也完成后再勾选。

## Task 1: 项目 Registry 查询 Helpers

**文件：**
- 修改：`internal/projects/registry.go`
- 修改：`internal/projects/registry_test.go`

- [ ] **Step 1: 编写失败测试**

新增测试覆盖：

- `List(registry)` 按 name、path 排序返回记录。
- `Resolve(registry, selector)` 可以按记录 name 查找。
- `Resolve(registry, selector)` 可以按完整 path 查找。
- `Resolve` 对未知 selector 返回 not-found 错误。
- `Resolve` 对匹配多个记录的 selector 返回 ambiguity 错误。

运行：

```bash
go test ./internal/projects
```

预期：失败，因为 helpers 尚不存在。

- [ ] **Step 2: 实现最小 helpers**

新增：

```go
type ProjectEntry struct {
    Path   string
    Record ProjectRecord
}

func List(registry Registry) []ProjectEntry
func Resolve(registry Registry, selector string) (ProjectEntry, error)
```

错误需要导出或至少可被 CLI 测试识别：

```go
var ErrProjectNotFound = errors.New("project not found")
var ErrProjectAmbiguous = errors.New("project selector is ambiguous")
```

运行：

```bash
go test ./internal/projects
```

预期：通过。

## Task 2: 删除和 Prune Helpers

**文件：**
- 修改：`internal/projects/registry.go`
- 修改：`internal/projects/registry_test.go`

- [ ] **Step 1: 编写失败测试**

新增测试覆盖：

- `Remove(registry, selector)` 只删除匹配到的记录。
- `Remove` 正确传播 not-found 和 ambiguity 错误。
- `PruneMissing(registry)` 删除路径不存在或不再是目录的记录。
- `PruneMissing` 保留仍然存在的目录。

运行：

```bash
go test ./internal/projects
```

预期：失败，因为 helpers 尚不存在。

- [ ] **Step 2: 实现最小 helpers**

新增：

```go
func Remove(registry Registry, selector string) (ProjectEntry, error)
func PruneMissing(registry Registry) []ProjectEntry
```

运行：

```bash
go test ./internal/projects
```

预期：通过。

## Task 3: CLI Flag 和早返回分发

**文件：**
- 新增：`projects_cli.go`
- 新增：`projects_cli_test.go`
- 修改：`main.go`
- 修改：`go.mod`
- 修改：`go.sum`

- [ ] **Step 1: 编写失败 CLI 测试**

新增测试覆盖：

- `--projects list` 在空 registry 下输出 `No saved projects.`
- `--projects show <name>` 输出项目字段。
- `--projects remove <name>` 保存删除后的 registry。
- `--projects prune` 删除 missing paths 后保存 registry。
- 管理命令执行后不会启动 server。

测试中通过注入 registry path 或 package-level 变量使用临时文件，避免访问真实 `~/.config/markview/markview-projects.json`。

运行：

```bash
go test .
```

预期：失败，因为 CLI 层尚不存在。

- [ ] **Step 2: 添加 `cliui` 依赖**

运行：

```bash
go get github.com/gookit/cliui
```

预期：`go.mod` 和 `go.sum` 新增 `github.com/gookit/cliui` 相关记录。

- [ ] **Step 3: 实现 CLI handler**

新增 package-level option：

```go
var projectsAction string
```

注册 flag：

```go
cmd.StrVar(&projectsAction, "projects", "", "Manage saved projects: list, show, remove, prune")
```

在 `run(c)` 的 `prepare(args)` 之前处理：

```go
if projectsAction != "" {
    return runProjectsAction(projectsAction, c.RemainArgs(), os.Stdout)
}
```

运行：

```bash
go test .
```

预期：通过。

## Task 4: 用户可见输出和错误

**文件：**
- 修改：`projects_cli.go`
- 修改：`projects_cli_test.go`

- [ ] **Step 1: 添加输出格式测试**

验证：

- list 通过 `cliui` 表格输出，并包含 `NAME`、`PORT`、`ADDED`、`PATH`
- show 通过 `cliui` 信息块输出，并包含 `Name`、`Path`、`Port`、`Added`、`Exists`
- remove 输出包含 name 和 path
- prune 输出报告 removed count
- 缺少 selector 返回清晰错误
- 未知 action 返回清晰错误

运行：

```bash
go test .
```

预期：失败，直到格式化和错误处理实现完成。

- [ ] **Step 2: 使用 `cliui` 实现输出格式和错误处理**

`list` 和 `show` 使用 `github.com/gookit/cliui`；错误消息保持普通 `error` 返回，避免把业务错误绑定到终端展示库。

运行：

```bash
go test .
```

预期：通过。

## Task 5: 验证

**文件：**
- 所有修改文件

- [ ] **Step 1: 格式化 Go 代码**

```bash
gofmt -w internal/projects/registry.go internal/projects/registry_test.go projects_cli.go projects_cli_test.go main.go
go mod tidy
```

- [ ] **Step 2: 运行全量 Go 测试**

```bash
go test ./...
```

预期：通过。

- [ ] **Step 3: 检查 diff**

```bash
git diff --stat
git diff -- internal/projects projects_cli.go projects_cli_test.go main.go docs/TODO.md
```

预期：改动只集中在项目管理命令相关文件。
