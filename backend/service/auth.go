package service

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vpn-location/backend/config"
	"golang.org/x/crypto/bcrypt"
)

// Claims JWT声明
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// AuthService 认证服务
type AuthService struct{}

var authService *AuthService
var authOnce sync.Once

// GetAuthService 获取认证服务单例
func GetAuthService() *AuthService {
	authOnce.Do(func() {
		authService = &AuthService{}
	})
	return authService
}

// GenerateToken 生成JWT令牌
func (s *AuthService) GenerateToken(userID uint, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(config.Config.Auth.ExpireHours))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Config.Auth.JWTSecret))
}

// ParseToken 解析JWT令牌
func (s *AuthService) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Config.Auth.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的令牌")
}

// HashPassword 密码 bcrypt 哈希
func (s *AuthService) HashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return password // fallback，不应该发生
	}
	return string(hash)
}

// VerifyPassword 验证 bcrypt 密码
func (s *AuthService) VerifyPassword(hashedPassword, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}
