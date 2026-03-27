# MarkView Examples

这个目录包含几个用于本地预览的 Markdown 示例文件。

## 示例列表

- [基础排版](./basics.md)
- [Mermaid 图表](./mermaid.md)
- [长文档与目录](./guide.md)

## 使用方式

在仓库根目录运行：

```bash
./markview.exe ./example index.md
```

或使用环境变量指定端口：

```bash
SERVER_PORT=3001 ./markview.exe ./example index.md
```
