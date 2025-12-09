package auth

import (
	"errors"
	"time"

	"project-management/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID    uint     `json:"user_id"`
	Username  string   `json:"username"`
	Roles     []string `json:"roles"`
	TokenType string   `json:"token_type"` // "access" 或 "refresh"
	jwt.RegisteredClaims
}

// getJWTSecret 获取JWT密钥
func getJWTSecret() []byte {
	if config.AppConfig == nil {
		return []byte("default-secret-key-change-in-production")
	}
	return []byte(config.AppConfig.JWT.Secret)
}

// GenerateToken 生成JWT Token (Access Token)
func GenerateToken(userID uint, username string, roles []string) (string, error) {
	if config.AppConfig == nil {
		return "", errors.New("config not initialized")
	}

	// Access Token 有效期从配置文件读取（默认 24 小时）
	expirationTime := time.Now().Add(time.Duration(config.AppConfig.JWT.Expiration) * time.Hour)

	claims := &Claims{
		UserID:    userID,
		Username:  username,
		Roles:     roles,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// GenerateRefreshToken 生成Refresh Token
// Refresh Token 的有效期通常比 Access Token 长得多（例如 7 天或 30 天）
func GenerateRefreshToken(userID uint, username string, roles []string) (string, error) {
	if config.AppConfig == nil {
		return "", errors.New("config not initialized")
	}

	// Refresh Token 有效期设置为 7 天
	expirationTime := time.Now().Add(7 * 24 * time.Hour)

	claims := &Claims{
		UserID:    userID,
		Username:  username,
		Roles:     roles,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// ParseToken 解析JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid token signing method")
		}
		return getJWTSecret(), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
