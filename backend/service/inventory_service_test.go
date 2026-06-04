package service

import (
	"exchange_cdk/model"
	"testing"
)

func TestInventoryServiceCountsOnlyReadyUnusedStock(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "owner")
	tpl := seedTestTemplate(t, db, user.ID, "gptplus", "GPT Plus")
	readyItem := seedTestRedeemItem(t, db, user.ID, "ready", "content", &tpl.ID)
	readyCdk := seedTestCdk(t, db, readyItem.ID)
	_ = seedTestPurchaseTask(t, db, user.ID, readyCdk.ID, readyItem.ID, "ready", "paid", "content")
	emptyItem := seedTestRedeemItem(t, db, user.ID, "empty", "", &tpl.ID)
	_ = seedTestCdk(t, db, emptyItem.ID)

	svc := NewInventoryService(db)
	rows, _, err := svc.ListTemplateInventory(InventoryListInput{Page: 1, PageSize: 20, CurrentUserID: user.ID})
	if err != nil {
		t.Fatalf("list inventory: %v", err)
	}
	if len(rows) != 1 || rows[0].ReadyStock != 1 || rows[0].IncomingStock != 0 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestInventoryPolicyRejectsUnsafeValues(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "owner")
	tpl := seedTestTemplate(t, db, user.ID, "gptplus", "GPT Plus")

	svc := NewInventoryService(db)
	err := svc.UpdatePolicy(tpl.ID, InventoryPolicyInput{SafeStock: -1, ReplenishQuantity: 1}, user.ID)
	if err == nil || err.Error() != "安全库存不能小于 0" {
		t.Fatalf("unexpected err: %v", err)
	}
	err = svc.UpdatePolicy(tpl.ID, InventoryPolicyInput{SafeStock: 1, ReplenishQuantity: 21}, user.ID)
	if err == nil || err.Error() != "单次补货数量必须在 1 到 20 之间" {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestInventoryServiceCountsOnlyAccessibleOwnerStock(t *testing.T) {
	db := openTestDB(t)
	owner := seedTestUser(t, db, "owner")
	outsider := seedTestUser(t, db, "outsider")
	tpl := seedTestTemplate(t, db, owner.ID, "gptplus", "GPT Plus")

	ownerItem := seedTestRedeemItem(t, db, owner.ID, "owner-ready", "content", &tpl.ID)
	ownerCdk := seedTestCdk(t, db, ownerItem.ID)
	_ = seedTestPurchaseTask(t, db, owner.ID, ownerCdk.ID, ownerItem.ID, "ready", "paid", "content")

	outsiderItem := seedTestRedeemItem(t, db, outsider.ID, "outsider-ready", "content", &tpl.ID)
	_ = seedTestCdk(t, db, outsiderItem.ID)
	_ = seedTestPurchaseTask(t, db, outsider.ID, 0, 0, "pending", "unpaid", "")

	rows, _, err := NewInventoryService(db).ListTemplateInventory(InventoryListInput{Page: 1, PageSize: 20, CurrentUserID: owner.ID})
	if err != nil {
		t.Fatalf("list inventory: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one template row, got %+v", rows)
	}
	if rows[0].ReadyStock != 1 || rows[0].IncomingStock != 0 {
		t.Fatalf("expected only owner stock, got %+v", rows[0])
	}
}

func TestReplenishTemplateCreatesOnlyNeededTasks(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "owner")
	tpl := seedTestTemplate(t, db, user.ID, "gptplus", "GPT Plus")
	if err := db.Model(tpl).Updates(map[string]interface{}{"safe_stock": 3, "replenish_quantity": 2}).Error; err != nil {
		t.Fatalf("update policy: %v", err)
	}

	svc := NewInventoryService(db)
	tasks, err := svc.ReplenishTemplate(tpl.ID, user.ID)
	if err != nil {
		t.Fatalf("replenish: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	for _, task := range tasks {
		if task.Source != "replenishment" || task.CdkID != nil || task.RedeemItemID != nil {
			t.Fatalf("unexpected task: %+v", task)
		}
	}

	var itemCount int64
	if err := db.Model(&model.RedeemItem{}).Count(&itemCount).Error; err != nil {
		t.Fatalf("count redeem items: %v", err)
	}
	if itemCount != 0 {
		t.Fatalf("expected no redeem item before replenishment task completion, got %d", itemCount)
	}
	var cdkCount int64
	if err := db.Model(&model.Cdk{}).Count(&cdkCount).Error; err != nil {
		t.Fatalf("count cdks: %v", err)
	}
	if cdkCount != 0 {
		t.Fatalf("expected no cdk before replenishment task completion, got %d", cdkCount)
	}
}

func TestReplenishTemplateIncludesIncomingStockToAvoidDuplicates(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "owner")
	tpl := seedTestTemplate(t, db, user.ID, "gptplus", "GPT Plus")
	if err := db.Model(tpl).Updates(map[string]interface{}{"safe_stock": 1, "replenish_quantity": 2}).Error; err != nil {
		t.Fatalf("update policy: %v", err)
	}
	if _, err := NewPurchaseTaskService(db, nil).CreateForTemplate(tpl.ID, user.ID); err != nil {
		t.Fatalf("seed incoming task: %v", err)
	}

	tasks, err := NewInventoryService(db).ReplenishTemplate(tpl.ID, user.ID)
	if err != nil {
		t.Fatalf("replenish: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected no duplicate tasks, got %+v", tasks)
	}

	rows, _, err := NewInventoryService(db).ListTemplateInventory(InventoryListInput{Page: 1, PageSize: 20, CurrentUserID: user.ID})
	if err != nil {
		t.Fatalf("list inventory: %v", err)
	}
	if len(rows) != 1 || rows[0].NeedsReplenishment {
		t.Fatalf("expected incoming stock to cover replenishment need, got %+v", rows)
	}
}

func TestReplenishTemplateRequiresOwner(t *testing.T) {
	db := openTestDB(t)
	owner := seedTestUser(t, db, "owner")
	other := seedTestUser(t, db, "other")
	tpl := seedTestTemplate(t, db, owner.ID, "gptplus", "GPT Plus")
	if err := db.Model(tpl).Updates(map[string]interface{}{"safe_stock": 3, "replenish_quantity": 2}).Error; err != nil {
		t.Fatalf("update policy: %v", err)
	}

	_, err := NewInventoryService(db).ReplenishTemplate(tpl.ID, other.ID)
	if err == nil || err.Error() != "无权操作该模板或模板不存在" {
		t.Fatalf("unexpected err: %v", err)
	}
}
