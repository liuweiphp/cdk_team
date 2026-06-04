package service

import (
	"errors"
	"exchange_cdk/model"
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CreatePendingTaskInput struct {
	TeamOwnerID   uint
	TemplateID    uint
	RedeemItemID  uint
	CdkID         uint
	CreatedBy     uint
	AccountPrefix string
	TemplateCode  string
	TargetCode    string
	TargetName    string
	Provider      string
	Source        string
}

type PurchaseTaskService struct {
	db     *gorm.DB
	runner AutomationExecutor

	mu        sync.Mutex
	sequences map[string]uint
}

const (
	externalAccountEmailPrefix = "liuweiphp"
	externalAccountEmailDomain = "gmail.com"
	externalAccountPassword    = "12345678"
	externalAccountStartSeq    = 1000
)

type PurchaseTaskListInput struct {
	Page          int
	PageSize      int
	Status        string
	PaymentStatus string
	CurrentUserID uint
}

func NewPurchaseTaskService(db *gorm.DB, runner AutomationExecutor) *PurchaseTaskService {
	return &PurchaseTaskService{
		db:        db,
		runner:    runner,
		sequences: make(map[string]uint),
	}
}

func (s *PurchaseTaskService) AllocateNextSequence(teamOwnerID, templateID uint) (uint, error) {
	if teamOwnerID == 0 || templateID == 0 {
		return 0, errors.New("team owner and template are required")
	}
	if s.db == nil {
		return s.allocateNextSequenceInMemory(teamOwnerID, templateID), nil
	}

	var next uint
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var seq model.TeamTemplateSequence
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("team_owner_id = ? AND template_id = ?", teamOwnerID, templateID).
			First(&seq).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			seq = model.TeamTemplateSequence{
				TeamOwnerID: teamOwnerID,
				TemplateID:  templateID,
				CurrentSeq:  0,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seq).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("team_owner_id = ? AND template_id = ?", teamOwnerID, templateID).
				First(&seq).Error; err != nil {
				return err
			}
		}

		next = seq.CurrentSeq + 1
		return tx.Model(&model.TeamTemplateSequence{}).
			Where("id = ?", seq.ID).
			Update("current_seq", next).Error
	})
	if err != nil {
		return 0, err
	}
	return next, nil
}

func (s *PurchaseTaskService) CreatePendingTask(in CreatePendingTaskInput) (*model.PurchaseTask, error) {
	sequenceNo, err := s.AllocateNextSequence(in.TeamOwnerID, in.TemplateID)
	if err != nil {
		return nil, err
	}

	provider := in.Provider
	if provider == "" {
		provider = "yfjc"
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = "manual"
	}

	task := &model.PurchaseTask{
		TeamOwnerID:      in.TeamOwnerID,
		TemplateID:       in.TemplateID,
		CreatedBy:        in.CreatedBy,
		AccountPrefix:    in.AccountPrefix,
		AccountName:      buildPurchaseTaskAccountName(in.AccountPrefix, in.TemplateCode, sequenceNo),
		TemplateCodePart: in.TemplateCode,
		SequenceNo:       sequenceNo,
		TargetCode:       in.TargetCode,
		TargetName:       in.TargetName,
		Provider:         provider,
		Source:           source,
		Status:           "pending",
		PaymentStatus:    "unpaid",
	}
	if in.RedeemItemID != 0 {
		redeemItemID := in.RedeemItemID
		task.RedeemItemID = &redeemItemID
	}
	if in.CdkID != 0 {
		cdkID := in.CdkID
		task.CdkID = &cdkID
	}

	if s.db == nil {
		return task, nil
	}
	if err := s.db.Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

func (s *PurchaseTaskService) CreateForTemplate(templateID, currentUserID uint) (*model.PurchaseTask, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("数据库未配置")
	}
	if templateID == 0 {
		return nil, errors.New("请选择模板")
	}
	if currentUserID == 0 {
		return nil, errors.New("创建人不能为空")
	}
	tpl, err := NewTemplateService(s.db).GetActiveAccessibleByID(templateID, currentUserID)
	if err != nil {
		return nil, err
	}
	user, err := NewUserService(s.db, 0).GetByID(currentUserID)
	if err != nil {
		return nil, err
	}
	return s.CreatePendingTask(CreatePendingTaskInput{
		TeamOwnerID:   currentUserID,
		TemplateID:    tpl.ID,
		CreatedBy:     currentUserID,
		AccountPrefix: user.ExternalAccountPrefix,
		TemplateCode:  tpl.ExternalTargetCode,
		TargetCode:    tpl.ExternalTargetCode,
		TargetName:    tpl.ExternalTargetName,
		Provider:      tpl.ExternalProvider,
		Source:        "manual",
	})
}

func (s *PurchaseTaskService) List(in PurchaseTaskListInput) ([]model.PurchaseTask, int64, error) {
	var list []model.PurchaseTask
	var total int64

	if in.Page <= 0 {
		in.Page = 1
	}
	if in.PageSize <= 0 {
		in.PageSize = 20
	}

	ownerIDs, err := accessibleOwnerIDs(s.db, in.CurrentUserID)
	if err != nil {
		return nil, 0, err
	}

	q := s.db.Model(&model.PurchaseTask{}).
		Preload("TeamOwner").
		Preload("Creator").
		Preload("Template").
		Preload("RedeemItem").
		Preload("Cdk").
		Where("team_owner_id IN ?", ownerIDs)
	if in.Status != "" {
		q = q.Where("status = ?", in.Status)
	}
	if in.PaymentStatus != "" {
		q = q.Where("payment_status = ?", in.PaymentStatus)
	}
	q.Count(&total)
	if err := q.Order("id DESC").Offset((in.Page - 1) * in.PageSize).Limit(in.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *PurchaseTaskService) ManualComplete(taskID uint, subscribeURL string, currentUserID uint) (*model.PurchaseTask, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("数据库未配置")
	}
	subscribeURL = strings.TrimSpace(subscribeURL)
	if taskID == 0 {
		return nil, errors.New("采购任务不存在")
	}
	if currentUserID == 0 {
		return nil, errors.New("无权操作该采购任务或任务不存在")
	}
	if subscribeURL == "" {
		return nil, errors.New("订阅链接不能为空")
	}

	var task model.PurchaseTask
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&task, taskID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("采购任务不存在")
			}
			return err
		}
		if task.TeamOwnerID != currentUserID {
			return errors.New("无权操作该采购任务或任务不存在")
		}

		redeemItemID, cdkID, err := s.ensureRedeemContent(tx, &task, subscribeURL)
		if err != nil {
			return err
		}

		updates := map[string]interface{}{
			"redeem_item_id": redeemItemID,
			"cdk_id":         cdkID,
			"subscribe_url":  subscribeURL,
			"status":         "manual_completed",
			"payment_status": "paid",
		}
		if err := tx.Model(&model.PurchaseTask{}).
			Where("id = ?", task.ID).
			Updates(updates).Error; err != nil {
			return err
		}

		task.SubscribeURL = subscribeURL
		task.Status = "manual_completed"
		task.PaymentStatus = "paid"
		task.RedeemItemID = &redeemItemID
		task.CdkID = &cdkID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *PurchaseTaskService) Process(taskID, currentUserID uint) (*model.PurchaseTask, error) {
	task, err := s.prepareRun(taskID, currentUserID, "process")
	if err != nil {
		return nil, err
	}
	result, err := s.runner.Run(AutomationRunInput{
		Action:           "prepare_order",
		TaskID:           task.ID,
		AccountName:      task.AccountName,
		AccountPrefix:    task.AccountPrefix,
		TemplateCode:     task.TemplateCodePart,
		TargetCode:       task.TargetCode,
		TargetName:       task.TargetName,
		Provider:         task.Provider,
		ExternalUsername: task.ExternalUsername,
		ExternalPassword: task.ExternalPassword,
		ExternalOrderNo:  task.ExternalOrderNo,
		PayloadJSON:      derefString(task.PayloadJSON),
	})
	if err != nil {
		return s.moveToManualReview(task.ID, err.Error(), err.Error())
	}
	return s.applyAutomationResult(task.ID, result, false)
}

func (s *PurchaseTaskService) FetchSubscribe(taskID, currentUserID uint) (*model.PurchaseTask, error) {
	task, err := s.prepareRun(taskID, currentUserID, "fetch_subscribe")
	if err != nil {
		return nil, err
	}
	result, err := s.runner.Run(AutomationRunInput{
		Action:           "fetch_subscribe",
		TaskID:           task.ID,
		AccountName:      task.AccountName,
		AccountPrefix:    task.AccountPrefix,
		TemplateCode:     task.TemplateCodePart,
		TargetCode:       task.TargetCode,
		TargetName:       task.TargetName,
		Provider:         task.Provider,
		ExternalUsername: task.ExternalUsername,
		ExternalPassword: task.ExternalPassword,
		ExternalOrderNo:  task.ExternalOrderNo,
		PayloadJSON:      derefString(task.PayloadJSON),
	})
	if err != nil {
		return s.moveToManualReview(task.ID, err.Error(), err.Error())
	}
	return s.applyAutomationResult(task.ID, result, true)
}

func (s *PurchaseTaskService) allocateNextSequenceInMemory(teamOwnerID, templateID uint) uint {
	key := fmt.Sprintf("%d:%d", teamOwnerID, templateID)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sequences[key]++
	return s.sequences[key]
}

func buildPurchaseTaskAccountName(accountPrefix, templateCode string, sequenceNo uint) string {
	parts := make([]string, 0, 3)
	if trimmed := strings.TrimSpace(accountPrefix); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if trimmed := strings.TrimSpace(templateCode); trimmed != "" {
		parts = append(parts, trimmed)
	}
	parts = append(parts, fmt.Sprintf("%04d", sequenceNo))
	return strings.Join(parts, "-")
}

func (s *PurchaseTaskService) prepareRun(taskID, currentUserID uint, action string) (*model.PurchaseTask, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("数据库未配置")
	}
	if s.runner == nil {
		return nil, errors.New("自动化执行器未配置")
	}
	if currentUserID == 0 {
		return nil, errors.New("无权操作该采购任务或任务不存在")
	}

	var task model.PurchaseTask
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, taskID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("采购任务不存在")
			}
			return err
		}
		if task.TeamOwnerID != currentUserID {
			return errors.New("无权操作该采购任务或任务不存在")
		}

		switch action {
		case "process":
			if task.Status != "pending" && task.Status != "needs_manual_review" && task.Status != "entry_challenge_required" {
				return errors.New("当前任务状态不可处理")
			}
			if strings.TrimSpace(task.TargetCode) == "" {
				return errors.New("购买目标编码不能为空")
			}
			if strings.TrimSpace(task.AccountName) == "" {
				return errors.New("外部账号名不能为空")
			}
			if err := s.ensureExternalAccountCredentials(tx, &task); err != nil {
				return err
			}
			return tx.Model(&model.PurchaseTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
				"status":               "registering",
				"manual_review_reason": "",
				"last_error":           nil,
				"external_account_seq": task.ExternalAccountSeq,
				"external_username":    task.ExternalUsername,
				"external_password":    task.ExternalPassword,
			}).Error
		case "fetch_subscribe":
			if task.Status != "pending_payment" && task.Status != "needs_manual_review" && task.Status != "fetching_subscribe" {
				return errors.New("当前任务状态不可抓取订阅")
			}
			return tx.Model(&model.PurchaseTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
				"status":               "fetching_subscribe",
				"payment_status":       "paid",
				"manual_review_reason": "",
				"last_error":           nil,
			}).Error
		default:
			return errors.New("不支持的任务操作")
		}
	})
	if err != nil {
		return nil, err
	}
	task.Status = map[string]string{
		"process":         "registering",
		"fetch_subscribe": "fetching_subscribe",
	}[action]
	if action == "fetch_subscribe" {
		task.PaymentStatus = "paid"
	}
	return &task, nil
}

func (s *PurchaseTaskService) ensureExternalAccountCredentials(tx *gorm.DB, task *model.PurchaseTask) error {
	if strings.TrimSpace(task.ExternalUsername) != "" && strings.TrimSpace(task.ExternalPassword) != "" {
		return nil
	}
	seq, err := s.allocateExternalAccountSequence(tx, task.Provider)
	if err != nil {
		return err
	}
	task.ExternalAccountSeq = seq
	task.ExternalUsername = fmt.Sprintf("%s%d@%s", externalAccountEmailPrefix, seq, externalAccountEmailDomain)
	task.ExternalPassword = externalAccountPassword
	return nil
}

func (s *PurchaseTaskService) allocateExternalAccountSequence(tx *gorm.DB, provider string) (uint, error) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "yfjc"
	}

	var seq model.ExternalAccountSequence
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("provider = ?", provider).
		First(&seq).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}

		seq = model.ExternalAccountSequence{
			Provider:   provider,
			CurrentSeq: externalAccountStartSeq,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seq).Error; err != nil {
			return 0, err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("provider = ?", provider).
			First(&seq).Error; err != nil {
			return 0, err
		}
	}

	next := seq.CurrentSeq + 1
	if err := tx.Model(&model.ExternalAccountSequence{}).
		Where("id = ?", seq.ID).
		Update("current_seq", next).Error; err != nil {
		return 0, err
	}
	return next, nil
}

func (s *PurchaseTaskService) applyAutomationResult(taskID uint, result *AutomationResult, markPaid bool) (*model.PurchaseTask, error) {
	if result == nil {
		return s.moveToManualReview(taskID, "自动化未返回结果", "自动化未返回结果")
	}
	status := strings.TrimSpace(result.Status)
	if status == "" {
		return s.moveToManualReview(taskID, "自动化返回缺少 status", "自动化返回缺少 status")
	}

	switch status {
	case "ready":
		subscribeURL := strings.TrimSpace(result.SubscribeURL)
		if subscribeURL == "" {
			return s.moveToManualReview(taskID, "自动化未返回订阅链接", "自动化未返回订阅链接")
		}
		return s.completeWithSubscribe(taskID, subscribeURL, result, markPaid)
	case "pending_payment":
		return s.updateTask(taskID, map[string]interface{}{
			"status":               "pending_payment",
			"payment_status":       "unpaid",
			"external_order_no":    strings.TrimSpace(result.ExternalOrderNo),
			"manual_review_reason": "",
			"last_error":           nil,
			"browser_trace_path":   strings.TrimSpace(result.BrowserTracePath),
			"screenshot_path":      strings.TrimSpace(result.ScreenshotPath),
			"html_dump_path":       strings.TrimSpace(result.HTMLDumpPath),
			"payload_json":         nullableString(result.PayloadJSON),
		})
	case "entry_challenge_required":
		reason := firstNonEmpty(result.ManualReviewReason, result.Error, "需要手动通过 Cloudflare 前置验证")
		return s.updateTask(taskID, map[string]interface{}{
			"status":               "entry_challenge_required",
			"manual_review_reason": truncateManualReviewReason(reason),
			"last_error":           nullableString(result.Error),
			"external_order_no":    strings.TrimSpace(result.ExternalOrderNo),
			"browser_trace_path":   strings.TrimSpace(result.BrowserTracePath),
			"screenshot_path":      strings.TrimSpace(result.ScreenshotPath),
			"html_dump_path":       strings.TrimSpace(result.HTMLDumpPath),
			"payload_json":         nullableString(result.PayloadJSON),
			"retry_count":          gorm.Expr("retry_count + 1"),
		})
	case "needs_manual_review":
		reason := firstNonEmpty(result.ManualReviewReason, result.Error, "自动化处理失败")
		return s.updateTask(taskID, map[string]interface{}{
			"status":               "needs_manual_review",
			"manual_review_reason": truncateManualReviewReason(reason),
			"last_error":           nullableString(result.Error),
			"external_order_no":    strings.TrimSpace(result.ExternalOrderNo),
			"browser_trace_path":   strings.TrimSpace(result.BrowserTracePath),
			"screenshot_path":      strings.TrimSpace(result.ScreenshotPath),
			"html_dump_path":       strings.TrimSpace(result.HTMLDumpPath),
			"payload_json":         nullableString(result.PayloadJSON),
			"retry_count":          gorm.Expr("retry_count + 1"),
		})
	case "failed":
		reason := firstNonEmpty(result.Error, "自动化处理失败")
		return s.updateTask(taskID, map[string]interface{}{
			"status":               "failed",
			"manual_review_reason": truncateManualReviewReason(reason),
			"last_error":           nullableString(result.Error),
			"browser_trace_path":   strings.TrimSpace(result.BrowserTracePath),
			"screenshot_path":      strings.TrimSpace(result.ScreenshotPath),
			"html_dump_path":       strings.TrimSpace(result.HTMLDumpPath),
			"payload_json":         nullableString(result.PayloadJSON),
		})
	default:
		return s.updateTask(taskID, map[string]interface{}{
			"status":             status,
			"external_order_no":  strings.TrimSpace(result.ExternalOrderNo),
			"browser_trace_path": strings.TrimSpace(result.BrowserTracePath),
			"screenshot_path":    strings.TrimSpace(result.ScreenshotPath),
			"html_dump_path":     strings.TrimSpace(result.HTMLDumpPath),
			"payload_json":       nullableString(result.PayloadJSON),
			"last_error":         nullableString(result.Error),
		})
	}
}

func (s *PurchaseTaskService) completeWithSubscribe(taskID uint, subscribeURL string, result *AutomationResult, markPaid bool) (*model.PurchaseTask, error) {
	var task model.PurchaseTask
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, taskID).Error; err != nil {
			return err
		}
		redeemItemID, cdkID, err := s.ensureRedeemContent(tx, &task, subscribeURL)
		if err != nil {
			return err
		}

		updates := map[string]interface{}{
			"redeem_item_id":       redeemItemID,
			"cdk_id":               cdkID,
			"status":               "ready",
			"subscribe_url":        subscribeURL,
			"external_order_no":    strings.TrimSpace(result.ExternalOrderNo),
			"manual_review_reason": "",
			"last_error":           nil,
			"browser_trace_path":   strings.TrimSpace(result.BrowserTracePath),
			"screenshot_path":      strings.TrimSpace(result.ScreenshotPath),
			"html_dump_path":       strings.TrimSpace(result.HTMLDumpPath),
			"payload_json":         nullableString(result.PayloadJSON),
		}
		if markPaid || strings.TrimSpace(result.PaymentStatus) == "paid" {
			updates["payment_status"] = "paid"
		}
		if err := tx.Model(&model.PurchaseTask{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&task, task.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *PurchaseTaskService) ensureRedeemContent(tx *gorm.DB, task *model.PurchaseTask, subscribeURL string) (uint, uint, error) {
	if task.RedeemItemID != nil && *task.RedeemItemID != 0 {
		if err := tx.Model(&model.RedeemItem{}).Where("id = ?", *task.RedeemItemID).Update("content", s.renderTaskContent(tx, task, subscribeURL)).Error; err != nil {
			return 0, 0, err
		}
		if task.CdkID != nil && *task.CdkID != 0 {
			return *task.RedeemItemID, *task.CdkID, nil
		}
		var cdk model.Cdk
		if err := tx.Where("item_id = ?", *task.RedeemItemID).First(&cdk).Error; err == nil {
			return *task.RedeemItemID, cdk.ID, nil
		}
		code, err := generateRedeemCode()
		if err != nil {
			return 0, 0, err
		}
		importID, err := s.createTaskImport(tx, task, "采购任务补生成")
		if err != nil {
			return 0, 0, err
		}
		cdk = model.Cdk{
			Code:     code,
			Amount:   0,
			ItemID:   task.RedeemItemID,
			Status:   "unused",
			ImportID: importID,
		}
		if err := tx.Create(&cdk).Error; err != nil {
			return 0, 0, err
		}
		return *task.RedeemItemID, cdk.ID, nil
	}

	content := s.renderTaskContent(tx, task, subscribeURL)
	namePart := firstNonEmpty(task.TargetName, task.TemplateCodePart, task.AccountName, "采购内容")
	filename := normalizeFilename(fmt.Sprintf("%s-%s.txt", task.AccountName, namePart))
	item := &model.RedeemItem{
		Name:       fmt.Sprintf("%s %s", task.AccountName, namePart),
		Filename:   filename,
		Content:    content,
		TemplateID: &task.TemplateID,
		Status:     "active",
		CreatedBy:  task.TeamOwnerID,
	}
	if err := tx.Create(item).Error; err != nil {
		return 0, 0, err
	}
	code, err := generateRedeemCode()
	if err != nil {
		return 0, 0, err
	}
	importID, err := s.createTaskImport(tx, task, "采购任务生成")
	if err != nil {
		return 0, 0, err
	}
	cdk := &model.Cdk{
		Code:     code,
		Amount:   0,
		ItemID:   &item.ID,
		Status:   "unused",
		ImportID: importID,
	}
	if err := tx.Create(cdk).Error; err != nil {
		return 0, 0, err
	}
	return item.ID, cdk.ID, nil
}

func (s *PurchaseTaskService) createTaskImport(tx *gorm.DB, task *model.PurchaseTask, remark string) (uint, error) {
	imp := &model.CdkImport{
		Filename:  normalizeFilename(fmt.Sprintf("%s-%s", task.AccountName, firstNonEmpty(task.TargetName, task.TemplateCodePart, "purchase"))),
		Amount:    0,
		Total:     1,
		Inserted:  1,
		Remark:    remark,
		CreatedBy: task.TeamOwnerID,
	}
	if err := tx.Create(imp).Error; err != nil {
		return 0, err
	}
	return imp.ID, nil
}

func (s *PurchaseTaskService) renderTaskContent(tx *gorm.DB, task *model.PurchaseTask, subscribeURL string) string {
	var tpl model.RedeemTemplate
	if err := tx.First(&tpl, task.TemplateID).Error; err != nil {
		return subscribeURL
	}
	if strings.Contains(tpl.Content, "{{content}}") {
		return renderTemplate(tpl.Content, subscribeURL)
	}
	return subscribeURL
}

func (s *PurchaseTaskService) moveToManualReview(taskID uint, reason, lastError string) (*model.PurchaseTask, error) {
	return s.updateTask(taskID, map[string]interface{}{
		"status":               "needs_manual_review",
		"manual_review_reason": truncateManualReviewReason(firstNonEmpty(reason, "自动化处理失败")),
		"last_error":           nullableString(lastError),
		"retry_count":          gorm.Expr("retry_count + 1"),
	})
}

func (s *PurchaseTaskService) updateTask(taskID uint, updates map[string]interface{}) (*model.PurchaseTask, error) {
	if err := s.db.Model(&model.PurchaseTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		return nil, err
	}
	var task model.PurchaseTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func truncateManualReviewReason(reason string) string {
	const maxManualReviewReasonLength = 255
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) <= maxManualReviewReasonLength {
		return reason
	}
	return string([]rune(reason)[:maxManualReviewReasonLength])
}

func nullableString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
