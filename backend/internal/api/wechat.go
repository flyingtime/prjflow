package api

import (
	"project-management/internal/model"
	"project-management/internal/utils"
	"project-management/pkg/wechat"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WeChatHandler struct {
	db           *gorm.DB
	wechatClient *wechat.WeChatClient
}

func NewWeChatHandler(db *gorm.DB) *WeChatHandler {
	return &WeChatHandler{
		db:           db,
		wechatClient: wechat.NewWeChatClient(),
	}
}

// GetWeChatConfig 获取微信配置
func (h *WeChatHandler) GetWeChatConfig(c *gin.Context) {
	var wechatAppIDConfig model.SystemConfig
	var wechatAppSecretConfig model.SystemConfig
	var accountTypeConfig model.SystemConfig
	var scopeConfig model.SystemConfig

	wechatAppID := ""
	wechatAppSecret := ""
	accountType := ""
	scope := ""

	// 从数据库读取微信配置
	if err := h.db.Where("key = ?", "wechat_app_id").First(&wechatAppIDConfig).Error; err == nil {
		wechatAppID = wechatAppIDConfig.Value
	}

	if err := h.db.Where("key = ?", "wechat_app_secret").First(&wechatAppSecretConfig).Error; err == nil {
		wechatAppSecret = wechatAppSecretConfig.Value
	}

	if err := h.db.Where("key = ?", "wechat_account_type").First(&accountTypeConfig).Error; err == nil {
		accountType = accountTypeConfig.Value
	}

	if err := h.db.Where("key = ?", "wechat_scope").First(&scopeConfig).Error; err == nil {
		scope = scopeConfig.Value
	}

	utils.Success(c, gin.H{
		"wechat_app_id":     wechatAppID,
		"wechat_app_secret": wechatAppSecret,
		"account_type":      accountType,
		"scope":             scope,
	})
}

// SaveWeChatConfig 保存微信配置
func (h *WeChatHandler) SaveWeChatConfig(c *gin.Context) {
	var req struct {
		WeChatAppID     string `json:"wechat_app_id" binding:"required"`
		WeChatAppSecret string `json:"wechat_app_secret" binding:"required"`
		AccountType     string `json:"account_type"` // 可选：open_platform 或 official_account
		Scope           string `json:"scope"`         // 可选：snsapi_base 或 snsapi_userinfo
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

	// 保存AccountType（如果提供）
	if req.AccountType != "" {
		accountTypeConfig := model.SystemConfig{
			Key:   "wechat_account_type",
			Value: req.AccountType,
			Type:  "string",
		}
		if err := h.db.Where("key = ?", "wechat_account_type").Assign(model.SystemConfig{Value: req.AccountType, Type: "string"}).FirstOrCreate(&accountTypeConfig).Error; err != nil {
			utils.Error(c, utils.CodeError, "保存微信AccountType失败")
			return
		}
	}

	// 保存Scope（如果提供）
	if req.Scope != "" {
		scopeConfig := model.SystemConfig{
			Key:   "wechat_scope",
			Value: req.Scope,
			Type:  "string",
		}
		if err := h.db.Where("key = ?", "wechat_scope").Assign(model.SystemConfig{Value: req.Scope, Type: "string"}).FirstOrCreate(&scopeConfig).Error; err != nil {
			utils.Error(c, utils.CodeError, "保存微信Scope失败")
			return
		}
	}

	// 记录审计日志
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	if userID != nil && username != nil {
		utils.RecordAuditLog(h.db, userID.(uint), username.(string), "update", "system", 0, c, true, "", "更新微信配置")
	}

	utils.Success(c, gin.H{
		"message": "微信配置保存成功",
	})
}

