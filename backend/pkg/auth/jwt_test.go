package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prjflow/internal/config"
)

func TestGenerateToken(t *testing.T) {
	// 初始化JWT配置
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{
			JWT: config.JWTConfig{
				Secret:     "test-secret-key-for-unit-testing",
				Expiration: 24,
			},
		}
	}

	t.Run("生成Token成功", func(t *testing.T) {
		token, err := GenerateToken(1, "testuser", []string{"admin"})
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// 验证Token可以解析
		claims, err := ParseToken(token)
		require.NoError(t, err)
		assert.Equal(t, uint(1), claims.UserID)
		assert.Equal(t, "testuser", claims.Username)
		assert.Equal(t, []string{"admin"}, claims.Roles)
		assert.Equal(t, "access", claims.TokenType) // 验证Token类型为access
		assert.NotNil(t, claims.ExpiresAt)
	})

	t.Run("生成Token失败-配置未初始化", func(t *testing.T) {
		// 临时保存配置
		originalConfig := config.AppConfig
		config.AppConfig = nil

		token, err := GenerateToken(1, "testuser", []string{"admin"})
		assert.Error(t, err)
		assert.Empty(t, token)

		// 恢复配置
		config.AppConfig = originalConfig
	})
}

func TestGenerateRefreshToken(t *testing.T) {
	// 初始化JWT配置
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{
			JWT: config.JWTConfig{
				Secret:     "test-secret-key-for-unit-testing",
				Expiration: 24,
			},
		}
	}

	t.Run("生成RefreshToken成功", func(t *testing.T) {
		refreshToken, err := GenerateRefreshToken(1, "testuser", []string{"admin"})
		require.NoError(t, err)
		assert.NotEmpty(t, refreshToken)

		// 验证RefreshToken可以解析
		claims, err := ParseToken(refreshToken)
		require.NoError(t, err)
		assert.Equal(t, uint(1), claims.UserID)
		assert.Equal(t, "testuser", claims.Username)
		assert.Equal(t, []string{"admin"}, claims.Roles)
		assert.Equal(t, "refresh", claims.TokenType) // 验证Token类型为refresh
		assert.NotNil(t, claims.ExpiresAt)

		// 验证RefreshToken的有效期比AccessToken长（RefreshToken是7天，AccessToken是24小时）
		// 这里只验证RefreshToken的过期时间在未来
		assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
		assert.True(t, claims.ExpiresAt.Time.After(time.Now().Add(6*24*time.Hour))) // 至少6天后过期
	})

	t.Run("生成RefreshToken失败-配置未初始化", func(t *testing.T) {
		// 临时保存配置
		originalConfig := config.AppConfig
		config.AppConfig = nil

		refreshToken, err := GenerateRefreshToken(1, "testuser", []string{"admin"})
		assert.Error(t, err)
		assert.Empty(t, refreshToken)

		// 恢复配置
		config.AppConfig = originalConfig
	})

	t.Run("RefreshToken和AccessToken不同", func(t *testing.T) {
		accessToken, err := GenerateToken(1, "testuser", []string{"admin"})
		require.NoError(t, err)

		refreshToken, err := GenerateRefreshToken(1, "testuser", []string{"admin"})
		require.NoError(t, err)

		// 验证两个token不同
		assert.NotEqual(t, accessToken, refreshToken)

		// 验证两个token都可以解析
		accessClaims, err := ParseToken(accessToken)
		require.NoError(t, err)

		refreshClaims, err := ParseToken(refreshToken)
		require.NoError(t, err)

		// 验证用户信息相同
		assert.Equal(t, accessClaims.UserID, refreshClaims.UserID)
		assert.Equal(t, accessClaims.Username, refreshClaims.Username)
		assert.Equal(t, accessClaims.Roles, refreshClaims.Roles)

		// 验证Token类型不同
		assert.Equal(t, "access", accessClaims.TokenType)
		assert.Equal(t, "refresh", refreshClaims.TokenType)

		// 验证RefreshToken的过期时间更长
		assert.True(t, refreshClaims.ExpiresAt.Time.After(accessClaims.ExpiresAt.Time))
	})
}

func TestParseToken(t *testing.T) {
	// 初始化JWT配置
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{
			JWT: config.JWTConfig{
				Secret:     "test-secret-key-for-unit-testing",
				Expiration: 24,
			},
		}
	}

	t.Run("解析Token成功", func(t *testing.T) {
		token, err := GenerateToken(1, "testuser", []string{"admin"})
		require.NoError(t, err)

		claims, err := ParseToken(token)
		require.NoError(t, err)
		assert.Equal(t, uint(1), claims.UserID)
		assert.Equal(t, "testuser", claims.Username)
		assert.Equal(t, []string{"admin"}, claims.Roles)
	})

	t.Run("解析Token失败-无效的Token", func(t *testing.T) {
		claims, err := ParseToken("invalid-token")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("解析Token失败-空Token", func(t *testing.T) {
		claims, err := ParseToken("")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("解析Token失败-使用错误的密钥", func(t *testing.T) {
		// 使用一个密钥生成token
		originalSecret := config.AppConfig.JWT.Secret
		config.AppConfig.JWT.Secret = "secret1"
		token, err := GenerateToken(1, "testuser", []string{"admin"})
		require.NoError(t, err)

		// 使用另一个密钥解析token
		config.AppConfig.JWT.Secret = "secret2"
		claims, err := ParseToken(token)
		assert.Error(t, err)
		assert.Nil(t, claims)

		// 恢复密钥
		config.AppConfig.JWT.Secret = originalSecret
	})
}

