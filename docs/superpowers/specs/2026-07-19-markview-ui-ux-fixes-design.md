# MarkView UI/UX 问题修复设计

| 日期 | 修订人 | 变更 |
| --- | --- | --- |
| 2026-07-19 | Codex | 初版设计：UI005、UI006、UX001、UX002。 |
| 2026-07-19 | Codex | 补充 TOC 展开状态的控制图标尺寸要求。 |

相关文档：

- [TODO 需求](../../TODO.md)
- [实施计划](../plans/2026-07-19-markview-ui-ux-fixes.md)

## 目标

本次只修复 `docs/TODO.md` 中现有的四个 UI/UX 问题，不调整布局架构、不新增依赖：

1. UI005：compact 模式收起文件侧栏后，窄侧栏占满页面高度。
2. UI006：TOC 收起后显示为内容区域底部的独立按钮，不遮挡滚动条。
3. UX001：只有 `path:...` 条件、没有实际搜索词时，不调用搜索 API。
4. UX002：文件变动通知更宽，并提供安全、可点击的文档链接。

FEA002 global server 不属于本次实施范围；UI/UX 完成后另写设计文档。

## UI005：compact 收起侧栏

### 当前问题

compact 使用两行网格，Files 和 TOC 分别占左侧第一、第二行。收起侧栏只改变列宽，Files 仍停留在第一行，因此窄侧栏只显示在页面上部。

### 设计

仅调整 `web/src/style/layout.css`：

- compact 且 `body.sidebar-collapsed` 时，让 `.files-pane` 跨越全部网格行。
- 保持现有收起宽度、图标和展开行为。
- 不修改 TypeScript 折叠状态，不引入新的布局状态。

## UI006：TOC 收起按钮

### 当前问题

- toc-middle 收起后仍是贯穿页面的 44px 竖栏。
- toc-right 收起后仍保留完整宽度面板，只通过平移露出 44px，靠近并遮挡内容滚动条。

### 设计

继续复用 `toc-floating-open` 和现有 `.toc-section-toggle`，只改变桌面端收起样式：

- 收起面板为 `48px × 48px`，只显示 TOC 图标。
- `toc-middle`：位于内容区域左下角，左侧坐标跟随展开后的 TOC 左边界；文件侧栏收起时同步跟随收起宽度。
- `toc-right`：位于内容区域右下角，距右侧和底部各 `16px`。
- 收起时取消完整面板的平移做法，隐藏标题和目录内容。
- 展开时保留现有 TOC 面板尺寸和位置。
- 展开状态的控制按钮保留现有点击区，并将 SVG 图标同步放大到 `24px`，避免图标在面板标题区显得过小。
- preview 打开时继续复用现有右侧避让规则。

不新增第二个 TOC 按钮，也不持久化 TOC 展开状态。

## UX001：path-only 搜索

### 当前问题

前端只判断查询字符串总长度。`path:docs` 长度大于 2，因此会请求 `/api/search`，即使没有实际搜索词。

### 设计

在 `web/src/components/content-search.ts` 中增加一个最小查询判定：

- 移除空白分隔的 `path:...` token 后再检查剩余查询。
- 剩余内容少于 2 个字符时清空结果且不调用 API。
- `path:docs keyword` 正常搜索。
- 纯 `!exclude` 查询继续交给后端处理，不改变现有语义。

测试通过 mock `fetch` 验证是否实际发起请求。

## UX002：文件变动通知

### 当前问题

通知最大宽度只有 360px；单文件只显示不可点击的 basename；多文件只显示数量；文件名通过 `innerHTML` 拼接。

### 设计

继续使用现有 Toast，不增加通知框架：

- 宽度改为响应式，最大约 `520px`，小屏不超过视口可用宽度。
- 单文件显示一个可点击的项目内文档链接。
- 多文件最多显示 3 个可点击链接，剩余文件显示“还有 N 个文件”。
- 链接使用项目内绝对路径，复用现有 document 级 inline navigation。
- 文件路径按 URL path segment 编码。
- 文件名使用 DOM 节点和 `textContent`，不再拼接到 `innerHTML`。
- 关闭按钮保留阻止冒泡行为。

## 测试和验证

每个功能点独立执行 TDD，并独立提交：

1. 先添加能复现问题的测试并确认失败。
2. 实施最小修复并确认目标测试通过。
3. 更新对应 `docs/TODO.md` checkbox。
4. 运行完整 `bun test` 后提交该功能点。

全部 UI/UX 完成后运行：

```bash
cd web && bun test
bun run build
go test ./...
```

再通过真实浏览器验证：

- compact 收起侧栏覆盖完整高度。
- toc-middle 收起按钮位于内容区左下角。
- toc-right 收起按钮位于内容区右下角且不覆盖滚动条。
- path-only 输入不产生搜索请求。
- 单文件和多文件变动通知均可点击跳转。

## 提交拆分

按功能点生成四个实现提交：

1. `fix: fill compact collapsed sidebar height`
2. `fix: dock collapsed toc controls`
3. `fix: skip path-only content searches`
4. `feat: link file change notifications`

设计文档、实施计划和后续 FEA002 global server 设计分别独立提交。
