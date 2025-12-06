package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"project-management/internal/model"
	"project-management/internal/utils"
)

type AuditLogHandler struct {
	db *gorm.DB // 审计日志数据库（如果配置了独立数据库则使用独立数据库，否则使用主数据库）
}

func NewAuditLogHandler(db *gorm.DB) *AuditLogHandler {
	// 优先使用独立审计日志数据库
	auditDB := db
	if utils.AuditDB != nil {
		auditDB = utils.AuditDB
	}
	return &AuditLogHandler{db: auditDB}
}

// GetAuditLogs 获取审计日志列表
func (h *AuditLogHandler) GetAuditLogs(c *gin.Context) {
	var auditLogs []model.AuditLog
	query := h.db.Model(&model.AuditLog{})

	// 用户筛选
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// 操作类型筛选
	if actionType := c.Query("action_type"); actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}

	// 资源类型筛选
	if resourceType := c.Query("resource_type"); resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	// 操作结果筛选
	if success := c.Query("success"); success != "" {
		if success == "true" {
			query = query.Where("success = ?", true)
		} else if success == "false" {
			query = query.Where("success = ?", false)
		}
	}

	// 时间范围筛选
	if startDate := c.Query("start_date"); startDate != "" {
		if startTime, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if endTime, err := time.Parse("2006-01-02", endDate); err == nil {
			// 结束日期包含整天，所以加一天
			endTime = endTime.AddDate(0, 0, 1)
			query = query.Where("created_at < ?", endTime)
		}
	}

	// 搜索：用户名、资源类型、操作类型
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("username LIKE ? OR resource_type LIKE ? OR action_type LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 分页
	page := utils.GetPage(c)
	pageSize := utils.GetPageSize(c)
	offset := (page - 1) * pageSize

	var total int64
	// 计算总数时需要应用与查询相同的筛选条件
	countQuery := h.db.Model(&model.AuditLog{})

	// 应用相同的筛选条件
	if userID := c.Query("user_id"); userID != "" {
		countQuery = countQuery.Where("user_id = ?", userID)
	}
	if actionType := c.Query("action_type"); actionType != "" {
		countQuery = countQuery.Where("action_type = ?", actionType)
	}
	if resourceType := c.Query("resource_type"); resourceType != "" {
		countQuery = countQuery.Where("resource_type = ?", resourceType)
	}
	if success := c.Query("success"); success != "" {
		if success == "true" {
			countQuery = countQuery.Where("success = ?", true)
		} else if success == "false" {
			countQuery = countQuery.Where("success = ?", false)
		}
	}
	if startDate := c.Query("start_date"); startDate != "" {
		if startTime, err := time.Parse("2006-01-02", startDate); err == nil {
			countQuery = countQuery.Where("created_at >= ?", startTime)
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if endTime, err := time.Parse("2006-01-02", endDate); err == nil {
			endTime = endTime.AddDate(0, 0, 1)
			countQuery = countQuery.Where("created_at < ?", endTime)
		}
	}
	if keyword := c.Query("keyword"); keyword != "" {
		countQuery = countQuery.Where("username LIKE ? OR resource_type LIKE ? OR action_type LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	countQuery.Count(&total)

	// 排序：按时间倒序
	query = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&auditLogs)

	if err := query.Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, gin.H{
		"list":  auditLogs,
		"total": total,
	})
}

// GetAuditLog 获取审计日志详情
func (h *AuditLogHandler) GetAuditLog(c *gin.Context) {
	id := c.Param("id")
	var auditLog model.AuditLog
	if err := h.db.First(&auditLog, id).Error; err != nil {
		utils.Error(c, 404, "审计日志不存在")
		return
	}

	utils.Success(c, auditLog)
}


