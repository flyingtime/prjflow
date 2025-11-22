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
	query := h.db.Preload("User").Preload("Project").Preload("Tasks").Preload("Approvers").Preload("ApprovalRecords.Approver")

	// 检查是否是获取审批列表
	forApproval := c.Query("for_approval") == "true"
	uid := userID.(uint)
	if !utils.IsAdmin(c) {
		if forApproval {
			// 获取需要当前用户审批的报告
			query = query.Where("id IN (SELECT daily_report_id FROM daily_report_approvers WHERE user_id = ?)", uid)
		} else {
			// 只查询自己创建的
			query = query.Where("user_id = ?", uid)
		}
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
		if forApproval {
			// 获取需要当前用户审批的报告
			countQuery = countQuery.Where("id IN (SELECT daily_report_id FROM daily_report_approvers WHERE user_id = ?)", uid)
		} else {
			// 只查询自己创建的
			countQuery = countQuery.Where("user_id = ?", uid)
		}
	} else {
		// 管理员可以筛选用户
		if filterUserID := c.Query("user_id"); filterUserID != "" {
			countQuery = countQuery.Where("user_id = ?", filterUserID)
		}
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
	if err := h.db.Preload("User").Preload("Project").Preload("Tasks").Preload("Approvers").Preload("ApprovalRecords.Approver").First(&report, id).Error; err != nil {
		utils.Error(c, 404, "日报不存在")
		return
	}

	// 权限检查：普通用户可以查看自己的报告或需要审批的报告
	userID, _ := c.Get("user_id")
	uid := userID.(uint)
	if !utils.IsAdmin(c) {
		// 检查是否是报告创建者
		if report.UserID != uid {
			// 检查是否是审批人
			isApprover := false
			for _, approver := range report.Approvers {
				if approver.ID == uid {
					isApprover = true
					break
				}
			}
			if !isApprover {
				utils.Error(c, 403, "没有权限访问该报告")
				return
			}
		}
	}

	utils.Success(c, report)
}

// CreateDailyReport 创建日报
func (h *ReportHandler) CreateDailyReport(c *gin.Context) {
	var req struct {
		Date        string   `json:"date" binding:"required"`
		Content     string   `json:"content"`
		Hours       *float64 `json:"hours"`
		Status      string   `json:"status"`
		ProjectID   *uint    `json:"project_id"`
		TaskIDs     []uint   `json:"task_ids"`     // 任务ID数组（多选）
		ApproverIDs []uint   `json:"approver_ids"`  // 审批人ID数组（多选）
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
	}

	if req.Hours != nil {
		report.Hours = *req.Hours
	}

	// 创建日报
	if err := h.db.Create(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "创建失败")
		return
	}

	// 关联任务（多对多）
	if len(req.TaskIDs) > 0 {
		var tasks []model.Task
		if err := h.db.Where("id IN ?", req.TaskIDs).Find(&tasks).Error; err == nil {
			h.db.Model(&report).Association("Tasks").Replace(tasks)
		}
	}

	// 关联审批人（多对多）
	if len(req.ApproverIDs) > 0 {
		var approvers []model.User
		if err := h.db.Where("id IN ?", req.ApproverIDs).Find(&approvers).Error; err == nil {
			h.db.Model(&report).Association("Approvers").Replace(approvers)
			// 为每个审批人创建待审批记录
			for _, approver := range approvers {
				approval := model.DailyReportApproval{
					DailyReportID: report.ID,
					ApproverID:    approver.ID,
					Status:        "pending",
				}
				h.db.Create(&approval)
			}
		}
	}

	h.db.Preload("User").Preload("Project").Preload("Tasks").Preload("Approvers").Preload("ApprovalRecords.Approver").First(&report, report.ID)
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
		Date        *string `json:"date"`
		Content     *string `json:"content"`
		Hours       *float64 `json:"hours"`
		Status      *string `json:"status"`
		ProjectID   *uint   `json:"project_id"`
		TaskIDs     []uint  `json:"task_ids"`     // 任务ID数组（多选）
		ApproverIDs []uint  `json:"approver_ids"` // 审批人ID数组（多选）
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

	// 更新任务关联（多对多）
	if req.TaskIDs != nil {
		var tasks []model.Task
		if len(req.TaskIDs) > 0 {
			if err := h.db.Where("id IN ?", req.TaskIDs).Find(&tasks).Error; err == nil {
				h.db.Model(&report).Association("Tasks").Replace(tasks)
			}
		} else {
			// 如果传入空数组，清空所有任务关联
			h.db.Model(&report).Association("Tasks").Clear()
		}
	}

	// 更新审批人关联（多对多）
	if req.ApproverIDs != nil {
		var approvers []model.User
		if len(req.ApproverIDs) > 0 {
			if err := h.db.Where("id IN ?", req.ApproverIDs).Find(&approvers).Error; err == nil {
				h.db.Model(&report).Association("Approvers").Replace(approvers)
				// 删除旧的审批记录
				h.db.Where("daily_report_id = ?", report.ID).Delete(&model.DailyReportApproval{})
				// 为每个审批人创建待审批记录
				for _, approver := range approvers {
					approval := model.DailyReportApproval{
						DailyReportID: report.ID,
						ApproverID:    approver.ID,
						Status:        "pending",
					}
					h.db.Create(&approval)
				}
			}
		} else {
			// 如果传入空数组，清空所有审批人关联和审批记录
			h.db.Model(&report).Association("Approvers").Clear()
			h.db.Where("daily_report_id = ?", report.ID).Delete(&model.DailyReportApproval{})
		}
	}

	if err := h.db.Save(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	h.db.Preload("User").Preload("Project").Preload("Tasks").Preload("Approvers").Preload("ApprovalRecords.Approver").First(&report, report.ID)
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

	// 删除关联的审批记录
	h.db.Where("daily_report_id = ?", report.ID).Delete(&model.DailyReportApproval{})
	// 删除关联的审批人关联
	h.db.Model(&report).Association("Approvers").Clear()
	// 删除关联的任务关联
	h.db.Model(&report).Association("Tasks").Clear()

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

	h.db.Preload("User").Preload("Project").Preload("Tasks").Preload("Approvers").Preload("ApprovalRecords.Approver").First(&report, report.ID)
	utils.Success(c, report)
}

// ApproveDailyReport 审批日报
func (h *ReportHandler) ApproveDailyReport(c *gin.Context) {
	id := c.Param("id")
	var report model.DailyReport
	if err := h.db.Preload("Approvers").First(&report, id).Error; err != nil {
		utils.Error(c, 404, "日报不存在")
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	// 检查当前用户是否是审批人
	isApprover := false
	for _, approver := range report.Approvers {
		if approver.ID == uid {
			isApprover = true
			break
		}
	}

	if !isApprover && !utils.IsAdmin(c) {
		utils.Error(c, 403, "您不是该报告的审批人")
		return
	}

	var req struct {
		Status  string `json:"status" binding:"required"` // approved 或 rejected
		Comment string `json:"comment"`                 // 批注
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	if req.Status != "approved" && req.Status != "rejected" {
		utils.Error(c, 400, "状态必须是 approved 或 rejected")
		return
	}

	// 查找或创建审批记录
	var approval model.DailyReportApproval
	if err := h.db.Where("daily_report_id = ? AND approver_id = ?", report.ID, uid).First(&approval).Error; err != nil {
		// 如果不存在，创建新的审批记录
		approval = model.DailyReportApproval{
			DailyReportID: report.ID,
			ApproverID:    uid,
			Status:        req.Status,
			Comment:       req.Comment,
		}
		if err := h.db.Create(&approval).Error; err != nil {
			utils.Error(c, utils.CodeError, "创建审批记录失败")
			return
		}
	} else {
		// 更新现有审批记录
		approval.Status = req.Status
		approval.Comment = req.Comment
		if err := h.db.Save(&approval).Error; err != nil {
			utils.Error(c, utils.CodeError, "更新审批记录失败")
			return
		}
	}

	// 检查是否所有审批人都已审批
	var pendingCount int64
	h.db.Model(&model.DailyReportApproval{}).Where("daily_report_id = ? AND status = ?", report.ID, "pending").Count(&pendingCount)
	if pendingCount == 0 {
		// 所有审批人都已审批，检查是否有拒绝的
		var rejectedCount int64
		h.db.Model(&model.DailyReportApproval{}).Where("daily_report_id = ? AND status = ?", report.ID, "rejected").Count(&rejectedCount)
		if rejectedCount > 0 {
			report.Status = "rejected"
		} else {
			report.Status = "approved"
		}
		h.db.Save(&report)
	}

	h.db.Preload("User").Preload("Project").Preload("Tasks").Preload("Approvers").Preload("ApprovalRecords.Approver").First(&report, report.ID)
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
	query := h.db.Preload("User").Preload("Project").Preload("Tasks").Preload("Approvers").Preload("ApprovalRecords.Approver")

	// 检查是否是获取审批列表
	forApproval := c.Query("for_approval") == "true"
	uid := userID.(uint)
	if !utils.IsAdmin(c) {
		if forApproval {
			// 获取需要当前用户审批的报告
			query = query.Where("id IN (SELECT weekly_report_id FROM weekly_report_approvers WHERE user_id = ?)", uid)
		} else {
			// 只查询自己创建的
			query = query.Where("user_id = ?", uid)
		}
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
		if forApproval {
			// 获取需要当前用户审批的报告
			countQuery = countQuery.Where("id IN (SELECT weekly_report_id FROM weekly_report_approvers WHERE user_id = ?)", uid)
		} else {
			// 只查询自己创建的
			countQuery = countQuery.Where("user_id = ?", uid)
		}
	} else {
		// 管理员可以筛选用户
		if filterUserID := c.Query("user_id"); filterUserID != "" {
			countQuery = countQuery.Where("user_id = ?", filterUserID)
		}
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
	if err := h.db.Preload("User").Preload("Project").Preload("Tasks").Preload("Approvers").Preload("ApprovalRecords.Approver").First(&report, id).Error; err != nil {
		utils.Error(c, 404, "周报不存在")
		return
	}

	// 权限检查：普通用户可以查看自己的报告或需要审批的报告
	userID, _ := c.Get("user_id")
	uid := userID.(uint)
	if !utils.IsAdmin(c) {
		// 检查是否是报告创建者
		if report.UserID != uid {
			// 检查是否是审批人
			isApprover := false
			for _, approver := range report.Approvers {
				if approver.ID == uid {
					isApprover = true
					break
				}
			}
			if !isApprover {
				utils.Error(c, 403, "没有权限访问该报告")
				return
			}
		}
	}

	utils.Success(c, report)
}

// CreateWeeklyReport 创建周报
func (h *ReportHandler) CreateWeeklyReport(c *gin.Context) {
	var req struct {
		WeekStart    string  `json:"week_start" binding:"required"`
		WeekEnd      string  `json:"week_end" binding:"required"`
		Summary      string  `json:"summary"`
		NextWeekPlan string  `json:"next_week_plan"`
		Status       string  `json:"status"`
		ProjectID    *uint   `json:"project_id"`
		TaskIDs      []uint  `json:"task_ids"`     // 任务ID数组（多选）
		ApproverIDs  []uint  `json:"approver_ids"` // 审批人ID数组（多选）
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
	}

	// 创建周报
	if err := h.db.Create(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "创建失败")
		return
	}

	// 关联任务（多对多）
	if len(req.TaskIDs) > 0 {
		var tasks []model.Task
		if err := h.db.Where("id IN ?", req.TaskIDs).Find(&tasks).Error; err == nil {
			h.db.Model(&report).Association("Tasks").Replace(tasks)
		}
	}

	// 关联审批人（多对多）
	if len(req.ApproverIDs) > 0 {
		var approvers []model.User
		if err := h.db.Where("id IN ?", req.ApproverIDs).Find(&approvers).Error; err == nil {
			h.db.Model(&report).Association("Approvers").Replace(approvers)
			// 为每个审批人创建待审批记录
			for _, approver := range approvers {
				approval := model.WeeklyReportApproval{
					WeeklyReportID: report.ID,
					ApproverID:     approver.ID,
					Status:         "pending",
				}
				h.db.Create(&approval)
			}
		}
	}

	h.db.Preload("User").Preload("Project").Preload("Tasks").Preload("Approvers").Preload("ApprovalRecords.Approver").First(&report, report.ID)
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
		TaskIDs      []uint  `json:"task_ids"`     // 任务ID数组（多选）
		ApproverIDs  []uint  `json:"approver_ids"` // 审批人ID数组（多选）
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

	// 更新任务关联（多对多）
	if req.TaskIDs != nil {
		var tasks []model.Task
		if len(req.TaskIDs) > 0 {
			if err := h.db.Where("id IN ?", req.TaskIDs).Find(&tasks).Error; err == nil {
				h.db.Model(&report).Association("Tasks").Replace(tasks)
			}
		} else {
			// 如果传入空数组，清空所有任务关联
			h.db.Model(&report).Association("Tasks").Clear()
		}
	}

	// 更新审批人关联（多对多）
	if req.ApproverIDs != nil {
		var approvers []model.User
		if len(req.ApproverIDs) > 0 {
			if err := h.db.Where("id IN ?", req.ApproverIDs).Find(&approvers).Error; err == nil {
				h.db.Model(&report).Association("Approvers").Replace(approvers)
				// 删除旧的审批记录
				h.db.Where("weekly_report_id = ?", report.ID).Delete(&model.WeeklyReportApproval{})
				// 为每个审批人创建待审批记录
				for _, approver := range approvers {
					approval := model.WeeklyReportApproval{
						WeeklyReportID: report.ID,
						ApproverID:     approver.ID,
						Status:         "pending",
					}
					h.db.Create(&approval)
				}
			}
		} else {
			// 如果传入空数组，清空所有审批人关联和审批记录
			h.db.Model(&report).Association("Approvers").Clear()
			h.db.Where("weekly_report_id = ?", report.ID).Delete(&model.WeeklyReportApproval{})
		}
	}

	if err := h.db.Save(&report).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	h.db.Preload("User").Preload("Project").Preload("Tasks").Preload("Approvers").Preload("ApprovalRecords.Approver").First(&report, report.ID)
	utils.Success(c, report)
}

// ApproveWeeklyReport 审批周报
func (h *ReportHandler) ApproveWeeklyReport(c *gin.Context) {
	id := c.Param("id")
	var report model.WeeklyReport
	if err := h.db.Preload("Approvers").First(&report, id).Error; err != nil {
		utils.Error(c, 404, "周报不存在")
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	// 检查当前用户是否是审批人
	isApprover := false
	for _, approver := range report.Approvers {
		if approver.ID == uid {
			isApprover = true
			break
		}
	}

	if !isApprover && !utils.IsAdmin(c) {
		utils.Error(c, 403, "您不是该报告的审批人")
		return
	}

	var req struct {
		Status  string `json:"status" binding:"required"` // approved 或 rejected
		Comment string `json:"comment"`                 // 批注
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	if req.Status != "approved" && req.Status != "rejected" {
		utils.Error(c, 400, "状态必须是 approved 或 rejected")
		return
	}

	// 查找或创建审批记录
	var approval model.WeeklyReportApproval
	if err := h.db.Where("weekly_report_id = ? AND approver_id = ?", report.ID, uid).First(&approval).Error; err != nil {
		// 如果不存在，创建新的审批记录
		approval = model.WeeklyReportApproval{
			WeeklyReportID: report.ID,
			ApproverID:    uid,
			Status:         req.Status,
			Comment:        req.Comment,
		}
		if err := h.db.Create(&approval).Error; err != nil {
			utils.Error(c, utils.CodeError, "创建审批记录失败")
			return
		}
	} else {
		// 更新现有审批记录
		approval.Status = req.Status
		approval.Comment = req.Comment
		if err := h.db.Save(&approval).Error; err != nil {
			utils.Error(c, utils.CodeError, "更新审批记录失败")
			return
		}
	}

	// 检查是否所有审批人都已审批
	var pendingCount int64
	h.db.Model(&model.WeeklyReportApproval{}).Where("weekly_report_id = ? AND status = ?", report.ID, "pending").Count(&pendingCount)
	if pendingCount == 0 {
		// 所有审批人都已审批，检查是否有拒绝的
		var rejectedCount int64
		h.db.Model(&model.WeeklyReportApproval{}).Where("weekly_report_id = ? AND status = ?", report.ID, "rejected").Count(&rejectedCount)
		if rejectedCount > 0 {
			report.Status = "rejected"
		} else {
			report.Status = "approved"
		}
		h.db.Save(&report)
	}

	h.db.Preload("User").Preload("Project").Preload("Tasks").Preload("Approvers").Preload("ApprovalRecords.Approver").First(&report, report.ID)
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

	// 删除关联的审批记录
	h.db.Where("weekly_report_id = ?", report.ID).Delete(&model.WeeklyReportApproval{})
	// 删除关联的审批人关联
	h.db.Model(&report).Association("Approvers").Clear()
	// 删除关联的任务关联
	h.db.Model(&report).Association("Tasks").Clear()

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

