package api

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"project-management/internal/model"
	"project-management/internal/utils"
	"project-management/pkg/auth"
	"project-management/pkg/wechat"
)

type InitHandler struct {
	db          *gorm.DB
	wechatClient *wechat.WeChatClient
}

func NewInitHandler(db *gorm.DB) *InitHandler {
	return &InitHandler{
		db:          db,
		wechatClient: wechat.NewWeChatClient(),
	}
}

// CheckInitStatus 检查初始化状态
func (h *InitHandler) CheckInitStatus(c *gin.Context) {
	var config model.SystemConfig
	result := h.db.Where("key = ?", "initialized").First(&config)
	
	if result.Error == gorm.ErrRecordNotFound {
		utils.Success(c, gin.H{
			"initialized": false,
		})
		return
	}
	
	utils.Success(c, gin.H{
		"initialized": config.Value == "true",
	})
}

// SaveWeChatConfig 保存微信配置（第一步）
func (h *InitHandler) SaveWeChatConfig(c *gin.Context) {
	// 检查是否已经初始化
	var existingConfig model.SystemConfig
	result := h.db.Where("key = ?", "initialized").First(&existingConfig)
	if result.Error == nil && existingConfig.Value == "true" {
		utils.Error(c, 400, "系统已经初始化，无法重复初始化")
		return
	}

	var req struct {
		WeChatAppID     string `json:"wechat_app_id" binding:"required"`
		WeChatAppSecret string `json:"wechat_app_secret" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 保存微信配置
	wechatAppIDConfig := model.SystemConfig{
		Key:   "wechat_app_id",
		Value: req.WeChatAppID,
		Type:  "string",
	}
	if err := h.db.Where("key = ?", "wechat_app_id").Assign(model.SystemConfig{Value: req.WeChatAppID, Type: "string"}).FirstOrCreate(&wechatAppIDConfig).Error; err != nil {
		utils.Error(c, utils.CodeError, "保存微信AppID失败")
		return
	}

	wechatAppSecretConfig := model.SystemConfig{
		Key:   "wechat_app_secret",
		Value: req.WeChatAppSecret,
		Type:  "string",
	}
	if err := h.db.Where("key = ?", "wechat_app_secret").Assign(model.SystemConfig{Value: req.WeChatAppSecret, Type: "string"}).FirstOrCreate(&wechatAppSecretConfig).Error; err != nil {
		utils.Error(c, utils.CodeError, "保存微信AppSecret失败")
		return
	}

	// 更新WeChatClient的配置（临时，用于后续获取二维码）
	h.wechatClient.AppID = req.WeChatAppID
	h.wechatClient.AppSecret = req.WeChatAppSecret

	utils.Success(c, gin.H{
		"message": "微信配置保存成功",
	})
}

// InitSystem 完成初始化（第二步：通过微信登录创建管理员）
func (h *InitHandler) InitSystem(c *gin.Context) {
	// 检查是否已经初始化
	var existingConfig model.SystemConfig
	result := h.db.Where("key = ?", "initialized").First(&existingConfig)
	if result.Error == nil && existingConfig.Value == "true" {
		utils.Error(c, 400, "系统已经初始化，无法重复初始化")
		return
	}

	// 检查微信配置是否已保存
	var wechatAppIDConfig model.SystemConfig
	if err := h.db.Where("key = ?", "wechat_app_id").First(&wechatAppIDConfig).Error; err != nil {
		utils.Error(c, 400, "请先配置微信AppID和AppSecret")
		return
	}

	var req struct {
		Code  string `json:"code" binding:"required"`  // 微信登录返回的code
		State string `json:"state"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 获取微信配置
	var wechatAppSecretConfig model.SystemConfig
	h.db.Where("key = ?", "wechat_app_secret").First(&wechatAppSecretConfig)
	
	// 临时设置WeChatClient配置
	h.wechatClient.AppID = wechatAppIDConfig.Value
	h.wechatClient.AppSecret = wechatAppSecretConfig.Value

	// 获取access_token
	accessTokenResp, err := h.wechatClient.GetAccessToken(req.Code)
	if err != nil {
		utils.Error(c, utils.CodeError, "获取access_token失败: "+err.Error())
		return
	}

	// 获取用户信息
	userInfo, err := h.wechatClient.GetUserInfo(accessTokenResp.AccessToken, accessTokenResp.OpenID)
	if err != nil {
		utils.Error(c, utils.CodeError, "获取用户信息失败: "+err.Error())
		return
	}

	// 开始事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 创建管理员角色
	var adminRole model.Role
	if err := tx.Where("code = ?", "admin").First(&adminRole).Error; err == gorm.ErrRecordNotFound {
		adminRole = model.Role{
			Name:        "管理员",
			Code:        "admin",
			Description: "系统管理员，拥有所有权限",
			Status:      1,
		}
		if err := tx.Create(&adminRole).Error; err != nil {
			tx.Rollback()
			utils.Error(c, utils.CodeError, "创建管理员角色失败")
			return
		}
	}

	// 2. 创建管理员用户（使用微信登录获取的用户信息）
	adminUser := model.User{
		WeChatOpenID: userInfo.OpenID,
		Username:     userInfo.Nickname,
		Avatar:       userInfo.HeadImgURL,
		Status:       1,
	}
	if err := tx.Create(&adminUser).Error; err != nil {
		tx.Rollback()
		utils.Error(c, utils.CodeError, "创建管理员用户失败")
		return
	}

	// 3. 分配管理员角色
	if err := tx.Model(&adminUser).Association("Roles").Append(&adminRole); err != nil {
		tx.Rollback()
		utils.Error(c, utils.CodeError, "分配管理员角色失败")
		return
	}

	// 4. 标记系统已初始化
	initConfig := model.SystemConfig{
		Key:   "initialized",
		Value: "true",
		Type:  "boolean",
	}
	if err := tx.Where("key = ?", "initialized").Assign(model.SystemConfig{Value: "true", Type: "boolean"}).FirstOrCreate(&initConfig).Error; err != nil {
		tx.Rollback()
		utils.Error(c, utils.CodeError, "保存初始化状态失败")
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		utils.Error(c, utils.CodeError, "初始化失败")
		return
	}

	// 生成管理员Token（可选，用于自动登录）
	roleNames := []string{"admin"}
	token, _ := auth.GenerateToken(adminUser.ID, adminUser.Username, roleNames)

	utils.Success(c, gin.H{
		"message": "系统初始化成功",
		"token":   token,
		"user": gin.H{
			"id":       adminUser.ID,
			"username": adminUser.Username,
			"avatar":   adminUser.Avatar,
			"roles":    roleNames,
		},
	})
}

// GetInitQRCode 获取初始化用的微信二维码
func (h *InitHandler) GetInitQRCode(c *gin.Context) {
	// 检查是否已经初始化
	var existingConfig model.SystemConfig
	result := h.db.Where("key = ?", "initialized").First(&existingConfig)
	if result.Error == nil && existingConfig.Value == "true" {
		utils.Error(c, 400, "系统已经初始化")
		return
	}

	// 检查微信配置是否已保存
	var wechatAppIDConfig model.SystemConfig
	if err := h.db.Where("key = ?", "wechat_app_id").First(&wechatAppIDConfig).Error; err != nil {
		utils.Error(c, 400, "请先配置微信AppID和AppSecret")
		return
	}

	// 获取微信配置
	var wechatAppSecretConfig model.SystemConfig
	h.db.Where("key = ?", "wechat_app_secret").First(&wechatAppSecretConfig)
	
	// 临时设置WeChatClient配置
	h.wechatClient.AppID = wechatAppIDConfig.Value
	h.wechatClient.AppSecret = wechatAppSecretConfig.Value

	qrCode, err := h.wechatClient.GetQRCode()
	if err != nil {
		utils.Error(c, utils.CodeError, "获取二维码失败")
		return
	}

	qrCodeURL := h.wechatClient.GetQRCodeURL(qrCode.Ticket)

	utils.Success(c, gin.H{
		"ticket":         qrCode.Ticket,
		"qr_code_url":    qrCodeURL,
		"expire_seconds": qrCode.ExpireSeconds,
	})
}

