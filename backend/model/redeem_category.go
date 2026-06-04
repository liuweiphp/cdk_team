package model

import (
	"time"

	"gorm.io/gorm"
)

type RedeemCategory struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:128" json:"name"`
	Status    string         `gorm:"type:enum('active','disabled');default:active" json:"status"`
	CreatedBy uint           `json:"created_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Creator *User `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}
