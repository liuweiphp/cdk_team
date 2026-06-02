package service

import (
	"exchange_cdk/model"
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
		RedeemItemID:  200,
		CdkID:         300,
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
			status TEXT DEFAULT 'active',
			created_by INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE INDEX idx_redeem_templates_deleted_at ON redeem_templates(deleted_at)`,
		`CREATE TABLE redeem_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			filename TEXT,
			content TEXT,
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
		`CREATE UNIQUE INDEX idx_purchase_tasks_redeem_item_id ON purchase_tasks(redeem_item_id)`,
		`CREATE UNIQUE INDEX idx_purchase_tasks_cdk_id ON purchase_tasks(cdk_id)`,
		`CREATE UNIQUE INDEX idx_purchase_owner_template_seq ON purchase_tasks(team_owner_id, template_id, sequence_no)`,
		`CREATE INDEX idx_purchase_owner_status_created ON purchase_tasks(team_owner_id, status, created_at)`,
		`CREATE INDEX idx_purchase_template_status_created ON purchase_tasks(template_id, status, created_at)`,
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

func generateTestCode(t *testing.T) string {
	t.Helper()

	code, err := generateRedeemCode()
	if err != nil {
		t.Fatalf("generate test code: %v", err)
	}
	return code
}
