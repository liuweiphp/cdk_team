package model

import "time"

type CdkImport struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Filename  string    `gorm:"size:255" json:"filename"`
	Amount    float64   `gorm:"type:decimal(12,2)" json:"amount"`
	ItemID    *uint     `json:"item_id"`
	Total     uint      `json:"total"`
	Inserted  uint      `json:"inserted"`
	Skipped   uint      `json:"skipped"`
	Invalid   uint      `json:"invalid"`
	Remark    string    `gorm:"size:255;default:''" json:"remark"`
	CreatedBy uint      `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`

	Creator    *User       `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	RedeemItem *RedeemItem `gorm:"foreignKey:ItemID" json:"redeem_item,omitempty"`
}
