# MarkView Frontend Interaction Fixes Design

| Revision | Date | Author | Notes |
| --- | --- | --- | --- |
| 1 | 2026-06-04 | Codex | 初始设计，覆盖 sidebar 拖拽、Mermaid 复制、高亮 fallback、链接预览刷新和长页面滚动优化。 |

实施计划：[2026-06-04-front-end-interaction-fixes.md](../plans/2026-06-04-front-end-interaction-fixes.md)

## 目标

恢复前几次布局改动中回退的关键前端体验：sidebar 可拖拽调宽、Mermaid 图可复制源码、未知语言代码块不破坏高亮体验、SSE 刷新后内部链接仍显示预览按钮，并降低长页面中表格和 Mermaid 图造成的滚动卡顿。

## 设计

sidebar 调宽继续沿用现有 `sidebar-resize.ts` 和 `--sidebar-width` 偏好存储，不改变用户配置格式。修复重点放在事件命中、拖拽过程禁用宽度过渡、以及 CSS 层级，确保 `files-pane` 右侧手柄在桌面布局中可操作，折叠状态仍隐藏。

Mermaid 复制按钮放入现有 `.mermaid-actions`，复制内容使用 `.mermaid-container` 的 `data-source` 原始源码。复制状态复用代码块按钮的交互语义：点击后显示短暂成功态，失败只记录控制台错误，不打断图表查看。

语法高亮保留当前轻量的 `highlight.js/lib/core` 按需注册方式，不在本轮切换库。新增统一高亮入口，对未注册语言 fallback 到 plaintext，并补充常见语言别名，避免未知语言触发不稳定体验。

链接预览增强函数改成幂等：已被 `.link-preview-wrapper` 包裹且已有按钮的链接跳过，刷新后重新写入的内容正常补按钮。这样 SSE 触发 inline refresh 后，`renderCurrentPage()` 再次增强链接仍稳定。

滚动性能先做低风险 CSS 优化：对表格包装容器和 Mermaid 容器使用 `content-visibility: auto` 与 `contain-intrinsic-size`，让视口外复杂内容减少布局和绘制成本；表格滚动区域限制 overscroll，降低嵌套滚动干扰。

## 测试

新增和更新前端单元测试覆盖拖拽宽度、Mermaid 复制按钮、未知语言 fallback、链接预览幂等增强、CSS 性能规则。完成后运行 `bun test`，并因改动触及 MVP 浏览主链路，运行 `go test ./...`。
