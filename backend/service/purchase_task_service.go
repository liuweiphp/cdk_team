package service

import (
	"errors"
	"exchange_cdk/model"
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AutomationRunner 在自动化流程落地前仅作为构造参数占位。
type AutomationRunner struct{}

type CreatePendingTaskInput struct {
	TeamOwnerID   uint
	TemplateID    uint
	RedeemItemID  uint
	CdkID         uint
	CreatedBy     uint
	AccountPrefix string
	TemplateCode  string
	TargetCode    string
	TargetName    string
	Provider      string
}

type PurchaseTaskService struct {
	db     *gorm.DB
	runner *AutomationRunner

	mu        sync.Mutex
	sequences map[string]uint
}

func NewPurchaseTaskService(db *gorm.DB, runner *AutomationRunner) *PurchaseTaskService {
	return &PurchaseTaskService{
		db:        db,
		runner:    runner,
		sequences: make(map[string]uint),
	}
}

func (s *PurchaseTaskService) AllocateNextSequence(teamOwnerID, templateID uint) (uint, error) {
	if teamOwnerID == 0 || templateID == 0 {
		return 0, errors.New("team owner and template are required")
	}
	if s.db == nil {
		return s.allocateNextSequenceInMemory(teamOwnerID, templateID), nil
	}

	var next uint
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var seq model.TeamTemplateSequence
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("team_owner_id = ? AND template_id = ?", teamOwnerID, templateID).
			First(&seq).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			seq = model.TeamTemplateSequence{
				TeamOwnerID: teamOwnerID,
				TemplateID:  templateID,
				CurrentSeq:  0,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seq).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("team_owner_id = ? AND template_id = ?", teamOwnerID, templateID).
				First(&seq).Error; err != nil {
				return err
			}
		}

		next = seq.CurrentSeq + 1
		return tx.Model(&model.TeamTemplateSequence{}).
			Where("id = ?", seq.ID).
			Update("current_seq", next).Error
	})
	if err != nil {
		return 0, err
	}
	return next, nil
}

func (s *PurchaseTaskService) CreatePendingTask(in CreatePendingTaskInput) (*model.PurchaseTask, error) {
	sequenceNo, err := s.AllocateNextSequence(in.TeamOwnerID, in.TemplateID)
	if err != nil {
		return nil, err
	}

	provider := in.Provider
	if provider == "" {
		provider = "yfjc"
	}

	task := &model.PurchaseTask{
		TeamOwnerID:      in.TeamOwnerID,
		TemplateID:       in.TemplateID,
		CdkID:            in.CdkID,
		CreatedBy:        in.CreatedBy,
		AccountPrefix:    in.AccountPrefix,
		AccountName:      buildPurchaseTaskAccountName(in.AccountPrefix, in.TemplateCode, sequenceNo),
		TemplateCodePart: in.TemplateCode,
		SequenceNo:       sequenceNo,
		TargetCode:       in.TargetCode,
		TargetName:       in.TargetName,
		Provider:         provider,
		Status:           "pending",
		PaymentStatus:    "unpaid",
	}
	if in.RedeemItemID != 0 {
		redeemItemID := in.RedeemItemID
		task.RedeemItemID = &redeemItemID
	}

	if s.db == nil {
		return task, nil
	}
	if err := s.db.Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

func (s *PurchaseTaskService) allocateNextSequenceInMemory(teamOwnerID, templateID uint) uint {
	key := fmt.Sprintf("%d:%d", teamOwnerID, templateID)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sequences[key]++
	return s.sequences[key]
}

func buildPurchaseTaskAccountName(accountPrefix, templateCode string, sequenceNo uint) string {
	parts := make([]string, 0, 3)
	if trimmed := strings.TrimSpace(accountPrefix); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if trimmed := strings.TrimSpace(templateCode); trimmed != "" {
		parts = append(parts, trimmed)
	}
	parts = append(parts, fmt.Sprintf("%04d", sequenceNo))
	return strings.Join(parts, "-")
}
