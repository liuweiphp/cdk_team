# User File Prefix Sequence Design

## Goal

每个用户可以配置生成文件名前缀。批量导入兑换内容时，生成文件名统一从 `1001` 开始递增，例如前缀为 `x` 时生成 `x1001.txt`、`x1002.txt`。

## Current State

批量导入兑换内容时，`RedeemItemService.createLineWithCode` 目前通过 `buildGeneratedItemName(sourceLine, categoryID, itemID)` 从导入行内容提取账号片段，再拼接分类 ID 和兑换内容 ID 生成名称与文件名。

用户表当前没有生成文件名前缀或序号字段。前端用户管理只支持角色、状态、密码；用户本人只支持修改密码。

## Data Model

`users` 表新增两个字段：

- `file_prefix VARCHAR(64) NOT NULL DEFAULT ''`
- `file_sequence_next INT UNSIGNED NOT NULL DEFAULT 1001`

`model.User` 增加对应 JSON 字段：

- `file_prefix`
- `file_sequence_next`

测试 SQLite schema 同步增加这两个字段。

## Filename Rules

批量导入生成兑换内容时使用当前导入用户的配置：

- 文件名格式：`{file_prefix}{sequence}.txt`
- 序号从当前用户 `file_sequence_next` 开始。
- 每成功创建一条兑换内容，序号递增 1。
- 导入结束后，用户的 `file_sequence_next` 更新为下一个可用序号。
- 只对批量导入生效，包括文本导入和文件导入。
- 手动新增或编辑兑换内容仍保留手动文件名逻辑。

如果用户前缀为空，生成结果为 `1001.txt`、`1002.txt`。

## Prefix Rules

前缀允许为空。非空前缀只允许：

- 英文字母
- 数字
- `-`
- `_`

非法前缀返回 `文件前缀只能包含字母、数字、-、_`。

修改前缀时，`file_sequence_next` 重置为 `1001`。即使新前缀与旧前缀相同，只要调用更新前缀接口，也按用户确认的修改操作重置序号。

## Backend API

管理员更新用户前缀：

- 复用 `PATCH /api/admin/users/:id`
- 请求字段增加 `file_prefix`
- 只要包含该字段，就调用前缀更新逻辑并重置序号为 `1001`

用户本人更新前缀：

- 新增 `PUT /api/user/file-prefix`
- 请求体：`{ "file_prefix": "x" }`
- 更新当前登录用户的前缀，并重置序号为 `1001`

`GET /api/user/me` 和 `GET /api/admin/users` 返回新增字段，用于前端展示当前前缀和下一个序号。

## Frontend UX

管理员入口：

- 在“用户管理”的新增/编辑弹窗中增加“文件前缀”输入框。
- 编辑用户时展示当前前缀。
- 保存前，如果文件前缀发生变化或用户明确提交前缀字段，弹确认框：
  `修改生成文件前缀后，后续生成文件序号将从 1001 重新开始，确认修改？`
- 确认后提交 `file_prefix`。

用户本人入口：

- 在后台侧边栏增加“生成设置”入口。
- 弹窗展示“文件前缀”输入框和当前“下一个序号”。
- 保存前弹同样确认框。
- 保存成功后更新本地 `localStorage.user` 中的 `file_prefix` 和 `file_sequence_next`。

## Import Behavior

导入时在数据库事务内分配文件名和递增序号，避免同一用户重复生成同名文件。

实现策略：

- `importLinesFromReader` 加载当前用户记录。
- 每成功创建一条兑换内容前，计算 `generatedName := fmt.Sprintf("%s%d", user.FilePrefix, nextSequence)`。
- 成功创建后 `nextSequence++`。
- 导入循环结束后保存 `users.file_sequence_next = nextSequence`。

如果单行创建失败，该行不消耗序号。

## Testing

后端服务测试：

- 用户前缀 `x` 且序号默认 `1001` 时，导入两行生成 `x1001.txt`、`x1002.txt`，用户下一个序号为 `1003`。
- 空前缀生成 `1001.txt`。
- 更新前缀后序号重置为 `1001`。
- 非法前缀被拒绝。

前端验证：

- `npm run build`
- 浏览器确认用户管理出现文件前缀输入框，后台侧边栏出现生成设置入口。

## Scope

本设计不处理历史文件名重命名，不保证重置前缀后与历史文件名全局唯一，不改变 CDK 生成规则，不改变模板渲染规则。
