# markview.json 完整配置示例

本文档展示 MarkView 当前支持的完整 `markview.json` 配置项。项目级配置文件可放在项目根目录，按以下优先级查找第一个存在的文件：

1. `markview.local.json`
2. `.markview.json`
3. `markview.json`

全局配置文件固定为用户配置目录下的 `markview/markview.json`：

- Windows: `%APPDATA%\markview\markview.json`
- macOS: `$HOME/Library/Application Support/markview/markview.json`
- Linux: `$XDG_CONFIG_HOME/markview/markview.json` 或 `$HOME/.config/markview/markview.json`

## 完整示例

```json
{
  "server": {
    "port": 6100,
    "private": false,
    "watch": true,
    "watch_dir": "docs,example",
    "watch_skip_dir": "append:.cache,coverage",
    "include_dir": ".docs,.wiki"
  },
  "ui": {
    "preview_exts": "append:.ini,.conf",
    "iframe_hosts": "intranet.local,192.168.1.20:8080,*.hyy.preview.test",
    "layout": "compact"
  }
}
```

## server

`port`: 预览服务监听端口。未配置时使用自动端口模式：优先使用已保存的项目端口；没有记录时优先尝试 `6100`，端口被占用会继续查找可用端口。

`private`: 是否只监听本机 `127.0.0.1`。默认 `false`，即允许局域网内通过本机 IP 访问。

`watch`: 是否监听文件变化并通过 SSE 实时刷新页面。默认 `true`。

`watch_dir`: 指定要监听的目录，多个目录用英文逗号分隔。未配置时监听目标目录。

`watch_skip_dir`: 指定监听时要跳过的目录，多个目录用英文逗号分隔。支持列表模式前缀：

```text
append:.cache,coverage
override:.cache
```

`append:` 会在默认跳过目录 `node_modules,dist,tmp,temp` 后追加新目录；`override:` 会覆盖默认列表，但 `node_modules` 始终会被跳过。

`include_dir`: 指定要在文件树中放行展示的跳过目录，多个目录用英文逗号分隔。常用于让 `.docs`、`.wiki` 这类点开头的文档目录显示在 file tree 中。`.git` 和 `node_modules` 始终不会展示。

## ui

`preview_exts`: 右侧预览面板支持的文件扩展名，多个扩展名用英文逗号分隔。支持列表模式前缀：

```text
append:.ini,.conf
override:.md,.txt
```

默认扩展名为 `.md,.json,.jsonl,.yaml,.yml,.toml,.html`。其中 `.html` 会在右侧预览面板中通过 iframe 渲染页面，其余内容文件默认以代码形式展示。未写点号的扩展名会自动补成 `.ext`，并统一转成小写。

`iframe_hosts`: 允许用 iframe 在右侧预览面板中打开的外部地址 host 白名单，多个 host 用英文逗号分隔。匹配规则使用浏览器 URL 的 `host`，包含端口但不包含协议和路径。例如 `http://192.168.1.20:8080/app` 对应 `192.168.1.20:8080`，`http://intranet.local/app` 对应 `intranet.local`。

后缀匹配支持 `*.hyy.preview.test` 或 `.hyy.preview.test`，可匹配 `foo.hyy.preview.test`、`bar.foo.hyy.preview.test` 等子域名；不会匹配根域 `hyy.preview.test`，也不会误匹配 `evil-hyy.preview.test`。

未配置 `iframe_hosts` 时，MarkView 不会给外部链接显示预览按钮。即使 host 已加入白名单，目标站点仍可能因为 `X-Frame-Options` 或 `Content-Security-Policy: frame-ancestors` 拒绝被 iframe 嵌入。

`layout`: 页面布局模式，支持以下值：

- `compact`: 默认布局，文件树和 TOC 合并在左侧，内容在右侧
- `toc-middle`: 三栏布局，文件树、TOC、正文依次排列
- `toc-right`: 文件树在左侧，正文居中，TOC 作为右侧浮动面板

## 环境变量

项目 `.env` 文件和系统环境变量支持以下键；优先级高于全局/项目配置文件：

```dotenv
MKVIEW_PORT=8080
MKVIEW_ENTRY=guide.md
MKVIEW_DEBUG=false
MKVIEW_WATCH=true
MKVIEW_WATCH_DIR=docs,example
MKVIEW_WATCH_SKIP_DIR=append:.cache,coverage
MKVIEW_INCLUDE_DIR=.docs,.wiki
MKVIEW_PREVIEW_EXTS=append:.html,.ini
MKVIEW_IFRAME_HOSTS=*.hyy.preview.test,intranet.local
```

环境变量可以覆盖运行时服务配置，也可以通过 `MKVIEW_PREVIEW_EXTS` 覆盖 `ui.preview_exts`，通过 `MKVIEW_IFRAME_HOSTS` 覆盖 `ui.iframe_hosts`。`ui.layout` 仍需通过 `markview.json` 配置。

## 配置优先级

运行时会按以下顺序合并配置，越靠后的来源优先级越高：

1. 全局 `markview.json`
2. 已保存的项目端口记录
3. 项目配置文件
4. 项目 `.env` 文件或系统环境变量
5. CLI 选项

因此，`markview --port 6543` 会覆盖 `MKVIEW_PORT` 和配置文件中的 `server.port`；`MKVIEW_PORT=8080` 会覆盖项目配置文件中的 `server.port`。
