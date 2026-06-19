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
# 本地开发 (SQLite，零依赖)
cp cmd/auth-server/conf/config.json.sample cmd/auth-server/conf/config.json
# 为所有服务复制 config.json.sample → config.json，确保 jwt_secret 一致

# 构建 + 启动
make auth && make run-auth     # 认证服务 :9722
make bff && make run-bff      # 门户首页 :9724
make admin && make run-admin  # 管理服务 :9723
make ts && make run-ts        # TS脚本   :9720
make sql && make run-sql      # SQL查询  :9721

# 或一键构建
make build

# 访问 http://127.0.0.1:9724 进入门户
```

## 内置账号

| 租户 | 用户 | 密码 | 角色 |
|------|------|------|------|
| `admin` | `admin` | `admin123` | 平台管理员 (可查看所有数据) |

## 开发

```bash
make test      # 运行全部自动化测试
make cover     # 生成覆盖率报告
make fmt       # 格式化代码
make vet       # 静态分析
make lint      # golangci-lint

# 启动开发依赖 (MySQL + Redis 容器)
docker compose -f docker-compose.dev.yml up -d
```

## 技术栈

- **语言**：Go 1.25+
- **JS 引擎**：quickjs-go / goja
- **TS 编译**：esbuild
- **数据库**：SQLite (modernc.org/sqlite) / MySQL (预留)
- **认证**：JWT HMAC-SHA256 + bcrypt
- **ID 生成**：24位 hex 前缀ID (001a/001b/001c/001d)
