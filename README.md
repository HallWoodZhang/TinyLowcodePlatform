# Tiny Lowcode Platform v2.0.0 "Gateway"

基于 QuickJS 的轻量级 TypeScript 脚本管理平台，支持多租户、认证鉴权、功能开关。

## 服务架构

| 服务 | 端口 | 说明 |
|------|------|------|
| **auth-server** | 9722 | 认证服务：登录/登出/签发JWT |
| **bff-server** | 9724 | 门户首页：功能入口聚合，按 betamap 控制可见性 |
| **admin-server** | 9723 | 管理服务：租户/用户 CRUD + betamap 管理 (admin only) |
| **ts-quickjs** | 9720 | TS 脚本编辑器 + CRUD + 执行/调试 (租户隔离) |
| **sql-runner** | 9721 | SQL 数据浏览器：只读查询目标数据库 (admin only) |

## 目录结构

```
tiny-lowcode-platform/
├── betamap.def.json              # 全平台功能模块定义
├── cmd/
│   ├── auth-server/              # 认证服务
│   ├── bff-server/               # 门户服务
│   ├── admin-server/             # 管理服务
│   ├── ts-quickjs/               # TS脚本运行服务
│   └── sql-runner/               # SQL数据浏览服务
├── core/
│   ├── idgen/                    # 前缀ID生成器 (24位hex)
│   ├── auth/                     # JWT签发/验证 + bcrypt + 中间件
│   ├── db/                       # 数据层接口 + SQLite/MySQL实现
│   ├── handler/                  # HTTP处理器
│   ├── sqlstore/                 # SQL查询目标库抽象 (多驱动)
│   ├── validator/                # JSON Schema校验
│   ├── logger/                   # slog日志 + 中间件
│   ├── config/                   # 配置加载
│   └── runtime/                  # TypeScript执行引擎
├── docs/                         # 设计文档
├── docker-compose.yml            # 5服务编排
├── docker-compose.dev.yml        # 本地开发依赖 (MySQL + Redis)
├── Makefile
└── go.mod
```

## 快速开始

```bash
# 一键启动 (无需 Docker)
./quick_dev_start.sh start

# 查看服务状态
./quick_dev_start.sh status

# 停止全部服务
./quick_dev_start.sh stop
```

访问 `http://127.0.0.1:9724` → 自动跳转登录 → 用内置账号登录。

### 手动启动

```bash
# 复制配置
for svc in auth-server admin-server bff-server ts-quickjs sql-runner; do
    cp cmd/$svc/conf/config.json.sample cmd/$svc/conf/config.json
done

# 构建 + 启动
make auth && make run-auth     # 认证服务 :9722
make bff && make run-bff      # 门户首页 :9724
make admin && make run-admin  # 管理服务 :9723
make ts && make run-ts        # TS脚本   :9720
make sql && make run-sql      # SQL查询  :9721
```

## 内置账号

| 租户 | 用户 | 密码 | 角色 |
|------|------|------|------|
| `admin` | `admin` | `admin123` | 平台管理员 (可查看所有数据) |

创建新租户时自动生成主账号 (username/password = 租户名, role = tenant_admin)。

## 开发与调试

### 项目结构约定

```
core/           # 共享库 (接口 + 实现 + 测试)
  idgen/        #   ID 生成器
  auth/         #   JWT / bcrypt / 中间件 (Auth, Admin, Betamap)
  db/           #   数据层接口 + SQLite 实现
  handler/      #   HTTP 处理器 (按领域分子包)
    authpkg/    #     认证处理器
    adminpkg/   #     管理处理器
    sqlpkg/     #     SQL 查询处理器
  sqlstore/     #   目标数据库抽象 (SQLite/MySQL/PG)
  validator/    #   JSON Schema 校验中间件
  logger/       #   slog 日志 + Trace/Access/Recovery 中间件
  config/       #   配置加载 (json + 环境变量)
  runtime/      #   JS 运行时 (QuickJS / Goja)
cmd/            # 服务入口 (一个目录一个 main.go)
tests/          # 集成测试
docs/           # 设计文档
patches/        # 第三方依赖补丁
scripts/        # 工具脚本
```

### 本地开发工作流

```bash
# 1. 启动全部服务 (一键)
./quick_dev_start.sh start

# 2. 修改代码 → 重新构建单个服务
make ts && make run-ts        # 仅重建 ts-quickjs
make auth && make run-auth    # 仅重建 auth-server

# 3. 运行测试 (修改哪个包就跑哪个)
go test ./core/auth/... -v           # 单元测试
go test ./tests/integration/... -v    # 集成测试
make test                             # 全部测试

# 4. 检查日志 (服务输出到 /tmp)
tail -f /tmp/auth-server.log
tail -f /tmp/ts-quickjs.log

# 5. 清理 + 重建
./quick_dev_start.sh stop
make clean && make build
```

### 调试技巧

**数据库直接查看：**
```bash
sqlite3 scripts.db ".tables"                           # 查看表
sqlite3 scripts.db "SELECT * FROM tenants"              # 租户
sqlite3 scripts.db "SELECT id,username,role FROM users" # 用户
sqlite3 scripts.db "SELECT id,name,tenant_id FROM scripts" # 脚本
```

**cURL 测试 API：**
```bash
# 登录获取 token
curl -s -X POST http://127.0.0.1:9722/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"tenant":"admin","username":"admin","password":"admin123"}'

# 使用 token 访问
TOKEN="<从登录响应中获取>"
curl -s http://127.0.0.1:9724/api/bff/entries \
  -H "Authorization: Bearer $TOKEN"

# 创建租户 (admin only)
curl -s -X POST http://127.0.0.1:9723/api/admin/tenants \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"dev","label":"Dev Team"}'
```

**QuickJS 引擎切换：**
```bash
# 默认使用 QuickJS (需先打补丁)
make patch-quickjs && make ts

# 或使用纯 Go Goja 引擎 (无需 CGO)
go build -tags noquickjs -o cmd/ts-quickjs/bin/ts-quickjs ./cmd/ts-quickjs/
```

**Betamap 功能开关调试：**
```bash
# 查看某租户的 betamap
curl -s http://127.0.0.1:9723/api/admin/tenants/{租户ID}/betamap \
  -H "Authorization: Bearer $TOKEN"

# 启用 debug 功能 (关闭后 API 返回 403)
curl -s -X PUT http://127.0.0.1:9723/api/admin/tenants/{租户ID}/betamap \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"script_debug":true}'
```

### 常用测试命令

```bash
# 单包测试
go test ./core/auth/... -v -count=1

# 单个测试函数
go test ./core/auth/... -run TestSignAndVerify -v

# 集成测试 (需要编译全部服务)
go test ./tests/integration/... -v

# 覆盖率
make cover
open coverage.html

# 竞态检测
go test -race ./core/...

# 基准测试
go test ./core/idgen/... -bench=. -benchmem
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
make build-linux-amd64         # Linux x86_64 (含 QuickJS)
make build-linux-arm64         # Linux ARM64
go build -tags noquickjs ...   # 纯 Go 引擎 (零 CGO, 跨平台最简单)
```

## 技术栈

- **语言**：Go 1.25+
- **JS 引擎**：quickjs-go / goja
- **TS 编译**：esbuild
- **数据库**：SQLite (modernc.org/sqlite) / MySQL (预留)
- **认证**：JWT HMAC-SHA256 + bcrypt
- **ID 生成**：24位 hex 前缀ID (001a/001b/001c/001d)
