package service

import (
	"errors"
	"exchange_cdk/model"
	"strings"

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
	AutoReplenish     bool `json:"auto_replenish"`
}

type TemplateInventoryRow struct {
	TemplateID         uint   `json:"template_id"`
	TemplateName       string `json:"template_name"`
	TargetCode         string `json:"target_code"`
	TargetName         string `json:"target_name"`
	Status             string `json:"status"`
	SafeStock          int    `json:"safe_stock"`
	ReplenishQuantity  int    `json:"replenish_quantity"`
	AutoReplenish      bool   `json:"auto_replenish"`
	ReadyStock         int64  `json:"ready_stock"`
	IncomingStock      int64  `json:"incoming_stock"`
	NeedsReplenishment bool   `json:"needs_replenishment"`
}

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
			"safe_stock":         in.SafeStock,
			"replenish_quantity": in.ReplenishQuantity,
			"auto_replenish":     in.AutoReplenish,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("无权操作该模板或模板不存在")
	}
	return nil
}

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
		ready, err := s.readyStock(tpl.ID, ownerIDs)
		if err != nil {
			return nil, 0, err
		}
		incoming, err := s.incomingStock(tpl.ID, ownerIDs)
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
			NeedsReplenishment: ready+incoming < int64(tpl.SafeStock),
		})
	}

	return rows, total, nil
}

func (s *InventoryService) ReplenishTemplate(templateID, currentUserID uint) ([]model.PurchaseTask, error) {
	if templateID == 0 {
		return nil, errors.New("请选择模板")
	}
	if currentUserID == 0 {
		return nil, errors.New("无权操作该模板或模板不存在")
	}

	var tpl model.RedeemTemplate
	if err := s.db.Where("id = ? AND created_by = ?", templateID, currentUserID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("无权操作该模板或模板不存在")
		}
		return nil, err
	}
	if strings.TrimSpace(tpl.ExternalTargetCode) == "" {
		return nil, errors.New("模板未配置购买目标")
	}
	if tpl.ReplenishQuantity < 1 || tpl.ReplenishQuantity > 20 {
		return nil, errors.New("单次补货数量必须在 1 到 20 之间")
	}

	ownerIDs := []uint{currentUserID}
	ready, err := s.readyStock(tpl.ID, ownerIDs)
	if err != nil {
		return nil, err
	}
	incoming, err := s.incomingStock(tpl.ID, ownerIDs)
	if err != nil {
		return nil, err
	}
	if ready+incoming >= int64(tpl.SafeStock) {
		return []model.PurchaseTask{}, nil
	}

	user, err := NewUserService(s.db, 0).GetByID(currentUserID)
	if err != nil {
		return nil, err
	}
	created := make([]model.PurchaseTask, 0, tpl.ReplenishQuantity)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		taskSvc := NewPurchaseTaskService(tx, nil)
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
				return err
			}
			created = append(created, *task)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *InventoryService) readyStock(templateID uint, ownerIDs []uint) (int64, error) {
	var count int64
	err := s.db.Model(&model.Cdk{}).
		Distinct("cdks.id").
		Joins("JOIN redeem_items ri ON ri.id = cdks.item_id AND ri.deleted_at IS NULL").
		Joins("LEFT JOIN purchase_tasks pt ON (pt.cdk_id = cdks.id OR pt.redeem_item_id = ri.id) AND pt.deleted_at IS NULL").
		Where("ri.template_id = ? AND ri.status = 'active' AND ri.content <> '' AND cdks.status = 'unused'", templateID).
		Where("ri.created_by IN ?", ownerIDs).
		Where("(pt.id IS NULL OR pt.status IN ?)", []string{"ready", "manual_completed"}).
		Count(&count).Error
	return count, err
}

func (s *InventoryService) incomingStock(templateID uint, ownerIDs []uint) (int64, error) {
	var count int64
	err := s.db.Model(&model.PurchaseTask{}).
		Where("template_id = ? AND status IN ?", templateID, []string{"pending", "registering", "ordering", "pending_payment", "fetching_subscribe", "needs_manual_review"}).
		Where("team_owner_id IN ?", ownerIDs).
		Count(&count).Error
	return count, err
}
