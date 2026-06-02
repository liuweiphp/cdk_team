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

func TestRedeemItemServiceCreateLineWithCodeUsesCategoryNamingRule(t *testing.T) {
	db := newTestDB(t)
	owner := createTestUser(t, db, "owner")

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

	importRecord := model.CdkImport{
		Filename:  "source.txt",
		Amount:    0,
		Total:     1,
		Inserted:  1,
		Remark:    "测试",
		CreatedBy: owner.ID,
	}
	if err := db.Create(&importRecord).Error; err != nil {
		t.Fatalf("create import failed: %v", err)
	}

	svc := NewRedeemItemService(db)
	item, err := svc.createLineWithCode("source.txt", 1, "acct001,secret", "内容 acct001", tpl.ID, category.ID, importRecord.ID, "ABCDEFGH23456789", owner.ID)
	if err != nil {
		t.Fatalf("create line item failed: %v", err)
	}

	if item.Name != "acct001"+itoa(category.ID)+itoa(item.ID) {
		t.Fatalf("unexpected item name: %s", item.Name)
	}
	if item.Filename != item.Name+".txt" {
		t.Fatalf("unexpected filename: %s", item.Filename)
	}
	if item.CategoryID == nil || *item.CategoryID != category.ID {
		t.Fatalf("expected category id %d, got %#v", category.ID, item.CategoryID)
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

func itoa(v uint) string {
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for v > 0 {
		buf = append([]byte{byte('0' + v%10)}, buf...)
		v /= 10
	}
	return string(buf)
}
