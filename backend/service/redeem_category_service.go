package service

import (
	"errors"
	"exchange_cdk/model"
	"strings"

	"gorm.io/gorm"
)

type RedeemCategoryService struct{ db *gorm.DB }

func NewRedeemCategoryService(db *gorm.DB) *RedeemCategoryService {
	return &RedeemCategoryService{db: db}
}

// List 获取当前用户可见的分类列表
func (s *RedeemCategoryService) List(page, pageSize int, keyword, status string, currentUserID uint) ([]model.RedeemCategory, int64, error) {
	var list []model.RedeemCategory
	var total int64

	ownerIDs, err := accessibleOwnerIDs(s.db, currentUserID)
	if err != nil {
		return nil, 0, err
	}

	q := s.db.Model(&model.RedeemCategory{}).Preload("Creator").Where("created_by IN ?", ownerIDs)
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

// Create 创建兑换内容分类
func (s *RedeemCategoryService) Create(name string, createdBy uint) (*model.RedeemCategory, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("分类名称不能为空")
	}
	category := &model.RedeemCategory{
		Name:      name,
		Status:    "active",
		CreatedBy: createdBy,
	}
	if err := s.db.Create(category).Error; err != nil {
		return nil, err
	}
	return category, nil
}

// Update 更新分类,仅拥有者可修改
func (s *RedeemCategoryService) Update(id uint, name, status string, currentUserID uint) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("分类名称不能为空")
	}
	if status != "active" && status != "disabled" {
		status = "active"
	}
	result := s.db.Model(&model.RedeemCategory{}).Where("id = ? AND created_by = ?", id, currentUserID).Updates(map[string]interface{}{
		"name":   name,
		"status": status,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("无权操作该分类或分类不存在")
	}
	return nil
}

// Delete 软删除分类,仅拥有者可删除
func (s *RedeemCategoryService) Delete(id uint, currentUserID uint) error {
	result := s.db.Where("id = ? AND created_by = ?", id, currentUserID).Delete(&model.RedeemCategory{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("无权操作该分类或分类不存在")
	}
	return nil
}
