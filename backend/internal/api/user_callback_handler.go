package api

import (
	"gorm.io/gorm"
	"prjflow/internal/model"
	"prjflow/pkg/permission"

	"github.com/gin-gonic/gin"
)

// AddUserCallbackHandler 添加用户场景的微信回调处理
type AddUserCallbackHandler struct {
	db *gorm.DB
}

func (h *AddUserCallbackHandler) Validate(ctx *WeChatCallbackContext) error {
	// 从state中提取用户ID（格式：adduser:{ticket}:{user_id}）
	var userIDStr string
	if len(ctx.State) > 8 && ctx.State[:8] == "adduser:" {
		parts := ctx.State[8:] // 去掉 "adduser:" 前缀
		// 找到最后一个冒号，后面是user_id
		lastColonIndex := -1
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] == ':' {
				lastColonIndex = i
				break
			}
		}
		if lastColonIndex > 0 {
			userIDStr = parts[lastColonIndex+1:]
		}
	}
	
	if userIDStr == "" {
		return &CallbackError{Message: "缺少用户信息，无法验证权限"}
	}
	
	// 查询用户
	var user model.User
	if err := ctx.DB.First(&user, userIDStr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &CallbackError{Message: "用户不存在"}
		}
		return &CallbackError{Message: "查询用户失败", Err: err}
	}
	
	// 获取用户角色
	var roles []model.Role
	ctx.DB.Model(&user).Association("Roles").Find(&roles)
	
	roleCodes := make([]string, 0, len(roles))
	for _, role := range roles {
		roleCodes = append(roleCodes, role.Code)
	}
	
	// 检查用户是否有创建用户的权限
	hasPermission, err := permission.CheckPermissionWithDB(ctx.DB, roleCodes, "user:create")
	if err != nil {
		return &CallbackError{Message: "权限检查失败", Err: err}
	}
	
	if !hasPermission {
		return &CallbackError{Message: "没有权限添加用户，请联系管理员"}
	}
	
	return nil
}

func (h *AddUserCallbackHandler) Process(ctx *WeChatCallbackContext) (interface{}, error) {
	// 检查用户是否已存在（包括软删除的记录）
	var existingUser model.User
	result := ctx.DB.Unscoped().Where("wechat_open_id = ?", ctx.UserInfo.OpenID).First(&existingUser)
	if result.Error == nil {
		// 用户已存在（可能是软删除的）
		if existingUser.DeletedAt.Valid {
			// 如果是软删除的用户，恢复它并更新信息
			// 使用 Unscoped().Update 清除软删除标记并更新字段
			updates := map[string]interface{}{
				"deleted_at": nil,
				"avatar":     ctx.UserInfo.HeadImgURL,
			}
			// 更新昵称（如果微信昵称不为空）
			if ctx.UserInfo.Nickname != "" {
				updates["nickname"] = ctx.UserInfo.Nickname
			} else if existingUser.Nickname == "" {
				// 如果昵称为空，使用用户名
				updates["nickname"] = existingUser.Username
			}
			if err := ctx.DB.Unscoped().Model(&existingUser).Updates(updates).Error; err != nil {
				return nil, &CallbackError{Message: "恢复用户失败", Err: err}
			}
			// 重新加载用户信息
			ctx.DB.Preload("Department").Preload("Roles").First(&existingUser, existingUser.ID)
			
			// 返回恢复的用户信息
			if ctx.Ticket != "" && ctx.Hub != nil {
				ctx.Hub.SendMessage(ctx.Ticket, "success", gin.H{
					"user": gin.H{
						"id":            existingUser.ID,
						"username":      existingUser.Username,
						"nickname":      existingUser.Nickname,
						"email":         existingUser.Email,
						"avatar":        existingUser.Avatar,
						"wechat_open_id": existingUser.WeChatOpenID,
					},
				}, "用户已恢复")
			}
			return gin.H{
				"user": gin.H{
					"id":            existingUser.ID,
					"username":      existingUser.Username,
					"nickname":      existingUser.Nickname,
					"email":         existingUser.Email,
					"avatar":        existingUser.Avatar,
					"wechat_open_id": existingUser.WeChatOpenID,
				},
			}, nil
		} else {
			// 用户已存在且未删除
		return nil, &CallbackError{Message: "该微信用户已存在"}
		}
	} else if result.Error != gorm.ErrRecordNotFound {
		// 查询出错
		return nil, &CallbackError{Message: "查询用户失败", Err: result.Error}
	}

	// 确保昵称不为空（如果微信昵称为空，使用默认值）
	nickname := ctx.UserInfo.Nickname
	if nickname == "" {
		nickname = "用户"
	}

	// 生成唯一的用户名并创建用户（处理并发冲突）
	username := GenerateUniqueUsername(ctx.DB, nickname, ctx.UserInfo.OpenID)
	
	// 如果昵称为空，使用用户名
	if nickname == "" {
		nickname = username
	}

	// 创建新用户
	wechatOpenID := ctx.UserInfo.OpenID
	user := model.User{
		WeChatOpenID: &wechatOpenID,
		Username:     username,
		Nickname:     nickname, // 设置昵称（从微信昵称获取，如果为空则使用用户名）
		Avatar:       ctx.UserInfo.HeadImgURL,
		Status:       1,
	}
	
	err := ctx.DB.Create(&user).Error
	if err != nil {
		// 检查错误类型
		errStr := err.Error()
		isWeChatOpenIDError := errStr == "UNIQUE constraint failed: users.wechat_open_id" ||
			contains(errStr, "UNIQUE constraint failed: users.wechat_open_id") ||
			contains(errStr, "Duplicate entry") && contains(errStr, "wechat_open_id") ||
			contains(errStr, "duplicate key") && contains(errStr, "wechat_open_id")
		
		isUsernameError := errStr == "UNIQUE constraint failed: users.username" ||
			contains(errStr, "UNIQUE constraint failed: users.username") ||
			contains(errStr, "Duplicate entry") && contains(errStr, "username") ||
			contains(errStr, "duplicate key") && contains(errStr, "username")

		if isWeChatOpenIDError {
			// wechat_open_id 冲突：可能是同一用户并发添加，重新查询用户
			var existingUser model.User
			if err := ctx.DB.Where("wechat_open_id = ?", ctx.UserInfo.OpenID).First(&existingUser).Error; err == nil {
				// 用户已存在，直接使用
				user = existingUser
			} else {
				return nil, &CallbackError{Message: "创建用户失败，请稍后重试或联系管理员", Err: err}
			}
		} else if isUsernameError {
			// username 冲突：两个不同的 OpenID 后8位相同，提示联系管理员
			return nil, &CallbackError{Message: "用户名冲突，请联系管理员处理", Err: err}
		} else {
			// 其他错误
			return nil, &CallbackError{Message: "创建用户失败，请联系管理员", Err: err}
		}
	}

	// 加载用户信息（包含关联数据）
	ctx.DB.Preload("Department").Preload("Roles").First(&user, user.ID)

	// 通过WebSocket通知成功
	if ctx.Ticket != "" && ctx.Hub != nil {
		ctx.Hub.SendMessage(ctx.Ticket, "info", nil, "用户添加成功")
		ctx.Hub.SendMessage(ctx.Ticket, "success", gin.H{
			"user": gin.H{
				"id":            user.ID,
				"username":      user.Username,
				"nickname":      user.Nickname,
				"email":         user.Email,
				"avatar":        user.Avatar,
				"wechat_open_id": user.WeChatOpenID,
			},
		}, "用户添加成功")
	}

	return gin.H{
		"user": gin.H{
			"id":            user.ID,
			"username":      user.Username,
			"nickname":      user.Nickname,
			"email":         user.Email,
			"avatar":        user.Avatar,
			"wechat_open_id": user.WeChatOpenID,
		},
	}, nil
}

func (h *AddUserCallbackHandler) GetSuccessHTML(ctx *WeChatCallbackContext, data interface{}) string {
	return GetDefaultSuccessHTML("用户添加成功", "请返回 PC 端查看")
}

func (h *AddUserCallbackHandler) GetErrorHTML(ctx *WeChatCallbackContext, err error) string {
	return GetDefaultErrorHTML("添加用户失败", err.Error())
}

