# TODO

<!--
简单的直接使用一行 checklist 说明即可。
需要附带较长说明的，使用标题+说明方式新建。使用emoji 表情状态图标(wait: ⏳|ing: 🔄|done: ✅)
-->

- [x] 新增 --private 选项，指定是否监听为本地server，不公开
- [x] 新增 --projects 选项，支持多项目管理和快速启动（详细说明见下面的章节）
- [x] 未指定端口时，按项目路径自动保存获取到的端口号（详细说明见下面的章节）
- [x] bug: 右侧打开的预览面板，mermaid, highlight 不会渲染显示
- [x] 点击 file-tree 的目录时，如果没有 entry file, 则默认列出目录下的所有文件/目录
- [ ] 新增支持分享单个页面的选项，用户可以在预览面板点击分享按钮，生成分享链接
  - 分享链接包含当前页面的 url，用户可以分享给其他人，其他人可以通过点击链接访问当前页面
  - 分享链接过期时间为1小时，过期后需要重新生成链接
  - 分享链接只能在当前项目内有效，不能跨项目使用，且只能查看当前页面，不显示目录列表
  - 要支持这个功能，可能需要简单的 auth 机制，例如使用 basic auth 等技术
- [x] UI优化 现在有的table 高度非常高，可能超出浏览器高度了 导致内容区域被大量占用，影响阅读体验。
  - 优化为 超过指定高度后启用y滚动，下边框添加可点击区，点击后也可以完全展开高度
- [x] feat: 现在页面拦截了内部的 .md 链接，增强支持 .json, .jsonl, .yaml, .toml 等常见内容文件链接
  - 点击链接后在预览面板查看高亮的文件内容，而不需要打开新页面
- [x] 新增支持全局和项目级别的配置文件 `markview.json`（详细说明见下面对应章节）
  - [x] 一期：配置文件读取/合并、页面配置注入、preview_exts 生效、layout 基础链路
  - [x] 二期：设置面板 layout 控件和完整布局模式

## 未指定端口时，自动保存获取到的端口号 ✅

未指定端口时，自动保存获取到的端口号，避免每次启动都需要手动指定端口号，或者自动端口变化导致之前的地址无法访问。

不再使用 `-p -1` 表示自动端口；直接不传 `-p/--port` 即进入自动端口模式，并保存获取到的端口号到 `markview-projects.json` 文件。

- 下次启动时，自动读取该文件，优先使用保存的端口号启动服务，如果端口号被占用，自动选择下一个可用端口号并更新文件

json 格式参考：

```json
{
  "project path": {
    "port": 6100,
    "name": "project name(default is directory name)",
    "added": "2025-08-03T15:00:00"
  }
}
```

## 支持多项目管理和快速启动 ✅

支持多项目管理和快速启动，用户可以在不同项目之间快速切换，而不需要cd到项目目录下才能启动服务。

命令：

```bash
# 列出所有已保存的项目
markview --projects list
# 清理不存在的项目记录
markview --projects prune
# 显示或删除指定项目
markview --projects show|remove <project name>
# 启动指定项目的预览服务 会自动切换到项目目录，然后启动服务
markview -P|--project <project name>
```

## 新增支持全局和项目级别的配置文件 ✅

相关文档：
- [设计文档](superpowers/specs/2026-05-28-markview-config-files-design.md)
- [一期实施计划](superpowers/plans/2026-05-28-markview-config-files-phase-1.md)
- [二期实施计划](superpowers/plans/2026-05-28-markview-config-files-phase-2.md)

新增支持全局和项目级别的配置文件 `markview.json`，用户可以在项目目录下创建该文件，来配置项目的预览服务参数。

项目下按 `markview.local.json`, `.markview.json`, `markview.json` 依次查找，第一个找到的文件会被使用。

可以配置：

server 配置：
- port: 预览服务监听的端口号；未配置时走自动端口
- private: 是否监听为本地server，默认false
- watch: bool 是否监听目录变化，默认 true
- watch_dir: string 监听的目录配置，多个目录用逗号分隔，默认项目目录
- watch_skip_dir: string 监听时，跳过的的目录配置，多个目录用逗号分隔，默认空字符串
  - 支持前缀 override: 覆盖默认的skip设置, append(default): 追加到默认的设置
- include_dir: string 放行被跳过的目录，多个目录用逗号分隔，例如 `.docs,.wiki`
  - 可通过 `MKVIEW_INCLUDE_DIR` 配置
  - 用于让点开头的文档目录显示到 file-tree；`.git` 和 `node_modules` 始终跳过

UI 页面配置（server渲染设置到页面）：
- preview_exts: string 支持的预览文件扩展名，多个逗号分隔，默认 .md, .json, .jsonl, .yaml, .yml, .toml, .html
  - 支持前缀 override: 覆盖默认的扩展名设置, append(default): 追加到默认的设置
  - 配置后将会在右侧预览面板显示对应ext的文件内容，而不会打开新页面；`.html` 使用 iframe 渲染页面
- iframe_hosts: string 允许在右侧预览面板用 iframe 打开的外部 host 白名单，多个 host 用逗号分隔，例如 `intranet.local,192.168.1.20:8080`
  - 未配置时不允许外部链接使用 iframe 预览
  - 匹配浏览器 URL 的 host，包含端口，不包含协议和路径
- layout: string 内容/TOC/目录布局位置，默认 compact
  - 支持 compact(就是现在的布局：toc与file-tree合并,右侧内容), toc-middle(file-tree|toc|body), toc-right(file-tree|body + floating toc)
  - toc-right 中 TOC 是右侧浮动面板，不为 TOC 预留固定列；打开预览面板时默认隐藏 TOC，但可通过按钮手动打开用于跳转
  - 默认是 compact模式，与file-tree合并，不占用额外空间。但是在文件多，内容长时，不方便查看，会影响阅读体验。

> 多种配置覆盖优先级：CLI 选项 > 项目 `.env` 文件 > 项目配置文件 > 全局 `markview-projects.json` > 全局 `markview.json` 文件

同时页面的设置面板也可以配置 布局设置
