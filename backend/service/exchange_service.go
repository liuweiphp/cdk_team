package service

import (
	"errors"
	"exchange_cdk/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExchangeService struct {
	db      *gorm.DB
	maxQty  int
}

func NewExchangeService(db *gorm.DB, maxQty int) *ExchangeService {
	return &ExchangeService{db: db, maxQty: maxQty}
}

// GetAvailableAmounts 获取仍有库存的面额列表
func (s *ExchangeService) GetAvailableAmounts() ([]struct {
	Amount    float64 `json:"amount"`
	Remaining int64   `json:"remaining"`
}, error) {
	var results []struct {
		Amount    float64 `json:"amount"`
		Remaining int64   `json:"remaining"`
	}
	err := s.db.Raw(`
		SELECT amount, COUNT(*) as remaining
		FROM cdks WHERE status = 'unused'
		GROUP BY amount HAVING remaining > 0 ORDER BY amount DESC
	`).Scan(&results).Error
	return results, err
}

// ExchangeResult 领取结果
type ExchangeResult struct {
	OrderID     uint     `json:"order_id"`
	Amount      float64  `json:"amount"`
	Quantity    uint     `json:"quantity"`
	TotalAmount float64  `json:"total_amount"`
	Codes       []string `json:"codes"`
}

// Exchange 核心领取逻辑,在单事务内: SELECT FOR UPDATE → UPDATE → INSERT
func (s *ExchangeService) Exchange(userID uint, amount float64, quantity uint, ip, ua string) (*ExchangeResult, error) {
	if amount <= 0 {
		return nil, errors.New("面额无效")
	}
	if quantity < 1 || int(quantity) > s.maxQty {
		return nil, errors.New("领取数量无效")
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

	// 1. SELECT FOR UPDATE 锁定要发的 N 行
	var cdks []model.Cdk
	err := tx.Where("amount = ? AND status = 'unused'", amount).
		Order("id ASC").Limit(int(quantity)).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Find(&cdks).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if len(cdks) < int(quantity) {
		// 库存不足,记录失败订单
		reason := "insufficient_stock"
		s.createOrder(tx, userID, amount, quantity, "failed", &reason, ip, ua)
		tx.Commit()
		return nil, errors.New("库存不足")
	}

	// 2. 收集要更新的 ID
	ids := make([]uint, len(cdks))
	codes := make([]string, len(cdks))
	now := time.Now()
	for i, c := range cdks {
		ids[i] = c.ID
		codes[i] = c.Code
	}

	// 3. UPDATE cdks SET status='exchanged'
	result := tx.Model(&model.Cdk{}).Where("id IN ? AND status = 'unused'", ids).
		Updates(map[string]interface{}{
			"status":       "exchanged",
			"exchanged_by": userID,
			"exchanged_at": now,
		})
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	if int(result.RowsAffected) != len(ids) {
		tx.Rollback()
		return nil, errors.New("并发冲突,请重试")
	}

	// 4. INSERT exchange_orders
	order := &model.ExchangeOrder{
		UserID:      userID,
		Amount:      amount,
		Quantity:    uint(len(codes)),
		TotalAmount: amount * float64(len(codes)),
		Status:      "success",
		IP:          ip,
		UserAgent:   ua,
	}
	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 5. 批量 INSERT exchange_order_items
	for i, c := range cdks {
		item := &model.ExchangeOrderItem{
			OrderID: order.ID,
			CdkID:   c.ID,
			Code:    codes[i],
		}
		if err := tx.Create(item).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &ExchangeResult{
		OrderID:     order.ID,
		Amount:      amount,
		Quantity:    uint(len(codes)),
		TotalAmount: amount * float64(len(codes)),
		Codes:       codes,
	}, nil
}

func (s *ExchangeService) createOrder(tx *gorm.DB, userID uint, amount float64, quantity uint, status string, failReason *string, ip, ua string) {
	order := &model.ExchangeOrder{
		UserID:      userID,
		Amount:      amount,
		Quantity:    quantity,
		TotalAmount: amount * float64(quantity),
		Status:      status,
		FailReason:  failReason,
		IP:          ip,
		UserAgent:   ua,
	}
	tx.Create(order)
}

// GetUserOrders 获取用户领取记录
func (s *ExchangeService) GetUserOrders(userID uint, page, pageSize int, amountFilter float64) ([]model.ExchangeOrder, int64, error) {
	var list []model.ExchangeOrder
	var total int64
	q := s.db.Where("user_id = ?", userID)
	if amountFilter > 0 {
		q = q.Where("amount = ?", amountFilter)
	}
	q.Model(&model.ExchangeOrder{}).Count(&total)
	if err := q.Order("id DESC").Offset((page-1)*pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetOrderDetail 获取订单详情(含所有 code)
func (s *ExchangeService) GetOrderDetail(orderID uint) (*model.ExchangeOrder, error) {
	var order model.ExchangeOrder
	if err := s.db.Preload("Items").First(&order, orderID).Error; err != nil {
		return nil, err
	}
	return &order, nil
}
