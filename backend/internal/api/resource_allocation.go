package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"project-management/internal/model"
	"project-management/internal/utils"
)

type ResourceAllocationHandler struct {
	db *gorm.DB
}

func NewResourceAllocationHandler(db *gorm.DB) *ResourceAllocationHandler {
	return &ResourceAllocationHandler{db: db}
}

// GetResourceAllocations 获取资源分配列表
func (h *ResourceAllocationHandler) GetResourceAllocations(c *gin.Context) {
	var allocations []model.ResourceAllocation
	query := h.db.Model(&model.ResourceAllocation{}).
		Preload("Resource").Preload("Resource.User").Preload("Resource.Project").Preload("Task").Preload("Bug").Preload("Project")

	// 用户筛选（通过资源）
	requestUserIDStr := c.Query("user_id")
	var requestUserID uint
	hasUserFilter := false
	if requestUserIDStr != "" {
		// 解析用户ID
		if _, err := fmt.Sscanf(requestUserIDStr, "%d", &requestUserID); err == nil {
			hasUserFilter = true
			query = query.Joins("JOIN resources AS filter_resources ON resource_allocations.resource_id = filter_resources.id").
				Where("filter_resources.user_id = ?", requestUserID)
		}
	}

	// 权限过滤：普通用户只能看到自己参与的项目相关的资源分配
	// 包括：1) 直接通过 project_id 关联的项目 2) 通过 resource 关联的项目
	if !utils.IsAdmin(c) {
		currentUserID := utils.GetUserID(c)
		if currentUserID == 0 {
			query = query.Where("1 = 0")
		} else {
			// 如果用户筛选的是自己，则已经通过JOIN过滤，不需要额外的权限检查
			if hasUserFilter && requestUserID == currentUserID {
				// 已经通过JOIN过滤，允许查看
			} else {
				// 获取用户参与的项目ID列表
				projectIDs := utils.GetUserProjectIDs(h.db, currentUserID)
				if len(projectIDs) > 0 {
					if hasUserFilter {
						// 如果已经JOIN了resources表，直接使用JOIN后的表
						query = query.Where(
							"(resource_allocations.project_id IN ? OR filter_resources.project_id IN ?)",
							projectIDs, projectIDs)
					} else {
						// 如果没有JOIN，使用EXISTS子查询
						query = query.Where(
							"(resource_allocations.project_id IN ? OR EXISTS (SELECT 1 FROM resources WHERE resources.id = resource_allocations.resource_id AND resources.project_id IN ?))",
							projectIDs, projectIDs)
					}
				} else {
					query = query.Where("1 = 0")
				}
			}
		}
	}

	// 资源筛选
	if resourceID := c.Query("resource_id"); resourceID != "" {
		query = query.Where("resource_allocations.resource_id = ?", resourceID)
	}

	// 项目筛选
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("resource_allocations.project_id = ?", projectID)
	}

	// 任务筛选
	if taskID := c.Query("task_id"); taskID != "" {
		query = query.Where("resource_allocations.task_id = ?", taskID)
	}

	// Bug筛选
	if bugID := c.Query("bug_id"); bugID != "" {
		query = query.Where("resource_allocations.bug_id = ?", bugID)
	}

	// 日期范围筛选
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("resource_allocations.date >= ?", t)
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			query = query.Where("resource_allocations.date <= ?", t)
		}
	}

	// 分页
	page := utils.GetPage(c)
	pageSize := utils.GetPageSize(c)
	offset := (page - 1) * pageSize

	var total int64
	countQuery := h.db.Model(&model.ResourceAllocation{})
	
	// 用户筛选（通过资源）
	countRequestUserIDStr := c.Query("user_id")
	var countRequestUserID uint
	countHasUserFilter := false
	if countRequestUserIDStr != "" {
		if _, err := fmt.Sscanf(countRequestUserIDStr, "%d", &countRequestUserID); err == nil {
			countHasUserFilter = true
			countQuery = countQuery.Joins("JOIN resources AS filter_resources ON resource_allocations.resource_id = filter_resources.id").
				Where("filter_resources.user_id = ?", countRequestUserID)
		}
	}

	// 权限过滤：普通用户只能看到自己参与的项目相关的资源分配
	// 包括：1) 直接通过 project_id 关联的项目 2) 通过 resource 关联的项目
	if !utils.IsAdmin(c) {
		currentUserID := utils.GetUserID(c)
		if currentUserID == 0 {
			countQuery = countQuery.Where("1 = 0")
		} else {
			// 如果用户筛选的是自己，则已经通过JOIN过滤，不需要额外的权限检查
			if countHasUserFilter && countRequestUserID == currentUserID {
				// 已经通过JOIN过滤，允许查看
			} else {
				// 获取用户参与的项目ID列表
				projectIDs := utils.GetUserProjectIDs(h.db, currentUserID)
				if len(projectIDs) > 0 {
					if countHasUserFilter {
						// 如果已经JOIN了resources表，直接使用JOIN后的表
						countQuery = countQuery.Where(
							"(resource_allocations.project_id IN ? OR filter_resources.project_id IN ?)",
							projectIDs, projectIDs)
					} else {
						// 如果没有JOIN，使用EXISTS子查询
						countQuery = countQuery.Where(
							"(resource_allocations.project_id IN ? OR EXISTS (SELECT 1 FROM resources WHERE resources.id = resource_allocations.resource_id AND resources.project_id IN ?))",
							projectIDs, projectIDs)
					}
				} else {
					countQuery = countQuery.Where("1 = 0")
				}
			}
		}
	}

	// 资源筛选
	if resourceID := c.Query("resource_id"); resourceID != "" {
		countQuery = countQuery.Where("resource_allocations.resource_id = ?", resourceID)
	}

	// 项目筛选
	if projectID := c.Query("project_id"); projectID != "" {
		countQuery = countQuery.Where("resource_allocations.project_id = ?", projectID)
	}

	// 任务筛选
	if taskID := c.Query("task_id"); taskID != "" {
		countQuery = countQuery.Where("resource_allocations.task_id = ?", taskID)
	}

	// Bug筛选
	if bugID := c.Query("bug_id"); bugID != "" {
		countQuery = countQuery.Where("resource_allocations.bug_id = ?", bugID)
	}

	// 日期范围筛选
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			countQuery = countQuery.Where("resource_allocations.date >= ?", t)
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			countQuery = countQuery.Where("resource_allocations.date <= ?", t)
		}
	}

	countQuery.Count(&total)

	if err := query.Offset(offset).Limit(pageSize).Order("resource_allocations.date DESC, resource_allocations.created_at DESC").Find(&allocations).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, gin.H{
		"list":      allocations,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetResourceAllocation 获取资源分配详情
func (h *ResourceAllocationHandler) GetResourceAllocation(c *gin.Context) {
	id := c.Param("id")
	var allocation model.ResourceAllocation
	if err := h.db.Preload("Resource").Preload("Resource.User").Preload("Resource.Project").Preload("Task").Preload("Bug").Preload("Project").First(&allocation, id).Error; err != nil {
		utils.Error(c, 404, "资源分配不存在")
		return
	}

	// 权限检查：普通用户只能查看自己参与的项目相关的资源分配
	if !utils.IsAdmin(c) {
		if allocation.ProjectID != nil && !utils.CheckProjectAccess(h.db, c, *allocation.ProjectID) {
			utils.Error(c, 403, "没有权限访问该资源分配")
			return
		}
	}

	utils.Success(c, allocation)
}

// CreateResourceAllocation 创建资源分配
func (h *ResourceAllocationHandler) CreateResourceAllocation(c *gin.Context) {
	var req struct {
		ResourceID  uint    `json:"resource_id" binding:"required"`
		Date        string  `json:"date" binding:"required"` // 接收字符串格式的日期
		Hours       float64 `json:"hours" binding:"required,gt=0"`
		TaskID      *uint   `json:"task_id"`
		BugID       *uint   `json:"bug_id"`
		ProjectID   *uint   `json:"project_id"`
		Description string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 验证资源是否存在
	var resource model.Resource
	if err := h.db.First(&resource, req.ResourceID).Error; err != nil {
		utils.Error(c, 404, "资源不存在")
		return
	}

	// 解析日期
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		utils.Error(c, 400, "日期格式错误，应为 YYYY-MM-DD")
		return
	}

	// 验证任务是否存在（如果提供了任务ID）
	if req.TaskID != nil && *req.TaskID > 0 {
		var task model.Task
		if err := h.db.First(&task, *req.TaskID).Error; err != nil {
			utils.Error(c, 404, "任务不存在")
			return
		}
	}

	// 验证Bug是否存在（如果提供了BugID）
	if req.BugID != nil && *req.BugID > 0 {
		var bug model.Bug
		if err := h.db.First(&bug, *req.BugID).Error; err != nil {
			utils.Error(c, 404, "Bug不存在")
			return
		}
	}

	// 验证项目是否存在（如果提供了项目ID）
	if req.ProjectID != nil && *req.ProjectID > 0 {
		var project model.Project
		if err := h.db.First(&project, *req.ProjectID).Error; err != nil {
			utils.Error(c, 404, "项目不存在")
			return
		}
	}

	// 检查资源冲突（同一资源在同一天的总工时不应超过24小时）
	var totalHours float64
	h.db.Model(&model.ResourceAllocation{}).
		Where("resource_id = ? AND date = ?", req.ResourceID, date).
		Select("COALESCE(SUM(hours), 0)").
		Scan(&totalHours)

	if totalHours+req.Hours > 24 {
		utils.Error(c, 400, "该资源在指定日期的总工时不能超过24小时")
		return
	}

	allocation := model.ResourceAllocation{
		ResourceID:  req.ResourceID,
		Date:        date,
		Hours:       req.Hours,
		TaskID:      req.TaskID,
		BugID:       req.BugID,
		ProjectID:   req.ProjectID,
		Description: req.Description,
	}

	if err := h.db.Create(&allocation).Error; err != nil {
		utils.Error(c, utils.CodeError, "创建失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Resource").Preload("Resource.User").Preload("Resource.Project").Preload("Task").Preload("Bug").Preload("Project").First(&allocation, allocation.ID)

	utils.Success(c, allocation)
}

// UpdateResourceAllocation 更新资源分配
func (h *ResourceAllocationHandler) UpdateResourceAllocation(c *gin.Context) {
	id := c.Param("id")
	var allocation model.ResourceAllocation
	if err := h.db.First(&allocation, id).Error; err != nil {
		utils.Error(c, 404, "资源分配不存在")
		return
	}

	var req struct {
		Date        *string  `json:"date"` // 接收字符串格式的日期
		Hours       *float64 `json:"hours"`
		TaskID      *uint    `json:"task_id"`
		BugID       *uint    `json:"bug_id"`
		ProjectID   *uint    `json:"project_id"`
		Description *string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 解析日期
	if req.Date != nil {
		if date, err := time.Parse("2006-01-02", *req.Date); err == nil {
			allocation.Date = date
		} else {
			utils.Error(c, 400, "日期格式错误，应为 YYYY-MM-DD")
			return
		}
	}

	if req.Hours != nil {
		if *req.Hours <= 0 {
			utils.Error(c, 400, "工时必须大于0")
			return
		}
		allocation.Hours = *req.Hours
	}
	if req.TaskID != nil {
		allocation.TaskID = req.TaskID
	}
	if req.BugID != nil {
		allocation.BugID = req.BugID
	}
	if req.ProjectID != nil {
		allocation.ProjectID = req.ProjectID
	}
	if req.Description != nil {
		allocation.Description = *req.Description
	}

	// 检查资源冲突（更新时排除当前记录）
	if req.Hours != nil || req.Date != nil {
		var totalHours float64
		date := allocation.Date
		if req.Date != nil {
			if t, err := time.Parse("2006-01-02", *req.Date); err == nil {
				date = t
			}
		}
		hours := allocation.Hours
		if req.Hours != nil {
			hours = *req.Hours
		}
		h.db.Model(&model.ResourceAllocation{}).
			Where("resource_id = ? AND date = ? AND id != ?", allocation.ResourceID, date, allocation.ID).
			Select("COALESCE(SUM(hours), 0)").
			Scan(&totalHours)

		if totalHours+hours > 24 {
			utils.Error(c, 400, "该资源在指定日期的总工时不能超过24小时")
			return
		}
	}

	if err := h.db.Save(&allocation).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Resource").Preload("Resource.User").Preload("Resource.Project").Preload("Task").Preload("Bug").Preload("Project").First(&allocation, allocation.ID)

	utils.Success(c, allocation)
}

// DeleteResourceAllocation 删除资源分配
func (h *ResourceAllocationHandler) DeleteResourceAllocation(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&model.ResourceAllocation{}, id).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除失败")
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}

// GetResourceCalendar 获取资源日历数据
func (h *ResourceAllocationHandler) GetResourceCalendar(c *gin.Context) {
	// 获取查询参数
	userID := c.Query("user_id")
	projectID := c.Query("project_id")
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	// 默认查询当前月份
	var startDate, endDate time.Time
	now := time.Now()
	if startDateStr == "" {
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	} else {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = t
		} else {
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		}
	}
	if endDateStr == "" {
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, 0).AddDate(0, 0, -1)
	} else {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = t
		} else {
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			endDate = startDate.AddDate(0, 1, 0).AddDate(0, 0, -1)
		}
	}

	query := h.db.Model(&model.ResourceAllocation{}).
		Preload("Resource").Preload("Resource.User").Preload("Resource.Project").Preload("Task").Preload("Bug").Preload("Project").
		Where("date >= ? AND date <= ?", startDate, endDate)

	// 用户筛选
	if userID != "" {
		query = query.Joins("JOIN resources ON resource_allocations.resource_id = resources.id").
			Where("resources.user_id = ?", userID)
	}

	// 项目筛选
	if projectID != "" {
		query = query.Where("resource_allocations.project_id = ?", projectID)
	}

	var allocations []model.ResourceAllocation
	if err := query.Order("date ASC").Find(&allocations).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	// 按日期分组
	calendarData := make(map[string][]model.ResourceAllocation)
	for _, allocation := range allocations {
		dateKey := allocation.Date.Format("2006-01-02")
		calendarData[dateKey] = append(calendarData[dateKey], allocation)
	}

	utils.Success(c, gin.H{
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
		"data":       calendarData,
	})
}

// CheckResourceConflict 检查资源冲突
func (h *ResourceAllocationHandler) CheckResourceConflict(c *gin.Context) {
	resourceID := c.Query("resource_id")
	dateStr := c.Query("date")

	if resourceID == "" || dateStr == "" {
		utils.Error(c, 400, "需要提供resource_id和date参数")
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		utils.Error(c, 400, "日期格式错误，应为 YYYY-MM-DD")
		return
	}

	var totalHours float64
	h.db.Model(&model.ResourceAllocation{}).
		Where("resource_id = ? AND date = ?", resourceID, date).
		Select("COALESCE(SUM(hours), 0)").
		Scan(&totalHours)

	conflicts := []string{}
	if totalHours > 24 {
		conflicts = append(conflicts, "总工时超过24小时")
	}

	utils.Success(c, gin.H{
		"resource_id": resourceID,
		"date":        dateStr,
		"total_hours": totalHours,
		"conflicts":   conflicts,
		"has_conflict": totalHours > 24,
	})
}

