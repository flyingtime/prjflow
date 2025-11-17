package model

import (
	"time"

	"gorm.io/gorm"
)

// SystemConfig 系统配置表
type SystemConfig struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Key   string `gorm:"size:100;not null;uniqueIndex" json:"key"`   // 配置键
	Value string `gorm:"type:text" json:"value"`                      // 配置值
	Type  string `gorm:"size:20" json:"type"`                        // 配置类型：string, number, boolean, json
}

// InitStatus 初始化状态
const (
	InitStatusNotInitialized = "not_initialized"
	InitStatusInitialized    = "initialized"
)

