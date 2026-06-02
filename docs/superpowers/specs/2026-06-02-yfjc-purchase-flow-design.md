# YFJC 采购流改造设计

## 背景

当前项目的兑换内容由后台用户手动上传，前台用户兑换时直接拿到 `redeem_items.content`。新的业务要求是将“现成库存”改为“后台生成兑换码后，系统异步去外部站点准备内容”，并保留失败时的人工补录能力。

外部站点为 `https://www.yfjc.xyz/#/register?code=d0CsArLt`。最终需要回填给兑换用户的内容，不是账号信息，也不是页面整段文本，而是接口 `GET /api/v1/user/getSubscribe` 返回中的 `data.subscribe_url`。

## 已确认约束

- 自动注册和下单发生在后台“生成兑换码”时，不在前台兑换时执行。
- 先生成本地兑换码，再异步注册和下单。
- 即使异步失败，也保留本地兑换码，允许后台人工补录最终内容。
- 最终兑换内容是 `subscribe_url`。
- 每个兑换模板绑定一个固定购买目标。
- 外部账号名格式为 `前缀-模板-序号`。
- `前缀` 由后台用户单独配置。
- `序号` 按“每个团队、每个模板”分别递增。
- 第一版采用本机单机模式。
- 自动化执行层采用 `Python + Playwright/Camoufox`。
- 页面动作失败后：先有限重试，超过阈值转人工复核。

## 目标

1. 后台生成兑换码时，自动创建对应的采购任务。
2. 后台用户手动完成外部支付后，系统能够继续抓取 `subscribe_url` 并回填。
3. 前台只有在内容准备完成后才允许成功兑换。
4. 自动化失败时，后台用户能够介入补录，不阻塞业务。

## 非目标

- 第一版不拆分独立自动化服务，不引入单独容器。
- 第一版不做复杂分布式队列，只支持单机本地异步执行。
- 第一版不自动完成支付。
- 第一版不尝试统一多外部站点，只支持当前 YFJC 采购流。

## 总体架构

系统拆为三层：

1. Go 后端
   - 管理 CDK、模板、兑换内容、团队权限、采购任务状态。
   - 提供后台管理接口和前台兑换接口。
   - 负责启动本地 Python 自动化脚本并处理结果。
2. Python 自动化脚本
   - 使用 `Playwright + Camoufox`。
   - 负责页面打开、等待页面加载完成、点击、注册、下单、抓取订阅链接。
   - 通过 JSON 与 Go 后端交互。
3. MySQL
   - 存储模板、兑换内容、CDK、团队、采购任务、团队模板序号等状态。

## 数据模型设计

### 1. 用户外部配置

在 `users` 表增加：

- `external_account_prefix`：外部账号名前缀

用途：

- 生成外部账号名时作为前缀部分

### 2. 模板扩展

在 `redeem_templates` 增加：

- `external_target_code`：固定购买目标编码
- `external_target_name`：固定购买目标名称
- `external_provider`：固定为 `yfjc`
- `result_content_mode`：固定为 `subscribe_url`

用途：

- 每个模板绑定一个固定购买目标
- 后续若扩其他外部站点，仍沿用同一模型

### 3. 团队模板序号表

新增 `team_template_sequences`：

- `id`
- `team_owner_id`
- `template_id`
- `current_seq`
- `created_at`
- `updated_at`

约束：

- `(team_owner_id, template_id)` 唯一

用途：

- 为账号命名规则 `前缀-模板-序号` 提供递增序号

### 4. 采购任务表

新增 `purchase_tasks`：

- `id`
- `team_owner_id`
- `template_id`
- `redeem_item_id`
- `cdk_id`
- `created_by`
- `account_prefix`
- `account_name`
- `template_code_part`
- `sequence_no`
- `target_code`
- `target_name`
- `provider`
- `status`
- `retry_count`
- `payment_status`
- `manual_review_reason`
- `external_order_no`
- `subscribe_url`
- `last_error`
- `browser_trace_path`
- `screenshot_path`
- `html_dump_path`
- `payload_json`
- `created_at`
- `updated_at`

说明：

- `payload_json` 用于保留脚本返回的原始结构化结果，便于排障。
- `subscribe_url` 一旦拿到，即写回 `redeem_items.content`。

## 状态机设计

`purchase_tasks.status`：

- `pending`
  - 本地任务已创建，等待异步执行
- `registering`
  - 正在注册外部账号
- `ordering`
  - 正在创建外部订单
- `pending_payment`
  - 已创建订单，等待人工支付
- `fetching_subscribe`
  - 支付完成后，正在抓取 `subscribe_url`
- `ready`
  - 已成功拿到 `subscribe_url` 并回填
- `needs_manual_review`
  - 自动化经过有限重试后仍失败，等待人工介入
- `manual_completed`
  - 后台人工补录了最终内容
- `failed`
  - 明确终止且不再自动重试

`payment_status`：

- `unpaid`
- `paid`
- `unknown`

## 账号命名规则

格式：

- `前缀-模板-序号`

规则：

- `前缀` 取后台用户配置的 `external_account_prefix`
- `模板` 取模板短码，建议由 `external_target_code` 或模板名归一化得到
- `序号` 使用 `team_template_sequences` 按团队拥有者和模板维度递增

示例：

- `vip-gptplus-0001`
- `vip-gptplus-0002`

## 后台生成流程

### 单条创建

后台生成兑换码时：

1. 创建本地 `redeem_item`
2. 创建本地 `cdk`
3. 分配团队模板序号
4. 生成 `account_name`
5. 创建 `purchase_task(status=pending, payment_status=unpaid)`
6. 异步触发 Python 自动化脚本

### 批量创建

若后续保留批量生成入口，则每一行生成独立的：

- `redeem_item`
- `cdk`
- `purchase_task`

每一条任务拥有独立外部账号名。

## 自动化执行层设计

### 技术选型

- `Python`
- `Playwright`
- `Camoufox`

### 执行边界

Go 后端不直接控制浏览器细节，只做：

- 构造任务上下文
- 启动本地 Python 脚本
- 接收 JSON 结果
- 落库更新状态

Python 脚本负责：

- 打开目标页面
- 等待页面可交互
- 点击按钮
- 注册账号
- 创建订单
- 在支付完成后请求 `getSubscribe`

### 调用方式

第一版采用命令行子进程：

```bash
python3 automation/yfjc_runner.py --task-id 123
```

输入上下文由 Go 后端通过：

- 命令行参数
- 或临时 JSON 文件

传入。

输出统一为 JSON，例如：

```json
{
  "status": "pending_payment",
  "external_order_no": "ORD123",
  "subscribe_url": "",
  "error": "",
  "artifacts": {
    "screenshot_path": "...",
    "html_dump_path": "..."
  }
}
```

## 页面等待与点击设计

### 目标

避免脚本因为页面未真正加载完成而误点、漏点或卡死。

### 原则

不使用固定 `sleep` 作为主策略，而是使用“明确就绪信号 + 动作后结果信号”。

### Python 侧统一封装

封装三个核心函数：

1. `wait_page_ready(page, ready_selectors, loading_selectors)`
2. `wait_actionable(locator)`
3. `click_and_verify(locator, verify_fn)`

### `wait_page_ready`

职责：

- 等 `domcontentloaded`
- 等主业务容器出现
- 等 loading / mask / skeleton 消失
- 必要时检查关键接口是否已完成

### `wait_actionable`

职责：

- 按钮存在
- 按钮可见
- 按钮未禁用
- 按钮未被遮挡

### `click_and_verify`

职责：

- 点击按钮
- 等待成功信号

成功信号可为：

- URL 变化
- 弹窗出现/关闭
- 成功提示出现
- 下一个表单步骤出现
- 指定请求发出并返回成功

### 错误处理策略

- 每一步最多重试 2 到 3 次
- 每次失败都保存：
  - 截图
  - 当前 URL
  - 页面标题
  - HTML dump
  - 最近错误信息
- 超过阈值后转 `needs_manual_review`

## 支付与订阅链接回填

### 自动流程推进到支付前

自动化脚本完成：

- 注册账号
- 创建订单

然后任务状态变为：

- `pending_payment`

### 人工支付

后台用户在外部站点完成支付。

### 支付完成后的继续处理

后台新增操作：

- `标记已支付并继续抓取`

触发后：

1. 任务状态改为 `fetching_subscribe`
2. 再次启动 Python 脚本
3. 脚本获取登录态后请求：
   - `GET /api/v1/user/getSubscribe`
4. 读取 `data.subscribe_url`
5. 成功后：
   - 更新 `purchase_tasks.subscribe_url`
   - 更新 `redeem_items.content = subscribe_url`
   - 状态改为 `ready`

## 人工补录设计

后台提供：

- 手动填写 `subscribe_url`

操作结果：

1. 更新 `purchase_tasks.subscribe_url`
2. 更新 `redeem_items.content`
3. 任务状态改为 `manual_completed`

适用场景：

- 自动化失败
- 支付完成但自动抓取失败
- 外部站点页面改动

## 前台兑换逻辑变更

当前逻辑是：

- 命中未使用 CDK 后直接核销并返回 `redeem_items.content`

改造后逻辑：

1. 先查 CDK
2. 再查关联 `purchase_task`
3. 只有当任务状态为：
   - `ready`
   - `manual_completed`
   时，才允许核销并返回内容
4. 如果任务状态为：
   - `pending`
   - `registering`
   - `ordering`
   - `pending_payment`
   - `fetching_subscribe`
   - `needs_manual_review`
   则不核销，直接返回“内容准备中”或“等待处理”

这样可以避免前台兑换掉一条尚未准备完成的内容。

## 后台页面改造

### 模板管理页

新增字段：

- 固定购买目标编码
- 固定购买目标名称
- 提供商

### 用户设置

新增字段：

- 外部账号名前缀

### 兑换内容 / CDK 管理页

新增展示：

- 采购任务状态
- 外部账号名
- 支付状态

### 采购任务管理页

新增独立页面：

- 列表展示任务状态
- 支持筛选：
  - 模板
  - 团队
  - 状态
  - 支付状态
- 支持操作：
  - 重试自动化
  - 标记已支付并继续抓取
  - 手动补录 `subscribe_url`
  - 查看错误信息
  - 查看截图和诊断文件

## 并发与一致性

### 序号分配

对 `team_template_sequences` 的读写必须放在事务内，并对对应行加锁，避免并发生成时重复序号。

### 任务执行互斥

同一 `purchase_task` 在任意时刻只能有一个 worker 在执行。第一版可通过数据库状态原子更新保证：

- 只有从预期前置状态成功更新到运行状态的 worker 才能继续执行

## 迁移方案

建议新增迁移：

- `0006_purchase_tasks.up.sql`
- `0006_purchase_tasks.down.sql`

内容包括：

1. `users` 增加 `external_account_prefix`
2. `redeem_templates` 增加外部采购配置字段
3. 新建 `team_template_sequences`
4. 新建 `purchase_tasks`

历史数据兼容策略：

- 历史 `redeem_items` 没有采购任务的，视为旧模式内容，不强制补任务
- 前台兑换时，如果某条 `redeem_item` 没有关联 `purchase_task`，仍按旧逻辑直接兑换

这样能兼容已有库存数据。

## 测试策略

### Go 测试

重点补充服务层测试：

1. 团队模板序号按团队+模板维度递增
2. 生成兑换码时自动创建 `purchase_task`
3. 未 `ready` 的任务不允许前台兑换成功
4. `manual_completed` 允许兑换
5. 历史无采购任务数据仍可兑换

### Python 测试

至少覆盖：

1. `wait_page_ready`
2. `wait_actionable`
3. `click_and_verify`
4. JSON 输出协议

### 集成验证

本地联调至少验证：

1. 后台生成兑换码 -> 创建采购任务
2. 手动将任务推进到 `pending_payment`
3. 后台手动补录 `subscribe_url`
4. 前台兑换成功返回最终链接

## 风险与后续演进

### 第一版风险

- 外部站点页面结构变化会影响自动化稳定性
- 本机子进程模式不适合高并发
- 浏览器环境依赖较重

### 后续演进方向

1. 将 Python 自动化拆成独立服务
2. 将任务执行改成真正队列
3. 为外部站点适配做插件化抽象

## 实施建议

按以下顺序实现：

1. 数据迁移与模型扩展
2. 后台任务模型与状态机
3. 前台兑换状态校验
4. 后台采购任务页面
5. Python 自动化脚手架和页面等待封装
6. 手动支付后的回填流程
