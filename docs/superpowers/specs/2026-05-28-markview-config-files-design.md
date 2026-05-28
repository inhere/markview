# MarkView 配置文件支持设计

## 背景

MarkView 当前主要通过 CLI 参数、项目 `.env`、环境变量和全局 `markview-projects.json` 项目端口注册表控制启动行为。`docs/TODO.md` 计划新增全局和项目级别的 `markview.json` 配置文件，用于配置服务端启动参数和页面运行时选项。

本设计目标是把 JSON 配置文件作为新的配置来源接入现有启动链路，同时保持已有行为稳定：

- 默认端口继续沿用现有 `6100`。
- CLI 参数仍然拥有最高优先级。
- `.env` 继续作为项目级别覆盖来源。
- `markview-projects.json` 继续只负责项目端口记忆。
- 页面本地偏好仍可覆盖项目默认 UI 设置。

## 目标

1. 支持全局配置文件：`<user-config-dir>/markview/markview.json`。
2. 支持项目配置文件，按顺序查找第一个存在的文件：
   - `markview.local.json`
   - `.markview.json`
   - `markview.json`
3. 支持服务端配置：
   - `server.port`
   - `server.private`
   - `server.watch`
   - `server.watch_dir`
   - `server.watch_skip_dir`
4. 支持页面配置：
   - `ui.preview_exts`
   - `ui.layout`
5. 明确配置来源优先级，并通过测试覆盖关键合并规则。

## 非目标

1. 不让多个项目配置文件叠加合并。
2. 不把 `markview-projects.json` 扩展成通用项目配置文件。
3. 不引入 YAML/TOML 配置格式。
4. 不在配置文件支持的一期开发中完成完整三栏布局；完整布局设计仍包含在本文档中，并拆到二期实施。

## 配置文件格式

推荐配置结构：

```json
{
  "server": {
    "port": 6100,
    "private": false,
    "watch": true,
    "watch_dir": "docs,example",
    "watch_skip_dir": "append:.tmp,coverage"
  },
  "ui": {
    "preview_exts": "append:.ini,.conf",
    "layout": "compact"
  }
}
```

字段说明：

- `server.port`: HTTP 监听端口。默认使用现有 `6100`。
- `server.private`: 是否只监听 `127.0.0.1`。
- `server.watch`: 是否启用文件监听。
- `server.watch_dir`: 监听目录，逗号分隔；空值表示项目目录。
- `server.watch_skip_dir`: 跳过监听目录，支持 `append:` 和 `override:` 前缀。
- `ui.preview_exts`: 右侧预览面板支持的内容文件扩展名，支持 `append:` 和 `override:` 前缀。
- `ui.layout`: 页面布局默认值，支持 `compact`、`toc-middle`、`toc-right`。

Go 侧读取 JSON 时应使用指针字段区分“未配置”和“显式配置为零值”，例如 `*int`、`*bool`、`*string`。这可以避免 `false`、`0` 被误判为未配置。

## 查找规则

全局配置文件路径：

```text
os.UserConfigDir()/markview/markview.json
```

该路径与现有全局项目注册表目录保持一致，便于用户理解和维护。

项目配置文件在最终 `targetDir` 下查找，按以下顺序使用第一个存在的文件：

```text
markview.local.json
.markview.json
markview.json
```

语义建议：

- `markview.local.json`: 本机私有配置，建议加入 `.gitignore`。
- `.markview.json`: 项目共享配置，适合提交到仓库。
- `markview.json`: 兼容显式命名场景。

项目配置文件之间不合并，避免用户误以为多个同级配置文件会叠加。

## 配置优先级

整体优先级：

```text
CLI 选项
> 项目 .env 文件
> 项目配置文件
> 全局 markview-projects.json
> 全局 markview.json
> 内置默认值
```

其中 `markview-projects.json` 只参与 `server.port`，不参与 `private`、`watch`、`ui` 等字段。

端口优先级细化：

1. CLI `--port/-p`
2. 项目 `.env` 的 `MKVIEW_PORT`
3. 项目配置文件 `server.port`
4. `markview-projects.json` 中该项目记住的端口
5. 全局配置文件 `server.port`
6. 内置默认 `6100`

`-p -1` 继续表示随机端口，并继续更新项目端口注册表。固定配置端口被占用时建议直接报错；只有未显式固定端口或 registry 记忆端口场景继续使用现有“向后寻找可用端口”的体验。

## 启动流程调整

当前 `.env` 在目标目录确定前加载。支持项目配置后，应调整为先确定项目目录，再加载项目级配置。

建议流程：

1. 解析 CLI 参数，识别 `--project/-P`、positional directory、entry file 和显式 CLI override。
2. 确定最终 `targetDir` 与 `entryFile`。
3. 加载 `targetDir/.env`。
4. 加载全局 `markview.json`。
5. 加载项目配置文件。
6. 加载全局 `markview-projects.json`，只提取当前项目端口。
7. 按优先级合并到 `config.Cfg`。
8. 校验目标目录、入口文件和配置值。
9. 启动 watcher、HTTP server 和页面渲染。

这样 `markview -P docs` 会读取 `docs/.env`，而不是命令执行目录的 `.env`。

## 模块设计

建议在 `internal/config` 中新增配置加载与合并逻辑：

- `FileConfig`: JSON 文件结构。
- `ServerConfig`: server 配置段。
- `UIConfig`: ui 配置段。
- `ResolvedConfig`: 合并后的中间配置，保留来源信息。
- `LoadGlobalFileConfig()`: 加载全局配置。
- `FindProjectConfig(targetDir string)`: 查找项目配置文件。
- `LoadProjectFileConfig(targetDir string)`: 加载项目配置。
- `MergeSources(...)`: 按优先级合并配置来源。
- `NormalizeListSetting(...)`: 处理 `append:` / `override:` 列表配置。

`config.Cfg` 可以继续作为运行时全局配置，但建议扩展字段：

- `PreviewExts []string`
- `UILayout string`
- `ConfigSources map[string]string` 或更轻量的 source 字段，用于测试和 debug 输出

`bootstrap.prepare()` 负责确定项目目录和加载项目 `.env`，但具体配置合并逻辑应放在 `internal/config`，避免 bootstrap 继续膨胀。

## 页面配置注入

服务端在完整页面渲染时注入页面配置：

```html
<script id="app-config-data" type="application/json">{{ .AppConfigJSON }}</script>
```

注入数据建议为：

```json
{
  "previewExts": [".json", ".jsonl", ".yaml", ".yml", ".toml"],
  "layout": "compact"
}
```

前端新增 `web/src/app-config.ts`：

- 从 `app-config-data` 读取配置。
- 缺失或无效时回退到默认值。
- 归一化扩展名，确保以 `.` 开头。
- 校验 layout，只允许 `compact`、`toc-middle`、`toc-right`。

`web/src/link-preview.ts` 中当前硬编码的 `PREVIEWABLE_CONTENT_EXTENSIONS` 改为读取 app config，这样 `ui.preview_exts` 能影响右侧预览面板。

## Layout 策略

`ui.layout` 按“项目默认值”处理，而不是强制覆盖用户浏览器设置。

优先级：

```text
用户 localStorage layout 偏好
> 服务端注入的项目默认 layout
> compact
```

原因：

- 项目可以提供推荐布局。
- 用户仍可在同一浏览器里保留自己的阅读偏好。
- 不会因为项目配置变化导致每次刷新都覆盖用户设置。

设置面板应新增 layout 控件：

- `compact`
- `toc-middle`
- `toc-right`
- reset/default 操作恢复到服务端注入的项目默认 layout

## 完整布局设计

布局模式定义：

```text
compact    : sidebar(files + toc) | body
toc-middle : files | toc | body
toc-right  : files | body | toc
```

`compact` 保持当前体验：文件树和 TOC 都在左侧 sidebar 中，用户可以分别折叠 Files 区块和整个 sidebar。

`toc-middle` 使用三列结构：左侧是文件树，中间是当前页面 TOC，右侧是正文。这个模式适合文件很多且页面较长的项目，文件导航和页面内导航同时可见，不互相挤占。

`toc-right` 使用三列结构：左侧是文件树，中间是正文，右侧是当前页面 TOC。这个模式适合更接近文档站的阅读体验，正文居中，TOC 作为阅读辅助停靠在右侧。

### DOM 结构

当前模板里的 sidebar 同时包含 Files 和 TOC。为了避免为每种布局维护两套 DOM，建议保留单一数据源和单一渲染入口，但把 Files、TOC、Body 作为三个可布局区域：

```html
<body data-layout="compact">
  <aside class="files-pane">...</aside>
  <aside class="toc-pane">...</aside>
  <main class="content-wrapper">...</main>
</body>
```

`compact` 模式下，CSS 将 `.files-pane` 和 `.toc-pane` 视觉上组合为当前 sidebar 样式；三栏模式下，两者作为独立列展示。

实施时可以先用最小 DOM 调整完成：

- 将现有 `.sidebar-panel-files` 保留为文件面板。
- 将现有 `.sidebar-panel-toc` 保留为 TOC 面板。
- 在三栏模式下通过 CSS grid/flex 将 TOC 面板移出合并视觉。
- 避免复制 TOC DOM，确保 `generateTOC()`、`highlightTOC()` 仍只操作一个 `#toc-list`。

### CSS 布局

桌面端建议使用页面级 grid：

```text
compact:
  grid-template-columns: var(--sidebar-width) minmax(0, 1fr)

toc-middle:
  grid-template-columns: var(--files-width) var(--toc-width) minmax(0, 1fr)

toc-right:
  grid-template-columns: var(--files-width) minmax(0, 1fr) var(--toc-width)
```

推荐默认宽度：

- files: 沿用当前 sidebar 宽度偏好，默认 `280px`。
- toc: 默认 `240px`，最小 `180px`，最大 `360px`。
- body: 继续使用 `--layout-max-width` 控制正文内部宽度。

`toc-middle` 和 `toc-right` 下，正文容器不应被强行拉满到失去阅读宽度控制；外层列占满剩余空间，正文内部仍按用户选择的 Width 设置居中。

移动端建议统一回退为 `compact` 的单栏/抽屉式行为，不在小屏强行展示三列。回退只影响视觉布局，不改变用户保存的 layout 偏好；当视口恢复到桌面宽度时继续使用用户选择的模式。

### 折叠和 Resize

现有 sidebar resize 可以继续控制 files 面板宽度。三栏模式新增 TOC 宽度时，不建议二期首版增加 TOC resize；默认固定宽度即可，降低交互复杂度。

折叠策略：

- `compact`: 沿用当前 sidebar 折叠和 Files 区块折叠。
- `toc-middle`: sidebar 折叠只影响 files 面板；TOC 保持可见。
- `toc-right`: sidebar 折叠只影响 files 面板；TOC 保持可见。
- 后续如需单独折叠 TOC，可增加独立 `markview:toc-collapsed` 偏好，不与当前 sidebar 状态复用。

### 设置面板交互

设置面板新增 Layout 分段控件：

```text
Layout: Compact | TOC Middle | TOC Right
```

行为：

- 页面初始化时读取 `localStorage` layout。
- 若本地无 layout，使用服务端注入的项目默认 `ui.layout`。
- 用户切换 layout 后立即应用，并保存到 `localStorage`。
- Reset/Default 操作恢复到服务端注入的项目默认 layout，并清除本地 layout 覆盖。

新增本地存储键：

```text
markview:layout-mode
```

### 前端状态流

页面初始化流程：

1. `readAppConfig()` 读取服务端注入配置。
2. `readStoredPreferences()` 读取本地偏好。
3. `resolveLayoutMode(localPreference, appConfig.layout)` 得到最终 layout。
4. `applyLayoutMode(layout)` 设置 `document.body.dataset.layout` 或 `document.documentElement.dataset.layout`。
5. 渲染文件树、TOC 和正文增强逻辑。

局部页面导航时不重新解析 layout，只更新正文、当前文件路径、文件树和 TOC。layout 是页面级状态，应在 SPA 风格导航中保持稳定。

### 可访问性和兼容性

- Layout 控件使用 button group 或 segmented control，并通过 `aria-pressed` 表示当前选中项。
- 三栏模式下 DOM 顺序建议保持 Files、TOC、Body，键盘导航顺序稳定；`toc-right` 只通过 CSS grid 调整视觉位置。
- 移动端回退时，不应隐藏正文内容或让 TOC 覆盖正文。
- `preview-active` 右侧预览面板打开时，三栏布局应优先保证正文和预览可读；可以临时隐藏独立 TOC 或降低 TOC 列宽，避免四列拥挤。

## 错误处理

配置错误应尽早失败，避免启动后行为不可预期。

- JSON 语法错误：启动失败，提示配置文件路径和解析错误。
- `server.port` 非法：启动失败。
- `ui.layout` 不在支持值内：启动失败。
- `append:` / `override:` 前缀非法：启动失败。
- `preview_exts` 中扩展名缺少 `.`：自动补齐。
- 配置文件不存在：忽略。
- 全局配置目录不存在：忽略。

## 测试计划

Go 测试：

- 全局配置文件读取。
- 项目配置文件查找优先级。
- CLI > `.env` > 项目配置 > registry > 全局配置 > 默认值。
- `server.watch_skip_dir` 的 `append:` 和 `override:`。
- `ui.preview_exts` 的 `append:` 和 `override:`。
- 无效 JSON、无效 port、无效 layout 的错误信息。
- `--project/-P` 场景下 `.env` 来自选中的项目目录。

前端测试：

- `app-config-data` 缺失时使用默认配置。
- `previewExts` 增加新扩展名后，链接预览识别生效。
- 无效 layout 回退到 `compact`。
- 本地 layout 偏好覆盖服务端默认值。
- Layout 设置面板切换 `compact`、`toc-middle`、`toc-right` 后写入本地偏好并更新页面 dataset。
- Reset/Default 清除本地 layout 覆盖并恢复服务端默认 layout。

集成验证：

- `go test ./...`
- `cd web && bun test`
- 手动验证：
  - 无配置文件时行为不变。
  - 全局配置端口生效。
  - 项目配置覆盖全局配置。
  - `.env` 覆盖项目配置。
  - CLI 覆盖 `.env`。
  - `preview_exts` 可让新增扩展名在右侧预览面板打开。

## 实施拆分

建议分两阶段实施。

阶段一：配置加载和页面配置注入

- 新增全局/项目配置文件加载。
- 调整 `.env` 加载到目标项目目录。
- 完成 server 配置合并。
- 注入 `preview_exts` 和 `layout` 到页面。
- 让 `preview_exts` 在前端链接预览中生效。
- 完成 `layout` 类型、默认值、localStorage key 和页面 dataset 的基础链路。
- 保持现有 `compact` 布局视觉不变。

阶段二：布局设置和三栏布局

- 新增设置面板 layout 控件。
- 实现 `compact`、`toc-middle`、`toc-right`。
- 本地 layout 偏好覆盖服务端项目默认值。
- 移动端统一回退或适配为单栏/compact 行为。
- 验证右侧预览面板打开时三栏布局不出现内容重叠。

## 开放问题

1. 固定配置端口被占用时是否严格报错，还是沿用自动寻找下一个可用端口。当前建议严格报错。
2. `watch_dir` 是否允许绝对路径。当前建议仅允许项目内相对路径，降低越权监听风险。
3. TOC 面板是否需要独立 resize。当前建议二期先使用固定宽度，后续按需要再增加。
