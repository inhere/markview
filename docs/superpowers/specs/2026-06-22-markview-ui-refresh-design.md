# MarkView UI Refresh Design

| Revision | Date | Author | Notes |
| --- | --- | --- | --- |
| 1 | 2026-06-22 | Codex | 初始设计，覆盖护眼默认配色、工具栏、内容搜索 overlay、CSS 拆分和验证边界。 |

参考预览稿：[../../ui-refresh-mockup.html](../../ui-refresh-mockup.html)

实施计划：[../plans/2026-06-22-markview-ui-refresh.md](../plans/2026-06-22-markview-ui-refresh.md)

## 目标

本轮 UI refresh 的目标是让 MarkView 更像一个安静、耐看的工程文档阅读器，而不是重新设计成新的应用。现有主链路保持不变：左侧文件树、正文阅读区、TOC、设置面板、内容搜索、链接预览、图片和 Mermaid 弹窗、SSE 刷新都继续存在。

本轮只改变视觉呈现、布局密度、搜索入口位置和 CSS 组织方式，不改变用户已有的控制逻辑和偏好存储语义。

## 非目标

不引入 Tailwind、shadcn/ui、图标库或新的前端框架。

不重写 `content-search.ts`、inline navigation、SSE、sidebar/toc collapse、preview panel resize 等核心交互逻辑。

不做大规模组件化重构。`app.css` 会拆分，但拆分应先作为机械搬运完成，避免和视觉修改混在一起。

## 设计原则

MarkView 是本地 Markdown 阅读和预览工具，界面应服务于长时间阅读和快速导航。视觉方向采用低饱和、低噪音、清晰边界的工具型风格。

内容优先：正文阅读区是第一视觉层级，文件树、TOC 和工具栏只提供辅助。

控制保留：设置面板仍负责布局、宽度、字体、颜色模式和主题；内容搜索是独立工具入口，不塞进设置面板。

少改逻辑：能用 CSS 和模板标记解决的，不改 TypeScript 控制流。必须改 TS 时，只补打开/关闭搜索 overlay 这类 UI 状态。

## 默认配色

默认主题改为更护眼的暖灰纸白配色，降低冷白背景和高饱和蓝色带来的刺眼感。

建议默认色板：

```css
:root {
    --bg-canvas: #f3f1ea;
    --bg-paper: #fffdf6;
    --bg-surface: #fbf8ef;
    --border-light: #d8d1c2;
    --border-focus: #a8c8bf;
    --text-heading: #22251f;
    --text-body: #3f443a;
    --text-muted: #74786c;
    --accent-primary: #2f6f68;
    --accent-subtle: #e8f2ee;
    --accent-border: #a8c8bf;
}
```

现有 `github`、`one-dark`、`dracula`、`nord` 主题继续保留。暗色主题不在本轮重做，只修必要对比度和边框可见性。

## 布局

桌面端继续保持三块结构：左侧文件树、中央正文、右侧或浮动 TOC。默认不改变布局模式配置和用户偏好。

正文区域减少传统纸张感，改为轻量阅读面：

- `content-wrapper` 使用响应式 padding，例如 `clamp(20px, 4vw, 52px)`。
- `.paper` 使用 `border-radius: 6px` 到 `8px`，不做大圆角卡片。
- `.paper` padding 使用 `clamp(34px, 5vw, 72px)`，预览面板打开时自然收紧。
- 阴影弱化，主边界靠背景差、细边框和留白表达。

移动端继续隐藏桌面侧栏和 TOC，但要保证正文 padding、工具按钮和搜索 overlay 不产生横向滚动。

## 工具栏

右上角工具栏保留设置入口，但增加独立内容搜索按钮：

```text
[ Search icon ] [ Settings icon ]
```

搜索和设置不合并。搜索是高频工作流入口，设置是低频偏好控制，二者相邻但不嵌套。

设置面板展开后的内容保持原有能力：

- Layout: compact / toc-middle / toc-right / default
- Width: S / M / L / Full
- Font: decrease / reset / increase
- Color: system / light / dark
- Theme: default / github / one-dark / dracula / nord

视觉层面调整按钮尺寸、hover、focus-visible 和分段控件样式；不改变现有事件绑定语义。

## 内容搜索

当前内容搜索位于正文左上角，容易和阅读区争抢注意力。本轮改为 command palette 式 overlay。

打开方式：

- 点击工具栏的 Search icon。
- 支持 `Ctrl+K` / `Cmd+K` 打开。
- `Esc` 关闭。
- 点击遮罩关闭。

桌面端位置：

```css
.content-search-panel {
    position: fixed;
    top: 14vh;
    left: 50%;
    width: min(720px, calc(100vw - 32px));
    transform: translateX(-50%);
}
```

使用中间偏上的位置，不使用真正居中。这样搜索结果向下展开时空间更充足，视线也更自然。

移动端采用接近全屏的面板，避免软键盘弹出后结果区域太小。

现有搜索逻辑应尽量复用：

- `/api/search?q=...` 不变。
- debounce 不变。
- `renderResults()` 可保留或只做图标与样式调整。
- 点击结果后仍通过现有 inline navigation 跳转。
- 文件搜索和内容搜索保持分工：左侧搜索只过滤文件树，右上角搜索查 Markdown 内容。

## 侧栏和 TOC

侧栏继续作为文件导航，不做结构改造。视觉上降低装饰密度：

- `.sidebar-panel` 去掉强渐变，使用纯 surface 背景。
- 面板圆角控制在 8px 内。
- 文件树行高稳定在 30px 到 32px。
- 当前文件用轻背景加左侧强调线，不只依赖文字颜色。
- 文件搜索框高度提高到 30px 左右。

TOC 保持扫读定位能力：

- active 项保留左侧强调线。
- H2/H3 缩进保留。
- 文字密度降低，减少过重的 uppercase 和 letter spacing。

## 图标和文本

UI 图标统一使用 inline SVG。避免 emoji 作为工具图标。

`查看 Markdown` 按钮从 `📄 查看Markdown` 改为 SVG 文件图标加文字。关闭按钮、搜索按钮、设置按钮统一尺寸和 hover/focus 反馈。

## CSS 拆分

`web/src/style/app.css` 当前过大，应先做一次机械拆分，再进行视觉改造。拆分阶段不改变选择器语义和视觉行为。

建议拆分：

```text
web/src/style/app.css
web/src/style/tokens.css
web/src/style/layout.css
web/src/style/toolbar.css
web/src/style/sidebar.css
web/src/style/content.css
web/src/style/overlays.css
```

`app.css` 作为入口文件，只保留 `@import` 和必要说明。Bun 会从 `app.ts` 引入 `app.css`，保持现有构建入口不变。

## 可访问性

所有 icon-only button 必须有 `aria-label` 和 `title`。

搜索 overlay 需要明确焦点行为：

- 打开后 focus 搜索输入框。
- `Esc` 关闭后焦点回到 Search icon。
- 点击结果后关闭 overlay。
- 点击遮罩关闭 overlay。

所有交互按钮使用 `:focus-visible` 焦点环。搜索结果可先保留鼠标点击主路径，键盘上下选择结果可作为后续增强，不放入本轮最低范围。

尊重 `prefers-reduced-motion`，关闭或缩短 pulse、toast slide、overlay 过渡等非必要动效。

## 实施边界

第一步：机械拆分 CSS，只验证构建和测试，不改变 UI。

第二步：应用默认护眼色板和阅读区、侧栏、TOC、工具栏样式。

第三步：迁移内容搜索入口为 overlay。复用现有搜索逻辑，只补打开/关闭状态、快捷键和焦点处理。

第四步：替换明显 emoji UI 图标，统一按钮尺寸和 focus-visible。

## 验证

前端验证：

```powershell
cd web
bun test
bun run build
```

Go 主链路验证：

```powershell
go test ./...
```

视觉检查：

- 1440px 桌面：默认布局、设置面板、搜索 overlay。
- 1024px 桌面：TOC right、preview panel 打开。
- 390px 移动端：正文、工具按钮、搜索 overlay。
- light / dark / system。
- sidebar collapsed / expanded。

## 风险

CSS 拆分可能遗漏 import 顺序，导致样式覆盖关系变化。拆分必须单独提交和验证。

搜索 overlay 如果复用现有 DOM 时处理不当，可能影响点击外部关闭和结果点击跳转。实现时保持 `content-search` 的事件逻辑集中，避免在 `app.ts` 和组件内部重复绑定。

护眼色板不能牺牲对比度。正文、按钮、搜索结果、代码块仍需满足可读性，尤其是浅色主题下的 muted 文本和边框。
