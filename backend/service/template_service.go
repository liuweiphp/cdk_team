package service

import (
	"errors"
	"exchange_cdk/model"
	"strings"

	"gorm.io/gorm"
)

const templatePlaceholder = "{{content}}"

type TemplateService struct{ db *gorm.DB }

func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{db: db}
}

func (s *TemplateService) GetActiveAccessibleByID(id, currentUserID uint) (*model.RedeemTemplate, error) {
	ownerIDs, err := accessibleOwnerIDs(s.db, currentUserID)
	if err != nil {
		return nil, err
	}

	var tpl model.RedeemTemplate
	if err := s.db.Where("id = ? AND status = 'active' AND created_by IN ?", id, ownerIDs).First(&tpl).Error; err != nil {
		return nil, errors.New("模板不存在或已禁用")
	}
	return &tpl, nil
}

// List 获取当前用户可见的模板列表
func (s *TemplateService) List(page, pageSize int, keyword, status string, currentUserID uint) ([]model.RedeemTemplate, int64, error) {
	var list []model.RedeemTemplate
	var total int64
	q := s.db.Model(&model.RedeemTemplate{}).Preload("Creator")
	ownerIDs, err := accessibleOwnerIDs(s.db, currentUserID)
	if err != nil {
		return nil, 0, err
	}
	q = q.Where("created_by IN ?", ownerIDs)
	if keyword != "" {
		q = q.Where("name LIKE ?", "%"+keyword+"%")
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

// Create 创建兑换模板
func (s *TemplateService) Create(name, content string, createdBy uint) (*model.RedeemTemplate, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("模板名称不能为空")
	}
	if !strings.Contains(content, templatePlaceholder) {
		return nil, errors.New("模板内容必须包含 {{content}} 占位符")
	}
	tpl := &model.RedeemTemplate{
		Name:      name,
		Content:   content,
		Status:    "active",
		CreatedBy: createdBy,
	}
	if err := s.db.Create(tpl).Error; err != nil {
		return nil, err
	}
	return tpl, nil
}

// Update 更新兑换模板,仅拥有者可修改
func (s *TemplateService) Update(id uint, name, content, status string, currentUserID uint) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("模板名称不能为空")
	}
	if !strings.Contains(content, templatePlaceholder) {
		return errors.New("模板内容必须包含 {{content}} 占位符")
	}
	if status != "active" && status != "disabled" {
		status = "active"
	}
	result := s.db.Model(&model.RedeemTemplate{}).Where("id = ? AND created_by = ?", id, currentUserID).Updates(map[string]interface{}{
		"name":    name,
		"content": content,
		"status":  status,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("无权操作该模板或模板不存在")
	}
	return nil
}

// Delete 软删除兑换模板,仅拥有者可删除
func (s *TemplateService) Delete(id uint, currentUserID uint) error {
	result := s.db.Where("id = ? AND created_by = ?", id, currentUserID).Delete(&model.RedeemTemplate{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("无权操作该模板或模板不存在")
	}
	return nil
}

func renderTemplate(template, content string) string {
	return strings.ReplaceAll(template, templatePlaceholder, strings.TrimSpace(content))
}
