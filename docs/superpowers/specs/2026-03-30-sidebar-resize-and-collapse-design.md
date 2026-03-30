# Sidebar Resize and Collapse Design

## Goal

为侧边栏添加三个交互功能：

1. **整体收起**：点击按钮将侧边栏收起为图标栏模式（48px）
2. **拖动调整宽度**：通过拖动把手调整侧边栏宽度（200-400px）
3. **Files 折叠**：Files 部分可以独立折叠/展开

这些功能提升阅读体验，让用户根据需要调整侧边栏占用空间。

## Current State

当前侧边栏：

- 固定宽度 280px
- 分上下两区：Files 文件树（flex:1）和 TOC 目录（flex:2）
- 无折叠或调整功能

相关文件：

- `frontend/template.html`：侧边栏 HTML 结构和 CSS
- `frontend/src/sidebar.ts`：文件树渲染逻辑
- `frontend/src/preferences.ts`：偏好设置存储
- `frontend/src/app.ts`：应用入口

## Architecture

### CSS Variables

```css
:root {
  --sidebar-width: 280px;
  --sidebar-collapsed-width: 48px;
}
```

`--sidebar-width` 通过 JS 动态更新，实现拖动调整。

### Storage Keys

| 功能 | 存储键 | 默认值 |
|------|--------|--------|
| 侧边栏收起 | `markview:sidebar-collapsed` | `false` |
| 侧边栏宽度 | `markview:sidebar-width` | `280` |
| Files 折叠 | `markview:files-collapsed` | `false` |

使用 `localStorage` 存储，页面刷新后恢复状态。

## Design Details

### 1. Sidebar Collapse (整体收起)

**收起状态**：

- 侧边栏宽度变为 48px（`--sidebar-collapsed-width`）
- `sidebar-panels` 容器隐藏
- 底部显示 Files/TOC 图标按钮（点击可快速切换展开内容）
- 顶部添加折叠按钮（chevron-left 图标）

**展开状态**：

- 恢复为 `--sidebar-width`（或默认 280px）
- 显示完整侧边栏内容

**实现**：

- 在 `sidebar-header` 右侧添加折叠按钮
- 按钮图标：收起时 `<i class="chevron-left">`，展开时 `<i class="chevron-right">`
- 点击切换 `sidebar-collapsed` 状态
- CSS 通过 `.sidebar-collapsed` 类控制样式

**收起时的底部图标栏**：

```html
<div class="sidebar-icons">
  <button class="sidebar-icon-btn" data-panel="files" title="Files">
    <i class="icon-files"></i>
  </button>
  <button class="sidebar-icon-btn" data-panel="toc" title="TOC">
    <i class="icon-toc"></i>
  </button>
</div>
```

点击图标按钮时：
1. 展开侧边栏
2. 如果点击的是 Files，确保 Files 部分展开
3. 如果点击的是 TOC，确保 Files 部分可以折叠（不影响 TOC）

### 2. Sidebar Resize (拖动调整宽度)

**拖动把手**：

- 位于侧边栏右侧边缘
- 4px 宽度的透明区域，hover 时显示浅色边框
- cursor: `col-resize`

**拖动行为**：

- 拖动时实时更新 `--sidebar-width` CSS 变量
- 范围限制：200px - 400px
- 松开后保存宽度到 `localStorage`

**实现**：

新增模块 `frontend/src/sidebar-resize.ts`：

```typescript
export function initSidebarResize() {
  // 监听 mousedown 开始拖动
  // mousemove 时更新宽度
  // mouseup 时保存并结束拖动
}

export function setSidebarWidth(width: number) {
  // 设置 CSS 变量并保存
}
```

### 3. Files Collapse (Files 折叠)

**折叠按钮**：

- 位于 Files 标题右侧
- 使用 chevron-down/up 图标
- 折叠时：`<i class="chevron-down">`，展开时：`<i class="chevron-up">`

**折叠状态**：

- 隐藏 `sidebar-scroll`（文件树列表区域）
- 标题栏保留，显示折叠按钮

**实现**：

- 在 `sidebar-section-title` 添加折叠按钮
- 点击切换 `files-collapsed` 状态
- CSS 通过 `.files-collapsed` 类隐藏文件列表

**HTML 结构调整**：

```html
<div class="sidebar-section files-section">
  <div class="sidebar-section-title">
    <span>Files</span>
    <button class="collapse-btn" data-target="files">
      <i class="chevron-up"></i>
    </button>
  </div>
  <div class="sidebar-scroll">
    <!-- 文件树内容 -->
  </div>
</div>
```

## Implementation Files

| 文件 | 修改内容 |
|------|----------|
| `frontend/template.html` | CSS 样式、HTML 结构调整 |
| `frontend/src/sidebar.ts` | Files 折叠逻辑、收起状态图标栏 |
| `frontend/src/sidebar-resize.ts` | **新增**：拖动调整宽度逻辑 |
| `frontend/src/preferences.ts` | 存取三个状态键 |
| `frontend/src/app.ts` | 初始化时恢复状态、调用 resize 初始化 |

## Risks

1. **拖动性能**：频繁更新 CSS 变量可能影响渲染性能，需要使用 `requestAnimationFrame` 或节流
2. **状态同步**：侧边栏收起状态下，Files 折叠状态可能被忽略，需要在展开时正确处理
3. **响应式**：小屏幕设备（<768px）可能不需要拖动调整功能，考虑禁用或调整范围
4. **图标资源**：需要确认现有图标库是否包含所需图标（chevron-left/right/up/down，files/toc）