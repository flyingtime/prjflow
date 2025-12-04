package unit

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"project-management/internal/api"
	"project-management/internal/model"
	"project-management/pkg/wechat"
	"project-management/tests/unit/mocks"
)

func TestInitCallbackHandlerImpl_Validate(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	// 注意：InitCallbackHandlerImpl的db字段是私有的，但Validate方法使用ctx.DB
	// 所以我们可以直接创建handler实例，只测试Validate方法
	handler := &api.InitCallbackHandlerImpl{}

	t.Run("验证成功-系统未初始化且微信配置已保存", func(t *testing.T) {
		// 设置微信配置
		appIDConfig := model.SystemConfig{
			Key:   "wechat_app_id",
			Value: "test_app_id",
			Type:  "string",
		}
		db.Create(&appIDConfig)

		ctx := &api.WeChatCallbackContext{
			DB: db,
		}

		err := handler.Validate(ctx)
		assert.NoError(t, err)
	})

	t.Run("验证失败-系统已初始化", func(t *testing.T) {
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

		ctx := &api.WeChatCallbackContext{
			DB: db,
		}

		err := handler.Validate(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "系统已经初始化")
	})

	t.Run("验证失败-微信配置未保存", func(t *testing.T) {
		// 确保没有初始化配置和微信配置
		db.Where("key IN ?", []string{"initialized", "wechat_app_id", "wechat_app_secret"}).Delete(&model.SystemConfig{})

		ctx := &api.WeChatCallbackContext{
			DB: db,
		}

		err := handler.Validate(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "请先配置微信AppID和AppSecret")
	})
}

func TestInitCallbackHandlerImpl_GetSuccessHTML(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	handler := &api.InitCallbackHandlerImpl{}

	t.Run("获取成功HTML", func(t *testing.T) {
		ctx := &api.WeChatCallbackContext{
			DB: db,
		}

		html := handler.GetSuccessHTML(ctx, nil)
		assert.Contains(t, html, "系统初始化成功")
		assert.Contains(t, html, "请返回 PC 端查看")
	})
}

func TestInitCallbackHandlerImpl_GetErrorHTML(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	handler := &api.InitCallbackHandlerImpl{}

	t.Run("获取错误HTML", func(t *testing.T) {
		ctx := &api.WeChatCallbackContext{
			DB: db,
		}

		err := &api.CallbackError{Message: "测试错误"}
		html := handler.GetErrorHTML(ctx, err)
		assert.Contains(t, html, "初始化失败")
		assert.Contains(t, html, "测试错误")
	})
}

func TestInitCallbackHandlerImpl_Process(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	// 注意：InitCallbackHandlerImpl的db字段是私有的，但Process方法使用ctx.DB
	// 所以我们可以直接创建handler实例
	handler := &api.InitCallbackHandlerImpl{}
	mockHub := mocks.NewMockWebSocketHub()

	t.Run("Process成功-创建管理员并初始化系统", func(t *testing.T) {
		// 确保系统未初始化
		db.Where("key = ?", "initialized").Delete(&model.SystemConfig{})

		// 创建WeChatCallbackContext
		ctx := &api.WeChatCallbackContext{
			DB:   db,
			Hub:  mockHub,
			Ticket: "test_ticket_123",
			UserInfo: &wechat.UserInfoResponse{
				OpenID:     "test_open_id",
				Nickname:   "管理员",
				Sex:        1,
				Province:   "广东",
				City:       "深圳",
				Country:    "中国",
				HeadImgURL: "http://example.com/admin.jpg",
				Privilege:  []string{},
				UnionID:    "test_union_id",
			},
		}

		result, err := handler.Process(ctx)

		// 验证没有错误
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// 验证管理员角色已创建
		var adminRole model.Role
		err = db.Where("code = ?", "admin").First(&adminRole).Error
		assert.NoError(t, err)
		assert.Equal(t, "管理员", adminRole.Name)

		// 验证管理员用户已创建
		var adminUser model.User
		err = db.Where("wechat_open_id = ?", "test_open_id").First(&adminUser).Error
		assert.NoError(t, err)
		assert.Equal(t, "管理员", adminUser.Nickname)
		assert.Equal(t, "http://example.com/admin.jpg", adminUser.Avatar)

		// 验证用户已分配管理员角色
		var roles []model.Role
		db.Model(&adminUser).Association("Roles").Find(&roles)
		assert.Greater(t, len(roles), 0)
		assert.Equal(t, "admin", roles[0].Code)

		// 验证系统已标记为初始化
		var initConfig model.SystemConfig
		err = db.Where("key = ?", "initialized").First(&initConfig).Error
		assert.NoError(t, err)
		assert.Equal(t, "true", initConfig.Value)

		// 验证WebSocket消息被发送
		messages := mockHub.GetMessagesByTicket("test_ticket_123")
		assert.Greater(t, len(messages), 0)
		assert.True(t, mockHub.HasMessage("test_ticket_123", "success", "系统初始化成功"))

		// 验证返回结果包含token和user
		resultMap, ok := result.(gin.H)
		assert.True(t, ok)
		assert.NotNil(t, resultMap["token"])
		assert.NotNil(t, resultMap["user"])
	})

	t.Run("Process成功-管理员角色已存在", func(t *testing.T) {
		// 清理之前的数据（包括第一个测试创建的数据）
		// 先删除用户和角色的关联关系
		db.Exec("DELETE FROM user_roles")
		// 删除用户
		db.Where("wechat_open_id IN ?", []string{"test_open_id", "test_open_id_2"}).Unscoped().Delete(&model.User{})
		// 删除所有角色（使用Unscoped确保完全删除，避免名称冲突）
		db.Exec("DELETE FROM roles")
		// 删除初始化配置
		db.Where("key = ?", "initialized").Delete(&model.SystemConfig{})

		// 先创建一个管理员角色（使用与Process方法相同的名称"管理员"）
		adminRole := model.Role{
			Name:        "管理员",
			Code:        "admin",
			Description: "系统管理员",
			Status:      1,
		}
		db.Create(&adminRole)

		mockHub.Reset()

		// 创建WeChatCallbackContext
		ctx := &api.WeChatCallbackContext{
			DB:   db,
			Hub:  mockHub,
			Ticket: "test_ticket_456",
			UserInfo: &wechat.UserInfoResponse{
				OpenID:     "test_open_id_2",
				Nickname:   "管理员2",
				Sex:        1,
				Province:   "广东",
				City:       "深圳",
				Country:    "中国",
				HeadImgURL: "http://example.com/admin2.jpg",
				Privilege:  []string{},
				UnionID:    "test_union_id_2",
			},
		}

		result, err := handler.Process(ctx)

		// 验证没有错误
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// 验证管理员用户已创建
		var adminUser model.User
		err = db.Where("wechat_open_id = ?", "test_open_id_2").First(&adminUser).Error
		assert.NoError(t, err)

		// 验证系统已标记为初始化
		var initConfig model.SystemConfig
		err = db.Where("key = ?", "initialized").First(&initConfig).Error
		assert.NoError(t, err)
	})
}

// 注意：Process方法的错误场景测试在TestInitCallbackHandlerImpl_Process_Errors中

