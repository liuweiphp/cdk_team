package service

import (
	"testing"

	"exchange_cdk/model"
)

func TestUserServiceUpdateFilePrefixResetsSequence(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "owner")
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"file_prefix":        "old",
		"file_sequence_next": 1030,
	}).Error; err != nil {
		t.Fatalf("seed user prefix failed: %v", err)
	}

	updated, err := NewUserService(db, 12).UpdateFilePrefix(user.ID, "x_1")
	if err != nil {
		t.Fatalf("update prefix failed: %v", err)
	}
	if updated.FilePrefix != "x_1" {
		t.Fatalf("expected prefix x_1, got %s", updated.FilePrefix)
	}
	if updated.FileSequenceNext != 1001 {
		t.Fatalf("expected sequence reset to 1001, got %d", updated.FileSequenceNext)
	}
}

func TestUserServiceUpdateFilePrefixRejectsInvalidPrefix(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "owner")

	_, err := NewUserService(db, 12).UpdateFilePrefix(user.ID, "中文")
	if err == nil || err.Error() != "文件前缀只能包含字母、数字、-、_" {
		t.Fatalf("expected invalid prefix error, got %v", err)
	}
}
