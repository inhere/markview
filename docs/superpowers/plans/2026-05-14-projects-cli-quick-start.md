# Projects CLI 快速启动实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 新增 `markview -P <project>` 和 `markview --project <project>`，允许用户不切换目录也能启动 registry 中已保存的项目。

**架构：** 复用阶段一的 selector 解析 helpers。在 `prepare()` 计算 target directory 之前解析选中的项目，然后把匹配到的路径传入现有 config 初始化和启动流程。

**技术栈：** Go、`github.com/gookit/goutil/cflag`、现有 `internal/projects`、现有 `github.com/gookit/goutil/x/assert`。

---

## 前置条件

本计划假设阶段一已完成：

- `internal/projects.Resolve` 已存在。
- `--projects list/show/remove/prune` 已存在。
- 阶段一已引入 `github.com/gookit/cliui`，快速启动阶段可以复用它输出项目解析错误或启动前提示，但不要让 `internal/projects` 依赖展示库。
- 测试可以安全注入临时 registry path。

## 文件结构

- 修改：`main.go`
  - 注册 `-P|--project`，在 `prepare()` 前解析选中项目，并确保普通启动流程使用匹配到的路径。
- 修改：`projects_cli.go`
  - 如果 registry 加载和解析 helpers 位于该文件，则复用它们。
- 修改：`main_test.go`
  - 添加项目路径优先级和端口 registry 激活规则测试。
- 修改：`docs/TODO.md`
  - 阶段二通过后，将完整 `--projects` TODO 标记完成。

## Task 1: 注册快速启动 Flag

**文件：**
- 修改：`main.go`
- 修改：`main_test.go`

- [x] **Step 1: 编写失败的 flag parse 测试**

新增测试解析：

```bash
-P markview
```

并验证 package-level `selectedProject` 或 config 字段被设置为 `markview`。

同时测试：

```bash
--project markview
```

运行：

```bash
go test .
```

预期：失败，因为 flag 尚不存在。

- [x] **Step 2: 注册 flag**

新增：

```go
var selectedProject string
cmd.StrVar(&selectedProject, "project", "", "Start a saved project by name or path;;P")
```

运行：

```bash
go test .
```

预期：通过。

## Task 2: 在 Prepare 前解析项目

**文件：**
- 修改：`main.go`
- 修改：`main_test.go`

- [x] **Step 1: 编写失败的解析测试**

新增测试覆盖：

- 按 name 选择项目，可以解析到 registry path。
- 按完整 path 选择项目，可以解析到 registry path。
- 未知项目返回错误。
- 歧义项目返回错误。

使用临时 registry 文件，不访问真实 home registry。

运行：

```bash
go test .
```

预期：失败，因为 startup 尚未解析 `selectedProject`。

- [x] **Step 2: 实现 selected project 解析**

新增 helper：

```go
func resolveSelectedProjectTarget(selector string) (string, error)
```

在 `run(c)` 中：

```go
args := c.RemainArgs()
if selectedProject != "" {
    targetDir, err := resolveSelectedProjectTarget(selectedProject)
    if err != nil { return err }
    args = append([]string{targetDir}, args...)
}
```

不要调用 `os.Chdir`。

运行：

```bash
go test .
```

预期：通过。

## Task 3: Entry File 优先级

**文件：**
- 修改：`main.go`
- 修改：`main_test.go`

- [x] **Step 1: 编写失败的 entry 优先级测试**

测试：

- `markview -P docs` 使用选中项目路径，并默认使用 `README.md`。
- `markview -P docs guide.md` 使用选中项目路径，并使用 `guide.md` 作为 entry file。
- 如果同时提供 selected project 和 positional directory，selected project 对 target directory 有优先权；positional 值只有在像 entry path 时才作为 entry。

运行：

```bash
go test .
```

预期：如果当前参数合并逻辑存在歧义，则失败。

- [x] **Step 2: 实现保守的参数合并**

推荐规则：

- 设置 `-P/--project` 时，忽略 positional directory。
- 最多允许一个 positional 参数，并把它当作 entry file。
- 如果 `-P` 同时收到多个 positional 参数，返回错误。

这样可以避免意外启动错误目录。

运行：

```bash
go test .
```

预期：通过。

## Task 4: 保持端口 Registry 行为

**文件：**
- 修改：`main_test.go`
- 如有必要，修改：`main.go`

- [x] **Step 1: 编写端口行为测试**

验证：

- `markview -P docs` 且没有 CLI/ENV port 时，使用项目端口 registry 模式。
- `markview -P docs -p -1` 使用项目端口 registry 模式。
- `markview -P docs -p 8080` 不使用项目端口 registry 模式。
- `MKVIEW_PORT=8080 markview -P docs` 不使用项目端口 registry 模式。

运行：

```bash
go test .
```

预期：如果阶段 0 的端口来源规则保持干净，则应通过；否则修复。

## Task 5: 最终 TODO 和验证

**文件：**
- 修改：`docs/TODO.md`
- 所有修改的 Go 文件

- [x] **Step 1: 标记 TODO 完成**

更新：

```markdown
- [x] 新增 --projects 选项，支持多项目管理和快速启动（详细说明见下面的章节）
```

- [x] **Step 2: 格式化 Go 代码**

```bash
gofmt -w main.go main_test.go projects_cli.go projects_cli_test.go
```

- [x] **Step 3: 运行全量 Go 测试**

```bash
go test ./...
```

预期：通过。

- [x] **Step 4: 检查 diff**

```bash
git diff --stat
git diff -- main.go main_test.go projects_cli.go projects_cli_test.go docs/TODO.md
```

预期：改动只集中在快速启动行为和 TODO 完成状态。
