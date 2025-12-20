package utils

import (
	"encoding/json"
	"time"

	"prjflow/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuditDB 审计日志数据库连接（如果配置了独立数据库则使用，否则为 nil 表示使用主数据库）
var AuditDB *gorm.DB

// RecordAuditLog 同步记录审计日志（直接写入，简单可靠）
// 如果配置了独立审计日志数据库（AuditDB），则使用独立数据库；否则使用传入的 db
func RecordAuditLog(db *gorm.DB, userID uint, username, actionType, resourceType string, resourceID uint, c *gin.Context, success bool, errorMsg, comment string) {
	// 优先使用独立审计日志数据库
	auditDB := db
	if AuditDB != nil {
		auditDB = AuditDB
	}

	// 直接同步写入，简单可靠，避免异步导致的复杂问题和SQLite并发锁定
	if err := recordAuditLogSync(auditDB, userID, username, actionType, resourceType, resourceID, c, success, errorMsg, comment); err != nil {
		// 记录错误但不影响主请求（即使写入失败也不返回错误）
		if Logger != nil {
			Logger.Errorf("记录审计日志失败: %v", err)
		}
	}
}

// RecordAuditLogSync 同步记录审计日志（用于关键操作，确保记录）
// 如果配置了独立审计日志数据库（AuditDB），则使用独立数据库；否则使用传入的 db
func RecordAuditLogSync(db *gorm.DB, userID uint, username, actionType, resourceType string, resourceID uint, c *gin.Context, success bool, errorMsg, comment string) error {
	// 优先使用独立审计日志数据库
	auditDB := db
	if AuditDB != nil {
		auditDB = AuditDB
	}
	return recordAuditLogSync(auditDB, userID, username, actionType, resourceType, resourceID, c, success, errorMsg, comment)
}

// recordAuditLogSync 内部实现：同步记录审计日志
func recordAuditLogSync(db *gorm.DB, userID uint, username, actionType, resourceType string, resourceID uint, c *gin.Context, success bool, errorMsg, comment string) error {
	auditLog := model.AuditLog{
		UserID:       userID,
		Username:     username,
		ActionType:   actionType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Success:      success,
		ErrorMsg:     errorMsg,
		Comment:      comment,
		CreatedAt:    time.Now(),
	}

	// 从gin.Context获取请求信息
	if c != nil {
		auditLog.IPAddress = c.ClientIP()
		auditLog.Path = c.Request.URL.Path
		auditLog.Method = c.Request.Method

		// 获取请求参数（仅记录关键参数，避免记录敏感信息）
		if params := getRequestParams(c); params != nil {
			paramsJSON, err := json.Marshal(params)
			if err == nil {
				auditLog.Params = string(paramsJSON)
			}
		}
	}

	if err := db.Create(&auditLog).Error; err != nil {
		return err
	}

	return nil
}

// getRequestParams 获取请求参数（仅记录关键参数）
func getRequestParams(c *gin.Context) map[string]interface{} {
	params := make(map[string]interface{})

	// 获取查询参数
	if c.Request.URL.RawQuery != "" {
		params["query"] = c.Request.URL.RawQuery
	}

	// 对于POST/PUT请求，可以记录部分参数（避免记录密码等敏感信息）
	if c.Request.Method == "POST" || c.Request.Method == "PUT" {
		// 只记录非敏感字段
		// 注意：这里不解析请求体，避免性能问题
		// 如果需要记录请求体，可以在调用时手动传入
	}

	return params
}

// CleanupOldAuditLogs 清理过期的审计日志
func CleanupOldAuditLogs(db *gorm.DB, retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	result := db.Where("created_at < ?", cutoffDate).Delete(&model.AuditLog{})
	
	if Logger != nil {
		Logger.Infof("清理审计日志: 删除了 %d 条记录（保留 %d 天）", result.RowsAffected, retentionDays)
	}

	return result.Error
}

