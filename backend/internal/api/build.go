package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"project-management/internal/model"
	"project-management/internal/utils"
)

type BuildHandler struct {
	db *gorm.DB
}

func NewBuildHandler(db *gorm.DB) *BuildHandler {
	return &BuildHandler{db: db}
}

// GetBuilds 获取构建列表
func (h *BuildHandler) GetBuilds(c *gin.Context) {
	var builds []model.Build
	query := h.db.Preload("Project").Preload("Creator")

	// 搜索
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("build_number LIKE ? OR branch LIKE ? OR commit LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 项目筛选
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 分支筛选
	if branch := c.Query("branch"); branch != "" {
		query = query.Where("branch = ?", branch)
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
	query.Model(&model.Build{}).Count(&total)

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&builds).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, gin.H{
		"list":      builds,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetBuild 获取构建详情
func (h *BuildHandler) GetBuild(c *gin.Context) {
	id := c.Param("id")
	var build model.Build
	if err := h.db.Preload("Project").Preload("Creator").Preload("Version").First(&build, id).Error; err != nil {
		utils.Error(c, 404, "构建不存在")
		return
	}

	utils.Success(c, build)
}

// CreateBuild 创建构建
func (h *BuildHandler) CreateBuild(c *gin.Context) {
	var req struct {
		BuildNumber string     `json:"build_number" binding:"required"`
		Status      string     `json:"status"`
		Branch      string     `json:"branch"`
		Commit      string     `json:"commit"`
		BuildTime   *string    `json:"build_time"` // 接收字符串格式的日期时间
		ProjectID   uint       `json:"project_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 验证状态
	if req.Status == "" {
		req.Status = "pending"
	}
	if !isValidBuildStatus(req.Status) {
		utils.Error(c, 400, "无效的构建状态")
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

	// 解析构建时间
	var buildTime *time.Time
	if req.BuildTime != nil && *req.BuildTime != "" {
		// 尝试解析多种时间格式
		formats := []string{
			"2006-01-02T15:04:05Z07:00", // RFC3339
			"2006-01-02 15:04:05",        // 标准格式
			"2006-01-02T15:04:05",        // ISO格式
		}
		for _, format := range formats {
			if t, err := time.Parse(format, *req.BuildTime); err == nil {
				buildTime = &t
				break
			}
		}
	}

	build := model.Build{
		BuildNumber: req.BuildNumber,
		Status:      req.Status,
		Branch:      req.Branch,
		Commit:      req.Commit,
		BuildTime:   buildTime,
		ProjectID:   req.ProjectID,
		CreatorID:   uid,
	}

	if err := h.db.Create(&build).Error; err != nil {
		utils.Error(c, utils.CodeError, "创建失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Project").Preload("Creator").Preload("Version").First(&build, build.ID)

	utils.Success(c, build)
}

// UpdateBuild 更新构建
func (h *BuildHandler) UpdateBuild(c *gin.Context) {
	id := c.Param("id")
	var build model.Build
	if err := h.db.First(&build, id).Error; err != nil {
		utils.Error(c, 404, "构建不存在")
		return
	}

	var req struct {
		BuildNumber *string `json:"build_number"`
		Status      *string `json:"status"`
		Branch      *string `json:"branch"`
		Commit      *string `json:"commit"`
		BuildTime   *string `json:"build_time"` // 接收字符串格式的日期时间
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	if req.BuildNumber != nil {
		build.BuildNumber = *req.BuildNumber
	}
	if req.Status != nil {
		if !isValidBuildStatus(*req.Status) {
			utils.Error(c, 400, "无效的构建状态")
			return
		}
		build.Status = *req.Status
	}
	if req.Branch != nil {
		build.Branch = *req.Branch
	}
	if req.Commit != nil {
		build.Commit = *req.Commit
	}
	// 解析构建时间
	if req.BuildTime != nil {
		if *req.BuildTime != "" {
			// 尝试解析多种时间格式
			formats := []string{
				"2006-01-02T15:04:05Z07:00", // RFC3339
				"2006-01-02 15:04:05",        // 标准格式
				"2006-01-02T15:04:05",        // ISO格式
			}
			parsed := false
			for _, format := range formats {
				if t, err := time.Parse(format, *req.BuildTime); err == nil {
					build.BuildTime = &t
					parsed = true
					break
				}
			}
			if !parsed {
				build.BuildTime = nil
			}
		} else {
			build.BuildTime = nil
		}
	}

	if err := h.db.Save(&build).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Project").Preload("Creator").Preload("Version").First(&build, build.ID)

	utils.Success(c, build)
}

// DeleteBuild 删除构建
func (h *BuildHandler) DeleteBuild(c *gin.Context) {
	id := c.Param("id")

	// 检查是否有关联的版本
	var count int64
	h.db.Model(&model.Version{}).Where("build_id = ?", id).Count(&count)
	if count > 0 {
		utils.Error(c, 400, "构建下有关联的版本，无法删除")
		return
	}

	if err := h.db.Delete(&model.Build{}, id).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除失败")
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}

// UpdateBuildStatus 更新构建状态
func (h *BuildHandler) UpdateBuildStatus(c *gin.Context) {
	id := c.Param("id")
	var build model.Build
	if err := h.db.First(&build, id).Error; err != nil {
		utils.Error(c, 404, "构建不存在")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	if !isValidBuildStatus(req.Status) {
		utils.Error(c, 400, "无效的构建状态")
		return
	}

	build.Status = req.Status
	// 如果状态为 success 或 failed，自动设置构建时间
	if (req.Status == "success" || req.Status == "failed") && build.BuildTime == nil {
		now := time.Now()
		build.BuildTime = &now
	}

	if err := h.db.Save(&build).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新状态失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Project").Preload("Creator").Preload("Version").First(&build, build.ID)

	utils.Success(c, build)
}

// isValidBuildStatus 检查构建状态是否合法
func isValidBuildStatus(status string) bool {
	switch status {
	case "pending", "building", "success", "failed":
		return true
	}
	return false
}

