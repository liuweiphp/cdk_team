package service

import (
	"bufio"
	"crypto/rand"
	"errors"
	"exchange_cdk/model"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"gorm.io/gorm"
)

type RedeemItemService struct{ db *gorm.DB }

func NewRedeemItemService(db *gorm.DB) *RedeemItemService {
	return &RedeemItemService{db: db}
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

// List 获取当前用户可见的兑换内容列表
func (s *RedeemItemService) List(page, pageSize int, keyword, status string, currentUserID uint) ([]model.RedeemItem, int64, error) {
	var list []model.RedeemItem
	var total int64
	q := s.db.Model(&model.RedeemItem{}).Preload("Cdk").Preload("Category").Preload("Template").Preload("Creator")
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
func (s *RedeemItemService) Create(name, filename, content string, categoryID, createdBy uint) (*model.RedeemItem, error) {
	name = strings.TrimSpace(name)
	filename = normalizeFilename(filename)
	if name == "" {
		return nil, errors.New("名称不能为空")
	}
	category, err := s.loadCategoryForUser(categoryID, createdBy)
	if err != nil {
		return nil, err
	}
	if content == "" {
		return nil, errors.New("文本内容不能为空")
	}
	code, err := generateRedeemCode()
	if err != nil {
		return nil, err
	}
	var item *model.RedeemItem
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
			CategoryID: &category.ID,
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
			ImportID: imp.ID,
		}
		return tx.Create(cdk).Error
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

// ImportText 从多行文本导入兑换内容,每个非空行使用模板生成一条兑换内容和一个 CDK
func (s *RedeemItemService) ImportText(text string, templateID, categoryID uint, createdBy uint) (*ImportRedeemItemsResult, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("请输入文本内容")
	}
	if len([]byte(text)) > maxFileSize {
		return nil, fmt.Errorf("文本内容超过上限 5MB")
	}
	return s.importLinesFromReader("manual-text", strings.NewReader(text), templateID, categoryID, createdBy, "请输入文本内容")
}

// ImportLines 上传单个文本文件,每个非空行使用模板生成一条兑换内容和一个 CDK
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

func (s *RedeemItemService) importLinesFromReader(sourceName string, reader io.Reader, templateID, categoryID uint, createdBy uint, emptyMessage string) (*ImportRedeemItemsResult, error) {
	var tpl model.RedeemTemplate
	ownerIDs, err := accessibleOwnerIDs(s.db, createdBy)
	if err != nil {
		return nil, err
	}
	if err := s.db.Where("id = ? AND status = 'active' AND created_by IN ?", templateID, ownerIDs).First(&tpl).Error; err != nil {
		return nil, errors.New("模板不存在或已禁用")
	}
	category, err := s.loadCategoryForUser(categoryID, createdBy)
	if err != nil {
		return nil, err
	}
	var user model.User
	if err := s.db.First(&user, createdBy).Error; err != nil {
		return nil, err
	}
	nextSequence := user.FileSequenceNext
	if nextSequence == 0 {
		nextSequence = 1001
	}

	result := &ImportRedeemItemsResult{}
	imp := &model.CdkImport{
		Filename:  sourceName,
		Amount:    0,
		Remark:    "自动生成",
		CreatedBy: createdBy,
	}
	if err := s.db.Create(imp).Error; err != nil {
		return nil, err
	}
	result.ImportID = imp.ID

	scanner := bufio.NewScanner(reader)
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
		generatedName := buildSequenceGeneratedItemName(user.FilePrefix, nextSequence)
		for retry := 0; retry < 5; retry++ {
			item, err = s.createLineWithCode(sourceName, lineNo, generatedName, content, tpl.ID, category.ID, imp.ID, code, createdBy)
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
		nextSequence++
		result.Codes = append(result.Codes, GeneratedRedeemCode{Code: code, ItemID: item.ID, ItemName: item.Name})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if result.Total == 0 {
		return nil, errors.New(emptyMessage)
	}
	imp.Total = uint(result.Total)
	imp.Inserted = uint(result.Inserted)
	imp.Invalid = uint(len(result.Invalid))
	s.db.Save(imp)
	if result.Inserted > 0 {
		s.db.Model(&model.User{}).Where("id = ?", createdBy).Update("file_sequence_next", nextSequence)
	}
	return result, nil
}

func (s *RedeemItemService) createLineWithCode(sourceFilename string, lineNo int, generatedName, content string, templateID, categoryID, importID uint, code string, createdBy uint) (*model.RedeemItem, error) {
	var item *model.RedeemItem
	err := s.db.Transaction(func(tx *gorm.DB) error {
		item = &model.RedeemItem{
			Name:       fmt.Sprintf("%s 第%d行", sourceFilename, lineNo),
			Filename:   normalizeFilename(fmt.Sprintf("%s-%d", sourceFilename, lineNo)),
			Content:    content,
			CategoryID: &categoryID,
			TemplateID: &templateID,
			Status:     "active",
			CreatedBy:  createdBy,
		}
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		filename := normalizeFilename(generatedName)
		if err := tx.Model(item).Updates(map[string]interface{}{
			"name":     generatedName,
			"filename": filename,
		}).Error; err != nil {
			return err
		}
		item.Name = generatedName
		item.Filename = filename
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

// Update 更新可兑换文本内容,仅拥有者可修改
func (s *RedeemItemService) Update(id uint, name, filename, content, status string, categoryID, currentUserID uint) error {
	name = strings.TrimSpace(name)
	filename = normalizeFilename(filename)
	if name == "" {
		return errors.New("名称不能为空")
	}
	category, err := s.loadCategoryForUser(categoryID, currentUserID)
	if err != nil {
		return err
	}
	if content == "" {
		return errors.New("文本内容不能为空")
	}
	if status != "active" && status != "disabled" {
		status = "active"
	}
	result := s.db.Model(&model.RedeemItem{}).Where("id = ? AND created_by = ?", id, currentUserID).Updates(map[string]interface{}{
		"name":        name,
		"filename":    filename,
		"content":     content,
		"category_id": category.ID,
		"status":      status,
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

func (s *RedeemItemService) loadCategoryForUser(categoryID, currentUserID uint) (*model.RedeemCategory, error) {
	if categoryID == 0 {
		return nil, errors.New("请选择分类")
	}
	ownerIDs, err := accessibleOwnerIDs(s.db, currentUserID)
	if err != nil {
		return nil, err
	}
	var category model.RedeemCategory
	if err := s.db.Where("id = ? AND status = 'active' AND created_by IN ?", categoryID, ownerIDs).First(&category).Error; err != nil {
		return nil, errors.New("分类不存在或已禁用")
	}
	return &category, nil
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

func buildSequenceGeneratedItemName(prefix string, sequence uint) string {
	return fmt.Sprintf("%s%d", prefix, sequence)
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
