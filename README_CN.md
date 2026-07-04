# SuperRead

订阅 RSS 信息源，AI 自动汇总为每日简报。五分钟浏览当日全部更新摘要。

## Features

- 添加 RSS 源，支持 OPML 批量导入
- 定时抓取（默认每 30 分钟），检测新文章
- 使用大模型将文章浓缩为单句摘要
- 每日简报汇总全部来源的更新
- 多来源报道同一事件时自动去重合并
- 已读/未读、星标收藏、标签分类、稍后阅读
- 平台内通知，可选邮件推送

## Architecture

```text
Browser
  ↓
React Frontend (Oxelia51 unified UI)
  ↓
Go Backend
  ├── RSS Fetcher (periodic cron jobs)
  ├── LLM Summarizer (user-provided API key)
  └── Dedup Engine
  ↓
PostgreSQL / SQLite (feeds, articles, user data)
```

在线版运行于 Oxelia51 平台。Go 后端的 cron 调度器定时抓取 RSS 源，去重引擎合并重复内容，大模型总结由用户提供的 API Key 驱动。桌面版使用 SQLite 存储。

## Requirements

- 在线版：Oxelia51 平台（Go + PostgreSQL + React）
- 桌面版：独立可执行文件，无需运行时依赖
- 大模型 API Key（OpenAI、Anthropic 等）

## Installation

### 桌面版

从 [GitHub Releases](https://github.com/XiaoleC05/SuperRead/releases) 下载 `SuperRead.exe`。

### 在线版

在线版集成于 Oxelia51 平台，参见 [Oxelia51 部署指南](https://github.com/XiaoleC05/Oxelia51)。

## Usage

### 在线

1. 访问 [oxelia51.com](https://oxelia51.com) 注册并登录
2. 进入 SuperRead 工具页
3. 添加 RSS 来源，在设置中填入大模型 API Key
4. 每日查看 AI 生成的简报

### 桌面

1. 双击 `SuperRead.exe` 启动
2. 添加 RSS 来源和 API Key，所有数据存储在本地

## Roadmap

- [ ] RSS 源管理与抓取
- [ ] AI 摘要生成
- [ ] 每日简报展示
- [ ] 智能去重

## Contributing

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/xxx`)
3. 提交变更 (`git commit -m 'Add xxx'`)
4. 推送分支 (`git push origin feature/xxx`)
5. 提交 Pull Request

## License

This project is licensed under the MIT License.
