# Inline Navigation And Cache Design

## Goal

将当前基于整页跳转的 Markdown 阅读体验调整为“同壳无刷新切页”：

- 左侧目录树点击切换 Markdown 时不再整页刷新
- 正文中的站内 Markdown 链接同样采用无刷新导航
- SSE 热更新不再调用 `window.location.reload()`，而是复用同一套局部渲染逻辑
- 静态资源设置明确缓存策略，减少重复校验和加载

## Root Cause

当前体验卡顿来自两个层面：

1. 手动点击目录树或正文站内链接时，浏览器执行完整页面跳转
2. SSE 收到 `reload` 后直接触发整页 `window.location.reload()`

这会导致：

- HTML 文档重新请求
- `app.js` 和 CSS 重新校验或重新拉取
- 工具栏、目录树、TOC、Mermaid、滚动状态全部重建

## Target UX

- Markdown 文档切换表现为同一阅读器外壳中的内容切换
- 地址栏正常更新，浏览器前进/后退可用
- 工具栏状态保留
- 切换文档时只替换必要内容，避免闪烁
- Markdown 变更热更新时尽量保留当前滚动位置

## Architecture

### Server remains SSR-first

服务端仍返回完整 HTML 页面，不新增专用 JSON 接口。前端通过 `fetch` 目标页面 HTML，然后解析并提取局部区域进行替换。

这保持了：

- 直接访问 URL 仍可工作
- 服务端渲染逻辑不需要分叉成 HTML + JSON 两套输出
- SSE 热更新和手动导航可以复用同一套“抓取 HTML -> 局部更新”的管线

### Frontend split

当前前端初始化拆分为两层：

- `setupOnce()`
  - 只执行一次
  - 负责工具栏、SSE、导航拦截、`popstate`、全局键盘事件
- `renderPage()`
  - 每次初始加载、局部导航、热更新都可重复执行
  - 负责目录树、TOC、高亮代码、Mermaid、标题和当前页面 DOM 的同步

## Replace Scope

每次局部导航或热更新时，仅替换：

- `document.title`
- `#content`
- `.file-meta`
- `#file-tree-data`
- `#current-file-path-data`

随后重新执行：

- `renderFileTree()`
- `generateTOC()`
- 代码高亮
- Mermaid 转换
- 当前 TOC 高亮计算

不替换：

- 工具栏 DOM
- Mermaid modal 外壳
- SSE 连接
- 全局事件绑定

## Navigation Rules

前端只接管这些链接：

- 同源链接
- 指向 Markdown 页面或目录入口的站内链接
- 没有 `target=_blank`
- 没有修饰键点击
- 不是下载链接

以下仍交给浏览器默认行为：

- 外链
- 非 Markdown 静态资源
- 新窗口行为
- 用户按住 `Ctrl` / `Cmd` / `Shift` / 中键

## History Behavior

- 手动切文档：`pushState`
- 浏览器前进/后退：监听 `popstate`，重新抓取目标页面并局部渲染
- 同页锚点：不抓取 HTML，只滚动到目标
- 跨页带锚点：完成局部导航后再滚动到锚点

## SSE Behavior

- 收到 `reload` 后不再执行 `window.location.reload()`
- 改为重新抓取当前 URL 并局部更新
- 热更新默认保留滚动位置

## Cache Strategy

- `/static/*`
  - 设置 `Cache-Control: public, max-age=0, must-revalidate`
  - 让浏览器复用缓存但在需要时校验
- Markdown HTML 页面
  - 设置 `Cache-Control: no-store`
  - 避免读取到旧页面壳
- Highlight CSS
  - 从 CDN 改为本地静态资源，减少跨站请求和重复校验

## Risks

- 前端需要避免重复绑定全局事件，否则切页多次后会产生重复响应
- Mermaid 节点需要在每次内容替换后重新处理，但不能污染 modal 的全局状态
- 内容区替换后，正文中的站内链接也需要自动具备无刷新导航能力
- 解析 HTML 时必须容忍目标页面内容缺失，发生异常时需要退回整页导航，避免卡死
