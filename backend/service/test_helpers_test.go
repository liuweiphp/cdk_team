package service

import (
	"exchange_cdk/model"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			status TEXT NOT NULL DEFAULT 'active',
			last_login_at DATETIME NULL,
			created_at DATETIME NULL,
			updated_at DATETIME NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			owner_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME NULL,
			updated_at DATETIME NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE team_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL,
			member_id INTEGER NOT NULL,
			created_at DATETIME NULL
		)`,
		`CREATE TABLE redeem_categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_by INTEGER NOT NULL,
			created_at DATETIME NULL,
			updated_at DATETIME NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE redeem_templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			content TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_by INTEGER NOT NULL,
			created_at DATETIME NULL,
			updated_at DATETIME NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE redeem_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			filename TEXT NOT NULL,
			content TEXT NOT NULL,
			category_id INTEGER NULL,
			template_id INTEGER NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_by INTEGER NOT NULL,
			created_at DATETIME NULL,
			updated_at DATETIME NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE cdk_imports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			item_id INTEGER NULL,
			total INTEGER NOT NULL DEFAULT 0,
			inserted INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0,
			invalid INTEGER NOT NULL DEFAULT 0,
			remark TEXT NOT NULL DEFAULT '',
			created_by INTEGER NOT NULL,
			created_at DATETIME NULL
		)`,
		`CREATE TABLE cdks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL,
			amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			item_id INTEGER NULL,
			status TEXT NOT NULL DEFAULT 'unused',
			import_id INTEGER NOT NULL,
			exchanged_by INTEGER NULL,
			exchanged_at DATETIME NULL,
			created_at DATETIME NULL,
			updated_at DATETIME NULL
		)`,
	}
	for _, stmt := range schema {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("migrate test db failed: %v", err)
		}
	}

	if err := db.Exec("CREATE UNIQUE INDEX idx_users_username ON users(username)").Error; err != nil {
		t.Fatalf("create users index failed: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX idx_cdks_code ON cdks(code)").Error; err != nil {
		t.Fatalf("create cdk code index failed: %v", err)
	}

	return db
}

func createTestUser(t *testing.T, db *gorm.DB, username string) model.User {
	t.Helper()

	user := model.User{
		Username:     username,
		PasswordHash: "hash",
		Role:         "admin",
		Status:       "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return user
}
