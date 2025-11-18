package api

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"project-management/internal/model"
	"project-management/internal/utils"
)

type TestReportHandler struct {
	db *gorm.DB
}

func NewTestReportHandler(db *gorm.DB) *TestReportHandler {
	return &TestReportHandler{db: db}
}

// GetTestReports 获取测试报告列表
func (h *TestReportHandler) GetTestReports(c *gin.Context) {
	var testReports []model.TestReport
	query := h.db.Preload("Creator").Preload("TestCases")

	// 搜索
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("title LIKE ? OR content LIKE ? OR summary LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 结果筛选
	if result := c.Query("result"); result != "" {
		query = query.Where("result = ?", result)
	}

	// 创建人筛选
	if creatorID := c.Query("creator_id"); creatorID != "" {
		query = query.Where("creator_id = ?", creatorID)
	}

	// 测试单筛选（通过关联表）
	if testCaseID := c.Query("test_case_id"); testCaseID != "" {
		query = query.Joins("JOIN test_case_reports ON test_reports.id = test_case_reports.test_report_id").
			Where("test_case_reports.test_case_id = ?", testCaseID)
	}

	// 分页
	page := utils.GetPage(c)
	pageSize := utils.GetPageSize(c)
	offset := (page - 1) * pageSize

	var total int64
	query.Model(&model.TestReport{}).Count(&total)

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&testReports).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, gin.H{
		"list":      testReports,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetTestReport 获取测试报告详情
func (h *TestReportHandler) GetTestReport(c *gin.Context) {
	id := c.Param("id")
	var testReport model.TestReport
	if err := h.db.Preload("Creator").Preload("TestCases").First(&testReport, id).Error; err != nil {
		utils.Error(c, 404, "测试报告不存在")
		return
	}

	utils.Success(c, testReport)
}

// CreateTestReport 创建测试报告
func (h *TestReportHandler) CreateTestReport(c *gin.Context) {
	var req struct {
		Title       string  `json:"title" binding:"required"`
		Content     string  `json:"content"`
		Result      string  `json:"result"`
		Summary     string  `json:"summary"`
		TestCaseIDs []uint  `json:"test_case_ids"` // 关联的测试单ID列表
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 验证结果
	if req.Result != "" && !isValidTestReportResult(req.Result) {
		utils.Error(c, 400, "无效的测试结果")
		return
	}

	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "未登录")
		return
	}
	uid := userID.(uint)

	testReport := model.TestReport{
		Title:     req.Title,
		Content:   req.Content,
		Result:    req.Result,
		Summary:   req.Summary,
		CreatorID: uid,
	}

	if err := h.db.Create(&testReport).Error; err != nil {
		utils.Error(c, utils.CodeError, "创建失败")
		return
	}

	// 关联测试单
	if len(req.TestCaseIDs) > 0 {
		var testCases []model.TestCase
		if err := h.db.Where("id IN ?", req.TestCaseIDs).Find(&testCases).Error; err == nil {
			h.db.Model(&testReport).Association("TestCases").Replace(testCases)
		}
	}

	// 重新加载关联数据
	h.db.Preload("Creator").Preload("TestCases").First(&testReport, testReport.ID)

	utils.Success(c, testReport)
}

// UpdateTestReport 更新测试报告
func (h *TestReportHandler) UpdateTestReport(c *gin.Context) {
	id := c.Param("id")
	var testReport model.TestReport
	if err := h.db.First(&testReport, id).Error; err != nil {
		utils.Error(c, 404, "测试报告不存在")
		return
	}

	var req struct {
		Title       *string `json:"title"`
		Content     *string `json:"content"`
		Result      *string `json:"result"`
		Summary     *string `json:"summary"`
		TestCaseIDs []uint  `json:"test_case_ids"` // 关联的测试单ID列表
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	if req.Title != nil {
		testReport.Title = *req.Title
	}
	if req.Content != nil {
		testReport.Content = *req.Content
	}
	if req.Result != nil {
		if !isValidTestReportResult(*req.Result) {
			utils.Error(c, 400, "无效的测试结果")
			return
		}
		testReport.Result = *req.Result
	}
	if req.Summary != nil {
		testReport.Summary = *req.Summary
	}

	if err := h.db.Save(&testReport).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	// 更新关联测试单
	if req.TestCaseIDs != nil {
		var testCases []model.TestCase
		if len(req.TestCaseIDs) > 0 {
			h.db.Where("id IN ?", req.TestCaseIDs).Find(&testCases)
		}
		h.db.Model(&testReport).Association("TestCases").Replace(testCases)
	}

	// 重新加载关联数据
	h.db.Preload("Creator").Preload("TestCases").First(&testReport, testReport.ID)

	utils.Success(c, testReport)
}

// DeleteTestReport 删除测试报告
func (h *TestReportHandler) DeleteTestReport(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&model.TestReport{}, id).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除失败")
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}

// GetTestReportStatistics 获取测试报告统计
func (h *TestReportHandler) GetTestReportStatistics(c *gin.Context) {
	baseQuery := h.db.Model(&model.TestReport{})

	// 创建人筛选
	if creatorID := c.Query("creator_id"); creatorID != "" {
		baseQuery = baseQuery.Where("creator_id = ?", creatorID)
	}

	// 搜索
	if keyword := c.Query("keyword"); keyword != "" {
		baseQuery = baseQuery.Where("title LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 使用独立的Session确保每个查询都是独立的
	var total, passed, failed, blocked int64
	baseQuery.Session(&gorm.Session{}).Count(&total)
	baseQuery.Session(&gorm.Session{}).Where("result = ?", "passed").Count(&passed)
	baseQuery.Session(&gorm.Session{}).Where("result = ?", "failed").Count(&failed)
	baseQuery.Session(&gorm.Session{}).Where("result = ?", "blocked").Count(&blocked)

	utils.Success(c, gin.H{
		"total":   total,
		"passed":  passed,
		"failed":  failed,
		"blocked": blocked,
	})
}

// isValidTestReportResult 检查测试报告结果是否合法
func isValidTestReportResult(result string) bool {
	switch result {
	case "passed", "failed", "blocked":
		return true
	}
	return false
}

