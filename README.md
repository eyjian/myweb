# myweb

基于浏览器的 MySQL 客户端，复用 [mysh](https://github.com/eyjian/mysh) 的连接池、SQL 执行和元数据缓存能力，提供 Web 端的 SQL 查询体验。

单二进制部署，前端资源通过 `go:embed` 嵌入，零外部依赖。

## 功能特性

- **SQL 编辑器** — 基于 CodeMirror 的 SQL 编辑器，支持语法高亮和 `Ctrl+Enter` 执行
- **实时查询** — REST API 同步查询 + WebSocket 流式查询
- **数据库浏览** — 查看数据库列表、表列表、列信息
- **连接管理** — 支持配置文件预配置 + 浏览器内动态连接
- **结果展示** — 表格化展示查询结果，显示影响行数和执行时间
- **单二进制** — 前端资源嵌入 Go 二进制，一个文件即可部署

## 安装

### 从源码构建

```bash
git clone https://github.com/eyjian/myweb.git
cd myweb
make build
```

> 构建需要 Go 1.24+ 和 Node.js 18+（用于构建前端）。

## 使用方法

### 快速开始

```bash
# 构建（首次或前端有变更时）
make build

# 直接启动（使用 ~/.mysh.yaml 中的连接配置）
./myweb

# 指定监听地址
./myweb --addr 0.0.0.0:9090

# 查看版本
./myweb --version
```

启动后会自动打开浏览器访问 `http://127.0.0.1:8080`。

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--addr` | `127.0.0.1:8080` | 监听地址 |
| `--dev` | `false` | 开发模式（代理到 Vite HMR） |
| `--open` | `true` | 自动打开浏览器 |
| `--version` | — | 打印版本号 |

### 开发模式

```bash
# 终端 1：启动 Vite 开发服务器
cd ui && npm run dev

# 终端 2：启动 Go 后端（代理前端请求到 Vite）
make dev
```

开发模式下，前端修改会实时热更新，无需重新构建。

## 配置数据库连接

myweb 复用 mysh 的配置系统，通过 `~/.mysh.yaml` 配置数据库连接。

### 方式一：配置文件（推荐）

创建或编辑 `~/.mysh.yaml`：

```yaml
connection:
  host: "127.0.0.1"      # MySQL 主机
  port: 3306              # MySQL 端口
  user: "root"            # 用户名
  password: ""            # 密码
  database: ""            # 默认数据库（可选）
  charset: "utf8mb4"      # 字符集（可选）
```

启动 myweb 时会自动使用该配置连接数据库。如果连接失败，服务仍会启动，可在浏览器中通过连接对话框手动连接。

### 方式二：浏览器内连接

如果启动时未配置数据库或连接失败，可以在浏览器界面中：

1. 点击状态栏的「未连接」按钮
2. 输入 DSN 连接字符串，格式：`user:password@tcp(host:port)/database`
3. 点击「连接」

示例：

```
root:mypassword@tcp(127.0.0.1:3306)/mydb
admin:secret@tcp(db.example.com:3306)/production
root@tcp(localhost:3306)/test
```

### 多数据库配置

使用 mysh 的会话管理功能，在 `~/.mysh.yaml` 中保存多个连接：

```yaml
# 默认连接
connection:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: ""
  database: "mydb"

# 已保存的会话（可在 mysh CLI 中用 \session 切换）
sessions:
  production:
    host: "db.prod.example.com"
    port: 3306
    user: "admin"
    password: "prod_password"
    database: "myapp"
  staging:
    host: "db.staging.example.com"
    port: 3306
    user: "dev"
    password: "dev_password"
    database: "myapp_dev"
```

> **注意**：myweb 当前仅使用 `connection` 段作为默认连接。如需切换数据库，请在浏览器中使用连接对话框。

### DSN 格式说明

DSN（Data Source Name）格式遵循 Go 的 `go-sql-driver/mysql` 规范：

```
[用户名]:[密码]@tcp([主机]:[端口])/[数据库名]
```

| 部分 | 必填 | 说明 |
|------|------|------|
| 用户名 | 是 | MySQL 用户名 |
| 密码 | 否 | 可省略（无密码时格式为 `root@tcp(...)`） |
| 主机 | 是 | MySQL 服务器地址 |
| 端口 | 是 | MySQL 端口号 |
| 数据库名 | 否 | 默认数据库，可留空（格式为 `root:pwd@tcp(host:port)/`） |

## API 接口

myweb 提供以下 REST API：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/status` | 获取连接状态 |
| POST | `/api/connect` | 连接数据库（JSON: `{"dsn": "..."}`) |
| GET | `/api/databases` | 列出所有数据库 |
| GET | `/api/tables?db=xxx` | 列出指定数据库的表 |
| GET | `/api/columns?db=xxx&table=yyy` | 列出表的列信息 |
| POST | `/api/query` | 执行 SQL 查询（JSON: `{"sql": "..."}`) |
| GET | `/ws` | WebSocket 连接（流式查询） |

WebSocket 协议：

```json
// 发送
{"action": "query", "sql": "SELECT * FROM users"}

// 接收（逐行流式返回）
{"type": "header", "columns": ["id", "name", "email"]}
{"type": "row", "data": [1, "Alice", "alice@example.com"]}
{"type": "footer", "rows_affected": 0, "time_ms": 12}
{"type": "error", "message": "..."}
```

## 项目结构

```
myweb/
├── cmd/myweb/main.go     # 入口（CLI flags + 初始化）
├── server/
│   ├── server.go         # HTTP 服务器 + embed.FS + 路由
│   ├── middleware.go      # 日志 + 恢复中间件
│   ├── api.go            # REST API 处理器
│   └── ws.go             # WebSocket 处理器 + Hub
├── ui/                   # Vue 3 前端
│   ├── src/
│   │   ├── composables/  # useApi, useWebSocket
│   │   └── components/   # SqlEditor, ResultTable, StatusBar, ConnectDialog
│   └── vite.config.ts
├── uiembed.go            # go:embed 前端资源
├── Makefile
└── docs/
```

## 开发

### 前置要求

- Go 1.24+
- Node.js 18+
- mysh（通过 go.work 链接）

### 构建与运行

```bash
make build       # 构建前端 + Go 二进制
make dev         # 开发模式（需要单独启动 Vite）
make test        # 运行测试
make clean       # 清理构建产物
```

### 贡献

1. Fork 本仓库
2. 创建功能分支（`git checkout -b feature/my-feature`）
3. 提交更改（`git commit -m 'Add my feature'`）
4. 确保构建通过（`make build`）
5. 推送到分支（`git push origin feature/my-feature`）
6. 发起 Pull Request

## 许可证

[Apache License 2.0](LICENSE)
