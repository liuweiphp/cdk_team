package model

import (
	"time"

	"gorm.io/gorm"
)

type RedeemTemplate struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Name               string         `gorm:"size:128" json:"name"`
	Content            string         `gorm:"type:text" json:"content"`
	ExternalTargetCode string         `gorm:"size:64;default:''" json:"external_target_code"`
	ExternalTargetName string         `gorm:"size:128;default:''" json:"external_target_name"`
	ExternalProvider   string         `gorm:"size:64;default:yfjc" json:"external_provider"`
	ResultContentMode  string         `gorm:"size:32;default:subscribe_url" json:"result_content_mode"`
	Status             string         `gorm:"type:enum('active','disabled');default:active" json:"status"`
	CreatedBy          uint           `json:"created_by"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	Creator *User `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}
