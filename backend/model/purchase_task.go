package model

import (
	"time"

	"gorm.io/gorm"
)

type TeamTemplateSequence struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TeamOwnerID uint      `gorm:"uniqueIndex:idx_team_template_sequence" json:"team_owner_id"`
	TemplateID  uint      `gorm:"uniqueIndex:idx_team_template_sequence" json:"template_id"`
	CurrentSeq  uint      `gorm:"default:0" json:"current_seq"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	TeamOwner *User           `gorm:"foreignKey:TeamOwnerID" json:"team_owner,omitempty"`
	Template  *RedeemTemplate `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
}

type ExternalAccountSequence struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Provider   string    `gorm:"size:64;uniqueIndex;default:yfjc" json:"provider"`
	CurrentSeq uint      `gorm:"default:1000" json:"current_seq"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type PurchaseTask struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	TeamOwnerID        uint           `gorm:"index:idx_purchase_owner_template_seq,priority:1;index:idx_purchase_owner_status_created,priority:1" json:"team_owner_id"`
	TemplateID         uint           `gorm:"index:idx_purchase_owner_template_seq,priority:2;index:idx_purchase_template_status_created,priority:1" json:"template_id"`
	RedeemItemID       *uint          `gorm:"uniqueIndex" json:"redeem_item_id"`
	CdkID              *uint          `gorm:"uniqueIndex" json:"cdk_id"`
	CreatedBy          uint           `gorm:"index" json:"created_by"`
	AccountPrefix      string         `gorm:"size:64;default:''" json:"account_prefix"`
	AccountName        string         `gorm:"size:128;default:''" json:"account_name"`
	TemplateCodePart   string         `gorm:"size:128;default:''" json:"template_code_part"`
	SequenceNo         uint           `gorm:"uniqueIndex:idx_purchase_owner_template_seq,priority:3" json:"sequence_no"`
	TargetCode         string         `gorm:"size:64;default:''" json:"target_code"`
	TargetName         string         `gorm:"size:128;default:''" json:"target_name"`
	Provider           string         `gorm:"size:64;default:yfjc" json:"provider"`
	ExternalAccountSeq uint           `gorm:"default:0" json:"external_account_seq"`
	ExternalUsername   string         `gorm:"size:128;default:''" json:"external_username"`
	ExternalPassword   string         `gorm:"size:128;default:''" json:"external_password"`
	Source             string         `gorm:"size:32;default:manual;index:idx_purchase_source_created,priority:1" json:"source"`
	Status             string         `gorm:"type:enum('pending','registering','ordering','entry_challenge_required','pending_payment','fetching_subscribe','ready','needs_manual_review','manual_completed','failed');default:pending;index:idx_purchase_owner_status_created,priority:2;index:idx_purchase_template_status_created,priority:2" json:"status"`
	RetryCount         uint           `gorm:"default:0" json:"retry_count"`
	PaymentStatus      string         `gorm:"type:enum('unpaid','paid','unknown');default:unpaid" json:"payment_status"`
	ManualReviewReason string         `gorm:"size:255;default:''" json:"manual_review_reason"`
	ExternalOrderNo    string         `gorm:"size:128;default:''" json:"external_order_no"`
	SubscribeURL       string         `gorm:"type:text" json:"subscribe_url"`
	LastError          *string        `gorm:"type:text" json:"last_error"`
	BrowserTracePath   string         `gorm:"size:255;default:''" json:"browser_trace_path"`
	ScreenshotPath     string         `gorm:"size:255;default:''" json:"screenshot_path"`
	HTMLDumpPath       string         `gorm:"size:255;default:''" json:"html_dump_path"`
	PayloadJSON        *string        `gorm:"type:longtext" json:"payload_json"`
	CreatedAt          time.Time      `gorm:"index:idx_purchase_source_created,priority:2" json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	TeamOwner  *User           `gorm:"foreignKey:TeamOwnerID" json:"team_owner,omitempty"`
	Template   *RedeemTemplate `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	RedeemItem *RedeemItem     `gorm:"foreignKey:RedeemItemID" json:"redeem_item,omitempty"`
	Cdk        *Cdk            `gorm:"foreignKey:CdkID" json:"cdk,omitempty"`
	Creator    *User           `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}
