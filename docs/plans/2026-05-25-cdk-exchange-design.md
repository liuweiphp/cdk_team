# CDK 兑换系统设计文档

## 角色

- **管理员**:导入 CDK、管理用户、查看报表、发布公告
- **普通用户**:登录后选择面额和数量,系统发放对应数量的 CDK

## 核心业务模型(关键)

- **CDK** 就是一段唯一的代码字符串,**自带金额(面额)**,只有 `unused / exchanged` 两种状态,一码只能被领取一次
- **导入** 时管理员指定**本次导入的金额**(同一次导入文件里所有 code 共享同一金额),不同次导入即使金额相同,统计时也按"金额"维度归并
- **领取** 时用户**不输入具体 code**,只选择**面额 + 数量 N**,系统从可用池中分配 N 个该面额的 unused CDK,原子标记为 exchanged,返回这 N 个具体 code 给用户
- **统计** 完全围绕"面额"展开:每个面额的总数 / 已领 / 剩余 / 总金额

## 技术栈

| 层 | 选型 |
|---|------|
| 后端 | Go + Gin + GORM |
| 前端 | Vue 3 + Vite + Element Plus + ECharts + Tiptap |
| 数据库 | MySQL 8 |
| 认证 | JWT(HS256) |
| 密码 | bcrypt(cost=12) |
| 迁移 | golang-migrate/migrate(版本化 SQL) |
| 日志 | zap(结构化) |
| 限流 | golang.org/x/time/rate |
| HTML 净化 | bluemonday |
| Excel 解析 | xuri/excelize |
| 部署 | Docker Compose(MySQL + 后端 + 前端 Nginx) |

## 项目结构

```
exchange_cdk/
├── backend/
│   ├── main.go
│   ├── config/                # 环境变量加载
│   ├── handler/               # auth, cdk, exchange, user, announce, stats
│   ├── middleware/            # JWT 鉴权、角色校验、限流、请求日志
│   ├── model/                 # GORM 模型
│   ├── service/               # 业务逻辑(领取原子操作、导入、统计)
│   ├── pkg/                   # 公共工具:bcrypt、jwt、sanitize
│   └── migration/             # *.up.sql / *.down.sql
├── frontend/
│   └── src/
│       ├── views/admin/       # 仪表盘、CDK 管理、导入、用户管理、公告管理
│       ├── views/user/        # 登录、领取页
│       ├── components/
│       ├── router/
│       ├── stores/            # Pinia
│       └── api/               # axios 封装
├── docker-compose.yml
├── .env.example
└── Dockerfile
```

## 数据模型

### users 用户表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| username | VARCHAR(64) UNIQUE | |
| password_hash | VARCHAR(255) | bcrypt cost=12 |
| role | ENUM('admin','user') | 默认 user |
| status | ENUM('active','disabled') | 默认 active,管理员可禁用 |
| last_login_at | TIMESTAMP NULL | 登录审计 |
| created_at / updated_at | TIMESTAMP | |
| deleted_at | TIMESTAMP NULL | 软删除 |

索引:`UNIQUE(username)`、`INDEX(status)`

### cdk_imports 导入记录表(仅作溯源,不参与业务统计)
| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| filename | VARCHAR(255) | 上传的原始文件名 |
| amount | DECIMAL(12,2) | **本次导入的金额(面额)** |
| total | INT UNSIGNED | 文件总行数 |
| inserted | INT UNSIGNED | 实际入库条数 |
| skipped | INT UNSIGNED | 重复跳过条数 |
| invalid | INT UNSIGNED | 格式无效条数 |
| remark | VARCHAR(255) | 备注 |
| created_by | BIGINT FK→users.id | |
| created_at | TIMESTAMP | |

索引:`INDEX(amount, created_at)`、`INDEX(created_by)`

### cdks 兑换码表(核心)
| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| code | VARCHAR(64) UNIQUE | CDK 码,统一大写,唯一索引;一码一兑 |
| amount | DECIMAL(12,2) | **面额**,继承自导入时的设定 |
| status | ENUM('unused','exchanged') | 默认 unused |
| import_id | BIGINT FK→cdk_imports.id | 溯源用,统计不依赖此字段 |
| exchanged_by | BIGINT FK→users.id NULL | 领取的用户 |
| exchanged_at | TIMESTAMP NULL | |
| created_at / updated_at | TIMESTAMP | |

索引(围绕领取与统计):
- `UNIQUE(code)`
- `INDEX(amount, status)` —— **核心索引**,支撑"按面额取 N 个 unused"和"按面额聚合统计"
- `INDEX(status, exchanged_at)` —— 每日趋势
- `INDEX(exchanged_by, exchanged_at)` —— 用户兑换记录、Top 用户

**CDK 不允许物理删除**,审计需要保留。
**未领取的 CDK 在 v1 不允许修改 amount**(避免破坏统计一致性);如需修正请删除该次导入下未使用的 CDK 后重新导入。

### exchange_orders 领取订单表
一次"用户提交 code+数量"对应一条订单,聚合本次领到的多个 CDK。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| user_id | BIGINT FK→users.id | |
| amount | DECIMAL(12,2) | 领取的面额 |
| quantity | INT UNSIGNED | 本次领取张数 |
| total_amount | DECIMAL(14,2) | `amount × quantity` |
| status | ENUM('success','failed') | |
| fail_reason | VARCHAR(64) NULL | `insufficient_stock` / `rate_limited` / `invalid_input` 等 |
| ip | VARCHAR(45) | |
| user_agent | VARCHAR(255) | |
| created_at | TIMESTAMP | |

索引:`INDEX(user_id, created_at)`、`INDEX(amount, status, created_at)`、`INDEX(status, created_at)`

### exchange_order_items 领取明细表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| order_id | BIGINT FK→exchange_orders.id | |
| cdk_id | BIGINT FK→cdks.id | 本次实际领到的 CDK |
| code | VARCHAR(64) | 冗余,便于查询不 join |
| created_at | TIMESTAMP | |

索引:`INDEX(order_id)`、`UNIQUE(cdk_id)`(防止同一 CDK 被算两次)

### announcements 公告表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| title | VARCHAR(255) | |
| content | TEXT | 富文本 HTML(入库前 sanitize) |
| is_pinned | BOOLEAN | 默认 false |
| created_by | BIGINT FK→users.id | |
| created_at / updated_at | TIMESTAMP | |
| deleted_at | TIMESTAMP NULL | 软删除 |

索引:`INDEX(is_pinned, created_at)`、`INDEX(deleted_at)`

## API 设计

所有响应统一 `{ code, message, data }`,业务码 0 = 成功,非 0 = 业务错误。

```
# 认证
POST   /api/auth/login              {username, password} → {token, user}
# 不开放公开注册

# 当前用户
GET    /api/user/me
GET    /api/user/orders             我的领取记录,?page=&page_size=&amount=

# 用户管理(admin)
GET    /api/admin/users             ?page=&page_size=&keyword=&status=
POST   /api/admin/users             {username, password, role}
PATCH  /api/admin/users/:id         {status?, role?, password?}

# CDK 导入(admin)
POST   /api/admin/cdk/import        multipart: file + amount + remark?
                                    返回 {import_id, total, inserted, skipped, invalid:[{line, code, reason}]}

# CDK 管理(admin)
GET    /api/admin/cdk/list          ?page=&page_size=&amount=&status=&code=&import_id=
GET    /api/admin/cdk/imports       导入记录列表 ?page=&page_size=&amount=

# 可领面额(用户端下拉)
GET    /api/amounts                 → [{amount, remaining}]  仅返回 remaining>0 的面额

# 领取(登录即可)
POST   /api/exchange                {amount, quantity}
                                    → {order_id, amount, quantity, total_amount, codes:[...]}

# 公告
GET    /api/announcements           ?page=&page_size=,is_pinned 排在前
POST   /api/admin/announcements
PUT    /api/admin/announcements/:id
DELETE /api/admin/announcements/:id

# 统计(admin)
GET    /api/admin/stats/overview        总用户数、CDK 总/已领/剩余、总金额、已领金额
GET    /api/admin/stats/by-amount       **核心**:按面额聚合 [{amount, total, exchanged, remaining, exchanged_amount}]
GET    /api/admin/stats/daily           ?start=&end= 每日领取张数与金额趋势
GET    /api/admin/stats/top-users       ?limit= 按累计领取金额 / 张数 排行
GET    /api/admin/stats/imports         按导入批次维度(amount + import 时间)
```

中间件:
- `/api/admin/*` —— JWT + role=admin
- `/api/exchange`、`/api/amounts`、`/api/user/*`、`/api/announcements` —— JWT
- `/api/exchange` —— 限流:同一 user 每分钟 ≤ 10 次,同一 IP 每分钟 ≤ 20 次
- 所有路由记请求日志(method、path、status、latency、user_id、trace_id)

## 核心逻辑

### CDK 码格式
- 长度 16 位,字母表 `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`(去除 0/O/1/I/L)
- 入库前 `strings.ToUpper(strings.TrimSpace(code))`
- 长度允许 8~64 位(兼容外部生成的 CDK),但唯一索引建在归一化后的 code 列

### 导入流程
1. 校验文件大小(≤ 5 MB)、行数(≤ 5 万)、`amount > 0 且 ≤ 99999999.99`
2. 流式读取(CSV/TXT 一行一码;Excel 取第一列,跳过表头)
3. 每行归一化为大写,长度 8~64,字符在字母表内,否则记为 invalid
4. 文件内去重 + 与 DB 既有 code 去重
5. 先创建 `cdk_imports` 记录(状态 running),分批 `INSERT IGNORE INTO cdks(code, amount, status, import_id)`,每批 500 条
6. 写回 `cdk_imports` 的 total/inserted/skipped/invalid 计数

### 领取流程(核心,必须原子)
用户提交 `{amount, quantity}`,后端在**单事务**内:

1. 输入校验:`amount > 0`,`1 ≤ quantity ≤ 50`(单次张数上限,防一次拖空库存)
2. `SELECT id FROM cdks WHERE amount=? AND status='unused' ORDER BY id LIMIT ? FOR UPDATE`
   - 取到的实际行数 `< quantity` → 回滚,返回 `insufficient_stock`,同时写一条 status=failed 的 order
3. `UPDATE cdks SET status='exchanged', exchanged_by=?, exchanged_at=NOW() WHERE id IN (...)`
   - 用 `RowsAffected == quantity` 兜底校验(必须等于,否则视为并发异常,回滚)
4. INSERT exchange_orders(status=success, total_amount=amount×quantity)
5. 批量 INSERT exchange_order_items(order_id, cdk_id, code)
6. 提交事务,响应用户 N 个 code

**并发正确性依赖**:
- 行锁 `FOR UPDATE` 锁定本次要发的 N 行
- `(amount, status)` 联合索引让 `WHERE amount=? AND status='unused'` 走索引扫描,避免锁全表
- 事务隔离级别用默认的 `REPEATABLE READ` 即可
- 测试要求:`go test -race`,起 100 个 goroutine 同时领同一面额,断言总领取数 == 库存数,无超发

### 公告富文本安全
- `POST/PUT /api/admin/announcements` 入口处对 `content` 调用 `bluemonday.UGCPolicy().Sanitize(html)`,**入库即净化**
- 前端 `v-html` 渲染净化后的内容

### 统计实现
- v1:直接 group by,依赖以下索引
  - `cdks(amount, status)` —— `GROUP BY amount, status`
  - `cdks(status, exchanged_at)` —— 每日趋势
  - `exchange_orders(user_id, ...)` —— 用户排行(更高效,避免扫 cdks)
  - `exchange_orders(amount, status, created_at)` —— 面额维度趋势
- v2(数据量 10 万+ 之后):新增 `daily_stats(date, amount, exchanged_count, exchanged_amount)`,定时任务每小时聚合

## 前端页面

### 用户端
| 页面 | 说明 |
|------|------|
| 登录 | 用户名 + 密码;无注册入口 |
| 领取 | 顶部公告栏(置顶醒目)。面额下拉(从 `/api/amounts` 拉取,展示"面额(剩余 X)"),数量输入框(1~50),"立即领取"按钮。结果展示:本次领到的所有 code(支持一键复制全部、单条复制) |
| 我的记录 | 分页表格:订单号、面额、数量、总金额、领取时间;点开详情显示该订单的所有 code |

### 管理端
| 页面 | 说明 |
|------|------|
| 仪表盘 | 概览卡片(用户数 / CDK 总数 / 已领 / 剩余 / 总金额 / 已领金额) + 按面额柱状图(总/已领/剩余 三色) + 每日趋势折线 + Top 用户表 |
| CDK 管理 | 列表筛选(面额、状态、code 模糊、导入批次) + 分页 |
| CDK 导入 | 上传文件 + 填写**金额**(必填,DECIMAL)+ 备注;返回成功/重复/失败明细;另有"导入历史"标签页 |
| 用户管理 | 列表 + 新增 + 禁用/启用 + 重置密码 |
| 公告管理 | 列表 + 新增/编辑(Tiptap 富文本)+ 置顶 + 软删除 |

## 安全要点

- **密码**:bcrypt cost=12;不存明文
- **JWT**:HS256,密钥从 `JWT_SECRET` 读取;access token 8h;`Authorization: Bearer`(避免 CSRF)
- **并发安全**:`SELECT ... FOR UPDATE` + `RowsAffected` 校验,杜绝超发
- **限流**:`golang.org/x/time/rate` token bucket,user+IP 双维度
- **XSS**:bluemonday `UGCPolicy()` 在公告入库前 sanitize
- **审计**:`exchange_orders` 记录所有领取尝试(成功+失败);管理员关键操作打应用日志
- **登录失败**:5 次失败后锁定账户 15 分钟(v1 可选,可先在日志层观测)
- **密码强度**:管理员创建用户时强制 ≥ 8 位
- **单次领取上限**:50 张/次,防一次性拖空库存

## 非功能性要求

- **分页**:所有列表接口默认 `page=1&page_size=20`,上限 100;返回 `{list, total, page, page_size}`
- **配置**:环境变量驱动,`.env.example` 列出所有 key
  - `DB_DSN`、`JWT_SECRET`、`SERVER_PORT`、`LOG_LEVEL`、`BCRYPT_COST`、`MAX_EXCHANGE_QUANTITY`(默认 50)
- **迁移**:`golang-migrate/migrate`,SQL 版本化(`backend/migration/0001_init.up.sql` 等)
- **日志**:zap JSON 输出;请求日志含 `trace_id`(uuid,中间件注入);错误日志含 stack
- **错误响应**:统一 `{code, message, data}`,常见错误码:
  - `40001` 参数无效
  - `40101` 未登录
  - `40301` 权限不足
  - `40901` 库存不足
  - `42901` 请求过于频繁
  - `50001` 服务异常
- **CORS**:开发环境允许 `localhost:5173`;生产同源部署(Nginx 转发 `/api/*`)
- **测试要求**:
  - service 层单元测试(领取成功 / 库存不足 / 限流 / 异常回滚)
  - **并发集成测试**(`-race`,100 goroutine 同时领同一面额,断言无超发)
- **可观测**:`/healthz` 存活、`/readyz` DB 可达
- **Docker Compose**:
  - MySQL 数据卷 `mysql_data:/var/lib/mysql`,健康检查 `mysqladmin ping`
  - 后端 `depends_on: { mysql: { condition: service_healthy } }`
  - 前端 nginx serve dist,`/api/*` 反代到后端
  - `.env` 不提交进 git(`.gitignore` 包含)

## 代码规范

- Go 导出结构体、函数、字段的注释使用中文,格式 `// 函数名 功能描述`
- 数据库表及字段 `COMMENT` 使用中文,迁移 SQL 中每列注明用途
- Vue 组件关键逻辑、props、emits 使用中文注释
- API 路由注册处中文注释标注功能分组
- 错误使用 `fmt.Errorf("...: %w", err)` 包裹,顶层统一转为业务错误码
- handler 只做参数解析与响应,业务逻辑全部下沉到 service

## 开放问题(待实现时确认)

1. 领取成功后是否需要回调外部系统(实际使用 CDK 的系统)?若需要,新增 `delivered_at` 字段 + outbox 表
2. 是否支持"按导入批次"撤回?当前模型导入后未使用的可以删,已领取的无法回收
3. 公告是否需要区分"用户可见"与"管理员可见"两类?
4. 是否需要"导出已领取记录"为 Excel 的能力(管理后台报表)?
