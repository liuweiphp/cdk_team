// Package bcrypt 提供密码哈希与验证
package bcrypt

import (
	"golang.org/x/crypto/bcrypt"
)

// Hash 使用给定 cost 生成 bcrypt 哈希
func Hash(password string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Compare 验证密码与哈希是否匹配
func Compare(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
