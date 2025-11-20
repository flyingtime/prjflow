package api

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"project-management/internal/model"
	"project-management/internal/utils"
)

type PermissionHandler struct {
	db *gorm.DB
}

func NewPermissionHandler(db *gorm.DB) *PermissionHandler {
	return &PermissionHandler{db: db}
}

// GetRoles 获取所有角色
func (h *PermissionHandler) GetRoles(c *gin.Context) {
	var roles []model.Role
	if err := h.db.Find(&roles).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, roles)
}

// GetRole 获取角色详情（包含权限）
func (h *PermissionHandler) GetRole(c *gin.Context) {
	id := c.Param("id")
	var role model.Role
	if err := h.db.Preload("Permissions").First(&role, id).Error; err != nil {
		utils.Error(c, 404, "角色不存在")
		return
	}

	utils.Success(c, role)
}

// CreateRole 创建角色
func (h *PermissionHandler) CreateRole(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
		Status      int    `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 检查角色代码是否已存在
	var existingRole model.Role
	if err := h.db.Where("code = ?", req.Code).First(&existingRole).Error; err == nil {
		utils.Error(c, 400, "角色代码已存在")
		return
	}

	// 设置默认状态
	if req.Status == 0 {
		req.Status = 1
	}

	role := model.Role{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Status:      req.Status,
	}

	if err := h.db.Create(&role).Error; err != nil {
		if utils.IsUniqueConstraintError(err) {
			utils.Error(c, 400, "角色代码或名称已存在")
			return
		}
		utils.Error(c, utils.CodeError, "创建失败: "+err.Error())
		return
	}

	utils.Success(c, role)
}

// UpdateRole 更新角色
func (h *PermissionHandler) UpdateRole(c *gin.Context) {
	id := c.Param("id")
	var role model.Role
	if err := h.db.First(&role, id).Error; err != nil {
		utils.Error(c, 404, "角色不存在")
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Code        *string `json:"code"`
		Description *string `json:"description"`
		Status      *int    `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Code != nil {
		// 检查角色代码是否已被其他角色使用
		var existingRole model.Role
		if err := h.db.Where("code = ? AND id != ?", *req.Code, id).First(&existingRole).Error; err == nil {
			utils.Error(c, 400, "角色代码已存在")
			return
		}
		role.Code = *req.Code
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.Status != nil {
		role.Status = *req.Status
	}

	if err := h.db.Save(&role).Error; err != nil {
		if utils.IsUniqueConstraintError(err) {
			utils.Error(c, 400, "角色代码或名称已存在")
			return
		}
		utils.Error(c, utils.CodeError, "更新失败: "+err.Error())
		return
	}

	utils.Success(c, role)
}

// DeleteRole 删除角色
func (h *PermissionHandler) DeleteRole(c *gin.Context) {
	id := c.Param("id")
	
	// 检查是否有用户使用此角色
	var count int64
	h.db.Table("user_roles").Where("role_id = ?", id).Count(&count)
	if count > 0 {
		utils.Error(c, 400, "该角色正在被用户使用，无法删除")
		return
	}

	if err := h.db.Delete(&model.Role{}, id).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除失败")
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}

// GetRolePermissions 获取角色权限
func (h *PermissionHandler) GetRolePermissions(c *gin.Context) {
	roleID := c.Param("id")
	var role model.Role
	if err := h.db.Preload("Permissions").First(&role, roleID).Error; err != nil {
		utils.Error(c, 404, "角色不存在")
		return
	}

	utils.Success(c, role.Permissions)
}

// GetPermissions 获取所有权限
func (h *PermissionHandler) GetPermissions(c *gin.Context) {
	var permissions []model.Permission
	if err := h.db.Find(&permissions).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, permissions)
}

// GetPermission 获取权限详情
func (h *PermissionHandler) GetPermission(c *gin.Context) {
	id := c.Param("id")
	var permission model.Permission
	if err := h.db.First(&permission, id).Error; err != nil {
		utils.Error(c, 404, "权限不存在")
		return
	}

	utils.Success(c, permission)
}

// CreatePermission 创建权限
func (h *PermissionHandler) CreatePermission(c *gin.Context) {
	var req struct {
		Code        string `json:"code" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Resource    string `json:"resource"`
		Action      string `json:"action"`
		Description string `json:"description"`
		Status      int    `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 检查权限代码是否已存在
	var existingPerm model.Permission
	if err := h.db.Where("code = ?", req.Code).First(&existingPerm).Error; err == nil {
		utils.Error(c, 400, "权限代码已存在")
		return
	}

	// 设置默认状态
	if req.Status == 0 {
		req.Status = 1
	}

	permission := model.Permission{
		Code:        req.Code,
		Name:        req.Name,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: req.Description,
		Status:      req.Status,
	}

	if err := h.db.Create(&permission).Error; err != nil {
		if utils.IsUniqueConstraintError(err) {
			utils.Error(c, 400, "权限代码已存在")
			return
		}
		utils.Error(c, utils.CodeError, "创建失败: "+err.Error())
		return
	}

	utils.Success(c, permission)
}

// UpdatePermission 更新权限
func (h *PermissionHandler) UpdatePermission(c *gin.Context) {
	id := c.Param("id")
	var permission model.Permission
	if err := h.db.First(&permission, id).Error; err != nil {
		utils.Error(c, 404, "权限不存在")
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Resource    *string `json:"resource"`
		Action      *string `json:"action"`
		Description *string `json:"description"`
		Status      *int    `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	if req.Name != nil {
		permission.Name = *req.Name
	}
	if req.Resource != nil {
		permission.Resource = *req.Resource
	}
	if req.Action != nil {
		permission.Action = *req.Action
	}
	if req.Description != nil {
		permission.Description = *req.Description
	}
	if req.Status != nil {
		permission.Status = *req.Status
	}

	if err := h.db.Save(&permission).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	utils.Success(c, permission)
}

// DeletePermission 删除权限
func (h *PermissionHandler) DeletePermission(c *gin.Context) {
	id := c.Param("id")
	
	// 检查是否有角色使用此权限
	var count int64
	h.db.Table("role_permissions").Where("permission_id = ?", id).Count(&count)
	if count > 0 {
		utils.Error(c, 400, "该权限正在被角色使用，无法删除")
		return
	}

	if err := h.db.Delete(&model.Permission{}, id).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除失败")
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}

// AssignRolePermissions 分配角色权限
func (h *PermissionHandler) AssignRolePermissions(c *gin.Context) {
	roleID := c.Param("id")
	var req struct {
		PermissionIDs []uint `json:"permission_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	var role model.Role
	if err := h.db.First(&role, roleID).Error; err != nil {
		utils.Error(c, 404, "角色不存在")
		return
	}

	var permissions []model.Permission
	if err := h.db.Where("id IN ?", req.PermissionIDs).Find(&permissions).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询权限失败")
		return
	}

	if err := h.db.Model(&role).Association("Permissions").Replace(permissions); err != nil {
		utils.Error(c, utils.CodeError, "分配权限失败")
		return
	}

	utils.Success(c, gin.H{"message": "分配成功"})
}

// GetUserRoles 获取用户角色
func (h *PermissionHandler) GetUserRoles(c *gin.Context) {
	userID := c.Param("id")
	var user model.User
	if err := h.db.Preload("Roles").First(&user, userID).Error; err != nil {
		utils.Error(c, 404, "用户不存在")
		return
	}

	utils.Success(c, user.Roles)
}

// AssignUserRoles 分配用户角色
func (h *PermissionHandler) AssignUserRoles(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		RoleIDs []uint `json:"role_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		utils.Error(c, 404, "用户不存在")
		return
	}

	var roles []model.Role
	if err := h.db.Where("id IN ?", req.RoleIDs).Find(&roles).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询角色失败")
		return
	}

	if err := h.db.Model(&user).Association("Roles").Replace(roles); err != nil {
		utils.Error(c, utils.CodeError, "分配角色失败")
		return
	}

	utils.Success(c, gin.H{"message": "分配成功"})
}

// GetUserPermissions 获取当前用户的权限列表
func (h *PermissionHandler) GetUserPermissions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "未登录")
		return
	}

	var user model.User
	if err := h.db.Preload("Roles.Permissions").First(&user, userID).Error; err != nil {
		utils.Error(c, 404, "用户不存在")
		return
	}

	// 收集所有权限（去重）
	permMap := make(map[string]model.Permission)
	for _, role := range user.Roles {
		if role.Status == 0 {
			continue // 跳过禁用的角色
		}
		for _, perm := range role.Permissions {
			if perm.Status == 1 {
				permMap[perm.Code] = perm
			}
		}
	}

	permissions := make([]model.Permission, 0, len(permMap))
	for _, perm := range permMap {
		permissions = append(permissions, perm)
	}

	utils.Success(c, permissions)
}
