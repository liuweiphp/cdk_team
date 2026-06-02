package model

import (
	"time"

	"gorm.io/gorm"
)

type RedeemItem struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Name       string         `gorm:"size:128" json:"name"`
	Filename   string         `gorm:"size:255" json:"filename"`
	Content    string         `gorm:"type:text" json:"content"`
	TemplateID *uint          `json:"template_id"`
	Status     string         `gorm:"type:enum('active','disabled');default:active" json:"status"`
	CreatedBy  uint           `json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	Creator  *User           `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Template *RedeemTemplate `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	Cdk      *Cdk            `gorm:"foreignKey:ItemID" json:"cdk,omitempty"`
}
