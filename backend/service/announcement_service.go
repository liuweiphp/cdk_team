package service

import (
	"errors"
	"exchange_cdk/model"
	"exchange_cdk/pkg/sanitize"
	"time"

	"gorm.io/gorm"
)

type AnnouncementService struct{ db *gorm.DB }

func NewAnnouncementService(db *gorm.DB) *AnnouncementService {
	return &AnnouncementService{db: db}
}

// ListPublic 获取公开公告列表,置顶优先,按时间倒序
func (s *AnnouncementService) ListPublic(page, pageSize int) ([]model.Announcement, int64, error) {
	var list []model.Announcement
	var total int64
	s.db.Model(&model.Announcement{}).Count(&total)
	if err := s.db.Order("is_pinned DESC, created_at DESC").
		Offset((page-1)*pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Create 管理员创建公告,入口 content 已由 handler 净化
func (s *AnnouncementService) Create(title, content string, isPinned bool, createdBy uint) (*model.Announcement, error) {
	a := &model.Announcement{
		Title:     title,
		Content:   sanitize.HTML(content),
		IsPinned:  isPinned,
		CreatedBy: createdBy,
	}
	if err := s.db.Create(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}

// Update 更新公告
func (s *AnnouncementService) Update(id uint, title, content string, isPinned bool) error {
	updates := map[string]interface{}{
		"title":     title,
		"content":   sanitize.HTML(content),
		"is_pinned": isPinned,
		"updated_at": time.Now(),
	}
	return s.db.Model(&model.Announcement{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 软删除公告
func (s *AnnouncementService) Delete(id uint) error {
	return s.db.Delete(&model.Announcement{}, id).Error
}

// GetByID 获取单条公告
func (s *AnnouncementService) GetByID(id uint) (*model.Announcement, error) {
	var a model.Announcement
	if err := s.db.First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("公告不存在")
		}
		return nil, err
	}
	return &a, nil
}
