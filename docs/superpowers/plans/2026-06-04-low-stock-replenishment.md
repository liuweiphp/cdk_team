# Low Stock Replenishment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build low-stock replenishment so admins keep small ready inventory, create paid purchase tasks when stock is low, and users can only receive ready inventory.

**Architecture:** Keep the current Go + Gin + GORM + Vue structure. Add inventory policy fields to templates, an inventory service for stock counts and replenishment decisions, and reuse the existing purchase task completion path that creates `redeem_item + cdk` only after `subscribe_url` is available.

**Tech Stack:** Go, Gin, GORM, MySQL migrations, Vue 3, Element Plus, existing Python automation runner.

---

## File Structure

- Create `backend/migration/0009_inventory_replenishment.up.sql`: add template inventory policy fields and purchase task source.
- Create `backend/migration/0009_inventory_replenishment.down.sql`: rollback those fields.
- Modify `backend/model/redeem_template.go`: add inventory policy fields.
- Modify `backend/model/purchase_task.go`: add `Source`.
- Create `backend/service/inventory_service.go`: template inventory counts, policy update, replenishment task creation.
- Create `backend/service/inventory_service_test.go`: stock counting, policy validation, replenishment idempotence.
- Modify `backend/service/purchase_task_service.go`: accept `Source` in `CreatePendingTaskInput`, add source filter, keep completion behavior.
- Modify `backend/service/exchange_service.go`: only issue CDKs whose linked redeem content is ready.
- Modify `backend/service/redeem_service.go`: return out-of-stock style error when content is missing or task is not ready.
- Keep `backend/handler/admin_template_handler.go` focused on template create/update; inventory policy is updated through the dedicated inventory handler.
- Create `backend/handler/admin_inventory_handler.go`: inventory overview and replenish endpoints.
- Modify `backend/router/router.go`: register inventory routes.
- Modify `backend/main.go`: initialize `InventoryService`.
- Modify `backend/service/purchase_task_service_test.go`: add schema fields to SQLite bootstrap.
- Modify `frontend/src/api/index.ts`: add inventory API helpers.
- Modify `frontend/src/views/admin/TemplateManageView.vue`: add policy controls.
- Modify `frontend/src/views/admin/PurchaseTaskManageView.vue`: add inventory overview and replenishment action.
- Modify `frontend/src/views/user/ExchangeView.vue`: show clear out-of-stock message when API returns it.

---

### Task 1: Database Migration And Models

**Files:**
- Create: `backend/migration/0009_inventory_replenishment.up.sql`
- Create: `backend/migration/0009_inventory_replenishment.down.sql`
- Modify: `backend/model/redeem_template.go`
- Modify: `backend/model/purchase_task.go`
- Modify: `backend/service/purchase_task_service_test.go`

- [ ] **Step 1: Write migration up**

Create `backend/migration/0009_inventory_replenishment.up.sql`:

```sql
ALTER TABLE `redeem_templates`
    ADD COLUMN `safe_stock` INT NOT NULL DEFAULT 0 COMMENT '安全库存阈值' AFTER `result_content_mode`,
    ADD COLUMN `replenish_quantity` INT NOT NULL DEFAULT 1 COMMENT '单次补货数量' AFTER `safe_stock`,
    ADD COLUMN `auto_replenish` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否自动补货' AFTER `replenish_quantity`;

ALTER TABLE `purchase_tasks`
    ADD COLUMN `source` VARCHAR(32) NOT NULL DEFAULT 'manual' COMMENT '任务来源: manual/replenishment' AFTER `provider`,
    ADD INDEX `idx_purchase_source_created` (`source`, `created_at`);
```

- [ ] **Step 2: Write migration down**

Create `backend/migration/0009_inventory_replenishment.down.sql`:

```sql
ALTER TABLE `purchase_tasks`
    DROP INDEX `idx_purchase_source_created`,
    DROP COLUMN `source`;

ALTER TABLE `redeem_templates`
    DROP COLUMN `auto_replenish`,
    DROP COLUMN `replenish_quantity`,
    DROP COLUMN `safe_stock`;
```

- [ ] **Step 3: Update model fields**

In `backend/model/redeem_template.go`, add fields after `ResultContentMode`:

```go
SafeStock         int  `gorm:"default:0" json:"safe_stock"`
ReplenishQuantity int  `gorm:"default:1" json:"replenish_quantity"`
AutoReplenish    bool `gorm:"default:false" json:"auto_replenish"`
```

In `backend/model/purchase_task.go`, add after `Provider`:

```go
Source string `gorm:"size:32;default:manual;index:idx_purchase_source_created,priority:1" json:"source"`
```

- [ ] **Step 4: Update SQLite test schema**

In `backend/service/purchase_task_service_test.go`, add these columns to bootstrap tables:

```sql
safe_stock INTEGER DEFAULT 0,
replenish_quantity INTEGER DEFAULT 1,
auto_replenish INTEGER DEFAULT 0,
```

inside `redeem_templates`, and:

```sql
source TEXT DEFAULT 'manual',
```

inside `purchase_tasks`.

- [ ] **Step 5: Run model compile check**

Run:

```bash
GOCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-build GOMODCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-mod go test ./...
```

Expected: compile succeeds.

---

### Task 2: Inventory Service Counts And Policy Validation

**Files:**
- Create: `backend/service/inventory_service.go`
- Create: `backend/service/inventory_service_test.go`

- [ ] **Step 1: Write failing tests for inventory counts**

Create `backend/service/inventory_service_test.go` with tests named:

```go
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
```

Add a second test:

```go
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
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
GOCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-build GOMODCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-mod go test ./service -run 'TestInventoryServiceCountsOnlyReadyUnusedStock|TestInventoryPolicyRejectsUnsafeValues' -v
```

Expected: FAIL because `NewInventoryService` and related types do not exist.

- [ ] **Step 3: Implement inventory service types**

Create `backend/service/inventory_service.go`:

```go
package service

import (
	"errors"
	"exchange_cdk/model"

	"gorm.io/gorm"
)

type InventoryService struct {
	db *gorm.DB
}

func NewInventoryService(db *gorm.DB) *InventoryService {
	return &InventoryService{db: db}
}

type InventoryListInput struct {
	Page          int
	PageSize      int
	Keyword       string
	Status        string
	CurrentUserID uint
}

type InventoryPolicyInput struct {
	SafeStock         int  `json:"safe_stock"`
	ReplenishQuantity int  `json:"replenish_quantity"`
	AutoReplenish    bool `json:"auto_replenish"`
}

type TemplateInventoryRow struct {
	TemplateID          uint   `json:"template_id"`
	TemplateName        string `json:"template_name"`
	TargetCode          string `json:"target_code"`
	TargetName          string `json:"target_name"`
	Status              string `json:"status"`
	SafeStock           int    `json:"safe_stock"`
	ReplenishQuantity   int    `json:"replenish_quantity"`
	AutoReplenish       bool   `json:"auto_replenish"`
	ReadyStock          int64  `json:"ready_stock"`
	IncomingStock       int64  `json:"incoming_stock"`
	NeedsReplenishment  bool   `json:"needs_replenishment"`
}
```

- [ ] **Step 4: Implement policy validation**

Add:

```go
func (s *InventoryService) UpdatePolicy(templateID uint, in InventoryPolicyInput, currentUserID uint) error {
	if templateID == 0 {
		return errors.New("模板不存在")
	}
	if in.SafeStock < 0 {
		return errors.New("安全库存不能小于 0")
	}
	if in.ReplenishQuantity < 1 || in.ReplenishQuantity > 20 {
		return errors.New("单次补货数量必须在 1 到 20 之间")
	}
	result := s.db.Model(&model.RedeemTemplate{}).
		Where("id = ? AND created_by = ?", templateID, currentUserID).
		Updates(map[string]interface{}{
			"safe_stock":          in.SafeStock,
			"replenish_quantity":  in.ReplenishQuantity,
			"auto_replenish":      in.AutoReplenish,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("无权操作该模板或模板不存在")
	}
	return nil
}
```

- [ ] **Step 5: Implement inventory list**

Add a method that loads accessible templates and computes counts:

```go
func (s *InventoryService) ListTemplateInventory(in InventoryListInput) ([]TemplateInventoryRow, int64, error) {
	if in.Page <= 0 {
		in.Page = 1
	}
	if in.PageSize <= 0 {
		in.PageSize = 20
	}
	if in.PageSize > 100 {
		in.PageSize = 100
	}
	ownerIDs, err := accessibleOwnerIDs(s.db, in.CurrentUserID)
	if err != nil {
		return nil, 0, err
	}

	q := s.db.Model(&model.RedeemTemplate{}).Where("created_by IN ?", ownerIDs)
	if in.Keyword != "" {
		q = q.Where("name LIKE ?", "%"+in.Keyword+"%")
	}
	if in.Status != "" {
		q = q.Where("status = ?", in.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var templates []model.RedeemTemplate
	if err := q.Order("id DESC").Offset((in.Page - 1) * in.PageSize).Limit(in.PageSize).Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]TemplateInventoryRow, 0, len(templates))
	for _, tpl := range templates {
		ready, err := s.readyStock(tpl.ID)
		if err != nil {
			return nil, 0, err
		}
		incoming, err := s.incomingStock(tpl.ID)
		if err != nil {
			return nil, 0, err
		}
		rows = append(rows, TemplateInventoryRow{
			TemplateID:         tpl.ID,
			TemplateName:       tpl.Name,
			TargetCode:         tpl.ExternalTargetCode,
			TargetName:         tpl.ExternalTargetName,
			Status:             tpl.Status,
			SafeStock:          tpl.SafeStock,
			ReplenishQuantity:  tpl.ReplenishQuantity,
			AutoReplenish:      tpl.AutoReplenish,
			ReadyStock:         ready,
			IncomingStock:      incoming,
			NeedsReplenishment: ready <= int64(tpl.SafeStock),
		})
	}
	return rows, total, nil
}
```

Add helper queries:

```go
func (s *InventoryService) readyStock(templateID uint) (int64, error) {
	var count int64
	err := s.db.Model(&model.Cdk{}).
		Joins("JOIN redeem_items ri ON ri.id = cdks.item_id AND ri.deleted_at IS NULL").
		Joins("LEFT JOIN purchase_tasks pt ON pt.cdk_id = cdks.id AND pt.deleted_at IS NULL").
		Where("ri.template_id = ? AND ri.status = 'active' AND ri.content <> '' AND cdks.status = 'unused'", templateID).
		Where("(pt.id IS NULL OR pt.status IN ?)", []string{"ready", "manual_completed"}).
		Count(&count).Error
	return count, err
}

func (s *InventoryService) incomingStock(templateID uint) (int64, error) {
	var count int64
	err := s.db.Model(&model.PurchaseTask{}).
		Where("template_id = ? AND status IN ?", templateID, []string{"pending", "registering", "ordering", "pending_payment", "fetching_subscribe", "needs_manual_review"}).
		Count(&count).Error
	return count, err
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
GOCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-build GOMODCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-mod go test ./service -run 'TestInventoryServiceCountsOnlyReadyUnusedStock|TestInventoryPolicyRejectsUnsafeValues' -v
```

Expected: PASS.

---

### Task 3: Replenishment Task Creation

**Files:**
- Modify: `backend/service/inventory_service.go`
- Modify: `backend/service/purchase_task_service.go`
- Modify: `backend/service/inventory_service_test.go`

- [ ] **Step 1: Write failing replenishment tests**

Add:

```go
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
}
```

- [ ] **Step 2: Run failing tests**

Run:

```bash
GOCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-build GOMODCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-mod go test ./service -run 'TestReplenishTemplate' -v
```

Expected: FAIL because `ReplenishTemplate` does not exist.

- [ ] **Step 3: Add source support to purchase task service**

In `CreatePendingTaskInput`, add:

```go
Source string
```

In `CreatePendingTask`, set:

```go
source := strings.TrimSpace(in.Source)
if source == "" {
	source = "manual"
}
task.Source = source
```

In `CreateForTemplate`, pass `Source: "manual"`.

- [ ] **Step 4: Implement replenishment creation**

In `backend/service/inventory_service.go`, add:

```go
func (s *InventoryService) ReplenishTemplate(templateID, currentUserID uint) ([]model.PurchaseTask, error) {
	if templateID == 0 {
		return nil, errors.New("请选择模板")
	}
	var tpl model.RedeemTemplate
	if err := s.db.Where("id = ? AND created_by = ?", templateID, currentUserID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("无权操作该模板或模板不存在")
		}
		return nil, err
	}
	if tpl.ExternalTargetCode == "" {
		return nil, errors.New("模板未配置购买目标")
	}
	if tpl.ReplenishQuantity < 1 || tpl.ReplenishQuantity > 20 {
		return nil, errors.New("单次补货数量必须在 1 到 20 之间")
	}
	ready, err := s.readyStock(tpl.ID)
	if err != nil {
		return nil, err
	}
	incoming, err := s.incomingStock(tpl.ID)
	if err != nil {
		return nil, err
	}
	if ready+incoming > int64(tpl.SafeStock) {
		return []model.PurchaseTask{}, nil
	}

	user, err := NewUserService(s.db, 0).GetByID(currentUserID)
	if err != nil {
		return nil, err
	}
	taskSvc := NewPurchaseTaskService(s.db, nil)
	created := make([]model.PurchaseTask, 0, tpl.ReplenishQuantity)
	for i := 0; i < tpl.ReplenishQuantity; i++ {
		task, err := taskSvc.CreatePendingTask(CreatePendingTaskInput{
			TeamOwnerID:   currentUserID,
			TemplateID:    tpl.ID,
			CreatedBy:     currentUserID,
			AccountPrefix: user.ExternalAccountPrefix,
			TemplateCode:  tpl.ExternalTargetCode,
			TargetCode:    tpl.ExternalTargetCode,
			TargetName:    tpl.ExternalTargetName,
			Provider:      tpl.ExternalProvider,
			Source:        "replenishment",
		})
		if err != nil {
			return nil, err
		}
		created = append(created, *task)
	}
	return created, nil
}
```

- [ ] **Step 5: Run replenishment tests**

Run:

```bash
GOCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-build GOMODCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-mod go test ./service -run 'TestReplenishTemplate' -v
```

Expected: PASS.

---

### Task 4: Admin Inventory API And Router Wiring

**Files:**
- Create: `backend/handler/admin_inventory_handler.go`
- Modify: `backend/router/router.go`
- Modify: `backend/main.go`
- Modify: `backend/handler/admin_template_handler.go`

- [ ] **Step 1: Add service to router services**

In `backend/router/router.go`, add to `Services`:

```go
Inventory *service.InventoryService
```

In `backend/main.go`, initialize:

```go
Inventory: service.NewInventoryService(db),
```

- [ ] **Step 2: Create inventory handler**

Create `backend/handler/admin_inventory_handler.go`:

```go
package handler

import (
	"exchange_cdk/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminInventoryHandler struct {
	svc *service.InventoryService
}

func NewAdminInventoryHandler(svc *service.InventoryService) *AdminInventoryHandler {
	return &AdminInventoryHandler{svc: svc}
}

func (h *AdminInventoryHandler) ListTemplates(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	list, total, err := h.svc.ListTemplateInventory(service.InventoryListInput{
		Page: page, PageSize: pageSize, Keyword: c.Query("keyword"), Status: c.Query("status"), CurrentUserID: getUserID(c),
	})
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{"list": list, "total": total, "page": page, "page_size": pageSize}})
}

func (h *AdminInventoryHandler) UpdatePolicy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "ID无效", "data": nil})
		return
	}
	var req service.InventoryPolicyInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	if err := h.svc.UpdatePolicy(uint(id), req, getUserID(c)); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
}

func (h *AdminInventoryHandler) Replenish(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "ID无效", "data": nil})
		return
	}
	tasks, err := h.svc.ReplenishTemplate(uint(id), getUserID(c))
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{"list": tasks, "total": len(tasks)}})
}
```

- [ ] **Step 3: Register routes**

In `router.Setup`, after template routes:

```go
adminInventoryH := handler.NewAdminInventoryHandler(svc.Inventory)
admin.GET("/inventory/templates", adminInventoryH.ListTemplates)
admin.PUT("/templates/:id/inventory-policy", adminInventoryH.UpdatePolicy)
admin.POST("/templates/:id/replenish", adminInventoryH.Replenish)
```

- [ ] **Step 4: Keep template create/update unchanged**

Do not add inventory policy fields to `templateReq` in `backend/handler/admin_template_handler.go`. Policy updates go through `PUT /api/admin/templates/:id/inventory-policy`, which keeps the existing template create/update behavior stable.

- [ ] **Step 5: Run backend tests**

Run:

```bash
GOCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-build GOMODCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-mod go test ./...
```

Expected: PASS.

---

### Task 5: Exchange And Redeem Stock Protection

**Files:**
- Modify: `backend/service/exchange_service.go`
- Modify: `backend/service/redeem_service.go`
- Modify: `backend/service/purchase_task_service_test.go`

- [ ] **Step 1: Write tests for exchange stock filtering**

Add service tests:

```go
func TestExchangeOnlyIssuesReadyContentCdks(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "buyer")
	tpl := seedTestTemplate(t, db, user.ID, "gptplus", "GPT Plus")
	empty := seedTestRedeemItem(t, db, user.ID, "empty", "", &tpl.ID)
	cdk := seedTestCdk(t, db, empty.ID)
	if err := db.Model(cdk).Update("amount", 46).Error; err != nil {
		t.Fatalf("set amount: %v", err)
	}

	_, err := NewExchangeService(db, 10).Exchange(user.ID, 46, 1, "127.0.0.1", "test")
	if err == nil || err.Error() != "当前商品暂时缺货，请稍后再试" {
		t.Fatalf("unexpected err: %v", err)
	}

	var fresh model.Cdk
	if err := db.First(&fresh, cdk.ID).Error; err != nil {
		t.Fatalf("reload cdk: %v", err)
	}
	if fresh.Status != "unused" {
		t.Fatalf("expected unused cdk, got %s", fresh.Status)
	}
}
```

- [ ] **Step 2: Modify available amounts query**

In `GetAvailableAmounts`, replace raw query with one that joins `redeem_items`:

```sql
SELECT c.amount, COUNT(*) as remaining
FROM cdks c
JOIN redeem_items ri ON ri.id = c.item_id AND ri.deleted_at IS NULL
LEFT JOIN purchase_tasks pt ON pt.cdk_id = c.id AND pt.deleted_at IS NULL
WHERE c.status = 'unused'
  AND ri.status = 'active'
  AND ri.content <> ''
  AND (pt.id IS NULL OR pt.status IN ('ready', 'manual_completed'))
GROUP BY c.amount HAVING remaining > 0 ORDER BY c.amount DESC
```

- [ ] **Step 3: Modify exchange selection**

In `Exchange`, replace:

```go
err := tx.Where("amount = ? AND status = 'unused'", amount)
```

with a joined query:

```go
err := tx.Model(&model.Cdk{}).
	Joins("JOIN redeem_items ri ON ri.id = cdks.item_id AND ri.deleted_at IS NULL").
	Joins("LEFT JOIN purchase_tasks pt ON pt.cdk_id = cdks.id AND pt.deleted_at IS NULL").
	Where("cdks.amount = ? AND cdks.status = 'unused'", amount).
	Where("ri.status = 'active' AND ri.content <> ''").
	Where("(pt.id IS NULL OR pt.status IN ?)", []string{"ready", "manual_completed"}).
	Order("cdks.id ASC").Limit(int(quantity)).
	Clauses(clause.Locking{Strength: "UPDATE"}).
	Find(&cdks).Error
```

When `len(cdks) < int(quantity)`, return:

```go
return nil, errors.New("当前商品暂时缺货，请稍后再试")
```

Do not create a failed order for this case because the spec says缺货不创建订单.

- [ ] **Step 4: Tighten redeem-by-code protection**

In `RedeemByCode`, after `cdk.RedeemItem == nil` check, add:

```go
if cdk.RedeemItem.Content == "" {
	tx.Rollback()
	return nil, errors.New("当前商品暂时缺货，请稍后再试")
}
```

For non-ready purchase task status, return the same message:

```go
return nil, errors.New("当前商品暂时缺货，请稍后再试")
```

- [ ] **Step 5: Run focused tests**

Run:

```bash
GOCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-build GOMODCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-mod go test ./service -run 'TestExchangeOnlyIssuesReadyContentCdks|TestRedeem' -v
```

Expected: PASS.

---

### Task 6: Frontend Inventory Controls

**Files:**
- Modify: `frontend/src/api/index.ts`
- Modify: `frontend/src/views/admin/TemplateManageView.vue`
- Modify: `frontend/src/views/admin/PurchaseTaskManageView.vue`
- Modify: `frontend/src/views/user/ExchangeView.vue`

- [ ] **Step 1: Add API helpers**

In `frontend/src/api/index.ts`, add:

```ts
export const getTemplateInventory = (params: Record<string, any>) =>
  api.get('/admin/inventory/templates', { params })
export const updateTemplateInventoryPolicy = (id: number, data: Record<string, any>) =>
  api.put(`/admin/templates/${id}/inventory-policy`, data)
export const replenishTemplate = (id: number) =>
  api.post(`/admin/templates/${id}/replenish`)
```

- [ ] **Step 2: Add template policy fields**

In `TemplateManageView.vue`, add to `form`:

```ts
safe_stock: 0,
replenish_quantity: 1,
auto_replenish: false,
```

Add fields to the dialog:

```vue
<el-form-item label="安全库存">
  <el-input-number v-model="form.safe_stock" :min="0" :max="999" />
</el-form-item>
<el-form-item label="单次补货">
  <el-input-number v-model="form.replenish_quantity" :min="1" :max="20" />
</el-form-item>
<el-form-item label="自动补货">
  <el-switch v-model="form.auto_replenish" />
</el-form-item>
```

After successful template save, call:

```ts
if (editingId.value) {
  await updateTemplateInventoryPolicy(editingId.value, {
    safe_stock: form.safe_stock,
    replenish_quantity: form.replenish_quantity,
    auto_replenish: form.auto_replenish,
  })
}
```

For create flow, use returned template id if `createTemplate` returns it; otherwise save base template first, reload list, and require policy edits on the edit dialog in first pass.

- [ ] **Step 3: Add purchase task inventory overview**

In `PurchaseTaskManageView.vue`, import helpers:

```ts
import { getTemplateInventory, replenishTemplate } from '@/api'
```

Add state:

```ts
const inventoryRows = ref<any[]>([])
const inventoryLoading = ref(false)
const replenishLoadingId = ref<number | null>(null)
```

Add `fetchInventory()`:

```ts
async function fetchInventory() {
  inventoryLoading.value = true
  try {
    const data: any = await getTemplateInventory({ page: 1, page_size: 100, status: 'active' })
    inventoryRows.value = data.list || []
  } catch {}
  inventoryLoading.value = false
}
```

Call it in `onMounted` and after task actions.

Add compact table above task list:

```vue
<div class="glass-card inventory-panel">
  <el-table :data="inventoryRows" v-loading="inventoryLoading" style="width:100%">
    <el-table-column prop="template_name" label="模板" min-width="160" />
    <el-table-column prop="ready_stock" label="可用库存" width="100" />
    <el-table-column prop="incoming_stock" label="待入库" width="100" />
    <el-table-column prop="safe_stock" label="安全库存" width="100" />
    <el-table-column prop="replenish_quantity" label="单次补货" width="100" />
    <el-table-column label="状态" width="120">
      <template #default="{ row }">
        <el-tag :type="row.needs_replenishment ? 'warning' : 'success'" size="small">
          {{ row.needs_replenishment ? '需补货' : '正常' }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column label="操作" width="120">
      <template #default="{ row }">
        <el-button text size="small" type="primary" :loading="replenishLoadingId === row.template_id" @click="handleReplenish(row)">
          补货
        </el-button>
      </template>
    </el-table-column>
  </el-table>
</div>
```

- [ ] **Step 4: Add replenishment action**

Add:

```ts
async function handleReplenish(row: any) {
  replenishLoadingId.value = row.template_id
  try {
    const data: any = await replenishTemplate(row.template_id)
    const total = data.total || 0
    ElMessage.success(total ? `已创建 ${total} 个补货任务` : '当前库存无需补货')
    fetchInventory()
    fetchData()
  } catch {}
  replenishLoadingId.value = null
}
```

- [ ] **Step 5: Show source in task list**

Add a column:

```vue
<el-table-column label="来源" width="100">
  <template #default="{ row }">{{ row.source === 'replenishment' ? '补货' : '手动' }}</template>
</el-table-column>
```

- [ ] **Step 6: Verify frontend type/build**

Run:

```bash
cd frontend && npm run build
```

Expected: PASS.

---

### Task 7: Full Verification And Container Restart

**Files:**
- No new files unless fixing failures.

- [ ] **Step 1: Format Go files**

Run:

```bash
gofmt -w backend/model/redeem_template.go backend/model/purchase_task.go backend/service/inventory_service.go backend/service/inventory_service_test.go backend/service/purchase_task_service.go backend/service/exchange_service.go backend/service/redeem_service.go backend/handler/admin_inventory_handler.go backend/router/router.go backend/main.go backend/service/purchase_task_service_test.go
```

- [ ] **Step 2: Run all backend tests**

Run:

```bash
cd backend && GOCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-build GOMODCACHE=/Users/liuw/Desktop/qdhx/java_demo/exchange_cdk/backend/.cache/go-mod go test ./...
```

Expected: all packages pass.

- [ ] **Step 3: Build frontend**

Run:

```bash
cd frontend && npm run build
```

Expected: build succeeds.

- [ ] **Step 4: Rebuild and restart containers**

Run:

```bash
docker compose up -d --build
```

Expected: `exc_mysql`, `exc_backend`, and `exc_frontend` are running.

- [ ] **Step 5: Verify health endpoint**

Run:

```bash
curl -i http://localhost:3000/readyz
```

Expected:

```text
HTTP/1.1 200 OK
{"status":"ready"}
```

- [ ] **Step 6: Verify inventory API manually**

Login:

```bash
curl -s -X POST http://localhost:3000/api/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}'
```

Use the token:

```bash
TOKEN=$(curl -s -X POST http://localhost:3000/api/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}' | node -e "let s='';process.stdin.on('data',d=>s+=d);process.stdin.on('end',()=>console.log(JSON.parse(s).data.token))")
curl -i 'http://localhost:3000/api/admin/inventory/templates?page=1&page_size=100&status=active' -H "Authorization: Bearer $TOKEN"
```

Expected: `200 OK` with `data.list` containing inventory fields.

- [ ] **Step 7: Verify no unrelated work was staged**

Run:

```bash
git status --short
git diff --stat
```

Expected: only intended implementation files are modified, plus existing unrelated dirty files remain untouched.
