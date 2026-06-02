# CDK Exchange

这是一个基于 `Go + Gin + GORM + Vue 3 + Element Plus` 的兑换码系统，当前业务模型已经改成：

- 后台按模板批量生成 `CDK`
- 生成时同步创建本地 `采购任务`
- 前台游客只在采购任务 `ready` / `manual_completed` 时才能成功兑换
- 后台可以手动补录 `subscribe_url`
- 团队成员可查看共享数据，但只能由拥有者修改

## 当前流程

1. 后台用户在“模板管理”配置固定购买目标
2. 后台用户设置自己的“账号前缀”
3. 在“兑换内容”上传单个文本文件
4. 文件每个非空行生成一条：
   - `redeem_item`
   - `cdk`
   - `purchase_task`
5. 采购任务初始状态为 `pending`
6. 后台在“采购任务”页手动补录 `subscribe_url` 后，任务进入 `manual_completed`
7. 前台游客输入兑换码后，系统返回对应订阅链接文本

## 主要页面

- 前台兑换：`http://localhost:3000/exchange`
- 后台登录：`http://localhost:3000/login`
- 采购任务：`http://localhost:3000/admin/purchase-tasks`

默认后台账号：

- 用户名：`admin`
- 密码：`admin123`

## 本地启动

```bash
docker compose up -d --build
```

服务端口：

- 前端：`3000`
- 后端：`8081`
- MySQL：`3307`

健康检查：

```bash
curl -i http://localhost:3000/readyz
```

## 前端构建

如果本机 `npm run build` 受 `rolldown` 原生绑定影响，可以直接用容器构建：

```bash
docker run --rm \
  -v /Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/frontend:/app \
  -v /app/node_modules \
  -w /app \
  node:25-alpine \
  sh -lc "npm install && npm run build"
```

## 环境变量

参考 `.env.example`：

- `DB_ROOT_PASSWORD`
- `DB_DSN`
- `JWT_SECRET`
- `SERVER_PORT`
- `LOG_LEVEL`
- `BCRYPT_COST`
- `MAX_EXCHANGE_QUANTITY`
- `AUTOMATION_PYTHON_BIN`
- `AUTOMATION_SCRIPT_PATH`
- `AUTOMATION_TIMEOUT_SECONDS`
- `AUTOMATION_MAX_RETRIES`

## 自动化边界

当前仓库已经包含本地自动化脚手架：

- `backend/service/automation_runner.go`
- `backend/automation/yfjc_runner.py`
- `backend/automation/browser_helpers.py`

这部分目前只完成了本地 runner、配置和 JSON 协议边界，还没有接入真实外部站点流程执行。

## 已验证

- 后端 `go test ./...` 通过
- 前端 `tsc --noEmit` 通过
- 容器内 `npm install && npm run build` 通过
- `docker compose up -d --build` 通过
- `http://localhost:3000/readyz` 返回 `200`
- 浏览器已验证：
  - 采购任务页可访问
  - 上传生成后会出现 `pending` 任务
  - 手动补录后任务变为 `manual_completed`
  - 前台用对应兑换码可成功兑换
