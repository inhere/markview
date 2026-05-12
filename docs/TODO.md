# TODO

<!--
简单的直接使用一行 checklist 说明即可。
需要附带较长说明的，使用标题+说明方式新建。使用emoji 表情状态图标(wait: ⏳|ing: 🔄|done: ✅)
-->

- [x] 新增 --private 选项，指定是否监听为本地server，不公开
- [ ] 支持多项目管理和快速启动
- [ ] 随机端口时，按项目路径自动保存获取到的端口号
- [x] bug: 右侧打开的预览面板，mermaid, highlight 不会渲染显示
- [x] 点击 file-tree 的目录时，如果没有 entry file, 则默认列出目录下的所有文件/目录

## 随机端口时，自动保存获取到的端口号 ⏳

随机端口时，自动保存获取到的端口号，避免每次启动都需要手动指定端口号 或者 每次随机端口号不一样，导致之前的无法访问。

`-p -1` 指定随机端口时，自动保存获取到的端口号等到 `markview-projects.json` 文件

- 下次启动时，自动读取该文件，优先使用保存的端口号启动服务，如果端口号被占用，自动选择下一个可用端口号并更新文件

json 格式参考：

```json
{
  "project path": {
    "port": 8080,
    "name": "project name(default is directory name)",
    "added": "2025-08-03T15:00:00"
  }
}
```

## 支持多项目管理和快速启动 ⏳

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

