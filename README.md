# Toy Platform

基于 QuickJS 的轻量级 TypeScript 脚本管理平台，包含 TS 执行器和 SQL 数据浏览器两个独立服务。

## 目录结构

```
toy-platform/
├── cmd/
│   ├── ts-quickjs/          # TS 脚本运行服务
│   │   ├── main.go
│   │   ├── conf/
│   │   │   ├── config.json          # 配置文件（不入库）
│   │   │   └── config.json.sample   # 配置模板
│   │   ├── static/
│   │   │   └── index.html
│   │   └── bin/
│   └── sql-runner/          # SQL 数据浏览服务
│       ├── main.go
│       ├── conf/
│       │   ├── config.json
│       │   └── config.json.sample
│       ├── static/
│       │   └── index.html
│       └── bin/
├── core/                    # 共享模块
│   ├── config/              # 配置读取
│   ├── db/                  # 数据库层（SQLite + Snowflake ID）
│   ├── handler/             # HTTP 请求处理
│   └── runtime/             # TypeScript 执行引擎
├── Makefile
└── go.mod
```

## 快速开始

```bash
# 构建所有服务
make build

# 分部构建
make ts          # 仅构建 TS 服务
make sql         # 仅构建 SQL 服务

# 启动服务
make run-ts      # TS Runner → http://127.0.0.1:9720
make run-sql     # SQL Runner → http://127.0.0.1:9721
```

访问 `http://127.0.0.1:9720` 自动重定向到 TS Runner 界面。

## 配置

每个服务通过配置文件和环境变量设置监听地址和端口：

| 服务 | 配置文件 | 环境变量 | 默认值 |
|---|---|---|---|
| ts-quickjs | `cmd/ts-quickjs/conf/config.json` | `TS_HOST` / `TS_PORT` | `127.0.0.1:9720` |
| sql-runner | `cmd/sql-runner/conf/config.json` | `SQL_HOST` / `SQL_PORT` | `127.0.0.1:9721` |

优先级：环境变量 > 配置文件 > 硬编码默认值

## 服务说明

### TS Runner（ts-quickjs）

TypeScript 脚本在线编辑与执行环境：

- **前端**：侧边栏脚本列表 + 代码编辑器 + 运行输出区
- **后端**：RESTful API，esbuild 编译 TypeScript → QuickJS 执行
- **特性**：雪花 ID、脚本 CRUD、10 秒超时保护、`Ctrl+S` 保存 / `Ctrl+Enter` 运行

API：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/ts-quickjs/ui/` | 前端页面 |
| GET | `/api/scripts` | 脚本列表 |
| POST | `/api/scripts` | 创建脚本 |
| GET | `/api/scripts/{id}` | 查询脚本 |
| PUT | `/api/scripts/{id}` | 更新脚本 |
| DELETE | `/api/scripts/{id}` | 删除脚本 |
| POST | `/api/scripts/{id}/run` | 执行脚本 |

### SQL Runner（sql-runner）

SQLite 数据库交互式查询工具：

- **前端**：左侧表列表 + SQL 输入区 + 结果展示（Table / JSON 切换）
- **后端**：只读连接，仅允许 SELECT / EXPLAIN / WITH
- **特性**：选中执行（`Ctrl+Shift+Enter`）、EXPLAIN 支持、防 SQL 注入

API：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/sql-runner/ui/` | 前端页面 |
| GET | `/api/sql/tables` | 表列表 |
| POST | `/api/sql/run` | 执行 SQL |

## 技术栈

- **语言**：Go 1.25
- **JS 引擎**：[quickjs-go](https://github.com/quickjs-go/quickjs-go)
- **TS 编译**：[esbuild](https://github.com/evanw/esbuild)
- **数据库**：SQLite（modernc.org/sqlite，纯 Go 实现）
- **ID 生成**：Snowflake 算法

## 开发

```bash
make fmt      # 格式化代码
make vet      # 静态分析
make lint     # golangci-lint 检查
make test     # 运行测试
make clean    # 清理构建产物
```
