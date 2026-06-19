# Tiny Lowcode Platform v2.0.0 "Gateway"

基于 QuickJS 的轻量级 TypeScript 脚本管理平台，支持多租户、认证鉴权、功能开关。Vue 3 前端 + Go 后端分离架构。

## 服务架构

```
前端 (Vue 3 + Vite :5173)
  │
  ├── /api/auth    → auth-server   :9722   登录/登出/签发JWT
  ├── /api/bff     → bff-server    :9724   门户入口聚合
  ├── /api/admin   → admin-server  :9723   租户/用户管理 (admin only)
  ├── /api/scripts → ts-quickjs    :9720   TS脚本CRUD/执行/调试 (租户隔离)
  └── /api/sql     → sql-runner    :9721   SQL只读查询 (admin only)
```

## 目录结构

```
tiny-lowcode-platform/
├── frontend/                      # Vue 3 前端 (Vite + Pinia + Vue Router)
│   └── src/
│       ├── views/                 # Login, Portal, Editor, Sql, Admin
│       ├── components/            # AppHeader
│       ├── stores/auth.js         # Pinia 认证状态
│       └── router/index.js        # 路由 + 导航守卫
├── cmd/                           # Go 后端服务 (纯 API)
│   ├── auth-server/ main.go
│   ├── bff-server/  main.go
│   ├── admin-server/main.go
│   ├── ts-quickjs/  main.go
│   └── sql-runner/  main.go
├── core/                          # 共享库
│   ├── idgen/       前缀ID生成器
│   ├── auth/        JWT + bcrypt + 中间件
│   ├── db/          数据层接口 + SQLite
│   ├── handler/      HTTP处理器 (authpkg/adminpkg/sqlpkg)
│   ├── sqlstore/     SQL目标库抽象
│   ├── validator/    JSON Schema校验
│   ├── logger/       slog日志 + CORS中间件
│   ├── config/       配置加载
│   └── runtime/      JS运行时 (QuickJS/Goja)
├── betamap.def.json              # 功能模块定义
├── tests/integration/            # 集成测试
├── docs/                         # 设计文档
├── patches/                      # 第三方补丁
├── scripts/                      # 工具脚本
├── docker-compose.yml            # 5服务编排
├── docker-compose.dev.yml        # 开发依赖 (MySQL + Redis)
├── quick_dev_start.sh            # 一键启动脚本
└── Makefile
```

## 快速开始

```bash
# 一键启动 (自动构建 5 个后端 + Vue 前端 + 配置)
./quick_dev_start.sh start

# 浏览器打开
open http://127.0.0.1:5173
```

内置账号: `admin` / `admin` / `admin123`

### 手动启动

```bash
# 1. 后端 (复制配置 → 构建 → 启动)
for svc in auth-server admin-server bff-server ts-quickjs sql-runner; do
    cp cmd/$svc/conf/config.json.sample cmd/$svc/conf/config.json
done
make build
make run-auth & make run-admin & make run-bff & make run-ts & make run-sql &

# 2. 前端
cd frontend && npm install && npm run dev
```

## 开发与调试

### 前后端分离工作流

```bash
# 后端：修改 Go 代码 → 重建单个服务
make ts && make run-ts

# 前端：Vite HMR 热更新，保存即刷新
cd frontend && npm run dev

# 测试
go test ./core/auth/... -v              # 单包
go test ./tests/integration/... -v      # 集成
make test                                # 全量
```

### 前端调试

```bash
# Vite dev server 自带 proxy → Go 后端，无需 CORS 配置
# 前端 :5173 → /api/* → 后端对应端口

# 生产构建
cd frontend && npm run build    # → dist/
```

### 后端调试

**数据库直接查看：**
```bash
sqlite3 scripts.db ".tables"
sqlite3 scripts.db "SELECT id,username,role FROM users"
```

**cURL 测试 API：**
```bash
curl -s -X POST http://127.0.0.1:9722/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"tenant":"admin","username":"admin","password":"admin123"}'
```

**QuickJS 引擎切换：**
```bash
make patch-quickjs && make ts          # QuickJS (CGO)
go build -tags noquickjs ./cmd/ts-quickjs/  # Goja (纯Go)
```

### 常用测试命令

```bash
go test ./core/auth/... -run TestSignAndVerify -v  # 单个函数
go test -race ./core/...                            # 竞态检测
make cover && open coverage.html                    # 覆盖率
```

### 配置说明

| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|--------|------|
| `host` | `*_HOST` | `127.0.0.1` | 监听地址 |
| `port` | `*_PORT` | 见服务端口 | 监听端口 |
| `jwt_secret` | — | 自动生成 | 32 字节随机密钥 |
| `token_expire_hours` | — | `1` | Token 过期时间 |
| `redis_addr` | — | `""` | Redis 地址 (空=禁用黑名单) |
| `engine` | — | `quickjs` | JS 引擎 (quickjs / goja) |

优先级: 环境变量 > config.json > 默认值

### 跨平台构建

```bash
make build                     # 当前平台
make build-linux-amd64         # Linux x86_64
make build-linux-arm64         # Linux ARM64
```

## 技术栈

| 层 | 技术 |
|----|------|
| 前端 | Vue 3 + Vite + Pinia + Vue Router + CodeMirror 6 |
| 后端 | Go 1.25+ + net/http (Go 1.22 routing) |
| JS 引擎 | quickjs-go / goja |
| TS 编译 | esbuild |
| 数据库 | SQLite (modernc.org/sqlite) / MySQL (预留) |
| 认证 | JWT HMAC-SHA256 + bcrypt + Redis 黑名单(可选) |
| ID 生成 | 24位 hex 前缀ID (001a/001b/001c/001d) |
