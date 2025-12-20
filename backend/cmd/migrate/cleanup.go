package main

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// CleanupDatabase 清理数据库中的迁移数据
func CleanupDatabase(config *MigrateConfig) error {
	log.Println("==========================================")
	log.Println("开始清理数据库...")
	log.Println("==========================================")

	// 连接prjflow数据库
	var dialector gorm.Dialector
	if config.PrjFlow.Type == "sqlite" {
		dialector = sqlite.Open(config.PrjFlow.DSN)
	} else {
		return fmt.Errorf("不支持的数据库类型: %s", config.PrjFlow.Type)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 清理顺序：先清理关联数据，再清理主数据
	log.Println("清理Bug数据...")
	if err := db.Exec("DELETE FROM bugs").Error; err != nil {
		log.Printf("清理Bug失败: %v", err)
	}

	log.Println("清理任务数据...")
	if err := db.Exec("DELETE FROM tasks").Error; err != nil {
		log.Printf("清理任务失败: %v", err)
	}

	log.Println("清理需求数据...")
	if err := db.Exec("DELETE FROM requirements").Error; err != nil {
		log.Printf("清理需求失败: %v", err)
	}

	log.Println("清理项目数据...")
	if err := db.Exec("DELETE FROM projects").Error; err != nil {
		log.Printf("清理项目失败: %v", err)
	}

	log.Println("==========================================")
	log.Println("数据库清理完成！")
	log.Println("==========================================")

	return nil
}

