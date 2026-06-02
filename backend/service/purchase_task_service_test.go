package service

import (
	"exchange_cdk/model"
	"fmt"
	"os"
	"strings"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
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

	_ = godotenv.Load("../../.env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	dsn := os.Getenv("DB_DSN")
	rootPassword := os.Getenv("DB_ROOT_PASSWORD")

	candidates := make([]string, 0, 2)
	if dsn != "" {
		candidates = append(candidates, dsn)
	}
	if rootPassword != "" {
		candidates = append(candidates, fmt.Sprintf("root:%s@tcp(127.0.0.1:3307)/exchange_cdk?charset=utf8mb4&parseTime=True&loc=Local", rootPassword))
	}
	if len(candidates) == 0 {
		t.Fatal("DB_DSN or DB_ROOT_PASSWORD is required for purchase task integration test")
	}

	var lastErr error
	for _, candidate := range candidates {
		db, err := openIsolatedTestDB(t, candidate)
		if err == nil {
			return db
		}
		lastErr = err
	}

	t.Fatalf("open test db: %v", lastErr)
	return nil
}

func openIsolatedTestDB(t *testing.T, dsn string) (*gorm.DB, error) {
	t.Helper()

	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}

	adminCfg := *cfg
	adminCfg.DBName = ""
	adminDB, err := gorm.Open(mysql.Open(adminCfg.FormatDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	dbName := "task3_" + strings.ToLower(generateTestCode(t)[:8])
	if err := adminDB.Exec("CREATE DATABASE `" + dbName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error; err != nil {
		return nil, err
	}

	testCfg := *cfg
	testCfg.DBName = dbName
	testDB, err := gorm.Open(mysql.Open(testCfg.FormatDSN()), &gorm.Config{})
	if err != nil {
		_ = adminDB.Exec("DROP DATABASE `" + dbName + "`").Error
		return nil, err
	}
	if err := testDB.AutoMigrate(
		&model.User{},
		&model.Team{},
		&model.TeamMember{},
		&model.RedeemTemplate{},
		&model.CdkImport{},
		&model.RedeemItem{},
		&model.Cdk{},
		&model.TeamTemplateSequence{},
		&model.PurchaseTask{},
	); err != nil {
		_ = adminDB.Exec("DROP DATABASE `" + dbName + "`").Error
		return nil, err
	}

	t.Cleanup(func() {
		sqlDB, err := testDB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		adminSQLDB, err := adminDB.DB()
		if err == nil {
			defer adminSQLDB.Close()
		}
		_ = adminDB.Exec("DROP DATABASE `" + dbName + "`").Error
	})

	return testDB, nil
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
