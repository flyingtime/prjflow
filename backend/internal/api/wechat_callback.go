package api

import (
	"fmt"
	"strings"
	"time"

	"prjflow/internal/config"
	"prjflow/internal/model"
	"prjflow/internal/websocket"
	"prjflow/pkg/wechat"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// WeChatCallbackContext 微信回调上下文
type WeChatCallbackContext struct {
	Code         string
	State        string
	Ticket       string
	WeChatClient wechat.WeChatClientInterface // 使用接口类型
	Hub          websocket.HubInterface        // 使用接口类型，用于发送WebSocket消息
	DB           *gorm.DB
	AccessToken  *wechat.AccessTokenResponse
	UserInfo     *wechat.UserInfoResponse
	Context      *gin.Context // Gin上下文，用于获取IP和请求路径等信息
}

// WeChatCallbackHandler 微信回调业务处理接口
type WeChatCallbackHandler interface {
	// Validate 验证前置条件（如检查系统是否已初始化等）
	Validate(ctx *WeChatCallbackContext) error

	// Process 处理业务逻辑（获取用户信息后）
	Process(ctx *WeChatCallbackContext) (interface{}, error)

	// GetSuccessHTML 获取成功页面的HTML
	GetSuccessHTML(ctx *WeChatCallbackContext, data interface{}) string

	// GetErrorHTML 获取错误页面的HTML
	GetErrorHTML(ctx *WeChatCallbackContext, err error) string
}

// ProcessWeChatCallback 处理微信回调的通用流程
func ProcessWeChatCallback(
	db *gorm.DB,
	wechatClient wechat.WeChatClientInterface, // 使用接口类型
	hub websocket.HubInterface,                // 使用接口类型，用于发送WebSocket消息
	code string,
	state string,
	handler WeChatCallbackHandler,
	c *gin.Context, // Gin上下文，用于获取IP和请求路径等信息
) (*WeChatCallbackContext, interface{}, error) {
	ctx := &WeChatCallbackContext{
		Code:         code,
		State:        state,
		WeChatClient: wechatClient,
		Hub:          hub,
		DB:           db,
		Context:      c, // 保存Gin上下文，用于记录审计日志时获取IP和请求路径
	}

	// 1. 从state中提取ticket和用户ID
	if state != "" && len(state) > 8 && state[:8] == "adduser:" {
		// 添加用户场景：格式 adduser:{ticket}:{user_id}
		parts := state[8:] // 去掉 "adduser:" 前缀
		// 找到第一个冒号，前面是ticket，后面是user_id
		colonIndex := strings.Index(parts, ":")
		if colonIndex > 0 {
			ctx.Ticket = parts[:colonIndex]
		} else {
			ctx.Ticket = parts
		}
	} else if state != "" && len(state) > 7 && state[:7] == "ticket:" {
		// 登录场景：格式 ticket:{ticket}
		ctx.Ticket = state[7:]
	} else if state != "" {
		ctx.Ticket = state
	}

	// 2. 检查code是否存在
	if code == "" {
		if ctx.Ticket != "" && ctx.Hub != nil {
			ctx.Hub.SendMessage(ctx.Ticket, "error", nil, "未获取到授权码")
		}
		return ctx, nil, &CallbackError{Message: "未获取到授权码"}
	}

	// 3. 读取微信配置
	var wechatAppIDConfig model.SystemConfig
	if err := db.Where("key = ?", "wechat_app_id").First(&wechatAppIDConfig).Error; err != nil {
		// 如果数据库中没有配置，尝试使用配置文件中的配置
		if config.AppConfig.WeChat.AppID == "" || config.AppConfig.WeChat.AppSecret == "" {
			if ctx.Ticket != "" && ctx.Hub != nil {
				ctx.Hub.SendMessage(ctx.Ticket, "error", nil, "请先配置微信AppID和AppSecret")
			}
			return ctx, nil, &CallbackError{Message: "请先配置微信AppID和AppSecret"}
		}
		wechatClient.SetAppID(config.AppConfig.WeChat.AppID)
		wechatClient.SetAppSecret(config.AppConfig.WeChat.AppSecret)
	} else {
		// 从数据库读取配置
		var wechatAppSecretConfig model.SystemConfig
		if err := db.Where("key = ?", "wechat_app_secret").First(&wechatAppSecretConfig).Error; err != nil {
			// 如果数据库中没有AppSecret，尝试使用配置文件中的配置
			if config.AppConfig.WeChat.AppSecret == "" {
				if ctx.Ticket != "" && ctx.Hub != nil {
					ctx.Hub.SendMessage(ctx.Ticket, "error", nil, "请先配置微信AppSecret")
				}
				return ctx, nil, &CallbackError{Message: "请先配置微信AppSecret"}
			}
			wechatClient.SetAppID(wechatAppIDConfig.Value)
			wechatClient.SetAppSecret(config.AppConfig.WeChat.AppSecret)
		} else {
			// 从数据库读取配置，去除首尾空格
			wechatClient.SetAppID(strings.TrimSpace(wechatAppIDConfig.Value))
			wechatClient.SetAppSecret(strings.TrimSpace(wechatAppSecretConfig.Value))
		}
		// 验证配置是否为空
		if wechatClient.GetAppID() == "" || wechatClient.GetAppSecret() == "" {
			if ctx.Ticket != "" && ctx.Hub != nil {
				ctx.Hub.SendMessage(ctx.Ticket, "error", nil, "微信AppID或AppSecret配置为空，请检查配置")
			}
			return ctx, nil, &CallbackError{Message: "微信AppID或AppSecret配置为空，请检查配置"}
		}
	}

	// 设置AccountType和Scope（优先从数据库读取，其次从配置文件，最后使用默认值）
	var accountTypeConfig model.SystemConfig
	if err := db.Where("key = ?", "wechat_account_type").First(&accountTypeConfig).Error; err == nil {
		wechatClient.SetAccountType(strings.TrimSpace(accountTypeConfig.Value))
	} else {
		wechatClient.SetAccountType(config.AppConfig.WeChat.AccountType)
	}
	if wechatClient.GetAccountType() == "" {
		wechatClient.SetAccountType("open_platform") // 默认使用开放平台
	}

	var scopeConfig model.SystemConfig
	if err := db.Where("key = ?", "wechat_scope").First(&scopeConfig).Error; err == nil {
		wechatClient.SetScope(strings.TrimSpace(scopeConfig.Value))
	} else {
		wechatClient.SetScope(config.AppConfig.WeChat.Scope)
	}
	if wechatClient.GetScope() == "" {
		wechatClient.SetScope("snsapi_userinfo") // 默认需要用户确认
	}

	// 4. 验证前置条件
	if err := handler.Validate(ctx); err != nil {
		if ctx.Ticket != "" && ctx.Hub != nil {
			ctx.Hub.SendMessage(ctx.Ticket, "error", nil, err.Error())
		}
		return ctx, nil, err
	}

	// 5. 通知已扫码
	if ctx.Ticket != "" && ctx.Hub != nil {
		ctx.Hub.SendMessage(ctx.Ticket, "info", nil, "已扫码，正在获取授权...")
	}

	// 6. 获取access_token
	// 添加调试信息：显示实际使用的配置（不显示完整的AppSecret，只显示前4位和后4位）
	appSecret := wechatClient.GetAppSecret()
	appSecretMasked := ""
	if len(appSecret) > 8 {
		appSecretMasked = appSecret[:4] + "****" + appSecret[len(appSecret)-4:]
	} else {
		appSecretMasked = "****"
	}
	debugInfo := fmt.Sprintf("使用配置: AppID=%s, AppSecret=%s, AccountType=%s, Scope=%s",
		wechatClient.GetAppID(), appSecretMasked, wechatClient.GetAccountType(), wechatClient.GetScope())

	accessTokenResp, err := wechatClient.GetAccessToken(code)
	if err != nil {
		errorMsg := fmt.Sprintf("获取access_token失败: %s。%s", err.Error(), debugInfo)
		if ctx.Ticket != "" && ctx.Hub != nil {
			ctx.Hub.SendMessage(ctx.Ticket, "error", nil, errorMsg)
		}
		return ctx, nil, &CallbackError{Message: errorMsg, Err: err}
	}
	ctx.AccessToken = accessTokenResp

	// 7. 通知正在获取用户信息
	if ctx.Ticket != "" && ctx.Hub != nil {
		ctx.Hub.SendMessage(ctx.Ticket, "info", nil, "正在获取用户信息...")
	}

	// 8. 获取用户信息
	userInfo, err := wechatClient.GetUserInfo(accessTokenResp.AccessToken, accessTokenResp.OpenID)
	if err != nil {
		if ctx.Ticket != "" && ctx.Hub != nil {
			ctx.Hub.SendMessage(ctx.Ticket, "error", nil, "获取用户信息失败")
		}
		return ctx, nil, &CallbackError{Message: "获取用户信息失败", Err: err}
	}
	ctx.UserInfo = userInfo

	// 9. 处理业务逻辑
	result, err := handler.Process(ctx)
	if err != nil {
		if ctx.Ticket != "" && ctx.Hub != nil {
			ctx.Hub.SendMessage(ctx.Ticket, "error", nil, err.Error())
		}
		return ctx, nil, err
	}

	return ctx, result, nil
}

// CallbackError 回调错误
type CallbackError struct {
	Message string
	Err     error
}

func (e *CallbackError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// GetDefaultErrorHTML 获取默认错误页面HTML
func GetDefaultErrorHTML(title, message string) string {
	return `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>` + title + `</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			display: flex;
			justify-content: center;
			align-items: center;
			height: 100vh;
			margin: 0;
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			color: white;
		}
		.container {
			text-align: center;
			padding: 20px;
		}
		h1 { font-size: 48px; margin: 0 0 20px 0; }
		p { font-size: 18px; margin: 0; }
	</style>
</head>
<body>
	<div class="container">
		<h1>✗</h1>
		<p>` + title + `</p>
		<p style="font-size: 14px; margin-top: 10px;">` + message + `</p>
	</div>
</body>
</html>`
}

// GenerateUniqueUsername 生成唯一的用户名
// 使用OpenID的后8位 + 年月日（格式：{openid后8位}_{YYYYMMDD}）
// 如果用户名冲突，不自动添加后缀，直接返回错误提示联系管理员
func GenerateUniqueUsername(db *gorm.DB, baseUsername string, openID string) string {
	// 无论是否有baseUsername，都使用OpenID后8位+日期格式
	// 获取OpenID的后8位
	openIDSuffix := openID
	if len(openID) > 8 {
		openIDSuffix = openID[len(openID)-8:]
	} else if len(openID) < 8 {
		// 如果OpenID长度不足8位，前面补0
		openIDSuffix = fmt.Sprintf("%08s", openID)
	}
	// 获取当前日期（格式：YYYYMMDD）
	dateStr := time.Now().Format("20060102")
	username := fmt.Sprintf("%s_%s", openIDSuffix, dateStr)
	
	return username
}

// GetDefaultSuccessHTML 获取默认成功页面HTML
func GetDefaultSuccessHTML(title, message string) string {
	return `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>` + title + `</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			display: flex;
			justify-content: center;
			align-items: center;
			height: 100vh;
			margin: 0;
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			color: white;
		}
		.container {
			text-align: center;
			padding: 20px;
		}
		h1 { font-size: 48px; margin: 0 0 20px 0; }
		p { font-size: 18px; margin: 0; }
	</style>
</head>
<body>
	<div class="container">
		<h1>✓</h1>
		<p>` + title + `</p>
		<p style="font-size: 14px; margin-top: 10px;">` + message + `</p>
	</div>
</body>
</html>`
}
