# CDK Delete Design

## Goal

CDK 管理页需要支持删除 CDK。删除只允许作用于当前用户自己归属的 `unused` CDK，不允许删除已领取 CDK，也不删除关联兑换内容。

## Current State

后端当前只有 `GET /api/admin/cdk/list`。`CdkService.List` 会列出当前用户可见的 CDK，包括团队共享数据。前端 `CdkManageView.vue` 表格操作列只有“下载”按钮。

## Backend Design

新增接口：

- `DELETE /api/admin/cdk/:id`

删除规则：

- 只能删除当前用户自己归属的 CDK。
- 团队共享可见但不归属当前用户的 CDK 不能删除。
- `status != unused` 时返回 `已领取的 CDK 不能删除`。
- 找不到或无权删除时返回 `无权操作该 CDK 或 CDK 不存在`。
- 删除只删除 `cdks` 记录，不删除 `redeem_items` 或 `cdk_imports`。

归属判断沿用列表的来源逻辑，但删除时只接受 `currentUserID`：

- 如果 CDK 关联兑换内容，则 `redeem_items.created_by` 必须等于当前用户。
- 如果 CDK 没有关联兑换内容，则 `cdk_imports.created_by` 必须等于当前用户。

## Frontend Design

`CDK 管理` 表格操作列增加删除按钮：

- `unused` 状态显示“删除”按钮。
- 非 `unused` 状态不显示删除按钮。
- 点击删除弹确认框。
- 确认后调用删除接口并刷新列表。

下载按钮保持原行为。

## Testing

后端服务层新增测试：

- 当前用户可以删除自己未使用的 CDK。
- 已领取 CDK 不能删除，返回 `已领取的 CDK 不能删除`。
- 团队成员可以看到团队 owner 的 CDK，但不能删除 owner 的 CDK。

前端通过 `npm run build` 验证类型和模板编译。运行环境通过 Docker 重建后用 HTTP 探活确认服务可用。

## Scope

本设计不做批量删除、不做软删除、不调整统计口径、不删除兑换内容、不删除导入批次。
