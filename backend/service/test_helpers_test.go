package service

import (
	"exchange_cdk/model"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db failed: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.Team{},
		&model.TeamMember{},
		&model.RedeemCategory{},
		&model.RedeemTemplate{},
		&model.RedeemItem{},
		&model.CdkImport{},
		&model.Cdk{},
	); err != nil {
		t.Fatalf("migrate test db failed: %v", err)
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
