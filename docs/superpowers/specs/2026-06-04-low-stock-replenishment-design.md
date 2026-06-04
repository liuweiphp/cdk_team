# 小库存补货采购模块设计

## 背景

现有兑换系统曾按“提前导入兑换内容”运行，这会导致外部商品提前购买、库存积压。之前讨论过“用户兑换时才自动购买”，但外部网站购买需要人工扫码支付，无法保证用户兑换链路内实时交付。因此第一版采购模块应改为“低库存补货”：后台只保持少量可交付库存，库存不足时生成待支付采购任务，由管理员扫码支付后继续抓取结果。

## 目标

- 避免大量提前购买造成压货。
- 用户兑换时只发放已经准备好的可交付内容。
- 库存不足时不消耗用户 CDK，不创建待交付排队订单。
- 系统根据库存策略生成补货任务，管理员完成扫码支付。
- 支付后系统继续抓取 `subscribe_url`，成功后生成可兑换库存。

## 非目标

- 第一版不自动完成扫码支付。
- 第一版不在用户兑换请求中触发外部购买。
- 第一版不支持缺货预约、排队、通知或延迟交付。
- 第一版不引入分布式队列或独立自动化服务。
- 第一版不扩展 CDK 状态机到复杂业务状态，仍保持现有 `unused/exchanged` 二态。

## 核心业务规则

### 后台库存策略

每个兑换模板或分类配置库存策略：

- `safe_stock`：安全库存阈值。当前可用库存数小于或等于该值时，需要补货。
- `replenish_quantity`：单次补货数量。每次生成多少个采购任务。
- `auto_replenish`：是否自动创建补货任务。

第一版建议策略挂在兑换模板上，因为现有采购任务已经以模板为购买目标，模板也绑定了外部商品编码。

### 库存口径

可用库存定义为：

- `cdks.status = unused`
- 关联的 `redeem_items.status = active`
- 关联 `redeem_items.content` 非空
- 如果有关联采购任务，则采购任务状态必须是 `ready` 或 `manual_completed`
- 按模板维度统计，必要时可附加分类筛选

待入库库存定义为：

- 采购任务状态为 `pending`、`ordering`、`pending_payment`、`fetching_subscribe` 或 `needs_manual_review`
- 尚未生成可用 CDK，或已生成但内容未准备好

库存看板同时展示可用库存和待入库库存，避免管理员重复创建过多待支付任务。

### 用户兑换规则

用户兑换时：

1. 系统根据用户输入的 CDK 找到对应可兑换内容。
2. 如果内容可用且库存满足发放条件，按现有事务逻辑完成兑换。
3. 如果没有可用库存，返回缺货错误：

```json
{
  "code": 40001,
  "message": "当前商品暂时缺货，请稍后再试",
  "data": null
}
```

缺货时不能更新 `cdks.status`，不能创建兑换订单，不能创建用户待交付任务。

## 采购补货流程

### 自动检查

系统在以下时机检查库存：

- 后台进入库存/采购页面时手动触发。
- 管理员点击“检查库存”按钮。
- 可选：后端启动一个本地定时任务，按固定间隔检查启用 `auto_replenish` 的模板。

第一版优先实现手动触发，定时任务作为后续增强，避免启动后自动创建大量待支付任务。

### 创建补货任务

当模板满足：

- `auto_replenish = true` 或管理员手动点击补货
- `ready_stock <= safe_stock`
- 当前待入库库存不足以覆盖目标库存

系统创建 `replenish_quantity` 条采购任务。

采购任务创建后进入 `pending`，管理员可点击“开始处理”，自动化脚本调用外部网站下单。下单成功后任务进入 `pending_payment`，等待管理员扫码支付。

### 支付后入库

管理员扫码支付后点击“已支付继续抓取”：

1. 自动化脚本调用外部站点接口抓取 `subscribe_url`。
2. 抓取成功后，系统创建 `redeem_item` 和 `cdk`。
3. `redeem_item.content` 写入 `subscribe_url`。
4. 采购任务绑定新创建的 `redeem_item_id` 和 `cdk_id`。
5. 采购任务状态改为 `ready`。

如果自动化失败，任务进入 `needs_manual_review`。管理员可以人工补录 `subscribe_url`，补录成功后任务进入 `manual_completed`，并同样生成可用库存。

## 数据模型调整

### 模板库存策略

在 `redeem_templates` 增加：

- `safe_stock INT NOT NULL DEFAULT 0`
- `replenish_quantity INT NOT NULL DEFAULT 1`
- `auto_replenish TINYINT(1) NOT NULL DEFAULT 0`

校验规则：

- `safe_stock >= 0`
- `replenish_quantity >= 1`
- `replenish_quantity <= 20`，防止误操作一次生成过多待支付任务

### 采购任务来源

在 `purchase_tasks` 增加：

- `source VARCHAR(32) NOT NULL DEFAULT 'manual'`

取值：

- `manual`：管理员手动创建
- `replenishment`：库存补货创建

### 采购任务绑定

沿用现有 `purchase_tasks.redeem_item_id` 和 `cdk_id` 可为空的模型。补货任务创建时不生成兑换内容和 CDK，只有支付并抓取结果成功后才生成。

## 接口设计

### 获取库存概览

`GET /api/admin/inventory/templates`

查询参数：

- `page`
- `page_size`
- `keyword`
- `status`

返回字段：

- `template_id`
- `template_name`
- `target_code`
- `safe_stock`
- `replenish_quantity`
- `auto_replenish`
- `ready_stock`
- `incoming_stock`
- `needs_replenishment`

### 更新库存策略

`PUT /api/admin/templates/:id/inventory-policy`

请求：

```json
{
  "safe_stock": 3,
  "replenish_quantity": 2,
  "auto_replenish": true
}
```

### 手动检查并补货

`POST /api/admin/templates/:id/replenish`

行为：

- 根据当前库存和策略计算需要创建的采购任务数量。
- 创建任务后返回任务列表。
- 如果不需要补货，返回空列表和说明。

### 采购任务列表

沿用现有 `GET /api/admin/purchase-tasks`，增加 `source` 字段展示和筛选。

## 后台页面设计

### 模板管理

在模板编辑弹窗增加库存策略字段：

- 安全库存
- 单次补货数量
- 自动补货开关

### 采购任务页

增加：

- 来源列：手动 / 补货
- 模板当前库存提示
- “按策略补货”按钮

### 库存看板

第一版可以放在采购任务页顶部，不单独建页面：

- 模板名称
- 可用库存
- 待入库库存
- 安全库存
- 是否需要补货
- 操作：检查 / 补货

## 错误处理

- 外部下单失败：任务进入 `needs_manual_review`，记录 `last_error` 和 `manual_review_reason`。
- 支付后抓取失败：任务进入 `needs_manual_review`，允许管理员人工补录。
- 人工补录内容为空：返回参数错误。
- 库存补货重复触发：创建任务前重新计算 `ready_stock + incoming_stock`，避免重复生成。
- 模板没有外部购买目标：禁止补货，返回“模板未配置购买目标”。

## 测试策略

### 服务测试

- 库存统计只计算 `unused + active + content 非空` 的 CDK。
- 缺货兑换不消耗 CDK，不创建订单。
- 库存低于安全阈值时创建正确数量的补货任务。
- 待入库库存参与计算，避免重复补货。
- 支付抓取成功后才创建 `redeem_item` 和 `cdk`。
- 人工补录成功后生成可用库存。

### 接口测试

- 库存策略更新校验边界值。
- 未登录访问库存接口返回 401。
- 非拥有者不能修改模板库存策略。
- 手动补货接口返回创建的采购任务。

### 前端验证

- 模板编辑能保存库存策略。
- 采购任务页能看到库存概览。
- 缺货兑换时显示“当前商品暂时缺货，请稍后再试”。
- 支付后继续抓取成功，库存数量增加。

## 实施分期

### 第一阶段：库存策略和统计

- 模板增加库存策略字段。
- 后端实现库存统计服务和接口。
- 后台展示每个模板的可用库存和待入库库存。

### 第二阶段：按策略补货

- 实现手动补货接口。
- 补货创建 `source = replenishment` 的采购任务。
- 创建任务时不生成 `redeem_item` 和 `cdk`。

### 第三阶段：支付后入库

- 修改采购任务完成逻辑。
- 抓取或人工补录 `subscribe_url` 后生成 `redeem_item + cdk`。
- 任务绑定生成的库存并进入可兑换状态。

### 第四阶段：兑换缺货保护

- 兑换接口增加库存可用性判断。
- 缺货时不消耗 CDK，不创建订单。
- 前端显示缺货提示。

## 验收标准

- 管理员可以为模板配置安全库存和单次补货数量。
- 当库存不足时，管理员可以一键生成待支付采购任务。
- 采购任务支付并抓取结果后，系统生成可兑换库存。
- 用户兑换时如果没有现货库存，CDK 状态保持不变。
- 用户兑换时如果有现货库存，仍按现有流程立即返回内容。
- 所有新增后端服务测试通过。
