package model

import "time"

type ExchangeOrder struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `json:"user_id"`
	Amount      float64   `gorm:"type:decimal(12,2)" json:"amount"`
	Quantity    uint      `json:"quantity"`
	TotalAmount float64   `gorm:"type:decimal(14,2)" json:"total_amount"`
	Status      string    `gorm:"type:enum('success','failed')" json:"status"`
	FailReason  *string   `gorm:"size:64" json:"fail_reason"`
	IP          string    `gorm:"size:45;default:''" json:"ip"`
	UserAgent   string    `gorm:"size:255;default:''" json:"user_agent"`
	CreatedAt   time.Time `json:"created_at"`

	User  *User               `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Items []ExchangeOrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

type ExchangeOrderItem struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OrderID   uint      `json:"order_id"`
	CdkID     uint      `gorm:"uniqueIndex" json:"cdk_id"`
	Code      string    `gorm:"size:64" json:"code"`
	CreatedAt time.Time `json:"created_at"`

	Cdk *Cdk `gorm:"foreignKey:CdkID" json:"cdk,omitempty"`
}
