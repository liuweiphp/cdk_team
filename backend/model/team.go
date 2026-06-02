package model

import (
	"time"

	"gorm.io/gorm"
)

type Team struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	OwnerID   uint           `gorm:"uniqueIndex" json:"owner_id"`
	Name      string         `gorm:"size:128" json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Owner   *User        `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members []TeamMember `gorm:"foreignKey:TeamID" json:"members,omitempty"`
}

type TeamMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TeamID    uint      `gorm:"uniqueIndex:idx_team_member" json:"team_id"`
	MemberID  uint      `gorm:"uniqueIndex:idx_team_member" json:"member_id"`
	CreatedAt time.Time `json:"created_at"`

	Team   *Team `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	Member *User `gorm:"foreignKey:MemberID" json:"member,omitempty"`
}
