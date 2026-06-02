package service

import (
	"errors"
	"exchange_cdk/model"
	"exchange_cdk/pkg/bcrypt"
	myjwt "exchange_cdk/pkg/jwt"
	"time"

	"gorm.io/gorm"
)

type AuthService struct {
	db        *gorm.DB
	jwtSecret string
	bcryptCst int
}

func NewAuthService(db *gorm.DB, jwtSecret string, bcryptCost int) *AuthService {
	return &AuthService{db: db, jwtSecret: jwtSecret, bcryptCst: bcryptCost}
}

// Login 验证用户名密码,返回 JWT token 和用户信息
func (s *AuthService) Login(username, password string) (string, *model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("用户名或密码错误")
		}
		return "", nil, err
	}
	if user.Status != "active" {
		return "", nil, errors.New("账户已被禁用")
	}
	if !bcrypt.Compare(password, user.PasswordHash) {
		return "", nil, errors.New("用户名或密码错误")
	}

	now := time.Now()
	user.LastLoginAt = &now
	s.db.Model(&user).Update("last_login_at", now)

	token, err := myjwt.Generate(s.jwtSecret, user.ID, user.Username, user.Role)
	if err != nil {
		return "", nil, err
	}
	return token, &user, nil
}
