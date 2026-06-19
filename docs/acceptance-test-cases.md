# 验收测试用例

> 版本: v1 | 日期: 2026-06-19 | 对应设计: account-system-design.md

---

## 1. auth-server (:9722)

### 1.1 登录

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| A-001 | 内置 admin 登录 | 服务启动，种子数据已写入 | `POST /api/auth/login {"tenant":"admin","username":"admin","password":"admin123"}` | 200, 返回 `{token, user}`，user.role=`admin`；Set-Cookie 包含 token |
| A-002 | 登录 — 错误密码 | 同上 | 密码改为 `wrong` | 401, `{"error":"invalid credentials"}` |
| A-003 | 登录 — 不存在的租户 | 同上 | tenant 改为 `noexist` | 401 |
| A-004 | 登录 — 缺少字段 | 同上 | body 缺 username | 400, validator 报错 `missing required field: "username"` |
| A-005 | 登录 — 租户主账号 | 通过 admin 创建了租户 `test` | `{"tenant":"test","username":"test","password":"test"}` | 200, user.role=`tenant_admin` |
| A-006 | JWT 内容校验 | 登录成功 | 解码 token payload | 包含 `sub`(用户ID 001b...), `tid`(租户ID 001a...), `tn`, `role`, `exp`, `iat`, `jti` |
| A-007 | Token 过期 | 登录成功 | 等待超过 `token_expire_hours` 后使用 | 401 |

### 1.2 登出

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| A-010 | 登出清除 Cookie | 已登录 | `POST /api/auth/logout` | 200, Set-Cookie 中 token 过期/清除 |
| A-011 | 登出后 Redis 黑名单 | Redis 已配置且已登录 | 登出后拿旧 token 访问 `/api/auth/me` | 401 |
| A-012 | 登出无 Redis 降级 | Redis 未配置，已登录 | 登出后拿旧 token 访问 `/api/auth/me` | 200（token 在有效期内仍可用，因为是纯前端删除） |

### 1.3 当前用户

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| A-020 | 获取当前用户信息 | 已登录 | `GET /api/auth/me` | 200, 返回 `{id, username, role, tenantId, tenantName}` |
| A-021 | 无 token 访问 | 未登录 | `GET /api/auth/me` | 401 |
| A-022 | 伪造 token | — | `Authorization: Bearer invalid.jwt.token` | 401 |

### 1.4 Betamap 获取

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| A-030 | 获取当前租户 betamap | 已登录 admin 租户 | `GET /api/auth/betamap` | 200, 返回 JSON，`modules.script_editor: true` |
| A-031 | 修改后实时生效 | admin 把 `script_debug` 改为 false | 该租户用户 `GET /api/auth/betamap` | `script_debug: false` |
| A-032 | 无 token 访问 | 未登录 | `GET /api/auth/betamap` | 401 |

### 1.5 Token 刷新

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| A-040 | 刷新 token | 已登录 | `POST /api/auth/refresh` | 200, 返回新 token，exp 晚于旧 token |
| A-041 | 过期 token 刷新 | token 已过期 | 同上 | 401 |

---

## 2. bff-server (:9724)

### 2.1 页面与跳转

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| B-001 | 未登录访问 | 无 token | `GET /` | 302 到 auth-server 登录页 |
| B-002 | 已登录访问门户 | 已登录 | `GET /bff/ui/index.html` | 200, 渲染门户页面 |
| B-003 | 点击"查看脚本" | 已登录，门户页面 | 点击按钮 | 跳转到 `http://127.0.0.1:9720/ts-quickjs/ui/index.html` |

### 2.2 功能入口 API

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| B-010 | 获取 entries — 普通租户 | 已登录，betamap 仅 `script_editor` 开启 | `GET /api/bff/entries` | 返回列表，`scripts.enabled: true`，`sql_runner.enabled: false`，`admin.enabled: false` |
| B-011 | 获取 entries — admin 用户 | admin 用户登录 | 同上 | 所有入口 `enabled: true` |
| B-012 | 获取 entries — 全部关闭 | 租户 betamap 全部 false | 同上 | 返回列表所有 `enabled: false` |
| B-013 | 无 token 访问 | 未登录 | `GET /api/bff/entries` | 401 |

---

## 3. admin-server (:9723)

### 3.1 权限控制

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| C-001 | 普通用户访问 admin | 普通 user 登录 | `GET /api/admin/tenants` | 403 |
| C-002 | tenant_admin 访问 admin | tenant_admin 登录 | 同上 | 403 |
| C-003 | admin 用户访问 admin | admin 登录 | 同上 | 200 |

### 3.2 租户管理

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| C-010 | 创建租户 | admin 登录 | `POST /api/admin/tenants {"name":"acme","label":"ACME Corp"}` | 201, ID 以 `001a` 开头，同时自动创建主账号（username=acme, password=acme, role=tenant_admin） |
| C-011 | 创建租户 — 重名 | admin 登录 | 再次创建 name=`acme` | 409 |
| C-012 | 创建租户 — betamap 默认值 | admin 登录 | 创建后查 betamap | 与 `betamap.def.json` 默认值一致 |
| C-013 | 创建租户 — 自定义 betamap | admin 登录 | body 包含 `betamap: {"script_debug":true}` | 创建后 betamap.script_debug=true |
| C-014 | 列出所有租户 | admin 登录 | `GET /api/admin/tenants` | 200, 至少包含 `admin` 和新创建的租户 |
| C-015 | 删除租户 | admin 登录 | `DELETE /api/admin/tenants/{id}` | 204, 租户及关联用户被级联删除 |

### 3.3 用户管理

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| C-020 | 在租户下创建用户 | admin 登录，已创建租户 acme | `POST /api/admin/tenants/{acme_id}/users {"username":"dev1","password":"pass1","role":"user"}` | 201, ID 以 `001b` 开头 |
| C-021 | 创建用户 — 重名 | 同上 | 再次创建 username=`dev1` | 409 |
| C-022 | 列出租户下用户 | admin 登录，acme 有 2 个用户 | `GET /api/admin/tenants/{acme_id}/users` | 200, 列表含主账号 acme 和新用户 dev1 |
| C-023 | 删除用户 | admin 登录 | `DELETE /api/admin/tenants/{acme_id}/users/{uid}` | 204 |

### 3.4 Backdoor

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| C-030 | 查看所有用户 | admin 登录，已有多个租户和用户 | `GET /api/admin/users` | 200, 跨租户列出全部用户（含 password_hash） |
| C-031 | 重置用户密码 | admin 登录 | `PUT /api/admin/users/{uid}/password {"password":"newpass"}` | 204, 后续用新密码可登录 |
| C-032 | 普通用户访问 backdoor | user 登录 | `GET /api/admin/users` | 403 |

### 3.5 Betamap 管理

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| C-040 | 获取租户 betamap | admin 登录 | `GET /api/admin/tenants/{id}/betamap` | 200, 返回 betamap JSON |
| C-041 | 修改租户 betamap | admin 登录 | `PUT /api/admin/tenants/{id}/betamap {"script_debug":true}` | 200, 该租户用户可访问 debug |
| C-042 | 获取 betamap 定义文件 | admin 登录 | `GET /api/admin/betamap/def` | 200, 返回 `betamap.def.json` 内容 |

---

## 4. ts-quickjs (:9720)

### 4.1 认证与隔离

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| T-001 | 未登录访问 | 无 token | `GET /api/scripts` | 401 |
| T-002 | 租户 A 用户创建脚本 | tenant-a 用户登录 | `POST /api/scripts {"name":"s1","label":"S1",...}` | 201, tenant_id = 租户A |
| T-003 | 租户 A 用户列脚本 | tenant-a 用户登录 | `GET /api/scripts` | 200, 仅返回租户A的脚本 |
| T-004 | 租户隔离 — 不可见其他租户脚本 | tenant-a 用户登录，租户B有脚本 | `GET /api/scripts/{租户B的脚本ID}` | 404 |
| T-005 | admin 可查看全部脚本 | admin 登录 | `GET /api/scripts` | 200, 返回所有租户的脚本 |

### 4.2 Betamap 后端拦截

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| T-010 | debug 被 betamap 关闭 | 租户 betamap.script_debug=false | `POST /api/scripts/{id}/debug` | 403, `{"error":"feature disabled: script_debug"}` |
| T-011 | debug 被 betamap 开启 | 租户 betamap.script_debug=true | 同上 | 正常执行，返回 debug 结果 |
| T-012 | admin 不受 betamap 限制 | admin 登录，租户 script_debug=false | 同上 | 正常执行（admin 绕过 betamap） |
| T-013 | 修改 betamap 后实时生效 | script_debug=false → admin 改为 true | 该租户用户调用 debug | 200 正常执行（立即生效，无需重登录） |

### 4.3 断点功能回归

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| T-020 | 添加断点 | 已登录，有脚本 | 点击行号 | API POST 200, 行号红色标记 |
| T-021 | 移除断点 | 已有断点 | 再次点击同一行号 | API DELETE 200, 标记消失 |
| T-022 | 断点调试执行 | 已设断点 | Debug 运行 | 在断点处暂停，返回 variables + call stack |

---

## 5. sql-runner (:9721)

### 5.1 权限控制

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| S-001 | 普通用户访问 | user 登录 | `GET /api/sql/tables` | 403 |
| S-002 | tenant_admin 访问 | tenant_admin 登录 | 同上 | 403 |
| S-003 | admin 用户访问 | admin 登录 | 同上 | 200, 返回表列表 |

### 5.2 多数据库连接

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| S-010 | SQLite 连接 | config: `db_driver: sqlite3` | admin 登录，`POST /api/sql/run {"sql":"SELECT 1"}` | 200 |
| S-011 | MySQL 连接 | MySQL 容器运行中，config 指向 MySQL | admin 登录，`POST /api/sql/run {"sql":"SELECT 1"}` | 200, 成功查询 |
| S-012 | 非法 SQL 拦截 | admin 登录 | `POST /api/sql/run {"sql":"DROP TABLE scripts"}` | 403 |
| S-013 | 空 SQL | admin 登录 | `POST /api/sql/run {"sql":""}` | 400 |

### 5.3 Betamap 控制

| 编号 | 用例 | 前置条件 | 步骤 | 预期 |
|------|------|----------|------|------|
| S-020 | sql_runner 被 betamap 关闭 | admin 的 sql_runner betamap=true，但当前为普通 admin 用户 | admin 该租户关闭 sql_runner 后访问 | 403 |
| S-021 | sql_runner betamap 开启 | admin 租户 sql_runner=true | admin 访问 | 200 |

---

## 6. core/idgen

| 编号 | 用例 | 步骤 | 预期 |
|------|------|------|------|
| I-001 | 租户 ID 格式 | `NewTenantID()` | 20 位 hex 字符串，以 `001a` 开头 |
| I-002 | 用户 ID 格式 | `NewUserID()` | 20 位 hex，以 `001b` 开头 |
| I-003 | 脚本 ID 格式 | `NewScriptID()` | 20 位 hex，以 `001c` 开头 |
| I-004 | 断点 ID 格式 | `NewBreakpointID()` | 20 位 hex，以 `001d` 开头 |
| I-005 | 全局唯一 | 连续生成 10 万个 ID | 无重复 |
| I-006 | 字典序递增 | 生成 ID1, sleep(5ms), 生成 ID2 | ID1 < ID2 (字符串比较) |

---

## 7. core/db

| 编号 | 用例 | 步骤 | 预期 |
|------|------|------|------|
| D-001 | SQLite 建表 | `NewStore({driver:"sqlite"})` | 自动创建 tenants/users/scripts/breakpoints 四张表 |
| D-002 | 种子数据 | 启动后查询 | tenants 表有 `admin` 租户，users 表有 `admin` 用户 (bcrypt hash) |
| D-003 | 种子数据幂等 | 重启服务 | 不重复插入 admin 数据 |
| D-004 | 旧数据迁移 | 存在旧的 integer ID scripts 表 | 迁移到新 TEXT ID + tenant_id=admin |
| D-005 | 工厂切换 MySQL | `NewStore({driver:"mysql","mysql_dsn":"..."})` | 返回 MySQL 实现，操作同一套接口 |

---

## 8. core/auth

| 编号 | 用例 | 步骤 | 预期 |
|------|------|------|------|
| U-001 | JWT 签发验证 | `Sign(claims) → token → Verify(token)` | 验证通过，返回 claims |
| U-002 | JWT 签名伪造 | 修改 token payload 后验证 | 失败 |
| U-003 | JWT 过期 | 签发 exp=now-1s 的 token | 验证失败 |
| U-004 | bcrypt 哈希 | `Hash("pass") → hash` | `Compare("pass", hash) == true`, `Compare("wrong", hash) == false` |
| U-005 | bcrypt 盐值 | 两次 `Hash("pass")` | 产生不同 hash |
| U-006 | AuthMiddleware 提取 Header token | `Authorization: Bearer <valid>` | ctx 注入 UserID/TenantID/Role |
| U-007 | AuthMiddleware 提取 Cookie token | `Cookie: token=<valid>` | 同上 |
| U-008 | AuthMiddleware 空 token | 无 Header/Cookie | 401 |
| U-009 | AdminMiddleware 放行 admin | ctx.Role=admin | 通过 |
| U-010 | AdminMiddleware 拒绝 user | ctx.Role=user | 403 |

---

## 9. 集成测试：完整流程

### 9.1 首次使用流程

| 编号 | 步骤 | 预期 |
|------|------|------|
| F-001 | 启动全部 5 服务 | auth(:9722), bff(:9724), admin(:9723), ts(:9720), sql(:9721) 均监听成功 |
| F-002 | 访问 `http://127.0.0.1:9724/` | 302 → auth-server 登录页 |
| F-003 | 用 admin/admin123 登录 | 302 → bff 门户，显示按钮 |
| F-004 | 点击"查看脚本" | 跳转到 ts-quickjs 编辑器 |
| F-005 | 创建脚本 `hello.ts` + Run | 执行成功，有输出 |
| F-006 | 回到 bff，admin 可见"管理面板"按钮 | 点击跳转 admin-server |

### 9.2 多租户隔离流程

| 编号 | 步骤 | 预期 |
|------|------|------|
| F-010 | admin 创建租户 `acme` | 自动创建主账号 acme/acme (tenant_admin) |
| F-011 | admin 为 acme 创建用户 `dev1/pass1` (role=user) | 成功 |
| F-012 | dev1 登录 → 创建脚本 A | 成功 |
| F-013 | admin 创建租户 `xyz` → 用户 `dev2` | 成功 |
| F-014 | dev2 登录 → 列脚本 | 看不到脚本 A |
| F-015 | dev2 尝试 `GET /api/scripts/{脚本A的ID}` | 404 |
| F-016 | admin 列脚本 | 看到所有脚本 (含 A + dev2 的脚本) |

### 9.3 Betamap 控制流程

| 编号 | 步骤 | 预期 |
|------|------|------|
| F-020 | admin 将 acme 的 betamap.script_debug 设为 false | 成功 |
| F-021 | acme 的 dev1 刷新门户页 | "查看脚本"仍可用，"调试功能"可能不可见（前端）+ 后端拦截 |
| F-022 | dev1 直接调用 `POST /api/scripts/{id}/debug` | 403 |
| F-023 | admin 将 script_debug 改回 true | 成功 |
| F-024 | dev1 再次调用 debug | 200，正常执行 |

### 9.4 Redis 黑名单流程

| 编号 | 步骤 | 预期 |
|------|------|------|
| F-030 | 启动 Redis 容器，配置 redis_addr | 服务启动正常 |
| F-031 | admin 登录 → 登出 | 登出成功 |
| F-032 | 拿旧 token 访问 `/api/auth/me` | 401（token 在黑名单中） |
| F-033 | 关闭 Redis，未配置 redis_addr | 服务降级，仅本地验签 |

---

## 10. 性能与边界

| 编号 | 用例 | 预期 |
|------|------|------|
| P-001 | 并发登录 100 个不同用户 | 全部成功，无死锁 |
| P-002 | ID 生成器并发 1000 goroutine 各生成 100 个 ID | 10 万个 ID 无重复 |
| P-003 | Token 即将过期时刷新 | 返回新 token，旧 token 在 Redis 中入黑名单 |
| P-004 | 超长 betamap JSON (100 个模块) | 存入/读取正确，无截断 |
| P-005 | 不存在的 script ID 执行 Run | 404 |
| P-006 | 空 body 的 POST 请求 | validator 正确处理 (body 为空 → 跳过或报错) |

---

> 验收条件：所有 [编号] 标记的用例通过，方可合入主干。
