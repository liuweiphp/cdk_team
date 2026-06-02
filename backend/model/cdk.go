package model

import "time"

type Cdk struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Code        string     `gorm:"size:64;uniqueIndex" json:"code"`
	Amount      float64    `gorm:"type:decimal(12,2)" json:"amount"`
	ItemID      *uint      `json:"item_id"`
	Status      string     `gorm:"type:enum('unused','exchanged');default:unused" json:"status"`
	ImportID    uint       `json:"import_id"`
	ExchangedBy *uint      `json:"exchanged_by"`
	ExchangedAt *time.Time `json:"exchanged_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	ImportRecord *CdkImport  `gorm:"foreignKey:ImportID" json:"import_record,omitempty"`
	RedeemItem   *RedeemItem `gorm:"foreignKey:ItemID" json:"redeem_item,omitempty"`
	Exchanger    *User       `gorm:"foreignKey:ExchangedBy" json:"exchanger,omitempty"`
}
