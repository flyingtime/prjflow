package main

import (
	"fmt"
	"log"

	_ "modernc.org/sqlite" // 纯Go SQLite驱动

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"prjflow/internal/config"
	"prjflow/internal/model"
)

func main() {
	log.Println("开始修复版本状态字段...")

	// 加载配置
	if err := config.LoadConfig(""); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	var dialector gorm.Dialector
	if config.AppConfig.Database.Type == "sqlite" {
		dialector = sqlite.Open(config.AppConfig.Database.DSN)
	} else if config.AppConfig.Database.Type == "mysql" {
		dsn := config.AppConfig.Database.DSN
		if dsn == "" {
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				config.AppConfig.Database.User,
				config.AppConfig.Database.Password,
				config.AppConfig.Database.Host,
				config.AppConfig.Database.Port,
				config.AppConfig.Database.DBName,
			)
		}
		dialector = mysql.Open(dsn)
	} else {
		log.Fatalf("不支持的数据库类型: %s", config.AppConfig.Database.Type)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 获取所有状态为 "draft" 的版本
	var versions []model.Version
	if err := db.Where("status = ?", "draft").Find(&versions).Error; err != nil {
		log.Fatalf("查询版本失败: %v", err)
	}

	log.Printf("找到 %d 个状态为 'draft' 的版本", len(versions))

	if len(versions) == 0 {
		log.Println("没有需要修复的版本")
		return
	}

	fixedCount := 0
	for _, version := range versions {
		// 将状态从 "draft" 改为 "wait"（未开始）
		if err := db.Model(&version).Update("status", "wait").Error; err != nil {
			log.Printf("修复版本 #%d (%s) 失败: %v", version.ID, version.VersionNumber, err)
			continue
		}
		log.Printf("修复版本 #%d (%s): draft -> wait", version.ID, version.VersionNumber)
		fixedCount++
	}

	log.Printf("修复完成！共修复 %d 个版本", fixedCount)
}

