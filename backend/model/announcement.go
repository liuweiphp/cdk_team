package model

import (
	"time"

	"gorm.io/gorm"
)

type Announcement struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Title     string         `gorm:"size:255" json:"title"`
	Content   string         `gorm:"type:text" json:"content"`
	IsPinned  bool           `gorm:"default:false" json:"is_pinned"`
	CreatedBy uint           `json:"created_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Creator *User `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}
