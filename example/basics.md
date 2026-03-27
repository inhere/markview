# 基础排版示例

这是一个用于验证常见 Markdown 渲染效果的示例文档。

## 文本样式

你可以看到 **粗体**、*斜体*、`行内代码`，以及一个 [外部链接](https://example.com)。

> 这是引用块，用来检查 blockquote 的样式。

## 列表

### 无序列表

- 苹果
- 香蕉
- 橙子

### 有序列表

1. 准备文档
2. 启动 MarkView
3. 观察实时刷新

## 代码块

```go
package main

import "fmt"

func main() {
    fmt.Println("hello from markview")
}
```

```json
{
  "name": "markview",
  "liveReload": true,
  "port": 3000
}
```

## 表格

| 功能 | 说明 |
| --- | --- |
| GFM | 支持 GitHub 风格 Markdown |
| Mermaid | 支持图表渲染 |
| SSE | 支持实时刷新 |
