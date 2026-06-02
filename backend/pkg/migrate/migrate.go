package migrate

import (
	"embed"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
)

// Run 用嵌入的迁移文件对目标 DSN 执行 migrate up。
// 已是最新版本则不报错;启动时调用一次,跑完连接立即关闭。
func Run(dsn string, fs embed.FS, subdir string, logger *zap.Logger) error {
	srcDSN, err := ensureMultiStatements(dsn)
	if err != nil {
		return fmt.Errorf("解析 DSN: %w", err)
	}

	srcDriver, err := iofs.New(fs, subdir)
	if err != nil {
		return fmt.Errorf("加载嵌入迁移: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", srcDriver, "mysql://"+srcDSN)
	if err != nil {
		return fmt.Errorf("初始化 migrate: %w", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			logger.Warn("关闭 migrate source", zap.Error(sourceErr))
		}
		if dbErr != nil {
			logger.Warn("关闭 migrate db", zap.Error(dbErr))
		}
	}()

	before, _, _ := m.Version()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("数据库已是最新版本", zap.Uint("version", before))
			return nil
		}
		return fmt.Errorf("执行迁移: %w", err)
	}

	after, _, _ := m.Version()
	logger.Info("数据库迁移完成", zap.Uint("from", before), zap.Uint("to", after))
	return nil
}

// ensureMultiStatements 给 DSN 添加 multiStatements=true。
// golang-migrate 把整个 .sql 文件作为单次 Exec 调用,需要 MySQL 允许多 statement。
func ensureMultiStatements(dsn string) (string, error) {
	idx := strings.LastIndex(dsn, "?")
	if idx < 0 {
		return dsn + "?multiStatements=true", nil
	}
	base, query := dsn[:idx], dsn[idx+1:]
	values, err := url.ParseQuery(query)
	if err != nil {
		return "", err
	}
	values.Set("multiStatements", "true")
	return base + "?" + values.Encode(), nil
}
