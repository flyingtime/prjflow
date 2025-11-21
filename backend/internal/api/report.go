package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"project-management/internal/model"
	"project-management/internal/utils"
)

type ReportHandler struct {
	db *gorm.DB
}

func NewReportHandler(db *gorm.DB) *ReportHandler {
	return &ReportHandler{db: db}
}

// GetDailyReports 获取日报列表
func (h *ReportHandler) GetDailyReports(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "未授权")
		return
	}

	var reports []model.DailyReport
	query := h.db.Preload("User").Preload("Project").Preload("Task")

	// 普通用户只能看到自己的报告
	uid := userID.(uint)
	if !utils.IsAdmin(c) {
		query = query.Where("user_id = ?", uid)
	} else {
		// 管理员可以筛选用户
		if filterUserID := c.Query("user_id"); filterUserID != "" {
			query = query.Where("user_id = ?", filterUserID)
		}
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 日期范围筛选
	if startDate := c.Query("start_date"); startDate != "" {
		query = query.Where("date >= ?", startDate)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		query = query.Where("date <= ?", endDate)
	}

	// 项目筛选
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	// 分页
	page := utils.GetPage(c)
	pageSize := utils.GetPageSize(c)
	offset := (page - 1) * pageSize

	var total int64
	countQuery := h.db.Model(&model.DailyReport{})
	if !utils.IsAdmin(c) {
		countQuery = countQuery.Where("user_id = ?", uid)
	}
	if status := c.Query("status"); status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}
	countQuery.Count(&total)

	if err := query.Offset(offset).Limit(pageSize).Order("date DESC").Find(&reports).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, gin.H{
		"list":      reports,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetDailyReport 获取日报详情
func (h *ReportHandler) GetDailyReport(c *gin.Context) {
	id := c.Param("id")
	var report model.DailyReport
	if err := h.db.Preload("User").Preload("Project").Preload("Task").First(&report, id).Error; err != nil {
		utils.Error(c, 404, "日报不存在")
		return
	}

	// 权限检查：普通用户只能查看自己的报告
	userID, _ := c.Get("user_id")
	if !utils.IsAdmin(c) && report.UserID != userID.(uint) {
		utils.Error(c, 403, "没有权限访问该报告")
		return
	}

	utils.Success(c, report)
}

// CreateDailyReport 创建日报
func (h *ReportHandler) CreateDailyReport(c *gin.Context) {
	var req struct {
		Date      string   `json:"date" binding:"required"`
		Content   string   `json:"content"`
		Hours     *float64 `json:"hours"`
		Status    string   `json:"status"`
		ProjectID *uint    `json:"project_id"`
		TaskID    *uint    `json:"task_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "未授权")
		return
	}

	// 解析日期
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		utils.Error(c, 400, "日期格式错误")
		return
	}

	// 检查是否已存在该日期的日报
	var existingReport model.DailyReport
	if err := h.db.Where("user_id = ? AND date = ?", userID.(uint), date).First(&existingReport).Error; err == nil {
		utils.Error(c, 400, "该日期已存在日报")
		return
	}

	// 设置默认状态
	if req.Status == "" {
		req.Status = "draft"
	}

	report := model.DailyReport{
		Date:      date,
		Content:   req.Content,
		Hours:     0,
		Status:    req.Status,
		UserID:    userID.(uint),
		ProjectID: req.ProjectID,
		TaskID:    req.TaskID,
	}

	if req.Hours != nil {
		report.Hours = *req.Hours
	}

	if err := h.db.Create(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "创建失败")
		return
	}

	h.db.Preload("User").Preload("Project").Preload("Task").First(&report, report.ID)
	utils.Success(c, report)
}

// UpdateDailyReport 更新日报
func (h *ReportHandler) UpdateDailyReport(c *gin.Context) {
	id := c.Param("id")
	var report model.DailyReport
	if err := h.db.First(&report, id).Error; err != nil {
		utils.Error(c, 404, "日报不存在")
		return
	}

	// 权限检查：普通用户只能更新自己的报告
	userID, _ := c.Get("user_id")
	if !utils.IsAdmin(c) && report.UserID != userID.(uint) {
		utils.Error(c, 403, "没有权限更新该报告")
		return
	}

	var req struct {
		Date      *string   `json:"date"`
		Content   *string   `json:"content"`
		Hours     *float64  `json:"hours"`
		Status    *string   `json:"status"`
		ProjectID *uint     `json:"project_id"`
		TaskID    *uint     `json:"task_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	// 如果更新日期，检查是否与其他日报冲突
	if req.Date != nil {
		date, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			utils.Error(c, 400, "日期格式错误")
			return
		}
		if date != report.Date {
			var existingReport model.DailyReport
			if err := h.db.Where("user_id = ? AND date = ? AND id != ?", report.UserID, date, report.ID).First(&existingReport).Error; err == nil {
				utils.Error(c, 400, "该日期已存在日报")
				return
			}
			report.Date = date
		}
	}

	if req.Content != nil {
		report.Content = *req.Content
	}
	if req.Hours != nil {
		report.Hours = *req.Hours
	}
	if req.Status != nil {
		report.Status = *req.Status
	}
	if req.ProjectID != nil {
		report.ProjectID = req.ProjectID
	}
	if req.TaskID != nil {
		report.TaskID = req.TaskID
	}

	if err := h.db.Save(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	h.db.Preload("User").Preload("Project").Preload("Task").First(&report, report.ID)
	utils.Success(c, report)
}

// DeleteDailyReport 删除日报
func (h *ReportHandler) DeleteDailyReport(c *gin.Context) {
	id := c.Param("id")
	var report model.DailyReport
	if err := h.db.First(&report, id).Error; err != nil {
		utils.Error(c, 404, "日报不存在")
		return
	}

	// 权限检查：普通用户只能删除自己的报告
	userID, _ := c.Get("user_id")
	if !utils.IsAdmin(c) && report.UserID != userID.(uint) {
		utils.Error(c, 403, "没有权限删除该报告")
		return
	}

	if err := h.db.Delete(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除失败")
		return
	}

	utils.Success(c, nil)
}

// UpdateDailyReportStatus 更新日报状态
func (h *ReportHandler) UpdateDailyReportStatus(c *gin.Context) {
	id := c.Param("id")
	var report model.DailyReport
	if err := h.db.First(&report, id).Error; err != nil {
		utils.Error(c, 404, "日报不存在")
		return
	}

	// 权限检查：普通用户只能更新自己的报告状态（提交），管理员可以审批
	userID, _ := c.Get("user_id")
	if !utils.IsAdmin(c) && report.UserID != userID.(uint) {
		utils.Error(c, 403, "没有权限更新该报告")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	validStatuses := map[string]bool{
		"draft":     true,
		"submitted": true,
		"approved":  true,
	}
	if !validStatuses[req.Status] {
		utils.Error(c, 400, "状态值无效")
		return
	}

	report.Status = req.Status
	if err := h.db.Save(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	h.db.Preload("User").Preload("Project").Preload("Task").First(&report, report.ID)
	utils.Success(c, report)
}

// GetWeeklyReports 获取周报列表
func (h *ReportHandler) GetWeeklyReports(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "未授权")
		return
	}

	var reports []model.WeeklyReport
	query := h.db.Preload("User").Preload("Project").Preload("Task")

	// 普通用户只能看到自己的报告
	uid := userID.(uint)
	if !utils.IsAdmin(c) {
		query = query.Where("user_id = ?", uid)
	} else {
		// 管理员可以筛选用户
		if filterUserID := c.Query("user_id"); filterUserID != "" {
			query = query.Where("user_id = ?", filterUserID)
		}
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 日期范围筛选
	if startDate := c.Query("start_date"); startDate != "" {
		query = query.Where("week_start >= ?", startDate)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		query = query.Where("week_end <= ?", endDate)
	}

	// 项目筛选
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	// 分页
	page := utils.GetPage(c)
	pageSize := utils.GetPageSize(c)
	offset := (page - 1) * pageSize

	var total int64
	countQuery := h.db.Model(&model.WeeklyReport{})
	if !utils.IsAdmin(c) {
		countQuery = countQuery.Where("user_id = ?", uid)
	}
	if status := c.Query("status"); status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}
	countQuery.Count(&total)

	if err := query.Offset(offset).Limit(pageSize).Order("week_start DESC").Find(&reports).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, gin.H{
		"list":      reports,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetWeeklyReport 获取周报详情
func (h *ReportHandler) GetWeeklyReport(c *gin.Context) {
	id := c.Param("id")
	var report model.WeeklyReport
	if err := h.db.Preload("User").Preload("Project").Preload("Task").First(&report, id).Error; err != nil {
		utils.Error(c, 404, "周报不存在")
		return
	}

	// 权限检查：普通用户只能查看自己的报告
	userID, _ := c.Get("user_id")
	if !utils.IsAdmin(c) && report.UserID != userID.(uint) {
		utils.Error(c, 403, "没有权限访问该报告")
		return
	}

	utils.Success(c, report)
}

// CreateWeeklyReport 创建周报
func (h *ReportHandler) CreateWeeklyReport(c *gin.Context) {
	var req struct {
		WeekStart   string  `json:"week_start" binding:"required"`
		WeekEnd     string  `json:"week_end" binding:"required"`
		Summary     string  `json:"summary"`
		NextWeekPlan string `json:"next_week_plan"`
		Status      string  `json:"status"`
		ProjectID   *uint   `json:"project_id"`
		TaskID      *uint   `json:"task_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "未授权")
		return
	}

	// 解析日期
	weekStart, err := time.Parse("2006-01-02", req.WeekStart)
	if err != nil {
		utils.Error(c, 400, "周开始日期格式错误")
		return
	}
	weekEnd, err := time.Parse("2006-01-02", req.WeekEnd)
	if err != nil {
		utils.Error(c, 400, "周结束日期格式错误")
		return
	}

	if weekStart.After(weekEnd) {
		utils.Error(c, 400, "周开始日期不能晚于周结束日期")
		return
	}

	// 设置默认状态
	if req.Status == "" {
		req.Status = "draft"
	}

	report := model.WeeklyReport{
		WeekStart:    weekStart,
		WeekEnd:      weekEnd,
		Summary:      req.Summary,
		NextWeekPlan: req.NextWeekPlan,
		Status:       req.Status,
		UserID:       userID.(uint),
		ProjectID:    req.ProjectID,
		TaskID:       req.TaskID,
	}

	if err := h.db.Create(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "创建失败")
		return
	}

	h.db.Preload("User").Preload("Project").Preload("Task").First(&report, report.ID)
	utils.Success(c, report)
}

// UpdateWeeklyReport 更新周报
func (h *ReportHandler) UpdateWeeklyReport(c *gin.Context) {
	id := c.Param("id")
	var report model.WeeklyReport
	if err := h.db.First(&report, id).Error; err != nil {
		utils.Error(c, 404, "周报不存在")
		return
	}

	// 权限检查：普通用户只能更新自己的报告
	userID, _ := c.Get("user_id")
	if !utils.IsAdmin(c) && report.UserID != userID.(uint) {
		utils.Error(c, 403, "没有权限更新该报告")
		return
	}

	var req struct {
		WeekStart    *string `json:"week_start"`
		WeekEnd      *string `json:"week_end"`
		Summary      *string `json:"summary"`
		NextWeekPlan *string `json:"next_week_plan"`
		Status       *string `json:"status"`
		ProjectID    *uint   `json:"project_id"`
		TaskID       *uint   `json:"task_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	if req.WeekStart != nil {
		weekStart, err := time.Parse("2006-01-02", *req.WeekStart)
		if err != nil {
			utils.Error(c, 400, "周开始日期格式错误")
			return
		}
		report.WeekStart = weekStart
	}

	if req.WeekEnd != nil {
		weekEnd, err := time.Parse("2006-01-02", *req.WeekEnd)
		if err != nil {
			utils.Error(c, 400, "周结束日期格式错误")
			return
		}
		report.WeekEnd = weekEnd
	}

	if req.WeekStart != nil && req.WeekEnd != nil {
		if report.WeekStart.After(report.WeekEnd) {
			utils.Error(c, 400, "周开始日期不能晚于周结束日期")
			return
		}
	}

	if req.Summary != nil {
		report.Summary = *req.Summary
	}
	if req.NextWeekPlan != nil {
		report.NextWeekPlan = *req.NextWeekPlan
	}
	if req.Status != nil {
		report.Status = *req.Status
	}
	if req.ProjectID != nil {
		report.ProjectID = req.ProjectID
	}
	if req.TaskID != nil {
		report.TaskID = req.TaskID
	}

	if err := h.db.Save(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	h.db.Preload("User").Preload("Project").Preload("Task").First(&report, report.ID)
	utils.Success(c, report)
}

// DeleteWeeklyReport 删除周报
func (h *ReportHandler) DeleteWeeklyReport(c *gin.Context) {
	id := c.Param("id")
	var report model.WeeklyReport
	if err := h.db.First(&report, id).Error; err != nil {
		utils.Error(c, 404, "周报不存在")
		return
	}

	// 权限检查：普通用户只能删除自己的报告
	userID, _ := c.Get("user_id")
	if !utils.IsAdmin(c) && report.UserID != userID.(uint) {
		utils.Error(c, 403, "没有权限删除该报告")
		return
	}

	if err := h.db.Delete(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除失败")
		return
	}

	utils.Success(c, nil)
}

// UpdateWeeklyReportStatus 更新周报状态
func (h *ReportHandler) UpdateWeeklyReportStatus(c *gin.Context) {
	id := c.Param("id")
	var report model.WeeklyReport
	if err := h.db.First(&report, id).Error; err != nil {
		utils.Error(c, 404, "周报不存在")
		return
	}

	// 权限检查：普通用户只能更新自己的报告状态（提交），管理员可以审批
	userID, _ := c.Get("user_id")
	if !utils.IsAdmin(c) && report.UserID != userID.(uint) {
		utils.Error(c, 403, "没有权限更新该报告")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	validStatuses := map[string]bool{
		"draft":     true,
		"submitted": true,
		"approved":  true,
	}
	if !validStatuses[req.Status] {
		utils.Error(c, 400, "状态值无效")
		return
	}

	report.Status = req.Status
	if err := h.db.Save(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	h.db.Preload("User").Preload("Project").Preload("Task").First(&report, report.ID)
	utils.Success(c, report)
}

