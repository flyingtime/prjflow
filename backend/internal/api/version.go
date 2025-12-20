package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"prjflow/internal/model"
	"prjflow/internal/utils"
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
	query := h.db.Preload("Project").Preload("Requirements").Preload("Bugs")

	// 搜索
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("version_number LIKE ? OR release_notes LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 项目筛选
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
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
	// 计算总数时需要应用与查询相同的筛选条件
	countQuery := h.db.Model(&model.Version{})

	// 搜索
	if keyword := c.Query("keyword"); keyword != "" {
		countQuery = countQuery.Where("version_number LIKE ? OR release_notes LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 项目筛选
	if projectID := c.Query("project_id"); projectID != "" {
		countQuery = countQuery.Where("project_id = ?", projectID)
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}

	countQuery.Count(&total)

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
	if err := h.db.Preload("Project").
		Preload("Requirements").Preload("Bugs").
		Preload("Attachments").Preload("Attachments.Creator").
		First(&version, id).Error; err != nil {
		utils.Error(c, 404, "版本不存在")
		return
	}

	utils.Success(c, version)
}

// CreateVersion 创建版本
func (h *VersionHandler) CreateVersion(c *gin.Context) {
	var req struct {
		VersionNumber  string   `json:"version_number" binding:"required"`
		ReleaseNotes   string   `json:"release_notes"`
		Status         string   `json:"status"`
		ProjectID      uint     `json:"project_id" binding:"required"`
		ReleaseDate    *string  `json:"release_date"` // 接收字符串格式的日期
		RequirementIDs []uint   `json:"requirement_ids"` // 关联的需求ID列表
		BugIDs         []uint   `json:"bug_ids"`         // 关联的Bug ID列表
		AttachmentIDs  *[]uint  `json:"attachment_ids"`  // 附件ID列表
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 验证状态
	if req.Status == "" {
		req.Status = "wait"
	}
	if !isValidVersionStatus(req.Status) {
		utils.Error(c, 400, "无效的版本状态，有效值：wait, normal, fail, terminate")
		return
	}

	// 验证项目是否存在
	var project model.Project
	if err := h.db.First(&project, req.ProjectID).Error; err != nil {
		utils.Error(c, 404, "项目不存在")
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
		ProjectID:     req.ProjectID,
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

	// 关联附件
	if req.AttachmentIDs != nil {
		projectID := version.ProjectID
		var attachments []model.Attachment
		if len(*req.AttachmentIDs) > 0 {
			// 验证附件是否存在且属于同一项目
			if err := h.db.Where("id IN ?", *req.AttachmentIDs).Find(&attachments).Error; err != nil {
				utils.Error(c, 400, "附件查询失败: "+err.Error())
				return
			}
			if len(attachments) != len(*req.AttachmentIDs) {
				// 检查是否有附件被软删除
				var deletedAttachments []model.Attachment
				h.db.Unscoped().Where("id IN ? AND deleted_at IS NOT NULL", *req.AttachmentIDs).Find(&deletedAttachments)
				if len(deletedAttachments) > 0 {
					utils.Error(c, 400, fmt.Sprintf("部分附件已被删除：期望 %d 个，实际找到 %d 个，已删除 %d 个", len(*req.AttachmentIDs), len(attachments), len(deletedAttachments)))
					return
				}
				utils.Error(c, 400, fmt.Sprintf("附件不存在：期望 %d 个，实际找到 %d 个", len(*req.AttachmentIDs), len(attachments)))
				return
			}
			// 验证附件是否属于同一项目（通过检查附件是否关联到项目）
			for _, attachment := range attachments {
				var count int64
				if err := h.db.Table("project_attachments").
					Where("attachment_id = ? AND project_id = ?", attachment.ID, projectID).
					Count(&count).Error; err != nil {
					utils.Error(c, 400, "验证附件项目关联失败: "+err.Error())
					return
				}
				if count == 0 {
					utils.Error(c, 400, fmt.Sprintf("附件 %d 不属于项目 %d", attachment.ID, projectID))
					return
				}
			}
		}
		// 使用Replace方法同步附件关联（空数组表示移除所有附件）
		if err := h.db.Model(&version).Association("Attachments").Replace(attachments); err != nil {
			utils.Error(c, utils.CodeError, "更新附件关联失败: "+err.Error())
			return
		}
	}

	// 重新加载关联数据（包含附件）
	h.db.Session(&gorm.Session{}).Preload("Project").
		Preload("Requirements").Preload("Bugs").
		Preload("Attachments").Preload("Attachments.Creator").
		First(&version, version.ID)

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
		AttachmentIDs  *[]uint  `json:"attachment_ids"` // 附件ID列表
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
		utils.Error(c, 400, "无效的版本状态，有效值：wait, normal, fail, terminate")
		return
	}
		version.Status = *req.Status
		// 如果状态为 normal，自动设置发布日期
		if *req.Status == "normal" && version.ReleaseDate == nil {
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

	// 更新关联需求和Bug（如果提供了这些字段）
	// 注意：空数组 [] 不是 nil，所以会执行更新（清空关联）
	if req.RequirementIDs != nil {
		var requirements []model.Requirement
		if len(req.RequirementIDs) > 0 {
			if err := h.db.Where("id IN ?", req.RequirementIDs).Find(&requirements).Error; err != nil {
				utils.Error(c, utils.CodeError, "查询关联需求失败")
				return
			}
		}
		if err := h.db.Model(&version).Association("Requirements").Replace(requirements); err != nil {
			utils.Error(c, utils.CodeError, "更新关联需求失败")
			return
		}
	}
	if req.BugIDs != nil {
		var bugs []model.Bug
		if len(req.BugIDs) > 0 {
			if err := h.db.Where("id IN ?", req.BugIDs).Find(&bugs).Error; err != nil {
				utils.Error(c, utils.CodeError, "查询关联Bug失败")
				return
			}
		}
		if err := h.db.Model(&version).Association("Bugs").Replace(bugs); err != nil {
			utils.Error(c, utils.CodeError, "更新关联Bug失败")
			return
		}
	}

	// 更新附件关联
	if req.AttachmentIDs != nil {
		projectID := version.ProjectID
		var attachments []model.Attachment
		if len(*req.AttachmentIDs) > 0 {
			// 验证附件是否存在且属于同一项目
			if err := h.db.Where("id IN ?", *req.AttachmentIDs).Find(&attachments).Error; err != nil {
				utils.Error(c, 400, "附件查询失败: "+err.Error())
				return
			}
			if len(attachments) != len(*req.AttachmentIDs) {
				// 检查是否有附件被软删除
				var deletedAttachments []model.Attachment
				h.db.Unscoped().Where("id IN ? AND deleted_at IS NOT NULL", *req.AttachmentIDs).Find(&deletedAttachments)
				if len(deletedAttachments) > 0 {
					utils.Error(c, 400, fmt.Sprintf("部分附件已被删除：期望 %d 个，实际找到 %d 个，已删除 %d 个", len(*req.AttachmentIDs), len(attachments), len(deletedAttachments)))
					return
				}
				utils.Error(c, 400, fmt.Sprintf("附件不存在：期望 %d 个，实际找到 %d 个", len(*req.AttachmentIDs), len(attachments)))
				return
			}
			// 验证附件是否属于同一项目（通过检查附件是否关联到项目）
			for _, attachment := range attachments {
				var count int64
				if err := h.db.Table("project_attachments").
					Where("attachment_id = ? AND project_id = ?", attachment.ID, projectID).
					Count(&count).Error; err != nil {
					utils.Error(c, 400, "验证附件项目关联失败: "+err.Error())
					return
				}
				if count == 0 {
					utils.Error(c, 400, fmt.Sprintf("附件 %d 不属于项目 %d", attachment.ID, projectID))
					return
				}
			}
		}
		// 使用Replace方法同步附件关联（空数组表示移除所有附件）
		if err := h.db.Model(&version).Association("Attachments").Replace(attachments); err != nil {
			utils.Error(c, utils.CodeError, "更新附件关联失败: "+err.Error())
			return
		}
	}

	// 重新加载关联数据（包含附件）
	h.db.Session(&gorm.Session{}).Preload("Project").
		Preload("Requirements").Preload("Bugs").
		Preload("Attachments").Preload("Attachments.Creator").
		First(&version, version.ID)

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
		utils.Error(c, 400, "无效的版本状态，有效值：wait, normal, fail, terminate")
		return
	}

	version.Status = req.Status
	// 如果状态为 normal，自动设置发布日期
	if req.Status == "normal" && version.ReleaseDate == nil {
		now := time.Now()
		version.ReleaseDate = &now
	}

	if err := h.db.Save(&version).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新状态失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Project").
		Preload("Requirements").Preload("Bugs").First(&version, version.ID)

	utils.Success(c, version)
}

// ReleaseVersion 发布版本
func (h *VersionHandler) ReleaseVersion(c *gin.Context) {
	id := c.Param("id")
	var version model.Version
	if err := h.db.First(&version, id).Error; err != nil {
		utils.Error(c, 404, "版本不存在")
		return
	}

	version.Status = "normal"
	if version.ReleaseDate == nil {
		now := time.Now()
		version.ReleaseDate = &now
	}

	if err := h.db.Save(&version).Error; err != nil {
		utils.Error(c, utils.CodeError, "发布失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Project").
		Preload("Requirements").Preload("Bugs").First(&version, version.ID)

	utils.Success(c, version)
}

// isValidVersionStatus 检查版本状态是否合法（禅道状态值）
func isValidVersionStatus(status string) bool {
	switch status {
	case "wait", "normal", "fail", "terminate":
		return true
	}
	return false
}

