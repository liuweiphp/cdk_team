# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

业务、API、技术栈、目录结构详见 @README.md —— 编辑代码前先读它。

## 开发工作流

日常开发是**本地进程**模式,不要默认 docker compose:

```bash
# MySQL 仍走 docker(本地无 MySQL 时)
docker compose up -d mysql

# 后端:本地 go run,改完直接重启
cd backend && go run main.go         # :8080

# 前端:本地 vite 热重载
cd frontend && npm run dev           # :5173,/api 代理到 :8080
```

docker compose 主要用于跑 MySQL 或一次性集成验证。

## 迁移

`golang-migrate/migrate` 管理 SQL,文件在 `backend/migration/`,命名 `NNNN_name.{up,down}.sql`。

- **迁移在后端启动时自动跑**(`main.go` 通过 `pkg/migrate` 用 `go:embed` 嵌入的 SQL),`schema_migrations` 表幂等保证不重跑。本地 `go run` 和 docker 启动都走同一路径,**不需要手动 `migrate up`**。
- 新增迁移**必须**同时写 up 和 down,序号严格递增 —— 新文件加到 `backend/migration/` 后重启服务即生效(因为 embed 是编译期的,所以 docker 模式下要重新 build 镜像)
- 启动时连接 MySQL 会自动追加 `multiStatements=true`(单文件多 statement 必需),GORM 的连接不受影响
- 回滚(`migrate down`)仍需手动跑,见 `/db-migrate` skill

## 数据持久化

MySQL 容器用 **bind mount** 到 `./data/mysql/`(已 gitignore),不是 docker named volume。

- **备份/整机搬迁**:停掉容器后直接拷贝 `./data/mysql/` 目录即可,不需要 `docker volume` 操作
- **不要 `rm -rf ./data/`**,这会清空整个数据库
- `DB_ROOT_PASSWORD` 在 `.env` 里配置,docker-compose 用 `${VAR:?...}` 强校验 —— 缺失时 compose 启动直接失败,**不要给它加默认值**

## 测试

目标是**高覆盖率**。改动任何 `backend/service/` 下的业务逻辑都应该写或更新对应单元测试(尤其是 `exchange_service.go`)。运行:

```bash
cd backend && go test ./...
```

如果暂时无法补全测试,在交付时明确指出未覆盖的路径,不要默默跳过。

## 业务 gotcha(改代码前必读)

### 认证/权限

- JWT HS256,过期 8h,密钥来自 `JWT_SECRET`,签发/校验在 `backend/pkg/jwt/`
- 密码 bcrypt cost=12(`BCRYPT_COST` 可调,不要硬编码)
- 路由保护通过 `middleware/`:`JWTAuth` 拦未登录,`AdminOnly` 拦非管理员
- 新加管理端路由必须挂 `AdminOnly`,新加用户端路由必须挂 `JWTAuth`

### CDK 领取并发(`backend/service/exchange_service.go`)

这是最容易写错的地方:

- 必须在**单一事务**内完成 `SELECT ... FOR UPDATE → UPDATE status → INSERT order → INSERT items → COMMIT`
- 必须用 `RowsAffected` 兜底校验,防止行锁竞争下的超发
- 限流在 `middleware/`:同用户 ≤10/分钟、同 IP ≤20/分钟,目前**单实例假设**,不要引入需要分布式锁的逻辑而不提示
- 单次领取上限 `MAX_EXCHANGE_QUANTITY`(默认 50)

### CDK 导入

- 文件内先去重,DB 用 `INSERT IGNORE` 防重复 —— **静默忽略**,如果加日志/审计要明确告知用户
- 码字母表固定 `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`,长度 8~64,不要扩字符集
- 上限 5MB / 5 万行,改动这个值时同步更新前端校验

### 统计口径

按**面额**统计,不是按导入批次。`stats_service.go` 里的聚合都基于面额维度,新增报表沿用同口径。

## 代码风格

- Go:`gofmt` 标准(有 PostToolUse hook 自动跑),包名小写无下划线,error 用 `fmt.Errorf("...: %w", err)` 包装
- 前端:暂无 ESLint/Prettier 配置,跟随现有 `.vue`/`.ts` 缩进(2 空格)和 Element Plus 命名
- 响应统一 `{ code, message, data }`,失败 code≠0,不要直接返回原始错误信息给前端

## 不要做的事

- 不要把 `JWT_SECRET`、`DB_DSN` 等敏感配置写进代码或日志
- 不要绕过 `service` 层从 `handler` 直接操作 GORM
- 不要把 CDK 状态机扩成 `unused/exchanged` 之外,业务模型刻意保持二态
