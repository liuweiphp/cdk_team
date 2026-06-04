package service

import (
	"errors"
	"exchange_cdk/model"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type purchaseTaskSequenceAllocator interface {
	AllocateNextSequence(teamOwnerID, templateID uint) (uint, error)
}

func newPurchaseTaskSequenceAllocatorForTest() purchaseTaskSequenceAllocator {
	return NewPurchaseTaskService(nil, nil)
}

func lookupPurchaseTaskSequenceAllocatorForTest() purchaseTaskSequenceAllocator {
	return NewPurchaseTaskService(nil, nil)
}

func TestPurchaseTaskSequenceAllocatorPersistsSequencePerTeamOwnerAndTemplate(t *testing.T) {
	allocator := newPurchaseTaskSequenceAllocatorForTest()
	if allocator == nil {
		t.Fatalf("purchase task sequence allocator not implemented: expected persistent sequence allocation per team owner and template")
	}

	sequence1, err := allocator.AllocateNextSequence(10, 20)
	if err != nil {
		t.Fatalf("allocate first sequence: %v", err)
	}
	sequence2, err := allocator.AllocateNextSequence(10, 20)
	if err != nil {
		t.Fatalf("allocate second sequence: %v", err)
	}
	otherTemplateSequence, err := allocator.AllocateNextSequence(10, 21)
	if err != nil {
		t.Fatalf("allocate other template sequence: %v", err)
	}
	otherOwnerSequence, err := allocator.AllocateNextSequence(11, 20)
	if err != nil {
		t.Fatalf("allocate other owner sequence: %v", err)
	}

	first := model.PurchaseTask{TeamOwnerID: 10, TemplateID: 20, SequenceNo: sequence1}
	second := model.PurchaseTask{TeamOwnerID: 10, TemplateID: 20, SequenceNo: sequence2}
	otherTemplate := model.PurchaseTask{TeamOwnerID: 10, TemplateID: 21, SequenceNo: otherTemplateSequence}
	otherOwner := model.PurchaseTask{TeamOwnerID: 11, TemplateID: 20, SequenceNo: otherOwnerSequence}

	if second.SequenceNo != first.SequenceNo+1 {
		t.Fatalf("expected allocator to increment persisted sequence for same team owner/template: first=%d second=%d", first.SequenceNo, second.SequenceNo)
	}
	if otherTemplate.SequenceNo != 1 {
		t.Fatalf("expected allocator to start sequence at 1 for a different template, got %d", otherTemplate.SequenceNo)
	}
	if otherOwner.SequenceNo != 1 {
		t.Fatalf("expected allocator to start sequence at 1 for a different team owner, got %d", otherOwner.SequenceNo)
	}
}

func TestCreateTaskBuildsAccountNameAndPendingStatus(t *testing.T) {
	svc := NewPurchaseTaskService(nil, nil)

	task, err := svc.CreatePendingTask(CreatePendingTaskInput{
		TeamOwnerID:   1,
		TemplateID:    10,
		CreatedBy:     1,
		AccountPrefix: "vip",
		TemplateCode:  "gptplus",
		TargetCode:    "PLAN001",
		TargetName:    "GPT Plus",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if task.Status != "pending" || task.PaymentStatus != "unpaid" {
		t.Fatalf("unexpected task status: %+v", task)
	}
	if task.AccountName != "vip-gptplus-0001" {
		t.Fatalf("unexpected account name: %s", task.AccountName)
	}
	if task.RedeemItemID != nil || task.CdkID != nil {
		t.Fatalf("expected standalone purchase task before completion, got redeem_item_id=%v cdk_id=%v", task.RedeemItemID, task.CdkID)
	}
}

func TestCreateStandalonePurchaseTaskDoesNotCreateRedeemContent(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "vip")
	template := seedTestTemplate(t, db, user.ID, "month_1", "月卡一月")

	svc := NewPurchaseTaskService(db, nil)
	task, err := svc.CreateForTemplate(template.ID, user.ID)
	if err != nil {
		t.Fatalf("create purchase task: %v", err)
	}
	if task.RedeemItemID != nil || task.CdkID != nil {
		t.Fatalf("expected purchase task to be standalone, got redeem_item_id=%v cdk_id=%v", task.RedeemItemID, task.CdkID)
	}

	var itemCount int64
	if err := db.Model(&model.RedeemItem{}).Count(&itemCount).Error; err != nil {
		t.Fatalf("count redeem items: %v", err)
	}
	if itemCount != 0 {
		t.Fatalf("expected no redeem item before task completion, got %d", itemCount)
	}
}

func TestCreateRedeemItemCreatesPurchaseTask(t *testing.T) {
	db := openTestDB(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin test tx: %v", tx.Error)
	}
	t.Cleanup(func() {
		_ = tx.Rollback().Error
	})

	user := seedTestUser(t, tx, "vip")
	template := seedTestTemplate(t, tx, user.ID, "gptplus", "GPT Plus")

	taskSvc := NewPurchaseTaskService(tx, &AutomationRunner{})
	itemSvc := NewRedeemItemService(tx)
	itemSvc.SetPurchaseTaskService(taskSvc)

	item, err := itemSvc.CreateFromTemplate(CreateRedeemItemFromTemplateInput{
		Name:       "GPT Plus 现货",
		Filename:   "gptplus.txt",
		TemplateID: template.ID,
		CreatedBy:  user.ID,
	})
	if err != nil {
		t.Fatalf("create item from template: %v", err)
	}

	if item.Content != "" {
		t.Fatalf("expected empty initial content, got %q", item.Content)
	}

	var task model.PurchaseTask
	if err := tx.Where("redeem_item_id = ?", item.ID).First(&task).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if task.AccountName != "vip-gptplus-0001" {
		t.Fatalf("unexpected account name: %s", task.AccountName)
	}
	if task.TargetCode != "gptplus" {
		t.Fatalf("unexpected target code: %s", task.TargetCode)
	}
	if task.TargetName != "GPT Plus" {
		t.Fatalf("unexpected target name: %s", task.TargetName)
	}
}

func TestRedeemBlockedWhenPurchaseTaskNotReady(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "vip")
	item := seedTestRedeemItem(t, db, user.ID, "待支付内容", "", nil)
	cdk := seedTestCdk(t, db, item.ID)
	seedTestPurchaseTask(t, db, user.ID, cdk.ID, item.ID, "pending_payment", "unpaid", "")

	svc := NewRedeemService(db)
	_, err := svc.RedeemByCode(cdk.Code, "127.0.0.1", "ua")
	if err == nil || err.Error() != "当前商品暂时缺货，请稍后再试" {
		t.Fatalf("expected out-of-stock error, got %v", err)
	}

	var fresh model.Cdk
	if err := db.First(&fresh, cdk.ID).Error; err != nil {
		t.Fatalf("reload cdk: %v", err)
	}
	if fresh.Status != "unused" {
		t.Fatalf("expected cdk to remain unused, got %s", fresh.Status)
	}
}

func TestRedeemBlockedWhenContentEmpty(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "vip")
	item := seedTestRedeemItem(t, db, user.ID, "空内容", "", nil)
	cdk := seedTestCdk(t, db, item.ID)

	svc := NewRedeemService(db)
	_, err := svc.RedeemByCode(cdk.Code, "127.0.0.1", "ua")
	if err == nil || err.Error() != "当前商品暂时缺货，请稍后再试" {
		t.Fatalf("expected out-of-stock error, got %v", err)
	}

	var fresh model.Cdk
	if err := db.First(&fresh, cdk.ID).Error; err != nil {
		t.Fatalf("reload cdk: %v", err)
	}
	if fresh.Status != "unused" {
		t.Fatalf("expected cdk to remain unused, got %s", fresh.Status)
	}
}

func TestExchangeOnlyIssuesReadyContentCdks(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "vip")
	amount := 8.8

	emptyItem := seedTestRedeemItem(t, db, user.ID, "空内容", "", nil)
	emptyCdk := seedTestCdk(t, db, emptyItem.ID)
	if err := db.Model(emptyCdk).Update("amount", amount).Error; err != nil {
		t.Fatalf("set empty cdk amount: %v", err)
	}

	pendingItem := seedTestRedeemItem(t, db, user.ID, "未完成内容", "pending-content", nil)
	pendingCdk := seedTestCdk(t, db, pendingItem.ID)
	if err := db.Model(pendingCdk).Update("amount", amount).Error; err != nil {
		t.Fatalf("set pending cdk amount: %v", err)
	}
	seedTestPurchaseTask(t, db, user.ID, pendingCdk.ID, pendingItem.ID, "pending_payment", "unpaid", "")

	readyItem := seedTestRedeemItem(t, db, user.ID, "可用内容", "ready-content", nil)
	readyCdk := seedTestCdk(t, db, readyItem.ID)
	if err := db.Model(readyCdk).Update("amount", amount).Error; err != nil {
		t.Fatalf("set ready cdk amount: %v", err)
	}
	seedTestPurchaseTask(t, db, user.ID, readyCdk.ID, readyItem.ID, "ready", "paid", "ready-content")

	svc := NewExchangeService(db, 10)
	amounts, err := svc.GetAvailableAmounts()
	if err != nil {
		t.Fatalf("get available amounts: %v", err)
	}
	if len(amounts) != 1 || amounts[0].Amount != amount || amounts[0].Remaining != 1 {
		t.Fatalf("expected only one ready cdk amount, got %+v", amounts)
	}

	result, err := svc.Exchange(user.ID, amount, 1, "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("exchange ready cdk: %v", err)
	}
	if len(result.Codes) != 1 || result.Codes[0] != readyCdk.Code {
		t.Fatalf("expected ready cdk only, got %+v", result.Codes)
	}

	for _, cdk := range []*model.Cdk{emptyCdk, pendingCdk} {
		var fresh model.Cdk
		if err := db.First(&fresh, cdk.ID).Error; err != nil {
			t.Fatalf("reload cdk %d: %v", cdk.ID, err)
		}
		if fresh.Status != "unused" {
			t.Fatalf("expected cdk %d to remain unused, got %s", cdk.ID, fresh.Status)
		}
	}

	_, err = svc.Exchange(user.ID, amount, 1, "127.0.0.1", "ua")
	if err == nil || err.Error() != "当前商品暂时缺货，请稍后再试" {
		t.Fatalf("expected out-of-stock error, got %v", err)
	}

	var failedOrders int64
	if err := db.Model(&model.ExchangeOrder{}).Where("status = ?", "failed").Count(&failedOrders).Error; err != nil {
		t.Fatalf("count failed orders: %v", err)
	}
	if failedOrders != 0 {
		t.Fatalf("expected no failed exchange order, got %d", failedOrders)
	}
}

func TestManualCompleteUpdatesRedeemContent(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "vip")
	item := seedTestRedeemItem(t, db, user.ID, "待回填内容", "", nil)
	cdk := seedTestCdk(t, db, item.ID)
	task := seedTestPurchaseTask(t, db, user.ID, cdk.ID, item.ID, "pending_payment", "unpaid", "")

	svc := NewPurchaseTaskService(db, nil)
	subscribeURL := "https://dash.yfjc.xyz/api/v1/client/subscribe?token=test-token"
	if _, err := svc.ManualComplete(task.ID, subscribeURL, user.ID); err != nil {
		t.Fatalf("manual complete: %v", err)
	}

	var freshTask model.PurchaseTask
	if err := db.First(&freshTask, task.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if freshTask.Status != "manual_completed" {
		t.Fatalf("expected manual_completed, got %s", freshTask.Status)
	}
	if freshTask.PaymentStatus != "paid" {
		t.Fatalf("expected paid, got %s", freshTask.PaymentStatus)
	}
	if freshTask.SubscribeURL != subscribeURL {
		t.Fatalf("unexpected subscribe url: %s", freshTask.SubscribeURL)
	}

	var freshItem model.RedeemItem
	if err := db.First(&freshItem, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if freshItem.Content != subscribeURL {
		t.Fatalf("expected redeem content updated, got %q", freshItem.Content)
	}
}

func TestManualCompleteRequiresTaskOwner(t *testing.T) {
	db := openTestDB(t)
	owner := seedTestUser(t, db, "vip")
	otherUser := seedTestUser(t, db, "guest")
	item := seedTestRedeemItem(t, db, owner.ID, "待回填内容", "", nil)
	cdk := seedTestCdk(t, db, item.ID)
	task := seedTestPurchaseTask(t, db, owner.ID, cdk.ID, item.ID, "pending_payment", "unpaid", "")

	svc := NewPurchaseTaskService(db, nil)
	errMsg := "无权操作该采购任务或任务不存在"
	if _, err := svc.ManualComplete(task.ID, "https://dash.yfjc.xyz/api/v1/client/subscribe?token=test-token", otherUser.ID); err == nil || err.Error() != errMsg {
		t.Fatalf("expected %q, got %v", errMsg, err)
	}
}

func TestPurchaseTaskListFiltersByOwnerScope(t *testing.T) {
	db := openTestDB(t)
	owner := seedTestUser(t, db, "owner")
	member := seedTestUser(t, db, "member")
	outsider := seedTestUser(t, db, "outsider")
	seedTestTeamShare(t, db, owner.ID, member.ID)

	ownerItem := seedTestRedeemItem(t, db, owner.ID, "owner-item", "", nil)
	ownerCdk := seedTestCdk(t, db, ownerItem.ID)
	seedTestPurchaseTask(t, db, owner.ID, ownerCdk.ID, ownerItem.ID, "pending_payment", "unpaid", "")

	outsiderItem := seedTestRedeemItem(t, db, outsider.ID, "outsider-item", "", nil)
	outsiderCdk := seedTestCdk(t, db, outsiderItem.ID)
	seedTestPurchaseTask(t, db, outsider.ID, outsiderCdk.ID, outsiderItem.ID, "pending_payment", "unpaid", "")

	svc := NewPurchaseTaskService(db, nil)
	list, total, err := svc.List(PurchaseTaskListInput{
		Page:          1,
		PageSize:      20,
		CurrentUserID: member.ID,
	})
	if err != nil {
		t.Fatalf("list purchase tasks: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("unexpected task scope: total=%d len=%d", total, len(list))
	}
	if list[0].TeamOwnerID != owner.ID {
		t.Fatalf("expected shared owner task, got owner_id=%d", list[0].TeamOwnerID)
	}
}

func TestImportLinesCreatesPendingTasks(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "vip")
	template := seedTestTemplate(t, db, user.ID, "gptplus", "GPT Plus")

	taskSvc := NewPurchaseTaskService(db, NewAutomationRunner("python3", "automation/yfjc_runner.py", 120, 2))
	itemSvc := NewRedeemItemService(db)
	itemSvc.SetPurchaseTaskService(taskSvc)

	result, err := itemSvc.importLinesFromReader(strings.NewReader("first\nsecond\n"), "batch.txt", template.ID, user.ID)
	if err != nil {
		t.Fatalf("import lines: %v", err)
	}
	if result.Inserted != 2 {
		t.Fatalf("expected 2 inserted, got %d", result.Inserted)
	}

	var items []model.RedeemItem
	if err := db.Order("id ASC").Find(&items).Error; err != nil {
		t.Fatalf("load redeem items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	for _, item := range items {
		if item.Content != "" {
			t.Fatalf("expected empty item content before manual completion, got %q", item.Content)
		}
	}

	var tasks []model.PurchaseTask
	if err := db.Order("id ASC").Find(&tasks).Error; err != nil {
		t.Fatalf("load purchase tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Status != "pending" || tasks[1].Status != "pending" {
		t.Fatalf("expected pending tasks, got %s and %s", tasks[0].Status, tasks[1].Status)
	}
	if tasks[0].AccountName != "vip-gptplus-0001" || tasks[1].AccountName != "vip-gptplus-0002" {
		t.Fatalf("unexpected account names: %s %s", tasks[0].AccountName, tasks[1].AccountName)
	}
}

func TestProcessTaskMovesToPendingPayment(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "vip")
	item := seedTestRedeemItem(t, db, user.ID, "待处理内容", "", nil)
	cdk := seedTestCdk(t, db, item.ID)
	task := seedTestPurchaseTask(t, db, user.ID, cdk.ID, item.ID, "pending", "unpaid", "")

	svc := NewPurchaseTaskService(db, stubAutomationExecutor{
		result: &AutomationResult{
			Status:          "pending_payment",
			ExternalOrderNo: "ORD-1001",
		},
	})
	updated, err := svc.Process(task.ID, user.ID)
	if err != nil {
		t.Fatalf("process task: %v", err)
	}
	if updated.Status != "pending_payment" {
		t.Fatalf("expected pending_payment, got %+v", updated)
	}
	if updated.ExternalOrderNo != "ORD-1001" {
		t.Fatalf("unexpected order no: %+v", updated)
	}
}

func TestProcessTaskMovesToManualReviewOnRunnerError(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "vip")
	item := seedTestRedeemItem(t, db, user.ID, "待处理内容", "", nil)
	cdk := seedTestCdk(t, db, item.ID)
	task := seedTestPurchaseTask(t, db, user.ID, cdk.ID, item.ID, "pending", "unpaid", "")

	svc := NewPurchaseTaskService(db, stubAutomationExecutor{
		err: errors.New("runner boom"),
	})
	updated, err := svc.Process(task.ID, user.ID)
	if err != nil {
		t.Fatalf("process task: %v", err)
	}
	if updated.Status != "needs_manual_review" {
		t.Fatalf("expected needs_manual_review, got %+v", updated)
	}
	if updated.LastError == nil || *updated.LastError != "runner boom" {
		t.Fatalf("unexpected last error: %+v", updated.LastError)
	}
}

func TestProcessTaskTruncatesLongManualReviewReason(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "vip")
	item := seedTestRedeemItem(t, db, user.ID, "待处理内容", "", nil)
	cdk := seedTestCdk(t, db, item.ID)
	task := seedTestPurchaseTask(t, db, user.ID, cdk.ID, item.ID, "pending", "unpaid", "")
	longErr := strings.Repeat("cloudflare challenge html ", 20)

	svc := NewPurchaseTaskService(db, stubAutomationExecutor{
		result: &AutomationResult{
			Status:             "needs_manual_review",
			ManualReviewReason: longErr,
			Error:              longErr,
		},
	})
	updated, err := svc.Process(task.ID, user.ID)
	if err != nil {
		t.Fatalf("process task: %v", err)
	}
	if updated.Status != "needs_manual_review" {
		t.Fatalf("expected needs_manual_review, got %+v", updated)
	}
	if len(updated.ManualReviewReason) > 255 {
		t.Fatalf("manual review reason was not truncated: %d", len(updated.ManualReviewReason))
	}
	if updated.LastError == nil || *updated.LastError != longErr {
		t.Fatalf("unexpected last error: %+v", updated.LastError)
	}
}

func TestFetchSubscribeMarksTaskReadyAndCreatesRedeemContent(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "vip")
	template := seedTestTemplate(t, db, user.ID, "month_1", "月卡一月")
	task, err := NewPurchaseTaskService(db, nil).CreateForTemplate(template.ID, user.ID)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	task.Status = "pending_payment"
	if err := db.Save(task).Error; err != nil {
		t.Fatalf("mark task pending payment: %v", err)
	}

	svc := NewPurchaseTaskService(db, stubAutomationExecutor{
		result: &AutomationResult{
			Status:       "ready",
			SubscribeURL: "https://dash.yfjc.xyz/api/v1/client/subscribe?token=auto-token",
		},
	})
	updated, err := svc.FetchSubscribe(task.ID, user.ID)
	if err != nil {
		t.Fatalf("fetch subscribe: %v", err)
	}
	if updated.Status != "ready" || updated.PaymentStatus != "paid" {
		t.Fatalf("unexpected updated task: %+v", updated)
	}
	if updated.SubscribeURL == "" {
		t.Fatalf("expected subscribe url to be set")
	}
	if updated.RedeemItemID == nil || updated.CdkID == nil {
		t.Fatalf("expected ready task to bind generated redeem content and cdk, got %+v", updated)
	}

	var freshItem model.RedeemItem
	if err := db.First(&freshItem, *updated.RedeemItemID).Error; err != nil {
		t.Fatalf("reload redeem item: %v", err)
	}
	if !strings.Contains(freshItem.Content, updated.SubscribeURL) {
		t.Fatalf("expected content to include subscribe url, got %q", freshItem.Content)
	}
	var freshCdk model.Cdk
	if err := db.First(&freshCdk, *updated.CdkID).Error; err != nil {
		t.Fatalf("reload cdk: %v", err)
	}
	if freshCdk.ItemID == nil || *freshCdk.ItemID != freshItem.ID {
		t.Fatalf("expected cdk to bind generated redeem item, got %+v", freshCdk)
	}
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err := bootstrapPurchaseTaskTestSchema(db); err != nil {
		t.Fatalf("bootstrap sqlite test db: %v", err)
	}
	return db
}

func bootstrapPurchaseTaskTestSchema(db *gorm.DB) error {
	statements := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT,
			password_hash TEXT,
			role TEXT DEFAULT 'user',
			status TEXT DEFAULT 'active',
			external_account_prefix TEXT DEFAULT '',
			last_login_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE UNIQUE INDEX idx_users_username ON users(username)`,
		`CREATE INDEX idx_users_deleted_at ON users(deleted_at)`,
		`CREATE TABLE teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			owner_id INTEGER,
			name TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE UNIQUE INDEX idx_teams_owner_id ON teams(owner_id)`,
		`CREATE INDEX idx_teams_deleted_at ON teams(deleted_at)`,
		`CREATE TABLE team_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER,
			member_id INTEGER,
			created_at DATETIME
		)`,
		`CREATE UNIQUE INDEX idx_team_member ON team_members(team_id, member_id)`,
		`CREATE TABLE redeem_templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			content TEXT,
			external_target_code TEXT DEFAULT '',
			external_target_name TEXT DEFAULT '',
			external_provider TEXT DEFAULT 'yfjc',
			result_content_mode TEXT DEFAULT 'subscribe_url',
			safe_stock INTEGER DEFAULT 0,
			replenish_quantity INTEGER DEFAULT 1,
			auto_replenish INTEGER DEFAULT 0,
			status TEXT DEFAULT 'active',
			created_by INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE INDEX idx_redeem_templates_deleted_at ON redeem_templates(deleted_at)`,
		`CREATE TABLE redeem_categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			status TEXT DEFAULT 'active',
			created_by INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE INDEX idx_redeem_categories_deleted_at ON redeem_categories(deleted_at)`,
		`CREATE TABLE redeem_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			filename TEXT,
			content TEXT,
			category_id INTEGER,
			template_id INTEGER,
			status TEXT DEFAULT 'active',
			created_by INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE INDEX idx_redeem_items_deleted_at ON redeem_items(deleted_at)`,
		`CREATE TABLE cdks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT,
			amount NUMERIC DEFAULT 0,
			item_id INTEGER,
			status TEXT DEFAULT 'unused',
			import_id INTEGER DEFAULT 0,
			exchanged_by INTEGER,
			exchanged_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE UNIQUE INDEX idx_cdks_code ON cdks(code)`,
		`CREATE TABLE cdk_imports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT,
			amount NUMERIC DEFAULT 0,
			item_id INTEGER,
			total INTEGER DEFAULT 0,
			inserted INTEGER DEFAULT 0,
			skipped INTEGER DEFAULT 0,
			invalid INTEGER DEFAULT 0,
			remark TEXT DEFAULT '',
			created_by INTEGER,
			created_at DATETIME
		)`,
		`CREATE TABLE team_template_sequences (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_owner_id INTEGER,
			template_id INTEGER,
			current_seq INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE UNIQUE INDEX idx_team_template_sequence ON team_template_sequences(team_owner_id, template_id)`,
		`CREATE TABLE purchase_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_owner_id INTEGER,
			template_id INTEGER,
			redeem_item_id INTEGER,
			cdk_id INTEGER,
			created_by INTEGER,
			account_prefix TEXT DEFAULT '',
			account_name TEXT DEFAULT '',
			template_code_part TEXT DEFAULT '',
			sequence_no INTEGER,
			target_code TEXT DEFAULT '',
			target_name TEXT DEFAULT '',
			provider TEXT DEFAULT 'yfjc',
			source TEXT DEFAULT 'manual',
			status TEXT DEFAULT 'pending',
			retry_count INTEGER DEFAULT 0,
			payment_status TEXT DEFAULT 'unpaid',
			manual_review_reason TEXT DEFAULT '',
			external_order_no TEXT DEFAULT '',
			subscribe_url TEXT,
			last_error TEXT,
			browser_trace_path TEXT DEFAULT '',
			screenshot_path TEXT DEFAULT '',
			html_dump_path TEXT DEFAULT '',
			payload_json TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE exchange_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			amount NUMERIC DEFAULT 0,
			quantity INTEGER,
			total_amount NUMERIC DEFAULT 0,
			status TEXT DEFAULT 'success',
			fail_reason TEXT,
			ip TEXT DEFAULT '',
			user_agent TEXT DEFAULT '',
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE exchange_order_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER,
			cdk_id INTEGER,
			code TEXT,
			created_at DATETIME
		)`,
		`CREATE UNIQUE INDEX idx_purchase_tasks_redeem_item_id ON purchase_tasks(redeem_item_id)`,
		`CREATE UNIQUE INDEX idx_purchase_tasks_cdk_id ON purchase_tasks(cdk_id)`,
		`CREATE UNIQUE INDEX idx_purchase_owner_template_seq ON purchase_tasks(team_owner_id, template_id, sequence_no)`,
		`CREATE INDEX idx_purchase_owner_status_created ON purchase_tasks(team_owner_id, status, created_at)`,
		`CREATE INDEX idx_purchase_template_status_created ON purchase_tasks(template_id, status, created_at)`,
		`CREATE INDEX idx_purchase_source_created ON purchase_tasks(source, created_at)`,
		`CREATE INDEX idx_purchase_tasks_deleted_at ON purchase_tasks(deleted_at)`,
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedTestUser(t *testing.T, db *gorm.DB, prefix string) *model.User {
	t.Helper()

	user := &model.User{
		Username:              "task3-user-" + generateTestCode(t),
		PasswordHash:          "hash",
		Role:                  "admin",
		Status:                "active",
		ExternalAccountPrefix: prefix,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed test user: %v", err)
	}
	return user
}

func seedTestTemplate(t *testing.T, db *gorm.DB, createdBy uint, targetCode, targetName string) *model.RedeemTemplate {
	t.Helper()

	template := &model.RedeemTemplate{
		Name:               "task3-template-" + generateTestCode(t),
		Content:            "订阅地址\n{{content}}",
		ExternalTargetCode: targetCode,
		ExternalTargetName: targetName,
		ExternalProvider:   "yfjc",
		ResultContentMode:  "subscribe_url",
		Status:             "active",
		CreatedBy:          createdBy,
	}
	if err := db.Create(template).Error; err != nil {
		t.Fatalf("seed test template: %v", err)
	}
	return template
}

func seedTestRedeemItem(t *testing.T, db *gorm.DB, createdBy uint, name, content string, templateID *uint) *model.RedeemItem {
	t.Helper()

	item := &model.RedeemItem{
		Name:       name,
		Filename:   generateTestCode(t) + ".txt",
		Content:    content,
		TemplateID: templateID,
		Status:     "active",
		CreatedBy:  createdBy,
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("seed test redeem item: %v", err)
	}
	return item
}

func seedTestCdk(t *testing.T, db *gorm.DB, itemID uint) *model.Cdk {
	t.Helper()

	cdk := &model.Cdk{
		Code:     generateTestCode(t),
		Amount:   0,
		ItemID:   &itemID,
		Status:   "unused",
		ImportID: 1,
	}
	if err := db.Create(cdk).Error; err != nil {
		t.Fatalf("seed test cdk: %v", err)
	}
	return cdk
}

func seedTestPurchaseTask(t *testing.T, db *gorm.DB, teamOwnerID, cdkID, itemID uint, status, paymentStatus, subscribeURL string) *model.PurchaseTask {
	t.Helper()

	sequenceNo := uint(1)
	if cdkID != 0 {
		sequenceNo = cdkID
	}
	task := &model.PurchaseTask{
		TeamOwnerID:        teamOwnerID,
		TemplateID:         1,
		RedeemItemID:       &itemID,
		CreatedBy:          teamOwnerID,
		AccountPrefix:      "vip",
		AccountName:        "vip-gptplus-0001",
		TemplateCodePart:   "gptplus",
		SequenceNo:         sequenceNo,
		TargetCode:         "gptplus",
		TargetName:         "GPT Plus",
		Provider:           "yfjc",
		Status:             status,
		PaymentStatus:      paymentStatus,
		SubscribeURL:       subscribeURL,
		ManualReviewReason: "",
	}
	if cdkID != 0 {
		task.CdkID = &cdkID
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("seed test purchase task: %v", err)
	}
	return task
}

func seedTestTeamShare(t *testing.T, db *gorm.DB, ownerID, memberID uint) {
	t.Helper()

	team := &model.Team{
		OwnerID: ownerID,
		Name:    "team-" + generateTestCode(t),
	}
	if err := db.Create(team).Error; err != nil {
		t.Fatalf("seed test team: %v", err)
	}
	member := &model.TeamMember{
		TeamID:   team.ID,
		MemberID: memberID,
	}
	if err := db.Create(member).Error; err != nil {
		t.Fatalf("seed test team member: %v", err)
	}
}

func generateTestCode(t *testing.T) string {
	t.Helper()

	code, err := generateRedeemCode()
	if err != nil {
		t.Fatalf("generate test code: %v", err)
	}
	return code
}

type stubAutomationExecutor struct {
	result *AutomationResult
	err    error
}

func (s stubAutomationExecutor) Run(AutomationRunInput) (*AutomationResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}
