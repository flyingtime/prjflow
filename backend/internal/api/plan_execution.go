package api

import (
	"time"

	"project-management/internal/model"
	"project-management/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PlanExecutionHandler struct {
	db *gorm.DB
}

func NewPlanExecutionHandler(db *gorm.DB) *PlanExecutionHandler {
	return &PlanExecutionHandler{db: db}
}

// GetPlanExecutions 获取计划执行列表
func (h *PlanExecutionHandler) GetPlanExecutions(c *gin.Context) {
	planID := c.Param("id")
	var executions []model.PlanExecution
	query := h.db.Where("plan_id = ?", planID).Preload("Assignee").Preload("Task")

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 负责人筛选
	if assigneeID := c.Query("assignee_id"); assigneeID != "" {
		query = query.Where("assignee_id = ?", assigneeID)
	}

	if err := query.Order("created_at ASC").Find(&executions).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, executions)
}

// GetPlanExecution 获取计划执行详情
func (h *PlanExecutionHandler) GetPlanExecution(c *gin.Context) {
	id := c.Param("execution_id")
	var execution model.PlanExecution
	if err := h.db.Preload("Plan").Preload("Assignee").Preload("Task").First(&execution, id).Error; err != nil {
		utils.Error(c, 404, "执行不存在")
		return
	}

	utils.Success(c, execution)
}

// CreatePlanExecution 创建计划执行
func (h *PlanExecutionHandler) CreatePlanExecution(c *gin.Context) {
	planID := c.Param("id")

	// 验证计划是否存在
	var plan model.Plan
	if err := h.db.First(&plan, planID).Error; err != nil {
		utils.Error(c, 404, "计划不存在")
		return
	}

	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		Status      string  `json:"status"`
		Progress    int     `json:"progress"`
		StartDate   *string `json:"start_date"` // 接收字符串格式的日期
		EndDate     *string `json:"end_date"`   // 接收字符串格式的日期
		AssigneeID  *uint   `json:"assignee_id"`
		TaskID      *uint   `json:"task_id"` // 关联的任务ID
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 解析日期
	var startDate, endDate *time.Time
	if req.StartDate != nil && *req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			startDate = &t
		}
	}
	if req.EndDate != nil && *req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			endDate = &t
		}
	}

	// 验证状态
	if req.Status == "" {
		req.Status = "pending"
	}
	if !isValidExecutionStatus(req.Status) {
		utils.Error(c, 400, "无效的执行状态")
		return
	}

	// 验证进度
	if req.Progress < 0 || req.Progress > 100 {
		utils.Error(c, 400, "进度必须在0-100之间")
		return
	}

	// 如果指定了任务，验证任务是否存在
	if req.TaskID != nil {
		var task model.Task
		if err := h.db.First(&task, *req.TaskID).Error; err != nil {
			utils.Error(c, 404, "任务不存在")
			return
		}
		// 验证任务是否属于计划关联的项目
		if plan.ProjectID != nil && task.ProjectID != *plan.ProjectID {
			utils.Error(c, 400, "任务不属于计划关联的项目")
			return
		}
	}

	execution := model.PlanExecution{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		Progress:    req.Progress,
		PlanID:      plan.ID,
		StartDate:   startDate,
		EndDate:     endDate,
		AssigneeID:  req.AssigneeID,
		TaskID:      req.TaskID,
	}

	if err := h.db.Create(&execution).Error; err != nil {
		utils.Error(c, utils.CodeError, "创建失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Plan").Preload("Assignee").Preload("Task").First(&execution, execution.ID)

	utils.Success(c, execution)
}

// UpdatePlanExecution 更新计划执行
func (h *PlanExecutionHandler) UpdatePlanExecution(c *gin.Context) {
	id := c.Param("execution_id")
	var execution model.PlanExecution
	if err := h.db.First(&execution, id).Error; err != nil {
		utils.Error(c, 404, "执行不存在")
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
		Progress    *int    `json:"progress"`
		StartDate   *string `json:"start_date"` // 接收字符串格式的日期
		EndDate     *string `json:"end_date"`   // 接收字符串格式的日期
		AssigneeID  *uint   `json:"assignee_id"`
		TaskID      *uint   `json:"task_id"` // 关联的任务ID
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 解析日期
	if req.StartDate != nil && *req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			execution.StartDate = &t
		}
	}
	if req.EndDate != nil && *req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			execution.EndDate = &t
		}
	}

	if req.Name != nil {
		execution.Name = *req.Name
	}
	if req.Description != nil {
		execution.Description = *req.Description
	}
	if req.Status != nil {
		if !isValidExecutionStatus(*req.Status) {
			utils.Error(c, 400, "无效的执行状态")
			return
		}
		execution.Status = *req.Status
		// 如果状态为completed，自动设置进度为100
		if *req.Status == "completed" {
			execution.Progress = 100
		}
	}
	if req.Progress != nil {
		if *req.Progress < 0 || *req.Progress > 100 {
			utils.Error(c, 400, "进度必须在0-100之间")
			return
		}
		execution.Progress = *req.Progress
		// 如果进度为100，自动设置状态为completed
		if *req.Progress == 100 && execution.Status != "cancelled" {
			execution.Status = "completed"
		} else if *req.Progress > 0 && execution.Status == "pending" {
			execution.Status = "in_progress"
		}
	}
	// 日期已在上面解析
	if req.AssigneeID != nil {
		execution.AssigneeID = req.AssigneeID
	}
	if req.TaskID != nil {
		// 验证任务是否存在
		var task model.Task
		if err := h.db.First(&task, *req.TaskID).Error; err != nil {
			utils.Error(c, 404, "任务不存在")
			return
		}
		// 验证任务是否属于计划关联的项目
		var plan model.Plan
		h.db.First(&plan, execution.PlanID)
		if plan.ProjectID != nil && task.ProjectID != *plan.ProjectID {
			utils.Error(c, 400, "任务不属于计划关联的项目")
			return
		}
		execution.TaskID = req.TaskID
	}

	if err := h.db.Save(&execution).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Plan").Preload("Assignee").Preload("Task").First(&execution, execution.ID)

	utils.Success(c, execution)
}

// DeletePlanExecution 删除计划执行
func (h *PlanExecutionHandler) DeletePlanExecution(c *gin.Context) {
	id := c.Param("execution_id")

	if err := h.db.Delete(&model.PlanExecution{}, id).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除失败")
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}

// UpdatePlanExecutionStatus 更新计划执行状态
func (h *PlanExecutionHandler) UpdatePlanExecutionStatus(c *gin.Context) {
	id := c.Param("execution_id")
	var execution model.PlanExecution
	if err := h.db.First(&execution, id).Error; err != nil {
		utils.Error(c, 404, "执行不存在")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	if !isValidExecutionStatus(req.Status) {
		utils.Error(c, 400, "无效的执行状态")
		return
	}

	execution.Status = req.Status
	if execution.Status == "completed" {
		execution.Progress = 100
		// 如果执行关联了任务，同步更新任务状态和进度
		if execution.TaskID != nil {
			h.syncExecutionToTask(execution)
		}
	}

	if err := h.db.Save(&execution).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新状态失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Plan").Preload("Assignee").Preload("Task").First(&execution, execution.ID)

	utils.Success(c, execution)
}

// UpdatePlanExecutionProgress 更新计划执行进度
func (h *PlanExecutionHandler) UpdatePlanExecutionProgress(c *gin.Context) {
	id := c.Param("execution_id")
	var execution model.PlanExecution
	if err := h.db.First(&execution, id).Error; err != nil {
		utils.Error(c, 404, "执行不存在")
		return
	}

	var req struct {
		Progress int `json:"progress" binding:"required,min=0,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	execution.Progress = req.Progress
	if execution.Progress == 100 {
		execution.Status = "completed"
	} else if execution.Progress > 0 && execution.Status == "pending" {
		execution.Status = "in_progress"
	}

	// 如果执行关联了任务，同步更新任务进度
	if execution.TaskID != nil {
		h.syncExecutionToTask(execution)
	}

	if err := h.db.Save(&execution).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新进度失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Plan").Preload("Assignee").Preload("Task").First(&execution, execution.ID)

	utils.Success(c, execution)
}

// syncExecutionToTask 同步执行进度到关联的任务
func (h *PlanExecutionHandler) syncExecutionToTask(execution model.PlanExecution) {
	if execution.TaskID == nil {
		return
	}

	var task model.Task
	if err := h.db.First(&task, *execution.TaskID).Error; err != nil {
		return
	}

	// 同步进度
	task.Progress = execution.Progress

	// 同步状态（映射执行状态到任务状态）
	if execution.Status == "completed" {
		task.Status = "done"
	} else if execution.Status == "in_progress" {
		task.Status = "in_progress"
	} else if execution.Status == "cancelled" {
		task.Status = "cancelled"
	}

	h.db.Save(&task)
}

// isValidExecutionStatus 检查执行状态是否合法
func isValidExecutionStatus(status string) bool {
	switch status {
	case "pending", "in_progress", "completed", "cancelled":
		return true
	}
	return false
}
