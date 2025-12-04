package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"project-management/internal/api"
	"project-management/internal/model"
	"project-management/pkg/wechat"
	"project-management/tests/unit/mocks"
)

func TestInitCallbackHandler_HandleCallback(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	handler := api.NewInitCallbackHandler(db)

	t.Run("缺少code参数", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/init/callback?state=ticket:test123", nil)

		handler.HandleCallback(c)

		// 应该返回错误页面（HTML）
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "html")
		assert.Contains(t, w.Body.String(), "初始化失败")
	})

	t.Run("缺少state参数", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/init/callback?code=testcode", nil)

		handler.HandleCallback(c)

		// 应该返回错误页面（HTML）
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "html")
		assert.Contains(t, w.Body.String(), "初始化失败")
	})

	t.Run("微信配置未设置", func(t *testing.T) {
		// 确保没有微信配置
		db.Where("key IN ?", []string{"wechat_app_id", "wechat_app_secret"}).Delete(&model.SystemConfig{})

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/init/callback?code=testcode&state=ticket:test123", nil)

		handler.HandleCallback(c)

		// 应该返回错误页面（HTML）
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "html")
		assert.Contains(t, w.Body.String(), "初始化失败")
	})

	t.Run("系统已初始化", func(t *testing.T) {
		// 清理之前的数据
		db.Where("key = ?", "initialized").Delete(&model.SystemConfig{})
		db.Where("key IN ?", []string{"wechat_app_id", "wechat_app_secret"}).Delete(&model.SystemConfig{})

		// 设置系统已初始化
		initConfig := model.SystemConfig{
			Key:   "initialized",
			Value: "true",
			Type:  "boolean",
		}
		db.Create(&initConfig)

		// 设置微信配置
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
		c.Request = httptest.NewRequest(http.MethodGet, "/api/init/callback?code=testcode&state=ticket:test123", nil)

		handler.HandleCallback(c)

		// 应该返回错误页面（HTML）
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "html")
		assert.Contains(t, w.Body.String(), "初始化失败")
	})

	t.Run("HandleCallback成功-完整流程", func(t *testing.T) {
		// 清理之前的数据（包括之前测试创建的数据）
		// 先删除关联关系
		db.Exec("DELETE FROM user_roles")
		// 删除所有用户（包括之前测试创建的）
		db.Exec("DELETE FROM users")
		// 删除所有角色
		db.Exec("DELETE FROM roles")
		// 确保删除所有initialized配置（可能有多个测试创建了）
		db.Exec("DELETE FROM system_configs WHERE key = 'initialized'")
		// 也清理微信配置，避免冲突
		db.Exec("DELETE FROM system_configs WHERE key IN ('wechat_app_id', 'wechat_app_secret')")

		// 设置微信配置
		appIDConfig := model.SystemConfig{
			Key:   "wechat_app_id",
			Value: "test_app_id_handle",
			Type:  "string",
		}
		db.Create(&appIDConfig)

		appSecretConfig := model.SystemConfig{
			Key:   "wechat_app_secret",
			Value: "test_app_secret_handle",
			Type:  "string",
		}
		db.Create(&appSecretConfig)

		// 创建Handler并替换WeChatClient为Mock
		handler := api.NewInitCallbackHandler(db)
		mockWeChatClient := mocks.NewMockWeChatClient()
		
		// 配置Mock返回值
		mockWeChatClient.AccessTokenResponse = &wechat.AccessTokenResponse{
			AccessToken:  "test_access_token_handle",
			ExpiresIn:    7200,
			RefreshToken: "test_refresh_token_handle",
			OpenID:       "test_open_id_handle",
			Scope:        "snsapi_userinfo",
			UnionID:      "test_union_id_handle",
		}

		mockWeChatClient.UserInfoResponse = &wechat.UserInfoResponse{
			OpenID:     "test_open_id_handle",
			Nickname:   "测试管理员",
			Sex:        1,
			Province:   "广东",
			City:       "深圳",
			Country:    "中国",
			HeadImgURL: "http://example.com/admin_handle.jpg",
			Privilege:  []string{},
			UnionID:    "test_union_id_handle",
		}

		// 替换WeChatClient（使用Setter方法）
		handler.SetWeChatClient(mockWeChatClient)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/init/callback?code=testcode_handle&state=ticket:test_ticket_handle", nil)

		handler.HandleCallback(c)

		// 验证返回成功HTML
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "html")
		assert.Contains(t, w.Body.String(), "系统初始化成功")
		
		// 验证WeChatClient方法被调用
		assert.Equal(t, 1, mockWeChatClient.GetAccessTokenCallCount)
		assert.Equal(t, 1, mockWeChatClient.GetUserInfoCallCount)

		// 验证管理员用户已创建
		var adminUser model.User
		err := db.Where("wechat_open_id = ?", "test_open_id_handle").First(&adminUser).Error
		assert.NoError(t, err)
		assert.Equal(t, "测试管理员", adminUser.Nickname)

		// 验证系统已标记为初始化
		var initConfig model.SystemConfig
		err = db.Where("key = ?", "initialized").First(&initConfig).Error
		assert.NoError(t, err)
		assert.Equal(t, "true", initConfig.Value)
	})
}

