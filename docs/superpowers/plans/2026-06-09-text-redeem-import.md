# Text Redeem Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add direct multiline text import to the admin redeem item import panel while keeping file upload support.

**Architecture:** The existing `/api/admin/redeem-items/import` endpoint remains the single import endpoint. Backend file and text imports share one reader-based import implementation. The frontend defaults to text mode and sends either `text` or `file` with the existing category and template fields.

**Tech Stack:** Go, Gin, GORM, Vue 3, Element Plus, Vite.

---

### Task 1: Backend Text Import Service

**Files:**
- Modify: `backend/service/redeem_item_service_test.go`
- Modify: `backend/service/redeem_item_service.go`

- [ ] **Step 1: Write the failing service tests**

Add tests to `backend/service/redeem_item_service_test.go`:

```go
func TestRedeemItemServiceImportTextCreatesItemsAndCdks(t *testing.T) {
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

	result, err := NewRedeemItemService(db).ImportText("acct001\n\nacct002", tpl.ID, category.ID, owner.ID)
	if err != nil {
		t.Fatalf("import text failed: %v", err)
	}
	if result.Total != 2 || result.Inserted != 2 || len(result.Codes) != 2 {
		t.Fatalf("unexpected result: total=%d inserted=%d codes=%d", result.Total, result.Inserted, len(result.Codes))
	}

	var itemCount int64
	if err := db.Model(&model.RedeemItem{}).Where("template_id = ? AND category_id = ?", tpl.ID, category.ID).Count(&itemCount).Error; err != nil {
		t.Fatalf("count items failed: %v", err)
	}
	if itemCount != 2 {
		t.Fatalf("expected 2 redeem items, got %d", itemCount)
	}

	var cdkCount int64
	if err := db.Model(&model.Cdk{}).Where("status = ?", "unused").Count(&cdkCount).Error; err != nil {
		t.Fatalf("count cdks failed: %v", err)
	}
	if cdkCount != 2 {
		t.Fatalf("expected 2 cdks, got %d", cdkCount)
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
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./service -run 'TestRedeemItemServiceImportText' -v`

Expected: FAIL because `ImportText` is undefined.

- [ ] **Step 3: Implement reader-based import**

Modify `backend/service/redeem_item_service.go`:

```go
import (
	"bufio"
	"crypto/rand"
	"errors"
	"exchange_cdk/model"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
)
```

Add `ImportText` and extract shared logic:

```go
func (s *RedeemItemService) ImportText(text string, templateID, categoryID uint, createdBy uint) (*ImportRedeemItemsResult, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("请输入文本内容")
	}
	if len([]byte(text)) > maxFileSize {
		return nil, fmt.Errorf("文本内容超过上限 5MB")
	}
	return s.importLinesFromReader("manual-text", strings.NewReader(text), templateID, categoryID, createdBy, "文本没有有效内容行")
}
```

Change `ImportLines` to open the file and call the shared method:

```go
func (s *RedeemItemService) ImportLines(header *multipart.FileHeader, templateID, categoryID uint, createdBy uint) (*ImportRedeemItemsResult, error) {
	if header == nil {
		return nil, errors.New("请上传文件")
	}
	if header.Size > maxFileSize {
		return nil, fmt.Errorf("文件大小超过上限 5MB")
	}
	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return s.importLinesFromReader(header.Filename, file, templateID, categoryID, createdBy, "文件没有有效内容行")
}
```

Move the existing import loop into:

```go
func (s *RedeemItemService) importLinesFromReader(sourceName string, reader io.Reader, templateID, categoryID uint, createdBy uint, emptyMessage string) (*ImportRedeemItemsResult, error) {
	// Existing template/category validation, import record creation, scanner loop,
	// result saving, and return value move here unchanged except:
	// - use sourceName instead of header.Filename
	// - scan from reader
	// - return errors.New(emptyMessage) when result.Total == 0
}
```

- [ ] **Step 4: Run service tests and verify GREEN**

Run: `go test ./service -run 'TestRedeemItemServiceImportText|TestRedeemItemServiceCreateLineWithCodeUsesCategoryNamingRule' -v`

Expected: PASS.

### Task 2: Backend Handler Text Mode

**Files:**
- Modify: `backend/handler/admin_redeem_item_handler.go`

- [ ] **Step 1: Add handler support for text mode**

Modify `ImportFiles` after category validation:

```go
text := c.PostForm("text")
if strings.TrimSpace(text) != "" {
	result, err := h.svc.ImportText(text, templateID, categoryID, getUserID(c))
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": result})
	return
}
```

Add `strings` to imports.

- [ ] **Step 2: Run backend tests**

Run: `go test ./...`

Expected: PASS.

### Task 3: Frontend Import Mode UI

**Files:**
- Modify: `frontend/src/views/admin/RedeemItemManageView.vue`

- [ ] **Step 1: Update the import panel template**

Change title to `导入兑换内容`, add mode control, show textarea by default, and keep file upload under file mode.

- [ ] **Step 2: Update component state and submit logic**

Use `uploadForm.mode = 'text'`, `uploadForm.text = ''`, validate by active mode, submit `text` or `file`, and clear only the active input after success.

- [ ] **Step 3: Run frontend build**

Run: `npm run build` in `frontend`.

Expected: PASS.

### Task 4: End-to-End Verification

**Files:**
- Generated: `frontend/dist/**`

- [ ] **Step 1: Rebuild frontend dist**

Already covered by Task 3 build.

- [ ] **Step 2: Restart frontend container**

Run: `docker compose restart frontend`

Expected: frontend container restarts.

- [ ] **Step 3: Verify HTTP and browser state**

Run: `curl -s http://localhost:8081/readyz`

Expected: `{"status":"ready"}`

Open `http://localhost:3000/admin?ts=<timestamp>` and verify the import panel defaults to text input and offers file upload mode.
