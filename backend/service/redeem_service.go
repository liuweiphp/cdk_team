package service

import (
	"errors"
	"exchange_cdk/model"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RedeemService struct{ db *gorm.DB }

func NewRedeemService(db *gorm.DB) *RedeemService {
	return &RedeemService{db: db}
}

type RedeemResult struct {
	Code       string    `json:"code"`
	ItemID     uint      `json:"item_id"`
	ItemName   string    `json:"item_name"`
	Filename   string    `json:"filename"`
	Content    string    `json:"content"`
	RedeemedAt time.Time `json:"redeemed_at"`
}

// RedeemByCode 游客凭兑换码兑换绑定的文本内容
func (s *RedeemService) RedeemByCode(code, ip, ua string) (*RedeemResult, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("数据库未配置")
	}
	norm := normalizeCode(code)
	if norm == "" {
		return nil, errors.New("请输入兑换码")
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var cdk model.Cdk
	err := tx.Preload("RedeemItem").
		Where("code = ? AND status = 'unused'", norm).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&cdk).Error
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("兑换码无效或已使用")
		}
		return nil, err
	}
	if cdk.RedeemItem == nil || cdk.RedeemItem.Status != "active" {
		tx.Rollback()
		return nil, errors.New("兑换内容不可用")
	}
	if strings.TrimSpace(cdk.RedeemItem.Content) == "" {
		tx.Rollback()
		return nil, errors.New(outOfStockMessage)
	}
	var task model.PurchaseTask
	taskErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("cdk_id = ? OR redeem_item_id = ?", cdk.ID, cdk.RedeemItem.ID).
		First(&task).Error
	if taskErr != nil && !errors.Is(taskErr, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return nil, taskErr
	}
	if taskErr == nil && task.Status != "ready" && task.Status != "manual_completed" {
		tx.Rollback()
		return nil, errors.New(outOfStockMessage)
	}

	now := time.Now()
	result := tx.Model(&model.Cdk{}).Where("id = ? AND status = 'unused'", cdk.ID).Updates(map[string]interface{}{
		"status":       "exchanged",
		"exchanged_at": now,
	})
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		tx.Rollback()
		return nil, errors.New("兑换码已被使用")
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &RedeemResult{
		Code:       cdk.Code,
		ItemID:     cdk.RedeemItem.ID,
		ItemName:   cdk.RedeemItem.Name,
		Filename:   cdk.RedeemItem.Filename,
		Content:    cdk.RedeemItem.Content,
		RedeemedAt: now,
	}, nil
}
