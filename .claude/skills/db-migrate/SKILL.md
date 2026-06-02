---
name: db-migrate
description: 数据库迁移辅助 —— 主要用于回滚(down)、新建迁移文件、查看版本。**up 已在后端启动时自动执行**,正常工作流不用手动跑。当用户说"回滚迁移""新建迁移""查看迁移版本"时使用。$ARGUMENTS 可以是 "down" / "new <name>" / "status" / "force <version>"。
disable-model-invocation: true
---

# /db-migrate — golang-migrate 辅助操作

迁移文件在 `backend/migration/`,格式 `NNNN_name.up.sql` + `NNNN_name.down.sql`。

**重要**:up 由 backend 启动时自动执行(`main.go` → `pkg/migrate.Run` → 嵌入的 SQL),用户不需要也不应该手动跑 up。本 skill 处理 up 之外的运维场景。

读取 `$ARGUMENTS` 决定动作:

## down

```bash
cd /Users/shuo/go/src/exchange_cdk/backend
migrate -path migration -database "${DB_DSN}&multiStatements=true" down 1
```

**先问用户**要回滚几步,默认 1 步。回滚不可逆(数据可能丢),操作前**明确告知用户哪几个 down.sql 会跑、影响哪些表**,等用户确认。

如果本地没装 migrate CLI:

```bash
brew install golang-migrate
```

## new <name>

1. 找当前最大序号:

   ```bash
   ls /Users/shuo/go/src/exchange_cdk/backend/migration | grep -oE '^[0-9]+' | sort -n | tail -1
   ```

2. 用序号 +1(4 位补零),创建两个文件:

   - `NNNN_<name>.up.sql`
   - `NNNN_<name>.down.sql`

3. up 写正向 DDL,down 写**完整反向** DDL(`DROP COLUMN`、`DROP INDEX`、`DROP TABLE` 等),不能留空。
4. 如果是给现有大表加列,提醒用户考虑 `ALGORITHM=INPLACE, LOCK=NONE`(MySQL 8 InnoDB)。
5. 写完提示用户:**重启 backend 即生效**(本地 `go run` 重跑;docker 模式需重 build 镜像因为 SQL 是 embed 的)。

## status / version

```bash
cd /Users/shuo/go/src/exchange_cdk/backend
migrate -path migration -database "${DB_DSN}&multiStatements=true" version
```

显示当前迁移版本号。

## force <version>

仅在迁移卡在 dirty 状态时用(上次 up 失败留下半成品)。**先问用户**确定数据库状态后再跑:

```bash
migrate -path migration -database "${DB_DSN}&multiStatements=true" force <version>
```

之后让用户重启 backend,自动迁移会接管。

## 规约

- 序号严格递增,不允许复用或跳号
- 迁移文件一旦合入主分支就**不再修改**,要改用新的迁移
- 不要在迁移里塞业务数据写入(seed),仅 DDL 或必须的初始化常量(如默认 admin 账号)
- 修改 SQL 后必须重启 backend(本地)或重 build 镜像(docker),因为 SQL 是编译期 embed
