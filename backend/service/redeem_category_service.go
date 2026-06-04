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

func (s *RedeemCategoryService) List(page, pageSize int, keyword, status string, currentUserID uint) ([]model.RedeemCategory, int64, error) {
	var list []model.RedeemCategory
	var total int64
	q := s.db.Model(&model.RedeemCategory{}).Preload("Creator")

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
