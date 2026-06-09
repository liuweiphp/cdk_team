package service

import (
	"testing"

	"exchange_cdk/model"
)

func TestCdkServiceDeleteRemovesOwnUnusedCdkOnly(t *testing.T) {
	db := newTestDB(t)
	owner := createTestUser(t, db, "owner")
	category, err := NewRedeemCategoryService(db).Create("账号类", owner.ID)
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	item, err := NewRedeemItemService(db).Create("资料", "guide", "content", category.ID, owner.ID)
	if err != nil {
		t.Fatalf("create redeem item failed: %v", err)
	}

	var cdk model.Cdk
	if err := db.Where("item_id = ?", item.ID).First(&cdk).Error; err != nil {
		t.Fatalf("load cdk failed: %v", err)
	}

	if err := NewCdkService(db).Delete(cdk.ID, owner.ID); err != nil {
		t.Fatalf("delete cdk failed: %v", err)
	}

	var cdkCount int64
	if err := db.Model(&model.Cdk{}).Where("id = ?", cdk.ID).Count(&cdkCount).Error; err != nil {
		t.Fatalf("count cdk failed: %v", err)
	}
	if cdkCount != 0 {
		t.Fatalf("expected cdk deleted, got count=%d", cdkCount)
	}

	var itemCount int64
	if err := db.Model(&model.RedeemItem{}).Where("id = ?", item.ID).Count(&itemCount).Error; err != nil {
		t.Fatalf("count redeem item failed: %v", err)
	}
	if itemCount != 1 {
		t.Fatalf("expected redeem item to remain, got count=%d", itemCount)
	}
}

func TestCdkServiceDeleteRejectsExchangedCdk(t *testing.T) {
	db := newTestDB(t)
	owner := createTestUser(t, db, "owner")
	category, err := NewRedeemCategoryService(db).Create("账号类", owner.ID)
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	item, err := NewRedeemItemService(db).Create("资料", "guide", "content", category.ID, owner.ID)
	if err != nil {
		t.Fatalf("create redeem item failed: %v", err)
	}

	var cdk model.Cdk
	if err := db.Where("item_id = ?", item.ID).First(&cdk).Error; err != nil {
		t.Fatalf("load cdk failed: %v", err)
	}
	if err := db.Model(&model.Cdk{}).Where("id = ?", cdk.ID).Update("status", "exchanged").Error; err != nil {
		t.Fatalf("mark cdk exchanged failed: %v", err)
	}

	err = NewCdkService(db).Delete(cdk.ID, owner.ID)
	if err == nil || err.Error() != "已领取的 CDK 不能删除" {
		t.Fatalf("expected exchanged cdk error, got %v", err)
	}
}

func TestCdkServiceDeleteRejectsTeamSharedCdk(t *testing.T) {
	db := newTestDB(t)
	owner := createTestUser(t, db, "owner")
	member := createTestUser(t, db, "member")
	team, err := NewTeamService(db).EnsureOwnerTeam(owner.ID)
	if err != nil {
		t.Fatalf("ensure owner team failed: %v", err)
	}
	if err := db.Create(&model.TeamMember{TeamID: team.ID, MemberID: member.ID}).Error; err != nil {
		t.Fatalf("create team member failed: %v", err)
	}
	category, err := NewRedeemCategoryService(db).Create("账号类", owner.ID)
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	item, err := NewRedeemItemService(db).Create("资料", "guide", "content", category.ID, owner.ID)
	if err != nil {
		t.Fatalf("create redeem item failed: %v", err)
	}

	list, total, err := NewCdkService(db).List(1, 20, 0, "", "", 0, 0, member.ID)
	if err != nil {
		t.Fatalf("list shared cdks failed: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected member to see shared cdk, got total=%d list=%d", total, len(list))
	}

	err = NewCdkService(db).Delete(list[0].ID, member.ID)
	if err == nil || err.Error() != "无权操作该 CDK 或 CDK 不存在" {
		t.Fatalf("expected no permission error, got %v", err)
	}

	var cdkCount int64
	if err := db.Model(&model.Cdk{}).Where("item_id = ?", item.ID).Count(&cdkCount).Error; err != nil {
		t.Fatalf("count cdk failed: %v", err)
	}
	if cdkCount != 1 {
		t.Fatalf("expected cdk to remain, got count=%d", cdkCount)
	}
}
