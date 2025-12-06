package model

import (
	"time"
)

// AuditLog 审计日志表
type AuditLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 用户信息
	UserID   uint   `gorm:"index" json:"user_id"`   // 用户ID（单列索引）
	Username string `gorm:"size:50" json:"username"` // 用户名（冗余字段，便于查询）

	// 操作信息
	ActionType string `gorm:"size:50;not null;index" json:"action_type"` // 操作类型：login, logout, create, update, delete等
	// CreatedAt 作为操作时间，有单列索引
	
	// 资源信息
	ResourceType string `gorm:"size:50" json:"resource_type"` // 资源类型：user, project, permission等
	ResourceID   uint   `json:"resource_id"`                  // 资源ID

	// 请求信息
	IPAddress string `gorm:"size:50" json:"ip_address"`   // IP地址
	Path      string `gorm:"size:200" json:"path"`        // 请求路径
	Method    string `gorm:"size:10" json:"method"`       // 请求方法：GET, POST, PUT, DELETE等
	Params    string `gorm:"type:text" json:"params"`     // 请求参数（JSON格式）

	// 操作结果
	Success   bool   `gorm:"default:true" json:"success"`  // 操作是否成功
	ErrorMsg  string `gorm:"type:text" json:"error_msg"`  // 错误信息（如果失败）
	Comment   string `gorm:"type:text" json:"comment"`    // 备注信息
}

