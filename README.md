# Mutesolo

让 Discord 成为多 Agent 协作中心 — 在 Web 端实时查看 Agent 项目进度

## 🎯 项目目标

Mutesolo 是一个协调层，让多个 AI Agent（如 OpenClaw）能够在 Discord 中协作完成项目任务，并通过 Web 界面可视化整个流程。

**核心价值：**
- **Discord 作为协作入口**：通过 IM 下发任务给 Agent，人工可随时介入
- **Web 端进度可视化**：项目看板、需求追踪、Agent 状态一目了然
- **任务自动分配**：根据 Agent 能力自动匹配任务
- **结果可追溯**：所有产出物（代码、文档）通过 Git 提交，完整记录

## ✨ 主要功能

### 1. Web 控制台
- 项目列表与看板视图
- 需求文档编辑器（支持截图、附件）
- 任务详情与进度追踪
- Agent 在线状态监控
- 技能市场浏览与安装

### 2. 本地文档解析
- 使用 MinerU 解析上传的文档（PDF → Markdown）
- 解析后的内容用于生成 LLM 提示词
- 本地处理，保护隐私

### 3. 本地对象存储
- 使用 MinIO 存储需求截图和附件
- 支持 Tailscale 跨设备访问
- 7 天自动清理过期文件

### 4. Agent 协调
- Agent 注册表（A2A 协议）
- 技能注册表（ClawHub）
- 基于能力的任务匹配
- 任务与 Session 状态管理

## 🚀 快速开始

### 启动 Web 控制台

```bash
go run ./cmd/mutesolo-web
```

访问 http://127.0.0.1:8787

### 启动本地存储（可选）

```bash
cp .env.example .env
docker compose up -d minio minio-init
```

### 配置环境变量

```bash
export MUTESOLO_MINIO_ENDPOINT=http://127.0.0.1:9000
export MUTESOLO_MINIO_PUBLIC_URL=http://127.0.0.1:9000
export MUTESOLO_MINIO_BUCKET=Mutesolo-assets
export MUTESOLO_MINIO_ACCESS_KEY=Mutesolo
export MUTESOLO_MINIO_SECRET_KEY=Mutesolo123
export MUTESOLO_ASSET_FALLBACK_DIR=.openclaw/assets
```

## 🔄 工作流程

```
需求编辑 → 文档解析 → 提示词生成 → Discord 下发 → Agent 执行 → Git 提交 → Web 看板更新
```

1. **需求输入**：在 Web 端编辑需求文档（支持截图、附件）
2. **文档解析**：MinerU 将文档转换为结构化 Markdown
3. **提示词生成**：根据需求生成 Agent 指令
4. **Discord 下发**：通过 Discord 发送给目标 Agent
5. **Agent 执行**：Agent 完成任务并提交到 Git
6. **进度更新**：Web 看板自动同步最新状态

## 🏗️ 架构概览

### 核心组件

- **Web 控制台**（Go + Vue）：项目管理与可视化
- **CLI 工具**（`opclawctl`）：Agent/Skill/Task 管理
- **文档解析器**（MinerU）：PDF → Markdown 转换
- **对象存储**（MinIO）：附件与截图存储
- **协调层**：任务匹配、状态管理、事件追踪

### 数据模型

- **Agent**：ID、地址、状态、技能列表
- **Skill**：ID、能力列表、版本
- **Task**：ID、所需能力、状态
- **Session**：ID、Agent ID、Task ID、状态
- **Event**：类型、实体 ID、载荷、时间戳

## 📝 技术栈

- **后端**：Go
- **前端**：Vue 3 + Vite
- **存储**：MinIO（对象存储）
- **文档解析**：MinerU
- **协议**：A2A（Agent-to-Agent）
- **技能市场**：ClawHub

## 🔒 安全说明

- 所有文档解析在本地完成，不上传到云端
- LLM 提示词生成受控，不会自动执行代码
- Discord 交互采用人工确认模式（human-in-the-loop）
- Git 提交需要手动确认

## 📚 相关文档

- [MinerU 本地解析配置](docs/mineru-native.md)
- [需求编辑器说明](webapps/requirement-editor/README.md)

---

**Mutesolo** — 让 AI Agent 协作像人类团队一样简单
