# CDK Exchange — 文本兑换码系统

管理员维护模板并上传单个内容文件 → 系统按文件每一行自动生成兑换内容和 CDK → 游客输入兑换码 → 系统返回并下载对应 TXT 文件。

## 快速开始

```bash
# 1. 启动所有服务
docker compose up -d

# 2. 跑数据库迁移
docker compose exec backend ./migrate up

# 3. 访问
# 游客兑换: http://localhost:3000
# 管理后台: admin / admin123
```

本地开发:

```bash
# 后端
cd backend
cp ../.env.example ../.env  # 按需修改 DB_DSN
go run main.go               # :8080

# 前端
cd frontend
npm install
npm run dev                  # :5173, /api 代理到 :8080
```

## 角色

| 角色 | 功能 |
|------|------|
| 管理员 | 管理模板、上传内容文件自动生成 CDK、管理用户、查看仪表盘、发布公告 |
| 游客 | 无需登录，输入兑换码兑换并下载 TXT 文件 |

## 业务模型

- **模板** 使用 `{{content}}` 作为占位符，上传文件的每一行会填入该位置
- **兑换内容** 是后台上传文件逐行生成的文本文件内容，包含名称、下载文件名和 TXT 内容
- **CDK** 是唯一代码串，绑定一个兑换内容，只有 unused / exchanged 两种状态，一码一兑
- **生成** 时选择模板并上传单个 TXT/CSV 文件，每一行自动生成一个兑换内容和一个 CDK
- **兑换** 时游客输入 code，系统原子标记为 exchanged，返回绑定的文本文件内容

## 技术栈

| 层 | 选型 |
|---|------|
| 后端 | Go + Gin + GORM |
| 前端 | Vue 3 + Vite + Element Plus + ECharts + Tiptap |
| 数据库 | MySQL 8 |
| 认证 | JWT (HS256, 8h) |
| 密码 | bcrypt (cost=12) |
| 迁移 | golang-migrate/migrate |
| 日志 | zap (结构化 JSON) |
| 部署 | Docker Compose |

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DB_DSN` | MySQL 连接串 | 必填 |
| `JWT_SECRET` | JWT 签名密钥 | 必填 |
| `SERVER_PORT` | 后端监听端口 | `8080` |
| `LOG_LEVEL` | 日志级别 | `info` |
| `BCRYPT_COST` | bcrypt 成本 | `12` |
| `MAX_EXCHANGE_QUANTITY` | 单次领取上限 | `50` |

## API 概览

```
# 认证
POST   /api/auth/login

# 用户端
GET    /api/user/me
POST   /api/redeem               {code}
GET    /api/announcements        公告列表

# 管理端
GET    /api/admin/users
POST   /api/admin/users
PATCH  /api/admin/users/:id
GET    /api/admin/redeem-items
POST   /api/admin/redeem-items
POST   /api/admin/redeem-items/import  multipart: file + template_id
PUT    /api/admin/redeem-items/:id
DELETE /api/admin/redeem-items/:id
GET    /api/admin/templates
POST   /api/admin/templates
PUT    /api/admin/templates/:id
DELETE /api/admin/templates/:id
GET    /api/admin/cdk/list
GET    /api/admin/cdk/imports
POST   /api/admin/announcements
PUT    /api/admin/announcements/:id
DELETE /api/admin/announcements/:id
GET    /api/admin/stats/overview
GET    /api/admin/stats/by-amount
GET    /api/admin/stats/daily
GET    /api/admin/stats/top-users
```

响应统一 `{ code, message, data }`，code=0 为成功。

## 核心实现

### 兑换原子操作

单事务内完成，防止超发：

```
SELECT ... FOR UPDATE → UPDATE status → COMMIT
RowsAffected 兜底校验
```

### CDK 导入

- 支持 CSV/TXT（一行一码）和 Excel（取第一列）
- 推荐流程：先在“模板管理”确认模板，再在“兑换内容”上传单个 TXT/CSV 内容文件
- 内容文件每一行是一条待填入模板 `{{content}}` 的兑换内容
- 后端会为每一行自动生成一个 CDK
- 码长 8~64 位，字母表 `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`
- 文件内去重 + DB `INSERT IGNORE` 防重复
- 上限 5MB / 5 万行

### 限流

`POST /api/exchange`：同用户每分钟 ≤ 10 次，同 IP 每分钟 ≤ 20 次

## 项目结构

```
exchange_cdk/
├── backend/
│   ├── main.go
│   ├── config/          # 环境变量加载
│   ├── handler/         # HTTP 处理器
│   ├── middleware/      # JWT / admin / 限流 / 日志
│   ├── model/           # GORM 数据模型
│   ├── service/         # 业务逻辑(核心: exchange_service)
│   ├── pkg/             # bcrypt / jwt / response / sanitize
│   ├── migration/       # SQL 迁移文件
│   └── router/          # 路由注册
├── frontend/
│   └── src/
│       ├── layouts/     # 用户端 + 管理端布局
│       ├── views/       # 页面组件
│       ├── router/      # 路由 + 守卫
│       ├── stores/      # Pinia 状态
│       ├── api/         # axios 封装
│       └── styles/      # 暗色主题 + 玻璃拟态
├── docker-compose.yml
├── Dockerfile
└── .env.example
```
