---
name: dev-up
description: 启动本地开发栈 —— docker 起 MySQL,后台跑 go run main.go,前端跑 npm run dev。当用户说"启动开发环境""跑起来""开发模式启动"时使用。
disable-model-invocation: true
---

# /dev-up — 启动本地开发栈

按以下顺序启动,**每一步都要等前一步就绪再继续**:

## 1. MySQL(docker)

```bash
docker compose up -d mysql
```

等待健康检查:

```bash
until docker compose ps mysql | grep -q healthy; do sleep 2; done
```

如果用户本地已有 MySQL 实例并配好 `DB_DSN`,可跳过此步。

## 2. 后端(本地进程,后台运行)

```bash
cd /Users/shuo/go/src/exchange_cdk/backend && go run main.go
```

用 `run_in_background: true` 让 Claude 在后台跑,记下 task_id,然后用 Monitor 等到日志里出现监听端口(如 `:8080`)或报错。

如果是首次启动且 migrate CLI 已装,先跑迁移:

```bash
cd /Users/shuo/go/src/exchange_cdk/backend && migrate -path migration -database "$DB_DSN" up
```

(没装 migrate CLI 时回退到 `docker compose run --rm backend ./migrate up`。)

## 3. 前端(本地进程,后台运行)

```bash
cd /Users/shuo/go/src/exchange_cdk/frontend && npm run dev
```

同样 `run_in_background: true`,等日志显示 vite 已在 `:5173` 监听。

## 4. 汇报

启动完成后告诉用户:

- 后端: http://localhost:8080
- 前端: http://localhost:5173
- 管理后台默认账号: admin / admin123
- 三个后台 task_id(MySQL 之外的两个进程)

如果任何一步失败,**先停止后续步骤**,把错误日志贴出来等用户决定。
