package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Username         string         `gorm:"size:64;uniqueIndex" json:"username"`
	PasswordHash     string         `gorm:"size:255" json:"-"`
	Role             string         `gorm:"type:enum('admin','user');default:user" json:"role"`
	Status           string         `gorm:"type:enum('active','disabled');default:active" json:"status"`
	FilePrefix       string         `gorm:"size:64;default:''" json:"file_prefix"`
	FileSequenceNext uint           `gorm:"default:1001" json:"file_sequence_next"`
	LastLoginAt      *time.Time     `json:"last_login_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}
