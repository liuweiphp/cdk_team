package service

import (
	"errors"
	"exchange_cdk/model"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db        *gorm.DB
	bcryptCst int
}

func NewUserService(db *gorm.DB, bcryptCost int) *UserService {
	return &UserService{db: db, bcryptCst: bcryptCost}
}

// GetByID 获取用户
func (s *UserService) GetByID(id uint) (*model.User, error) {
	var u model.User
	if err := s.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// List 管理员获取用户列表
func (s *UserService) List(page, pageSize int, keyword, status string) ([]model.User, int64, error) {
	var list []model.User
	var total int64
	q := s.db.Model(&model.User{})
	if keyword != "" {
		q = q.Where("username LIKE ?", "%"+keyword+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Create 管理员创建用户
func (s *UserService) Create(username, password, role, filePrefix string) (*model.User, error) {
	if len(password) < 8 {
		return nil, errors.New("密码长度不能少于8位")
	}
	filePrefix = strings.TrimSpace(filePrefix)
	if !isValidFilePrefix(filePrefix) {
		return nil, errors.New("文件前缀只能包含字母、数字、-、_")
	}
	hash, err := hashPwd(password, s.bcryptCst)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Username:         username,
		PasswordHash:     hash,
		Role:             role,
		Status:           "active",
		FilePrefix:       filePrefix,
		FileSequenceNext: 1001,
	}
	if err := s.db.Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

// Update 管理员更新用户状态/角色/密码/文件前缀
func (s *UserService) Update(id uint, status, role, password, filePrefix *string) error {
	updates := map[string]interface{}{}
	if status != nil {
		updates["status"] = *status
	}
	if role != nil {
		updates["role"] = *role
	}
	if password != nil {
		if len(*password) < 8 {
			return errors.New("密码长度不能少于8位")
		}
		hash, err := hashPwd(*password, s.bcryptCst)
		if err != nil {
			return err
		}
		updates["password_hash"] = hash
	}
	if filePrefix != nil {
		prefix := strings.TrimSpace(*filePrefix)
		if !isValidFilePrefix(prefix) {
			return errors.New("文件前缀只能包含字母、数字、-、_")
		}
		updates["file_prefix"] = prefix
		updates["file_sequence_next"] = 1001
	}
	return s.db.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
}

func (s *UserService) UpdateFilePrefix(id uint, prefix string) (*model.User, error) {
	prefix = strings.TrimSpace(prefix)
	if !isValidFilePrefix(prefix) {
		return nil, errors.New("文件前缀只能包含字母、数字、-、_")
	}
	var user model.User
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
			"file_prefix":        prefix,
			"file_sequence_next": 1001,
		}).Error; err != nil {
			return err
		}
		return tx.First(&user, id).Error
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ChangePassword 修改自己的密码,需验证旧密码
func (s *UserService) ChangePassword(id uint, oldPwd, newPwd string) error {
	if len(newPwd) < 8 {
		return errors.New("新密码长度不能少于8位")
	}
	var u model.User
	if err := s.db.First(&u, id).Error; err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPwd)) != nil {
		return errors.New("旧密码错误")
	}
	hash, err := hashPwd(newPwd, s.bcryptCst)
	if err != nil {
		return err
	}
	return s.db.Model(&u).Update("password_hash", hash).Error
}

func isValidFilePrefix(prefix string) bool {
	for _, r := range prefix {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			continue
		default:
			return false
		}
	}
	return true
}

func hashPwd(pwd string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pwd), cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
