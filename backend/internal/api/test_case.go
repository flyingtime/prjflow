package api

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"project-management/internal/model"
	"project-management/internal/utils"
)

type TestCaseHandler struct {
	db *gorm.DB
}

func NewTestCaseHandler(db *gorm.DB) *TestCaseHandler {
	return &TestCaseHandler{db: db}
}

// GetTestCases 获取测试单列表
func (h *TestCaseHandler) GetTestCases(c *gin.Context) {
	var testCases []model.TestCase
	query := h.db.Preload("Project").Preload("Creator").Preload("Bugs").Preload("Reports")

	// 搜索
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ? OR test_steps LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 项目筛选
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 类型筛选（支持JSON数组字段查询）
	if testType := c.Query("type"); testType != "" {
		// 使用LIKE查询JSON数组字段（适用于SQLite和MySQL）
		query = query.Where("types LIKE ?", "%\""+testType+"\"%")
	}

	// 创建人筛选
	if creatorID := c.Query("creator_id"); creatorID != "" {
		query = query.Where("creator_id = ?", creatorID)
	}

	// 分页
	page := utils.GetPage(c)
	pageSize := utils.GetPageSize(c)
	offset := (page - 1) * pageSize

	var total int64
	query.Model(&model.TestCase{}).Count(&total)

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&testCases).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, gin.H{
		"list":      testCases,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetTestCase 获取测试单详情
func (h *TestCaseHandler) GetTestCase(c *gin.Context) {
	id := c.Param("id")
	var testCase model.TestCase
	if err := h.db.Preload("Project").Preload("Creator").Preload("Bugs").Preload("Reports").First(&testCase, id).Error; err != nil {
		utils.Error(c, 404, "测试单不存在")
		return
	}

	utils.Success(c, testCase)
}

// CreateTestCase 创建测试单
func (h *TestCaseHandler) CreateTestCase(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		TestSteps   string   `json:"test_steps"`
		Types       []string `json:"types"` // 测试类型（多选）
		Status      string   `json:"status"`
		ProjectID   uint     `json:"project_id" binding:"required"`
		BugIDs      []uint   `json:"bug_ids"` // 关联的Bug ID列表
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 验证项目ID
	if req.ProjectID == 0 {
		utils.Error(c, 400, "项目ID不能为空")
		return
	}

	// 验证状态
	if req.Status == "" {
		req.Status = "pending"
	}
	if !isValidTestCaseStatus(req.Status) {
		utils.Error(c, 400, "无效的测试单状态")
		return
	}

	// 验证项目是否存在
	var project model.Project
	if err := h.db.First(&project, req.ProjectID).Error; err != nil {
		utils.Error(c, 404, "项目不存在")
		return
	}

	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "未登录")
		return
	}
	uid := userID.(uint)

	testCase := model.TestCase{
		Name:        req.Name,
		Description: req.Description,
		TestSteps:   req.TestSteps,
		Types:       model.StringArray(req.Types),
		Status:      req.Status,
		ProjectID:   req.ProjectID,
		CreatorID:   uid,
	}

	if err := h.db.Create(&testCase).Error; err != nil {
		utils.Error(c, utils.CodeError, "创建失败")
		return
	}

	// 关联Bug
	if len(req.BugIDs) > 0 {
		var bugs []model.Bug
		if err := h.db.Where("id IN ?", req.BugIDs).Find(&bugs).Error; err == nil {
			h.db.Model(&testCase).Association("Bugs").Replace(bugs)
		}
	}

	// 重新加载关联数据
	h.db.Preload("Project").Preload("Creator").Preload("Bugs").Preload("Reports").First(&testCase, testCase.ID)

	utils.Success(c, testCase)
}

// UpdateTestCase 更新测试单
func (h *TestCaseHandler) UpdateTestCase(c *gin.Context) {
	id := c.Param("id")
	var testCase model.TestCase
	if err := h.db.First(&testCase, id).Error; err != nil {
		utils.Error(c, 404, "测试单不存在")
		return
	}

	var req struct {
		Name        *string  `json:"name"`
		Description *string  `json:"description"`
		TestSteps   *string  `json:"test_steps"`
		Types       []string `json:"types"` // 测试类型（多选）
		Status      *string  `json:"status"`
		BugIDs      []uint   `json:"bug_ids"` // 关联的Bug ID列表
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	if req.Name != nil {
		testCase.Name = *req.Name
	}
	if req.Description != nil {
		testCase.Description = *req.Description
	}
	if req.TestSteps != nil {
		testCase.TestSteps = *req.TestSteps
	}
	if req.Types != nil {
		testCase.Types = model.StringArray(req.Types)
	}
	if req.Status != nil {
		if !isValidTestCaseStatus(*req.Status) {
			utils.Error(c, 400, "无效的测试单状态")
			return
		}
		testCase.Status = *req.Status
	}

	if err := h.db.Save(&testCase).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	// 更新关联Bug
	if req.BugIDs != nil {
		var bugs []model.Bug
		if len(req.BugIDs) > 0 {
			h.db.Where("id IN ?", req.BugIDs).Find(&bugs)
		}
		h.db.Model(&testCase).Association("Bugs").Replace(bugs)
	}

	// 重新加载关联数据
	h.db.Preload("Project").Preload("Creator").Preload("Bugs").Preload("Reports").First(&testCase, testCase.ID)

	utils.Success(c, testCase)
}

// DeleteTestCase 删除测试单
func (h *TestCaseHandler) DeleteTestCase(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&model.TestCase{}, id).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除失败")
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}

// UpdateTestCaseStatus 更新测试单状态
func (h *TestCaseHandler) UpdateTestCaseStatus(c *gin.Context) {
	id := c.Param("id")
	var testCase model.TestCase
	if err := h.db.First(&testCase, id).Error; err != nil {
		utils.Error(c, 404, "测试单不存在")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	if !isValidTestCaseStatus(req.Status) {
		utils.Error(c, 400, "无效的测试单状态")
		return
	}

	testCase.Status = req.Status
	if err := h.db.Save(&testCase).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新状态失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Project").Preload("Creator").Preload("Bugs").Preload("Reports").First(&testCase, testCase.ID)

	utils.Success(c, testCase)
}

// GetTestCaseStatistics 获取测试单统计
func (h *TestCaseHandler) GetTestCaseStatistics(c *gin.Context) {
	baseQuery := h.db.Model(&model.TestCase{})

	// 项目筛选
	if projectID := c.Query("project_id"); projectID != "" {
		baseQuery = baseQuery.Where("project_id = ?", projectID)
	}

	// 搜索
	if keyword := c.Query("keyword"); keyword != "" {
		baseQuery = baseQuery.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 使用独立的Session确保每个查询都是独立的
	var total, pending, running, passed, failed int64
	baseQuery.Session(&gorm.Session{}).Count(&total)
	baseQuery.Session(&gorm.Session{}).Where("status = ?", "pending").Count(&pending)
	baseQuery.Session(&gorm.Session{}).Where("status = ?", "running").Count(&running)
	baseQuery.Session(&gorm.Session{}).Where("status = ?", "passed").Count(&passed)
	baseQuery.Session(&gorm.Session{}).Where("status = ?", "failed").Count(&failed)

	utils.Success(c, gin.H{
		"total":   total,
		"pending": pending,
		"running": running,
		"passed":  passed,
		"failed":  failed,
	})
}

// isValidTestCaseStatus 检查测试单状态是否合法
func isValidTestCaseStatus(status string) bool {
	switch status {
	case "pending", "running", "passed", "failed":
		return true
	}
	return false
}

