package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"project-management/internal/model"
	"project-management/internal/utils"
)

type VersionHandler struct {
	db *gorm.DB
}

func NewVersionHandler(db *gorm.DB) *VersionHandler {
	return &VersionHandler{db: db}
}

// GetVersions 获取版本列表
func (h *VersionHandler) GetVersions(c *gin.Context) {
	var versions []model.Version
	query := h.db.Preload("Build").Preload("Build.Project").Preload("Build.Creator")

	// 搜索
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("version_number LIKE ? OR release_notes LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 构建筛选
	if buildID := c.Query("build_id"); buildID != "" {
		query = query.Where("build_id = ?", buildID)
	}

	// 项目筛选（通过构建）
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Joins("JOIN builds ON versions.build_id = builds.id").
			Where("builds.project_id = ?", projectID)
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 分页
	page := utils.GetPage(c)
	pageSize := utils.GetPageSize(c)
	offset := (page - 1) * pageSize

	var total int64
	query.Model(&model.Version{}).Count(&total)

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&versions).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, gin.H{
		"list":      versions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetVersion 获取版本详情
func (h *VersionHandler) GetVersion(c *gin.Context) {
	id := c.Param("id")
	var version model.Version
	if err := h.db.Preload("Build").Preload("Build.Project").Preload("Build.Creator").
		Preload("Requirements").Preload("Bugs").First(&version, id).Error; err != nil {
		utils.Error(c, 404, "版本不存在")
		return
	}

	utils.Success(c, version)
}

// CreateVersion 创建版本
func (h *VersionHandler) CreateVersion(c *gin.Context) {
	var req struct {
		VersionNumber string   `json:"version_number" binding:"required"`
		ReleaseNotes  string   `json:"release_notes"`
		Status        string   `json:"status"`
		BuildID       uint     `json:"build_id" binding:"required"`
		ReleaseDate   *string  `json:"release_date"` // 接收字符串格式的日期
		RequirementIDs []uint  `json:"requirement_ids"` // 关联的需求ID列表
		BugIDs        []uint   `json:"bug_ids"`         // 关联的Bug ID列表
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 验证状态
	if req.Status == "" {
		req.Status = "draft"
	}
	if !isValidVersionStatus(req.Status) {
		utils.Error(c, 400, "无效的版本状态")
		return
	}

	// 验证构建是否存在
	var build model.Build
	if err := h.db.First(&build, req.BuildID).Error; err != nil {
		utils.Error(c, 404, "构建不存在")
		return
	}

	// 检查该构建是否已有版本
	var existingVersion model.Version
	if err := h.db.Where("build_id = ?", req.BuildID).First(&existingVersion).Error; err == nil {
		utils.Error(c, 400, "该构建已有关联的版本")
		return
	}

	// 解析发布日期
	var releaseDate *time.Time
	if req.ReleaseDate != nil && *req.ReleaseDate != "" {
		if t, err := time.Parse("2006-01-02", *req.ReleaseDate); err == nil {
			releaseDate = &t
		}
	}

	version := model.Version{
		VersionNumber: req.VersionNumber,
		ReleaseNotes:  req.ReleaseNotes,
		Status:        req.Status,
		BuildID:       req.BuildID,
		ReleaseDate:   releaseDate,
	}

	if err := h.db.Create(&version).Error; err != nil {
		utils.Error(c, utils.CodeError, "创建失败")
		return
	}

	// 关联需求和Bug
	if len(req.RequirementIDs) > 0 {
		var requirements []model.Requirement
		if err := h.db.Where("id IN ?", req.RequirementIDs).Find(&requirements).Error; err == nil {
			h.db.Model(&version).Association("Requirements").Replace(requirements)
		}
	}
	if len(req.BugIDs) > 0 {
		var bugs []model.Bug
		if err := h.db.Where("id IN ?", req.BugIDs).Find(&bugs).Error; err == nil {
			h.db.Model(&version).Association("Bugs").Replace(bugs)
		}
	}

	// 重新加载关联数据
	h.db.Preload("Build").Preload("Build.Project").Preload("Build.Creator").
		Preload("Requirements").Preload("Bugs").First(&version, version.ID)

	utils.Success(c, version)
}

// UpdateVersion 更新版本
func (h *VersionHandler) UpdateVersion(c *gin.Context) {
	id := c.Param("id")
	var version model.Version
	if err := h.db.First(&version, id).Error; err != nil {
		utils.Error(c, 404, "版本不存在")
		return
	}

	var req struct {
		VersionNumber  *string `json:"version_number"`
		ReleaseNotes   *string `json:"release_notes"`
		Status         *string `json:"status"`
		ReleaseDate    *string `json:"release_date"` // 接收字符串格式的日期
		RequirementIDs []uint  `json:"requirement_ids"` // 关联的需求ID列表
		BugIDs         []uint  `json:"bug_ids"`         // 关联的Bug ID列表
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	if req.VersionNumber != nil {
		version.VersionNumber = *req.VersionNumber
	}
	if req.ReleaseNotes != nil {
		version.ReleaseNotes = *req.ReleaseNotes
	}
	if req.Status != nil {
		if !isValidVersionStatus(*req.Status) {
			utils.Error(c, 400, "无效的版本状态")
			return
		}
		version.Status = *req.Status
		// 如果状态为 released，自动设置发布日期
		if *req.Status == "released" && version.ReleaseDate == nil {
			now := time.Now()
			version.ReleaseDate = &now
		}
	}
	// 解析发布日期
	if req.ReleaseDate != nil {
		if *req.ReleaseDate != "" {
			if t, err := time.Parse("2006-01-02", *req.ReleaseDate); err == nil {
				version.ReleaseDate = &t
			}
		} else {
			version.ReleaseDate = nil
		}
	}

	if err := h.db.Save(&version).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	// 更新关联需求和Bug
	if req.RequirementIDs != nil {
		var requirements []model.Requirement
		if len(req.RequirementIDs) > 0 {
			h.db.Where("id IN ?", req.RequirementIDs).Find(&requirements)
		}
		h.db.Model(&version).Association("Requirements").Replace(requirements)
	}
	if req.BugIDs != nil {
		var bugs []model.Bug
		if len(req.BugIDs) > 0 {
			h.db.Where("id IN ?", req.BugIDs).Find(&bugs)
		}
		h.db.Model(&version).Association("Bugs").Replace(bugs)
	}

	// 重新加载关联数据
	h.db.Preload("Build").Preload("Build.Project").Preload("Build.Creator").
		Preload("Requirements").Preload("Bugs").First(&version, version.ID)

	utils.Success(c, version)
}

// DeleteVersion 删除版本
func (h *VersionHandler) DeleteVersion(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&model.Version{}, id).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除失败")
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}

// UpdateVersionStatus 更新版本状态
func (h *VersionHandler) UpdateVersionStatus(c *gin.Context) {
	id := c.Param("id")
	var version model.Version
	if err := h.db.First(&version, id).Error; err != nil {
		utils.Error(c, 404, "版本不存在")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	if !isValidVersionStatus(req.Status) {
		utils.Error(c, 400, "无效的版本状态")
		return
	}

	version.Status = req.Status
	// 如果状态为 released，自动设置发布日期
	if req.Status == "released" && version.ReleaseDate == nil {
		now := time.Now()
		version.ReleaseDate = &now
	}

	if err := h.db.Save(&version).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新状态失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Build").Preload("Build.Project").Preload("Build.Creator").
		Preload("Requirements").Preload("Bugs").First(&version, version.ID)

	utils.Success(c, version)
}

// ReleaseVersion 发布版本
func (h *VersionHandler) ReleaseVersion(c *gin.Context) {
	id := c.Param("id")
	var version model.Version
	if err := h.db.Preload("Build").First(&version, id).Error; err != nil {
		utils.Error(c, 404, "版本不存在")
		return
	}

	// 检查构建状态
	if version.Build.Status != "success" {
		utils.Error(c, 400, "只有构建成功的版本才能发布")
		return
	}

	version.Status = "released"
	if version.ReleaseDate == nil {
		now := time.Now()
		version.ReleaseDate = &now
	}

	if err := h.db.Save(&version).Error; err != nil {
		utils.Error(c, utils.CodeError, "发布失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Build").Preload("Build.Project").Preload("Build.Creator").
		Preload("Requirements").Preload("Bugs").First(&version, version.ID)

	utils.Success(c, version)
}

// isValidVersionStatus 检查版本状态是否合法
func isValidVersionStatus(status string) bool {
	switch status {
	case "draft", "released", "archived":
		return true
	}
	return false
}

