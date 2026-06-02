# CDK Exchange

这是一个基于 `Go + Gin + GORM + Vue 3 + Element Plus` 的兑换码系统，当前业务模型已经改成：

- 后台按模板创建 `采购任务`
- 采购任务完成后生成 `兑换内容` 和 `CDK`
- 前台游客只在采购任务 `ready` / `manual_completed` 时才能成功兑换
- 后台可以手动补录 `subscribe_url`
- 团队成员可查看共享数据，但只能由拥有者修改

## 当前流程

1. 后台用户在“模板管理”配置固定购买目标
2. 后台用户设置自己的“账号前缀”
3. 在“采购任务”页新建采购任务
4. 采购任务初始状态为 `pending`，此时还没有兑换内容和 CDK
5. 点击“开始处理”后，runner 按 `order/save -> order/detail -> 结账` 推进到待支付
6. 支付完成后点击“已支付继续抓取”，或后台手动补录 `subscribe_url`
7. 系统生成 `redeem_item` 和 `cdk`，任务进入 `ready` / `manual_completed`
8. 前台游客输入兑换码后，系统返回对应文本内容

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
- `YFJC_BASE_URL`
- `YFJC_COOKIE`
- `YFJC_AUTH_TOKEN`
- `YFJC_ORDER_PAYLOAD_JSON`
- `YFJC_USE_BROWSER`
- `YFJC_HEADLESS`

## 自动化边界

当前仓库已经包含本地自动化脚手架：

- `backend/service/automation_runner.go`
- `backend/automation/yfjc_runner.py`
- `backend/automation/browser_helpers.py`

`prepare_order` 会调用 `order/save`，再调用 `order/detail`，最后尝试在订单详情页点击结账按钮。`fetch_subscribe` 会调用 `getSubscribe` 并读取 `data.subscribe_url`。

## 已验证

- 后端 `go test ./...` 通过
- 前端 `tsc --noEmit` 通过
- 容器内 `npm install && npm run build` 通过
- `docker compose up -d --build` 通过
- `http://localhost:3000/readyz` 返回 `200`
- 浏览器已验证：
  - 采购任务页可访问
  - 新建采购任务后会出现 `pending` 任务
  - 手动补录后任务变为 `manual_completed`
  - 前台用对应兑换码可成功兑换
