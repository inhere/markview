# Projects CLI 设计

## 目标

在现有 `~/.config/markview/markview-projects.json` 项目注册表基础上，新增项目管理和快速启动命令，让用户可以查看、维护已保存项目，并且不需要先 `cd` 到项目目录就能启动指定项目。

## 范围

本设计覆盖：

- `markview --projects list`
- `markview --projects show <project>`
- `markview --projects remove <project>`
- `markview --projects prune`
- `markview -P <project>`
- `markview --project <project>`

本设计不覆盖单页分享、鉴权、分享链接过期，也不改变 registry 文件位置。

## 复用 Registry

这些命令复用项目端口记忆功能已经引入的 registry：

```text
~/.config/markview/markview-projects.json
```

当前记录结构保持兼容：

```json
{
  "/abs/project/path": {
    "port": 6100,
    "name": "project-name",
    "added": "2026-05-14T15:00:00+08:00"
  }
}
```

第一版不做 schema migration。以后如果需要 `lastUsed` 等字段，可以在不破坏旧记录的前提下追加。

## 项目标识与匹配规则

registry 的 key 是项目绝对路径。项目命令接收的 `<project>` 选择器支持以下匹配方式：

1. 按 registry key 的完整路径匹配，路径先经过 `filepath.Abs` 和 `filepath.Clean`
2. 按记录里的 `name` 精确匹配
3. 按 registry key 的目录 basename 精确匹配

第一版不做模糊匹配。误删或误启动项目的代价比多输入几个字符更高，所以匹配规则应保持保守、可解释。

如果同一个选择器匹配到多个记录，命令必须失败，并输出清晰的歧义信息，列出匹配到的项目名和路径。用户可以改用完整路径选择项目。

## 阶段一：Registry 管理命令

阶段一新增依赖：

```text
github.com/gookit/cliui
```

`cliui` 只用于 CLI 展示层：

- `list` 使用表格展示项目列表。
- `show` 使用 key/value 信息块展示单个项目。
- `remove` 和 `prune` 使用简洁状态消息。

`internal/projects` 仍保持纯数据包，不依赖 `cliui`，不直接输出内容。

### `markview --projects list`

列出所有已保存项目，然后退出，不启动 HTTP server。

使用 `cliui` 渲染项目表格，展示字段：

```text
Saved projects:
NAME          PORT  ADDED                 PATH
markview      6100  2026-05-14T15:00:00  D:\work\aidev\lite-tools\markview
docs          6101  2026-05-14T16:00:00  D:\work\docs
```

如果 registry 为空：

```text
No saved projects.
```

排序规则保持稳定、可预测：

1. 按 `name` 升序
2. 同名时按路径升序

### `markview --projects show <project>`

显示单个项目记录，然后退出，不启动 HTTP server。

使用 `cliui` 渲染 key/value 信息，展示：

```text
Name: markview
Path: D:\work\aidev\lite-tools\markview
Port: 6100
Added: 2026-05-14T15:00:00+08:00
Exists: yes
```

如果项目不存在，返回错误。

### `markview --projects remove <project>`

删除一个项目记录，然后退出，不启动 HTTP server。

行为：

- 使用统一的 selector 规则解析 `<project>`
- 找不到时返回错误
- 匹配多个项目时返回歧义错误
- 删除 registry 记录后保存文件
- 不删除磁盘上的项目文件

输出：

```text
Removed project: markview (D:\work\aidev\lite-tools\markview)
```

### `markview --projects prune`

移除项目路径已经不存在、或不再是目录的记录，然后退出，不启动 HTTP server。

输出：

```text
Removed 2 missing project records.
```

如果没有移除任何记录：

```text
No missing project records.
```

## 阶段二：快速启动

### `markview -P <project>`

### `markview --project <project>`

启动指定已保存项目的预览服务。

行为：

1. 加载 registry。
2. 使用统一 selector 规则解析 `<project>`。
3. 找不到或匹配多个项目时，在启动 server 前返回错误。
4. 将 target directory 设置为匹配到的 registry 路径。
5. 复用现有 entry file 逻辑，默认使用 `README.md`；如果用户额外传入 positional `default-entry`，则使用该 entry。
6. 后续走现有启动流程。

端口行为：

- 如果用户没有传 CLI port，也没有设置 `MKVIEW_PORT`，继续使用项目端口记忆。
- 如果用户传 `-p -1`，继续使用项目端口记忆。
- 如果用户传固定 port 或设置了 `MKVIEW_PORT`，不通过 registry 更新端口，保持现有显式配置优先规则。

命令不应调用 `os.Chdir`，除非未来有明确需求。把解析后的 target path 传给 `config.Cfg.Init` 已经足够，也能避免改变进程工作目录带来的隐式影响。

## CLI 形态

新增两个 flag：

```text
--projects string   管理已保存项目：list, show, remove, prune
-P, --project       按名称或路径启动已保存项目
```

`show` 和 `remove` 的项目选择器来自 flag 后面的第一个 positional 参数：

```bash
markview --projects show markview
markview --projects remove markview
```

这样可以保持当前单命令 `cflag` 结构，不需要为了这几个轻量命令引入完整 subcommand 框架。

## 错误处理

registry 对普通启动保持宽容，但对管理命令应更严格：

- registry 文件不存在：视为空 registry。
- registry JSON 损坏：`--projects` 命令返回错误，因为无法安全编辑未知内容。
- `remove` 和 `prune` 保存失败：返回错误。
- `show`、`remove` 或 `-P` 缺少项目选择器：返回类似 usage 的错误。
- selector 匹配多个项目：返回错误，并列出所有匹配项。

## 代码边界

扩展 `internal/projects`，新增纯 registry 查询和更新能力：

- 按名称和路径排序列出记录
- 解析单个 selector
- 删除单个记录
- 清理不存在的路径

CLI 解析和输出放在 `main` package 中，可以放在 `main.go` 或新增 `projects_cli.go`。`projects_cli.go` 可以依赖 `github.com/gookit/cliui` 负责终端展示；`internal/projects` 不应该打印 stdout，也不应该知道 `cflag` 或 `cliui`。

## 测试策略

阶段一测试：

- list 排序
- selector 按 name 和 path 解析
- selector 歧义错误
- remove 只删除 registry 记录
- prune 删除不存在目录并保留存在目录
- 管理命令不会启动 server

阶段二测试：

- `-P` 解析项目并在 config 初始化前设置 target directory
- 未知 selector 和歧义 selector 在 server 启动前失败
- 默认 entry 行为保持不变
- `-P` 场景下端口 registry 激活规则保持不变

## 推进方式

分两个阶段实现：

1. Registry 管理命令。
2. 按已保存项目快速启动。

每个阶段都应该能独立测试、独立提交。
