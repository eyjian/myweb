# myweb 架构设计

## 概述

**myweb** 是一个基于浏览器的 MySQL 客户端，提供与 [mysh](https://github.com/eyjian/mysh) 一致的核心体验——SQL 编辑（语法高亮、自动补全、结果展示），但通过 Web 界面代替终端。

核心原则：
- **单二进制** — Go 后端 + 嵌入式 SPA 前端，与 mysh 哲学一致
- **本地优先** — 默认监听 `127.0.0.1`，无需认证
- **代码复用** — 共享 mysh 的 `connection`、`metadata`、`executor`、`output`、`config` 包

## 架构

```
┌─────────────────────────────────────────────────────────┐
│                    浏览器 (SPA)                         │
│  ┌──────────┐ ┌──────────┐ ┌────────┐ ┌─────────────┐ │
│  │ CodeMirror│ │ 结果     │ │侧边栏  │ │ 历史面板    │ │
│  │ SQL 编辑器│ │ 表格     │ │数据库浏览│ │             │ │
│  └─────┬─────┘ └────┬─────┘ └───┬────┘ └──────┬──────┘ │
│        └────────┬────┘───────────┘─────────────┘        │
│            HTTP REST + WebSocket                        │
├─────────────────────────────────────────────────────────┤
│                  Go HTTP 服务器                         │
│  ┌──────────┐ ┌───────────┐ ┌───────────────────────┐  │
│  │ REST API │ │ WebSocket │ │ 静态文件 (embed.FS)   │  │
│  │ 处理器   │ │ 处理器    │ │ ui/dist/              │  │
│  └────┬─────┘ └─────┬─────┘ └───────────────────────┘  │
├───────┴──────────────┴──────────────────────────────────┤
│              共享包 (来自 mysh)                          │
│  ┌──────────┐ ┌──────────┐ ┌────────┐ ┌────────────┐  │
│  │connection│ │ metadata │ │executor│ │  output     │  │
│  │  连接池  │ │ 元数据缓存│ │ 执行器 │ │ 格式化器    │  │
│  └──────────┘ └──────────┘ └────────┘ └────────────┘  │
│  ┌──────────┐                                           │
│  │  config  │                                           │
│  └──────────┘                                           │
├─────────────────────────────────────────────────────────┤
│                    MySQL 服务器                          │
└─────────────────────────────────────────────────────────┘
```

## 项目结构

```
myweb/
├── cmd/
│   └── myweb/            # 入口
│       └── main.go
├── server/
│   ├── server.go         # HTTP 服务器、embed.FS、路由
│   ├── api.go            # REST API 处理器
│   ├── ws.go             # WebSocket 处理器
│   └── middleware.go     # CORS、日志、恢复
├── ui/
│   ├── src/              # Vue 3 前端源码
│   │   ├── App.vue
│   │   ├── components/
│   │   │   ├── SqlEditor.vue      # CodeMirror 6 封装
│   │   │   ├── ResultTable.vue    # 虚拟滚动结果表格
│   │   │   ├── Sidebar.vue        # 数据库/表/列浏览器
│   │   │   ├── ConnectDialog.vue  # 连接对话框
│   │   │   ├── HistoryPanel.vue   # 查询历史面板
│   │   │   └── StatusBar.vue      # 连接信息 + 格式切换
│   │   ├── composables/
│   │   │   ├── useWebSocket.ts    # WebSocket 连接管理
│   │   │   └── useApi.ts          # REST API 辅助
│   │   └── main.ts
│   ├── dist/             # 构建产物 (go:embed 目标)
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── docs/
│   └── design/
│       └── ARCHITECTURE.md  # 本文件
├── Makefile
├── install.sh
├── go.mod
└── go.sum
```

## Go 后端

### 入口

`cmd/myweb/main.go`：
- 解析 CLI 参数：`-addr`（默认 `127.0.0.1:8080`）、`-open`（自动打开浏览器）、`-dsn`（初始连接）
- 加载配置 `~/.myweb.yaml`（复用 mysh `config` 包）
- 创建连接池、元数据缓存、执行器（来自 mysh 包）
- 启动 HTTP 服务器，嵌入 SPA
- 可选自动打开浏览器

### REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/status` | 连接状态（host、db、user、服务器版本、运行时长） |
| `POST` | `/api/connect` | 连接数据库 `{"dsn":"user:pass@tcp(host:port)/db"}` |
| `POST` | `/api/reconnect` | 重新连接当前服务器 |
| `GET` | `/api/databases` | 列出所有数据库 |
| `GET` | `/api/tables?db=xxx` | 列出指定数据库的表 |
| `GET` | `/api/columns?table=xxx&db=xxx` | 列出指定表的列信息 |
| `POST` | `/api/query` | 同步执行短查询 `{"sql":"SELECT ..."}` |
| `POST` | `/api/export` | 导出最近一次查询结果 `{"format":"csv","path":"~/out.csv"}` |
| `GET` | `/api/history` | 获取查询历史（分页） |
| `GET` | `/api/aliases` | 获取配置的别名 |

### WebSocket 协议

端点：`ws://localhost:8080/ws`

用于长查询、流式结果和 watch 模式。

**客户端 → 服务端：**

```json
{"type": "query", "sql": "SELECT * FROM users"}
{"type": "cancel"}
{"type": "watch", "interval": 5, "sql": "SELECT COUNT(*) FROM processes"}
{"type": "watch_stop"}
```

**服务端 → 客户端：**

```json
{"type": "result_start", "columns": ["id","name","email"], "format": "table"}
{"type": "result_row", "row": [1, "Alice", "alice@example.com"]}
{"type": "result_row", "row": [2, "Bob", "bob@example.com"]}
{"type": "result_end", "rows": 2, "time": "0.03s", "affected": 0}
{"type": "error", "message": "Table 'foo' doesn't exist"}
{"type": "watch_tick", "seq": 3, "columns": [...], "rows": [...], "time": "0.01s"}
{"type": "connected", "host": "localhost", "db": "mydb", "user": "root"}
{"type": "disconnected"}
```

流式 `result_row` 消息允许大结果集逐步渲染，无需等待查询完全结束。

### 静态文件服务

前端构建产物通过 `go:embed` 嵌入：

```go
//go:embed ui/dist/*
var staticFiles embed.FS
```

挂载到 `/`，SPA fallback（未知路径返回 `index.html`，支持客户端路由）。

## 前端 (React SPA)

### 技术栈

| 库 | 用途 |
|----|------|
| Vue 3 | UI 框架（Composition API + `<script setup>`） |
| TypeScript | 类型安全 |
| Vite | 构建工具（开发时快速 HMR，原生 Vue 支持） |
| CodeMirror 6 | SQL 编辑器，使用 `vue-codemirror` + `@codemirror/lang-sql` |
| @tanstack/vue-table | 虚拟滚动结果表格 |
| Tailwind CSS | 样式 |

### 页面布局

```
┌──────────────────────────────────────────────────────┐
│  🔵 myweb    mydb@localhost:3306       [连接]        │  ← 顶栏
├──────────┬───────────────────────────────────────────┤
│          │  ┌─────────────────────────────────────┐  │
│ 表列表   │  │  SQL 编辑器 (CodeMirror 6)          │  │
│──────────│  │  SELECT * FROM users WHERE ...       │  │
│ ▸ users  │  │                                     │  │
│ ▸ orders │  └─────────────────────────────────────┘  │
│ ▸ products│  [▶ 执行]  [⏹ 取消]  [📥 导出]          │  ← 操作栏
│          │  ┌─────────────────────────────────────┐  │
│ 列信息   │  │  结果表格                            │  │
│──────────│  │  id | name  | email                 │  │
│ id  INT  │  │  1  | Alice | alice@...             │  │
│ name VARC│  │  2  | Bob   | bob@...               │  │
│ email VAR│  └─────────────────────────────────────┘  │
│          │  2 行, 耗时 0.03s  [表格|JSON|Markdown]   │  ← 状态栏
└──────────┴───────────────────────────────────────────┘
```

### 组件

#### SqlEditor (CodeMirror 6)
- SQL 语法高亮（`@codemirror/lang-sql`）
- 自动补全：`Ctrl+Space` 触发，从 `/api/tables`、`/api/columns` 获取建议
- 多行编辑、括号匹配、自动闭合
- `Ctrl+Enter` 或 `Cmd+Enter` 执行
- 支持 `\G`、`\j`、`\m` 后缀（与 mysh 一致）

#### ResultTable
- 虚拟滚动：大结果集仅渲染可见行
- 列头点击排序（客户端排序）
- 列宽自动适配，支持手动拖拽
- NULL 值显示为暗色斜体 `NULL`（与 mysh 一致）
- 格式切换：表格 / JSON / Markdown 标签页

#### Sidebar
- 手风琴折叠：数据库 → 表 → 列
- 单击表名/列名 → 插入到编辑器光标处
- 双击表名 → 自动填充 `SELECT * FROM table LIMIT 100`
- 刷新按钮重新加载元数据

#### ConnectDialog
- 模态对话框：Host、Port、User、Password、Database 字段
- 或直接粘贴 DSN 字符串
- 保存为会话（存储在配置文件）

#### HistoryPanel
- 滑出面板，显示查询历史
- 关键词搜索/过滤
- 点击重填编辑器

#### StatusBar
- 连接指示器（绿点 = 已连接，红点 = 已断开）
- 当前数据库和主机
- 最近查询行数和耗时
- 输出格式切换按钮

### WebSocket 连接管理

```
useWebSocket composable:
  - 组件挂载时自动连接
  - 断线指数退避自动重连
  - 离线期间消息队列缓冲
  - 流式 result_row → 增量更新表格
```

## 安全

- **默认绑定**：`127.0.0.1:8080` — 仅本地访问
- **默认无认证** — 本地工具，与 mysh 一致
- **可选 Token**：`--token` 参数设置 Bearer token 用于 API/WebSocket 认证
- **无 CORS 外部头** — 仅限 localhost
- SQL 注入：不适用 — 用户直接编写 SQL，与任何数据库客户端相同

## 共享 mysh 代码

| 包 | myweb 中的用途 |
|----|---------------|
| `connection` | MySQL 连接池、重连、DB/表/列查询 |
| `metadata` | Schema 缓存（延迟刷新、脏标记） |
| `executor` | SQL 执行、自动重连重试、取消支持 |
| `output` | 结果格式化（表格/垂直/JSON/Markdown） |
| `config` | 配置文件加载/保存、别名、会话 |
| `history` | 查询历史持久化 |

**不共享的包**：
- `tui` — 终端专属 Bubble Tea 模型
- `editor` — 终端行编辑器
- `highlight` — ANSI 终端高亮（前端用 CodeMirror）
- `completer` — 终端补全菜单（前端用 CodeMirror 补全）

## 开发工作流

### 本地开发

```bash
# 终端 1：启动前端开发服务器（HMR）
cd ui && npm run dev

# 终端 2：启动 Go 后端，代理到前端
go run ./cmd/myweb --dev
```

`--dev` 模式下，Go 服务器将 `/` 请求代理到 Vite 开发服务器（默认 `http://localhost:5173`），而非使用嵌入文件。

### 生产构建

```bash
# 构建前端
cd ui && npm run build   # → ui/dist/

# 构建 Go 二进制（嵌入 ui/dist/）
go build -o myweb ./cmd/myweb

# 或通过 Makefile
make build
```

### 交叉编译

```bash
make cross-compile
# 产出：myweb-linux-amd64, myweb-darwin-arm64 等
```

## CLI 参数

```
用法：myweb [选项] [dsn]

选项：
  -addr string    监听地址（默认 "127.0.0.1:8080"）
  -open           自动打开浏览器（默认 true）
  -dev            开发模式（代理到 Vite）
  -token string   Bearer token 用于 API 认证
  -config string  配置文件路径（默认 "~/.myweb.yaml"）
  -version        显示版本号

参数：
  dsn             启动时连接的 MySQL DSN（可选）
                  user:pass@tcp(host:port)/dbname
```

## 配置

配置文件：`~/.myweb.yaml`（与 mysh 的 `~/.mysh.yaml` 格式兼容）

```yaml
# 连接默认值
host: "127.0.0.1"
port: 3306
user: "root"
password: ""

# 服务器设置
addr: "127.0.0.1:8080"
open_browser: true

# 显示
theme: "dark"              # dark / light
page_size: 100             # 默认结果限制
auto_vertical: false       # 宽结果自动切换垂直格式

# SQL 别名（与 mysh 共享）
aliases:
  top10: "SELECT * FROM users ORDER BY score DESC LIMIT 10"

# 保存的会话（与 mysh 共享）
sessions:
  prod:
    host: "db.prod.example.com"
    port: 3306
    user: "admin"
    database: "myapp"
```

## 分阶段实现计划

### Phase 1：骨架 + 基础查询
- Go HTTP 服务器 + embed.FS
- WebSocket 处理器（查询/取消）
- React 脚手架 + CodeMirror SQL 编辑器
- 结果表格（基础版，无虚拟滚动）
- 连接对话框

### Phase 2：Schema 浏览 + 补全
- 侧边栏数据库/表/列树
- CodeMirror 自动补全（来自元数据 API）
- 双击表名 → SELECT 查询
- 历史面板

### Phase 3：高级功能
- 虚拟滚动结果表格
- 导出为 CSV/JSON/Markdown
- Watch 模式（定期重执行）
- 格式切换（表格/JSON/Markdown）
- 会话管理

### Phase 4：打磨
- 暗色/亮色主题
- 快捷键面板
- install.sh + 交叉编译
- 配置文件与 mysh 共享
