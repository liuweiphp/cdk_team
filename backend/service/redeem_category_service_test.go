package service

import "testing"

func TestRedeemCategoryListFiltersByStatusAndAccessibleOwners(t *testing.T) {
	db := openTestDB(t)
	owner := seedTestUser(t, db, "owner")
	other := seedTestUser(t, db, "other")
	svc := NewRedeemCategoryService(db)

	active, err := svc.Create("可用分类", owner.ID)
	if err != nil {
		t.Fatalf("create active category: %v", err)
	}
	disabled, err := svc.Create("停用分类", owner.ID)
	if err != nil {
		t.Fatalf("create disabled category: %v", err)
	}
	if err := svc.Update(disabled.ID, disabled.Name, "disabled", owner.ID); err != nil {
		t.Fatalf("disable category: %v", err)
	}
	if _, err := svc.Create("他人分类", other.ID); err != nil {
		t.Fatalf("create other category: %v", err)
	}

	list, total, err := svc.List(1, 100, "", "active", owner.ID)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != active.ID {
		t.Fatalf("unexpected list result: total=%d list=%+v", total, list)
	}
}

func TestRedeemCategoryUpdateRequiresOwner(t *testing.T) {
	db := openTestDB(t)
	owner := seedTestUser(t, db, "owner")
	other := seedTestUser(t, db, "other")
	svc := NewRedeemCategoryService(db)

	category, err := svc.Create("原分类", owner.ID)
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	err = svc.Update(category.ID, "越权修改", "active", other.ID)
	if err == nil || err.Error() != "无权操作该分类或分类不存在" {
		t.Fatalf("unexpected err: %v", err)
	}
}
