package service

import (
	"bufio"
	"crypto/rand"
	"errors"
	"exchange_cdk/model"
	"fmt"
	"mime/multipart"
	"strings"

	"gorm.io/gorm"
)

type purchaseTaskCreator interface {
	CreatePendingTask(in CreatePendingTaskInput) (*model.PurchaseTask, error)
}

type RedeemItemService struct {
	db      *gorm.DB
	taskSvc purchaseTaskCreator
}

func NewRedeemItemService(db *gorm.DB) *RedeemItemService {
	return &RedeemItemService{db: db}
}

type CreateRedeemItemFromTemplateInput struct {
	Name       string
	Filename   string
	TemplateID uint
	CreatedBy  uint
}

type ImportRedeemItemsResult struct {
	ImportID uint                  `json:"import_id"`
	Total    int                   `json:"total"`
	Inserted int                   `json:"inserted"`
	Invalid  []string              `json:"invalid"`
	Codes    []GeneratedRedeemCode `json:"codes"`
}

type GeneratedRedeemCode struct {
	Code     string `json:"code"`
	ItemID   uint   `json:"item_id"`
	ItemName string `json:"item_name"`
}

func (s *RedeemItemService) SetPurchaseTaskService(taskSvc *PurchaseTaskService) {
	s.taskSvc = taskSvc
}

// List 获取当前用户可见的兑换内容列表
func (s *RedeemItemService) List(page, pageSize int, keyword, status string, currentUserID uint) ([]model.RedeemItem, int64, error) {
	var list []model.RedeemItem
	var total int64
	q := s.db.Model(&model.RedeemItem{}).Preload("Cdk").Preload("Template").Preload("Creator")
	ownerIDs, err := accessibleOwnerIDs(s.db, currentUserID)
	if err != nil {
		return nil, 0, err
	}
	q = q.Where("created_by IN ?", ownerIDs)
	if keyword != "" {
		q = q.Where("name LIKE ? OR filename LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Create 创建可兑换文本内容
func (s *RedeemItemService) Create(name, filename, content string, createdBy uint) (*model.RedeemItem, error) {
	name = strings.TrimSpace(name)
	filename = normalizeFilename(filename)
	if name == "" {
		return nil, errors.New("名称不能为空")
	}
	if content == "" {
		return nil, errors.New("文本内容不能为空")
	}
	item, _, err := s.createItemWithCode(name, filename, content, nil, createdBy)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *RedeemItemService) CreateFromTemplate(in CreateRedeemItemFromTemplateInput) (*model.RedeemItem, error) {
	if s.taskSvc == nil {
		return nil, errors.New("采购任务服务未配置")
	}

	name := strings.TrimSpace(in.Name)
	filename := normalizeFilename(in.Filename)
	if name == "" {
		return nil, errors.New("名称不能为空")
	}
	if in.TemplateID == 0 {
		return nil, errors.New("模板不存在或已禁用")
	}
	if in.CreatedBy == 0 {
		return nil, errors.New("创建人不能为空")
	}

	tpl, err := NewTemplateService(s.db).GetActiveAccessibleByID(in.TemplateID, in.CreatedBy)
	if err != nil {
		return nil, err
	}
	user, err := NewUserService(s.db, 0).GetByID(in.CreatedBy)
	if err != nil {
		return nil, err
	}

	item, cdk, err := s.createItemWithCode(name, filename, "", &tpl.ID, in.CreatedBy)
	if err != nil {
		return nil, err
	}

	_, err = s.taskSvc.CreatePendingTask(CreatePendingTaskInput{
		TeamOwnerID:   in.CreatedBy,
		TemplateID:    tpl.ID,
		RedeemItemID:  item.ID,
		CdkID:         cdk.ID,
		CreatedBy:     in.CreatedBy,
		AccountPrefix: user.ExternalAccountPrefix,
		TemplateCode:  tpl.ExternalTargetCode,
		TargetCode:    tpl.ExternalTargetCode,
		TargetName:    tpl.ExternalTargetName,
		Provider:      tpl.ExternalProvider,
	})
	if err != nil {
		return nil, err
	}

	return item, nil
}

// ImportLines 上传单个文本文件,每个非空行使用模板生成一条兑换内容和一个 CDK
func (s *RedeemItemService) ImportLines(header *multipart.FileHeader, templateID uint, createdBy uint) (*ImportRedeemItemsResult, error) {
	if header == nil {
		return nil, errors.New("请上传文件")
	}
	if header.Size > maxFileSize {
		return nil, fmt.Errorf("文件大小超过上限 5MB")
	}
	var tpl model.RedeemTemplate
	ownerIDs, err := accessibleOwnerIDs(s.db, createdBy)
	if err != nil {
		return nil, err
	}
	if err := s.db.Where("id = ? AND status = 'active' AND created_by IN ?", templateID, ownerIDs).First(&tpl).Error; err != nil {
		return nil, errors.New("模板不存在或已禁用")
	}

	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := &ImportRedeemItemsResult{}
	imp := &model.CdkImport{
		Filename:  header.Filename,
		Amount:    0,
		Remark:    "自动生成",
		CreatedBy: createdBy,
	}
	if err := s.db.Create(imp).Error; err != nil {
		return nil, err
	}
	result.ImportID = imp.ID

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		result.Total++
		content := renderTemplate(tpl.Content, line)
		code, err := generateRedeemCode()
		if err != nil {
			return nil, err
		}
		var item *model.RedeemItem
		for retry := 0; retry < 5; retry++ {
			item, err = s.createLineWithCode(header.Filename, lineNo, content, tpl.ID, imp.ID, code, createdBy)
			if err == nil {
				break
			}
			if !strings.Contains(err.Error(), "Duplicate") && !strings.Contains(err.Error(), "duplicate") {
				break
			}
			code, err = generateRedeemCode()
			if err != nil {
				return nil, err
			}
		}
		if err != nil {
			result.Invalid = append(result.Invalid, fmt.Sprintf("第%d行: %s", lineNo, err.Error()))
			continue
		}
		result.Inserted++
		result.Codes = append(result.Codes, GeneratedRedeemCode{Code: code, ItemID: item.ID, ItemName: item.Name})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if result.Total == 0 {
		return nil, errors.New("文件没有有效内容行")
	}
	imp.Total = uint(result.Total)
	imp.Inserted = uint(result.Inserted)
	imp.Invalid = uint(len(result.Invalid))
	s.db.Save(imp)
	return result, nil
}

func (s *RedeemItemService) createLineWithCode(sourceFilename string, lineNo int, content string, templateID uint, importID uint, code string, createdBy uint) (*model.RedeemItem, error) {
	var item *model.RedeemItem
	err := s.db.Transaction(func(tx *gorm.DB) error {
		filename := normalizeFilename(fmt.Sprintf("%s-%d", sourceFilename, lineNo))
		item = &model.RedeemItem{
			Name:       fmt.Sprintf("%s 第%d行", sourceFilename, lineNo),
			Filename:   filename,
			Content:    content,
			TemplateID: &templateID,
			Status:     "active",
			CreatedBy:  createdBy,
		}
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		itemID := item.ID
		cdk := &model.Cdk{
			Code:     code,
			Amount:   0,
			ItemID:   &itemID,
			Status:   "unused",
			ImportID: importID,
		}
		if err := tx.Create(cdk).Error; err != nil {
			return err
		}
		return nil
	})
	return item, err
}

func (s *RedeemItemService) createItemWithCode(name, filename, content string, templateID *uint, createdBy uint) (*model.RedeemItem, *model.Cdk, error) {
	code, err := generateRedeemCode()
	if err != nil {
		return nil, nil, err
	}

	var item *model.RedeemItem
	var cdk *model.Cdk
	err = s.db.Transaction(func(tx *gorm.DB) error {
		imp := &model.CdkImport{
			Filename:  filename,
			Amount:    0,
			Total:     1,
			Inserted:  1,
			Remark:    "手动生成",
			CreatedBy: createdBy,
		}
		if err := tx.Create(imp).Error; err != nil {
			return err
		}
		item = &model.RedeemItem{
			Name:       name,
			Filename:   filename,
			Content:    content,
			TemplateID: templateID,
			Status:     "active",
			CreatedBy:  createdBy,
		}
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		itemID := item.ID
		cdk = &model.Cdk{
			Code:     code,
			Amount:   0,
			ItemID:   &itemID,
			Status:   "unused",
			ImportID: imp.ID,
		}
		return tx.Create(cdk).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return item, cdk, nil
}

// Update 更新可兑换文本内容,仅拥有者可修改
func (s *RedeemItemService) Update(id uint, name, filename, content, status string, currentUserID uint) error {
	name = strings.TrimSpace(name)
	filename = normalizeFilename(filename)
	if name == "" {
		return errors.New("名称不能为空")
	}
	if content == "" {
		return errors.New("文本内容不能为空")
	}
	if status != "active" && status != "disabled" {
		status = "active"
	}
	result := s.db.Model(&model.RedeemItem{}).Where("id = ? AND created_by = ?", id, currentUserID).Updates(map[string]interface{}{
		"name":     name,
		"filename": filename,
		"content":  content,
		"status":   status,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("无权操作该兑换内容或内容不存在")
	}
	return nil
}

// Delete 软删除兑换内容,仅拥有者可删除
func (s *RedeemItemService) Delete(id uint, currentUserID uint) error {
	result := s.db.Where("id = ? AND created_by = ?", id, currentUserID).Delete(&model.RedeemItem{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("无权操作该兑换内容或内容不存在")
	}
	return nil
}

func normalizeFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "redeem.txt"
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".txt") {
		return filename + ".txt"
	}
	return filename
}

func generateRedeemCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	var b strings.Builder
	b.Grow(16)
	for _, v := range bytes {
		b.WriteByte(alphabet[int(v)%len(alphabet)])
	}
	return b.String(), nil
}
