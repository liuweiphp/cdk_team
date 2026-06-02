package service

import (
	"exchange_cdk/model"
	"fmt"

	"gorm.io/gorm"
)

type StatsService struct{ db *gorm.DB }

func NewStatsService(db *gorm.DB) *StatsService { return &StatsService{db: db} }

// Overview 总览统计: 用户数 / CDK总数 / 已领 / 剩余 / 总金额 / 已领金额
func (s *StatsService) Overview() (map[string]interface{}, error) {
	var userCount, cdkTotal, cdkUsed int64
	s.db.Model(&model.User{}).Count(&userCount)
	s.db.Model(&model.Cdk{}).Count(&cdkTotal)
	s.db.Model(&model.Cdk{}).Where("status = 'exchanged'").Count(&cdkUsed)

	var totalAmount, exchangedAmount float64
	s.db.Model(&model.Cdk{}).Select("COALESCE(SUM(amount),0)").Scan(&totalAmount)
	s.db.Model(&model.Cdk{}).Where("status = 'exchanged'").Select("COALESCE(SUM(amount),0)").Scan(&exchangedAmount)

	return map[string]interface{}{
		"user_count":       userCount,
		"cdk_total":        cdkTotal,
		"cdk_exchanged":    cdkUsed,
		"cdk_remaining":    cdkTotal - cdkUsed,
		"total_amount":     totalAmount,
		"exchanged_amount": exchangedAmount,
	}, nil
}

type AmountStat struct {
	Amount          float64 `json:"amount"`
	Total           int64   `json:"total"`
	Exchanged       int64   `json:"exchanged"`
	Remaining       int64   `json:"remaining"`
	ExchangedAmount float64 `json:"exchanged_amount"`
}

// ByAmount 按面额聚合统计 (核心报表)
func (s *StatsService) ByAmount() ([]AmountStat, error) {
	var results []AmountStat
	err := s.db.Raw(`
		SELECT amount, COUNT(*) as total,
			SUM(CASE WHEN status='exchanged' THEN 1 ELSE 0 END) as exchanged,
			SUM(CASE WHEN status='unused' THEN 1 ELSE 0 END) as remaining,
			COALESCE(SUM(CASE WHEN status='exchanged' THEN amount ELSE 0 END),0) as exchanged_amount
		FROM cdks GROUP BY amount ORDER BY amount DESC
	`).Scan(&results).Error
	return results, err
}

type ItemStat struct {
	ItemID    uint   `json:"item_id"`
	ItemName  string `json:"item_name"`
	Total     int64  `json:"total"`
	Exchanged int64  `json:"exchanged"`
	Remaining int64  `json:"remaining"`
}

// ByItem 按兑换内容聚合统计
func (s *StatsService) ByItem() ([]ItemStat, error) {
	var results []ItemStat
	err := s.db.Raw(`
		SELECT COALESCE(ri.id, 0) as item_id,
			COALESCE(ri.name, '未绑定内容') as item_name,
			COUNT(c.id) as total,
			SUM(CASE WHEN c.status='exchanged' THEN 1 ELSE 0 END) as exchanged,
			SUM(CASE WHEN c.status='unused' THEN 1 ELSE 0 END) as remaining
		FROM cdks c
		LEFT JOIN redeem_items ri ON ri.id = c.item_id
		GROUP BY ri.id, ri.name ORDER BY total DESC
	`).Scan(&results).Error
	return results, err
}

type DailyStat struct {
	Date   string  `json:"date"`
	Count  int64   `json:"count"`
	Amount float64 `json:"amount"`
}

// Daily 每日领取趋势
func (s *StatsService) Daily(start, end string) ([]DailyStat, error) {
	var results []DailyStat
	err := s.db.Raw(`
		SELECT DATE(exchanged_at) as date, COUNT(*) as count,
			COALESCE(SUM(amount),0) as amount
		FROM cdks WHERE status='exchanged' AND exchanged_at >= ? AND exchanged_at < ?
		GROUP BY DATE(exchanged_at) ORDER BY date
	`, start, end).Scan(&results).Error
	return results, err
}

type TopUser struct {
	Username    string  `json:"username"`
	Count       int64   `json:"count"`
	TotalAmount float64 `json:"total_amount"`
}

// TopUsers 领取排行
func (s *StatsService) TopUsers(limit int) ([]TopUser, error) {
	var results []TopUser
	err := s.db.Raw(`
		SELECT u.username, COUNT(*) as count, COALESCE(SUM(c.amount),0) as total_amount
		FROM cdks c JOIN users u ON u.id = c.exchanged_by
		WHERE c.status = 'exchanged'
		GROUP BY u.id, u.username ORDER BY total_amount DESC LIMIT ?
	`, limit).Scan(&results).Error
	return results, err
}

// ImportList 按导入批次统计(分页)
func (s *StatsService) ImportList(page, pageSize int) ([]model.CdkImport, int64, error) {
	var list []model.CdkImport
	var total int64
	s.db.Model(&model.CdkImport{}).Count(&total)
	if err := s.db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

type UserAmountPeriodStat struct {
	Username string  `json:"username"`
	Amount   float64 `json:"amount"`
	Period   string  `json:"period"`
	Count    int64   `json:"count"`
}

// ByUserAmount 按用户+面额+时间周期统计领取
func (s *StatsService) ByUserAmount(period, start, end string, page, pageSize int) ([]UserAmountPeriodStat, int64, error) {
	var periodExpr string
	switch period {
	case "day":
		periodExpr = "DATE(c.exchanged_at)"
	case "week":
		periodExpr = "DATE_SUB(DATE(c.exchanged_at), INTERVAL WEEKDAY(DATE(c.exchanged_at)) DAY)"
	case "month":
		periodExpr = "DATE_FORMAT(c.exchanged_at, '%Y-%m')"
	case "year":
		periodExpr = "DATE_FORMAT(c.exchanged_at, '%Y')"
	default:
		periodExpr = "DATE(c.exchanged_at)"
	}

	var total int64
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM (
		SELECT 1 FROM cdks c JOIN users u ON u.id = c.exchanged_by
		WHERE c.status = 'exchanged' AND c.exchanged_at >= ? AND c.exchanged_at < ?
		GROUP BY u.id, u.username, c.amount, %s
	) t`, periodExpr)
	if err := s.db.Raw(countSQL, start, end).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	var results []UserAmountPeriodStat
	dataSQL := fmt.Sprintf(`
		SELECT u.username, c.amount, %s AS period, COUNT(*) AS count
		FROM cdks c JOIN users u ON u.id = c.exchanged_by
		WHERE c.status = 'exchanged' AND c.exchanged_at >= ? AND c.exchanged_at < ?
		GROUP BY u.id, u.username, c.amount, %s
		ORDER BY period DESC, c.amount DESC, u.username
		LIMIT ? OFFSET ?
	`, periodExpr, periodExpr)
	offset := (page - 1) * pageSize
	if err := s.db.Raw(dataSQL, start, end, pageSize, offset).Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}
