# Link Preview 分屏预览功能设计

## 功能概述

为 Markdown 内容中的链接添加分屏预览能力：用户 hover 链接时显示预览按钮，点击后在右侧分屏面板中显示目标内容，无需跳转页面。

## 设计决策

### 交互行为

| 项目 | 决策 |
|------|------|
| 按钮触发 | 鼠标 hover 链接时显示，移出后隐藏 |
| 分屏比例 | 左侧主内容 60%，右侧预览 40% |
| 关闭方式 | X 按钮 + ESC 键 + 再次点击同一链接按钮（组合） |
| 按钮图标 | 分屏图标 ⧉（SVG） |
| 头部信息 | 仅关闭按钮 X，无标题栏 |
| 移动端 | 不提供此功能（屏幕宽度 < 1024px） |

### 链接类型处理

| 链接类型 | 处理方式 |
|----------|----------|
| **站内 Markdown 链接** | 复用 `fetchPageSnapshot()`，只渲染 `#content` 内容（不含 sidebar） |
| **站内图片/静态资源** | 不显示预览按钮（排除 `.jpg/.png/.gif/.svg/.pdf` 等扩展名） |
| **站外链接** | iframe 尝试嵌入 → 8秒超时检测 → 失败时显示提示卡片 → 3秒后自动关闭面板 |
| **锚点链接 (#hash)** | 不显示预览按钮（已在同一页面内导航） |

## 技术实现

### 文件改动

| 文件 | 改动类型 | 描述 |
|------|----------|------|
| `frontend/src/link-preview.ts` | 新建 | 核心逻辑模块 |
| `frontend/src/app.ts` | 修改 | 引入并调用 `setupLinkPreview()` |
| `frontend/template.html` | 修改 | CSS 分屏布局 + 面板 HTML + 按钮样式 |

### CSS 结构

```
现有布局：
sidebar | content-wrapper

新增布局：
sidebar | split-container (flex)
           ├── main-panel (60%, 原 content-wrapper 内容)
           └── preview-panel (40%, 新增预览面板)
```

分屏激活时，`content-wrapper` 被替换为 `split-container`，内部包含 `main-panel` 和 `preview-panel`。

### 核心函数设计

```typescript
// link-preview.ts 导出函数

/**
 * 入口函数，初始化链接预览功能
 * - 监听 ESC 键关闭面板
 * - 增强当前页面所有链接
 */
export function setupLinkPreview(): void;

/**
 * 为指定容器内的链接添加 hover 预览按钮
 * @param root - 要扫描的 DOM 根节点
 */
function enhanceLinksInContent(root: HTMLElement): void;

/**
 * 打开或替换右侧预览面板内容
 * @param url - 要预览的目标 URL
 * @param triggerButton - 触发按钮（用于再次点击关闭）
 */
function openPreviewPanel(url: string, triggerButton: HTMLElement): void;

/**
 * 关闭预览面板，恢复原始布局
 */
function closePreviewPanel(): void;

/**
 * 加载站内 Markdown 内容
 * - 复用 fetchPageSnapshot() 获取页面
 * - 只渲染 #content 部分
 */
function loadInternalContent(url: string): Promise<void>;

/**
 * 加载站外链接内容
 * - iframe 嵌入尝试
 * - 8秒超时检测
 * - 失败时显示提示卡片
 */
function loadExternalContent(url: string): Promise<void>;
```

### 按钮样式参考

参考现有 `.mermaid-actions` hover 模式：

```css
.link-preview-btn {
  position: absolute;
  opacity: 0;
  transition: opacity 0.2s;
  /* 分屏图标 SVG */
}

.link-preview-btn.visible {
  opacity: 1;
}
```

### 预览面板结构

```html
<div id="preview-panel" class="preview-panel">
  <div class="preview-header">
    <button class="preview-close">×</button>
  </div>
  <div class="preview-content">
    <!-- 内容区域：iframe 或渲染的 Markdown -->
  </div>
</div>
```

### 状态管理

- 当前预览 URL（用于判断再次点击是否关闭）
- 当前触发按钮引用（用于状态同步）
- iframe 加载状态（用于超时检测）

## 错误处理

### iframe 加载失败场景

1. 目标网站设置 `X-Frame-Options: DENY` 或 `SameOrigin`
2. 目标网站不可达（网络错误）
3. 加载超时（8秒）

**处理方式**：显示友好的提示卡片，内容为"无法预览此链接"，3秒后自动关闭面板。

## 测试要点

1. 站内 Markdown 链接预览正常显示
2. 站外链接 iframe 嵌入或失败提示正确处理
3. ESC 键关闭面板
4. 再次点击同一按钮关闭面板
5. X 按钮关闭面板
6. hover 按钮显示/隐藏正确
7. 移动端（< 1024px）不显示预览按钮
8. 锚点链接不显示预览按钮
9. 图片/静态资源链接不显示预览按钮

## 约束与限制

- 仅在屏幕宽度 ≥ 1024px 时提供功能
- 不处理复杂的跨域 iframe 场景（接受失败提示作为正常行为）
- 不保存预览历史或状态