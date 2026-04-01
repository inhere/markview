# Link Preview 分屏预览功能设计

## 功能概述

为 Markdown 内容中的链接添加分屏预览能力：用户 hover 链接时显示预览按钮，点击后在右侧分屏面板中显示目标内容，无需跳转页面。

## 设计决策

### 交互行为

| 项目 | 决策 |
|------|------|
| 按钮触发 | 鼠标 hover 链接时显示，移出后隐藏 |
| 分屏比例 | 左侧主内容 60%，右侧预览 40% |
| 打开预览 | 点击按钮 → 打开面板；若面板已打开不同链接 → 替换内容 |
| 关闭方式 | X 按钮 + ESC 键 + 再次点击同一链接按钮（组合） |
| 按钮图标 | 分屏图标 ⧉（SVG） |
| 头部信息 | 仅关闭按钮 X，无标题栏 |
| 移动端 | 不提供此功能（屏幕宽度 < 1024px） |

### 链接类型处理

| 链接类型 | 处理方式 |
|----------|----------|
| **站内 Markdown 链接** | 复用 `fetchPageSnapshot()`，只渲染 `#content` 内容（不含 sidebar） |
| **站内图片/静态资源** | 不显示预览按钮（排除 `.jpg/.png/.gif/.svg/.webp/.avif/.mp4/.webm/.mp3/.pdf` 等扩展名） |
| **站外链接** | iframe 尝试嵌入 → 8秒超时检测 → 失败时显示提示卡片 → 3秒后自动关闭面板 |
| **锚点链接 (#hash)** | 不显示预览按钮（已在同一页面内导航） |

## 技术实现

### 文件改动

| 文件 | 改动类型 | 描述 |
|------|----------|------|
| `web/src/link-preview.ts` | 新建 | 核心逻辑模块 |
| `web/src/app.ts` | 修改 | 引入并调用 `setupLinkPreview()` |
| `web/template.html` | 修改 | CSS 分屏布局 + 面板 HTML + 按钮样式 |

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
 * - 若面板未打开：打开新面板，加载内容
 * - 若面板已打开同一链接：关闭面板
 * - 若面板已打开不同链接：替换内容，不关闭面板
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
    <!-- 加载状态指示器 -->
    <div class="preview-loading">
      <div class="loading-spinner"></div>
      <span>加载中...</span>
    </div>
    <!-- 内容区域：iframe 或渲染的 Markdown -->
    <div class="preview-body" style="display: none;">
      <!-- iframe 或渲染的 HTML 内容 -->
    </div>
    <!-- 错误提示卡片 -->
    <div class="preview-error" style="display: none;">
      <span>无法预览此链接</span>
    </div>
  </div>
</div>
```

**加载状态行为**：
- 打开面板时默认显示 `.preview-loading`
- 站内链接：内容渲染完成后隐藏 loading，显示 `.preview-body`
- 站外链接：iframe onload 后隐藏 loading，显示 iframe；超时/失败时隐藏 loading，显示 `.preview-error`，3秒后自动关闭面板

**面板尺寸**：
- 预览面板高度：100vh（全视口高度）
- `.preview-content` 区域：overflow: auto，内容可滚动

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

### 功能测试场景

| ID | 场景 | 操作步骤 | 预期结果 |
|----|------|----------|----------|
| T1 | 站内 Markdown 链接预览 | 1. 创建 test.md 含链接到 other.md<br>2. hover 链接 → 点击预览按钮 | 右侧面板显示 other.md 内容（无 sidebar），样式正确渲染 |
| T2 | 站外链接 iframe 成功 | 1. Markdown 含外部链接（如 wikipedia.org）<br>2. hover → 点击预览按钮 | iframe 正常加载目标页面内容 |
| T3 | 站外链接 iframe 失败 | 1. Markdown 含 X-Frame-Options: DENY 的链接<br>2. hover → 点击预览按钮 | 显示"无法预览此链接"提示，3秒后面板自动关闭 |
| T4 | ESC 键关闭面板 | 1. 打开预览面板<br>2. 按 ESC 键 | 预览面板关闭，恢复原始布局 |
| T5 | 再次点击同一按钮关闭 | 1. 点击按钮 A 打开面板<br>2. 再次点击按钮 A | 预览面板关闭，按钮状态恢复 |
| T6 | 点击不同链接替换内容 | 1. 点击按钮 A 打开预览<br>2. 点击按钮 B（不同链接） | 预览面板保持打开，内容替换为链接 B 的内容 |
| T7 | X 按钮关闭面板 | 1. 打开预览面板<br>2. 点击面板头部 X 按钮 | 预览面板关闭 |
| T8 | hover 显示/隐藏按钮 | 1. 鼠标进入链接区域<br>2. 鼠标离开链接区域 | 进入时按钮 opacity: 1，离开时 opacity: 0 |
| T9 | 移动端隐藏功能 | 1. 浏览器宽度 < 1024px<br>2. hover 链接 | 不显示预览按钮 |
| T10 | 锚点链接无预览 | 1. Markdown 含 `#section` 锚点链接<br>2. hover 链接 | 不显示预览按钮 |
| T11 | 静态资源链接无预览 | 1. Markdown 含 `.png/.jpg/.pdf` 链接<br>2. hover 链接 | 不显示预览按钮 |
| T12 | 加载状态显示 | 1. 点击预览按钮<br>2. 观察面板打开过程 | 首先显示"加载中..."指示器，内容加载后消失 |

### 站内链接定义

**"站内链接"判断规则**：
- 相对路径链接（如 `./other.md`、`../folder/file.md`）
- 以 `/` 开头的绝对路径链接（如 `/docs/readme.md`）
- 同 origin 的完整 URL（如 `http://localhost:8080/file.md`）

**排除规则**：
- URL 以 `#` 开头（锚点链接）
- URL 含静态资源扩展名（`.jpg/.png/.gif/.svg/.webp/.avif/.mp4/.webm/.mp3/.pdf`）

## 约束与限制

- 仅在屏幕宽度 ≥ 1024px 时提供功能
- 不处理复杂的跨域 iframe 场景（接受失败提示作为正常行为）
- 不保存预览历史或状态