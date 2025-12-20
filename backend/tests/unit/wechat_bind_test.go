package unit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"prjflow/internal/api"
	"prjflow/internal/model"
	"prjflow/pkg/auth"
)

// CreateTestUserWithoutWeChat 创建没有微信OpenID的测试用户
func CreateTestUserWithoutWeChat(t *testing.T, db *gorm.DB, username, nickname string) *model.User {
	user := &model.User{
		Username: username,
		Nickname: nickname,
		Email:    username + "@test.com",
		Status:   1,
		// WeChatOpenID 为 nil
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user
}

func TestAuthHandler_GetWeChatBindQRCode(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	// 创建测试用户（未绑定微信）
	user := CreateTestUserWithoutWeChat(t, db, "binduser", "绑定用户")

	// 生成JWT Token
	token, err := auth.GenerateToken(user.ID, user.Username, []string{})
	require.NoError(t, err)

	handler := api.NewAuthHandler(db)

	t.Run("获取绑定二维码成功", func(t *testing.T) {
		// 设置微信配置（GetWeChatBindQRCode需要）
		appIDConfig := model.SystemConfig{
			Key:   "wechat_app_id",
			Value: "test_app_id",
			Type:  "string",
		}
		db.Create(&appIDConfig)

		appSecretConfig := model.SystemConfig{
			Key:   "wechat_app_secret",
			Value: "test_app_secret",
			Type:  "string",
		}
		db.Create(&appSecretConfig)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/wechat/bind/qrcode", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)
		// 设置user_id到上下文（模拟Auth中间件）
		c.Set("user_id", user.ID)

		handler.GetWeChatBindQRCode(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["ticket"])
		assert.NotEmpty(t, data["qr_code_url"])
		assert.NotEmpty(t, data["auth_url"])
	})

	t.Run("未登录用户无法获取二维码", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/wechat/bind/qrcode", nil)
		// 不设置Authorization头

		handler.GetWeChatBindQRCode(c)

		// 应该返回401或错误
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusUnauthorized || (response["code"] != nil && response["code"] != float64(200)))
	})
}

func TestAuthHandler_WeChatBindCallback(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	// 创建测试用户（未绑定微信）
	user := CreateTestUserWithoutWeChat(t, db, "callbackuser", "回调用户")

	handler := api.NewAuthHandler(db)

	// 注意：这个测试需要模拟微信回调，实际测试中需要mock微信客户端
	// 这里只测试基本的参数验证和错误处理

	t.Run("缺少code参数", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		// state格式：bind:{ticket}:{user_id}
		state := fmt.Sprintf("bind:ticket123:%d", user.ID)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/wechat/bind/callback?state="+state, nil)

		handler.WeChatBindCallback(c)

		// 应该返回错误页面（HTML）
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "html")
	})

	t.Run("缺少state参数", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/wechat/bind/callback?code=testcode", nil)

		handler.WeChatBindCallback(c)

		// 应该返回错误页面
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "html")
	})
}

func TestAuthHandler_UnbindWeChat(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	// 创建已绑定微信的用户
	wechatOpenID := "test_bind_openid"
	user := &model.User{
		Username:     "bounduser",
		Nickname:     "已绑定用户",
		Email:        "bounduser@test.com",
		Status:       1,
		WeChatOpenID: &wechatOpenID,
	}
	require.NoError(t, db.Create(user).Error)

	// 生成JWT Token
	token, err := auth.GenerateToken(user.ID, user.Username, []string{})
	require.NoError(t, err)

	handler := api.NewAuthHandler(db)

	t.Run("解绑微信成功", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/wechat/unbind", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)
		c.Set("user_id", user.ID)

		handler.UnbindWeChat(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		// 验证用户的wechat_open_id已被清空
		var updatedUser model.User
		require.NoError(t, db.First(&updatedUser, user.ID).Error)
		assert.Nil(t, updatedUser.WeChatOpenID)
	})

	t.Run("解绑未绑定的微信", func(t *testing.T) {
		// 创建未绑定微信的用户
		unboundUser := CreateTestUserWithoutWeChat(t, db, "unbounduser", "未绑定用户")
		unboundToken, err := auth.GenerateToken(unboundUser.ID, unboundUser.Username, []string{})
		require.NoError(t, err)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/wechat/unbind", nil)
		c.Request.Header.Set("Authorization", "Bearer "+unboundToken)
		c.Set("user_id", unboundUser.ID)

		handler.UnbindWeChat(c)

		// 应该返回错误（未绑定）
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, response["code"] != nil && response["code"] != float64(200))
	})

	t.Run("未登录用户无法解绑", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/wechat/unbind", nil)
		// 不设置Authorization头

		handler.UnbindWeChat(c)

		// 应该返回401或错误
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusUnauthorized || (response["code"] != nil && response["code"] != float64(200)))
	})
}

