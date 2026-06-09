package service

import (
	"testing"

	"exchange_cdk/model"
)

func TestRedeemItemServiceCreateRequiresCategory(t *testing.T) {
	db := newTestDB(t)
	owner := createTestUser(t, db, "owner")
	svc := NewRedeemItemService(db)

	_, err := svc.Create("资料", "guide", "content", 0, owner.ID)
	if err == nil || err.Error() != "请选择分类" {
		t.Fatalf("expected category required error, got %v", err)
	}
}

func TestRedeemItemServiceImportTextUsesUserFilePrefixSequence(t *testing.T) {
	db := newTestDB(t)
	owner := createTestUser(t, db, "owner")
	if err := db.Model(&model.User{}).Where("id = ?", owner.ID).Updates(map[string]interface{}{
		"file_prefix":        "x",
		"file_sequence_next": 1001,
	}).Error; err != nil {
		t.Fatalf("update user prefix failed: %v", err)
	}

	categorySvc := NewRedeemCategoryService(db)
	category, err := categorySvc.Create("账号类", owner.ID)
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	templateSvc := NewTemplateService(db)
	tpl, err := templateSvc.Create("模板", "内容 {{content}}", owner.ID)
	if err != nil {
		t.Fatalf("create template failed: %v", err)
	}

	result, err := NewRedeemItemService(db).ImportText("acct001,secret\nacct002,secret", tpl.ID, category.ID, owner.ID)
	if err != nil {
		t.Fatalf("import text failed: %v", err)
	}
	if result.Total != 2 || result.Inserted != 2 {
		t.Fatalf("unexpected import result: total=%d inserted=%d", result.Total, result.Inserted)
	}

	var items []model.RedeemItem
	if err := db.Where("created_by = ?", owner.ID).Order("id ASC").Find(&items).Error; err != nil {
		t.Fatalf("list redeem items failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Name != "x1001" || items[0].Filename != "x1001.txt" {
		t.Fatalf("unexpected first generated file: name=%s filename=%s", items[0].Name, items[0].Filename)
	}
	if items[1].Name != "x1002" || items[1].Filename != "x1002.txt" {
		t.Fatalf("unexpected second generated file: name=%s filename=%s", items[1].Name, items[1].Filename)
	}

	var updated model.User
	if err := db.First(&updated, owner.ID).Error; err != nil {
		t.Fatalf("reload user failed: %v", err)
	}
	if updated.FileSequenceNext != 1003 {
		t.Fatalf("expected next sequence 1003, got %d", updated.FileSequenceNext)
	}
}

func TestRedeemItemServiceImportTextUsesEmptyPrefixSequence(t *testing.T) {
	db := newTestDB(t)
	owner := createTestUser(t, db, "owner")
	category, err := NewRedeemCategoryService(db).Create("账号类", owner.ID)
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	tpl, err := NewTemplateService(db).Create("模板", "内容 {{content}}", owner.ID)
	if err != nil {
		t.Fatalf("create template failed: %v", err)
	}

	result, err := NewRedeemItemService(db).ImportText("acct001", tpl.ID, category.ID, owner.ID)
	if err != nil {
		t.Fatalf("import text failed: %v", err)
	}
	if result.Total != 1 || result.Inserted != 1 || len(result.Codes) != 1 {
		t.Fatalf("unexpected result: total=%d inserted=%d codes=%d", result.Total, result.Inserted, len(result.Codes))
	}

	var item model.RedeemItem
	if err := db.Where("created_by = ?", owner.ID).First(&item).Error; err != nil {
		t.Fatalf("load redeem item failed: %v", err)
	}
	if item.Name != "1001" || item.Filename != "1001.txt" {
		t.Fatalf("unexpected generated file: name=%s filename=%s", item.Name, item.Filename)
	}
}

func TestRedeemItemServiceImportTextRejectsBlankText(t *testing.T) {
	db := newTestDB(t)
	owner := createTestUser(t, db, "owner")
	category, err := NewRedeemCategoryService(db).Create("账号类", owner.ID)
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	tpl, err := NewTemplateService(db).Create("模板", "内容 {{content}}", owner.ID)
	if err != nil {
		t.Fatalf("create template failed: %v", err)
	}

	_, err = NewRedeemItemService(db).ImportText(" \n\t\n", tpl.ID, category.ID, owner.ID)
	if err == nil || err.Error() != "请输入文本内容" {
		t.Fatalf("expected blank text error, got %v", err)
	}
}

func TestCdkServiceListIncludesTeamOwnerSharedItems(t *testing.T) {
	db := newTestDB(t)
	owner := createTestUser(t, db, "owner")
	member := createTestUser(t, db, "member")

	teamSvc := NewTeamService(db)
	team, err := teamSvc.EnsureOwnerTeam(owner.ID)
	if err != nil {
		t.Fatalf("ensure owner team failed: %v", err)
	}
	if err := db.Create(&model.TeamMember{TeamID: team.ID, MemberID: member.ID}).Error; err != nil {
		t.Fatalf("create team member failed: %v", err)
	}

	category, err := NewRedeemCategoryService(db).Create("账号分类", owner.ID)
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	itemSvc := NewRedeemItemService(db)
	item, err := itemSvc.Create("资料A", "a", "shared-content", category.ID, owner.ID)
	if err != nil {
		t.Fatalf("create redeem item failed: %v", err)
	}

	list, total, err := NewCdkService(db).List(1, 20, 0, "", "", 0, 0, member.ID)
	if err != nil {
		t.Fatalf("list cdks failed: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected 1 cdk, got total=%d list=%d", total, len(list))
	}
	if list[0].RedeemItem == nil || list[0].RedeemItem.ID != item.ID || list[0].RedeemItem.Content != "shared-content" {
		t.Fatalf("expected shared redeem item content, got %#v", list[0].RedeemItem)
	}
}
