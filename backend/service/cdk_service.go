package service

import (
	"exchange_cdk/model"

	"gorm.io/gorm"
)

type CdkService struct{ db *gorm.DB }

func NewCdkService(db *gorm.DB) *CdkService { return &CdkService{db: db} }

// List 当前用户可见的 CDK 管理列表
func (s *CdkService) List(page, pageSize int, amount float64, status, code string, importID, itemID uint, currentUserID uint) ([]model.Cdk, int64, error) {
	var list []model.Cdk
	var total int64

	ownerIDs, err := accessibleOwnerIDs(s.db, currentUserID)
	if err != nil {
		return nil, 0, err
	}

	q := s.db.Model(&model.Cdk{}).
		Preload("RedeemItem").
		Preload("RedeemItem.Creator").
		Preload("RedeemItem.Category").
		Joins("JOIN cdk_imports ON cdk_imports.id = cdks.import_id").
		Joins("LEFT JOIN redeem_items ON redeem_items.id = cdks.item_id AND redeem_items.deleted_at IS NULL").
		Where("COALESCE(redeem_items.created_by, cdk_imports.created_by) IN ?", ownerIDs)
	if amount > 0 {
		q = q.Where("cdks.amount = ?", amount)
	}
	if itemID > 0 {
		q = q.Where("cdks.item_id = ?", itemID)
	}
	if status != "" {
		q = q.Where("cdks.status = ?", status)
	}
	if code != "" {
		q = q.Where("cdks.code LIKE ?", "%"+code+"%")
	}
	if importID > 0 {
		q = q.Where("cdks.import_id = ?", importID)
	}
	q.Count(&total)
	if err := q.Order("cdks.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
