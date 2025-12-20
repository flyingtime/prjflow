package permission

import (
	"gorm.io/gorm"
	"prjflow/internal/model"
)

// CheckPermission 检查用户角色是否有权限（需要传入db）
// 注意：这个函数已被废弃，请使用 CheckPermissionWithDB
func CheckPermission(roles []string, permCode string) bool {
	// 为了向后兼容，暂时返回 true
	// 实际应该使用 CheckPermissionWithDB
	return true
}

// GetRolePermissions 获取角色的所有权限
func GetRolePermissions(db *gorm.DB, roleCodes []string) ([]string, error) {
	var permissions []model.Permission

	err := db.Table("permissions").
		Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Joins("JOIN roles ON role_permissions.role_id = roles.id").
		Where("roles.code IN ?", roleCodes).
		Find(&permissions).Error

	if err != nil {
		return nil, err
	}

	permCodes := make([]string, 0, len(permissions))
	for _, perm := range permissions {
		permCodes = append(permCodes, perm.Code)
	}

	return permCodes, nil
}

// CheckPermissionWithDB 使用数据库检查权限
func CheckPermissionWithDB(db *gorm.DB, roleCodes []string, permCode string) (bool, error) {
	// 管理员角色自动拥有所有权限
	for _, roleCode := range roleCodes {
		if roleCode == "admin" {
			return true, nil
		}
	}

	permissions, err := GetRolePermissions(db, roleCodes)
	if err != nil {
		return false, err
	}

	for _, perm := range permissions {
		if perm == permCode {
			return true, nil
		}
	}

	return false, nil
}

