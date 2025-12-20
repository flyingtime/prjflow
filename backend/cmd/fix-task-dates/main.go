package main

import (
	"log"
	"time"

	_ "modernc.org/sqlite" // 纯Go SQLite驱动

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"prjflow/internal/config"
	"prjflow/internal/model"
)

func main() {
	log.Println("开始修复任务日期字段...")

	// 加载配置
	config.LoadConfig()

	// 连接数据库
	var dialector gorm.Dialector
	if config.AppConfig.Database.Type == "sqlite" {
		dialector = sqlite.Open(config.AppConfig.Database.DSN)
	} else {
		log.Fatalf("不支持的数据库类型: %s", config.AppConfig.Database.Type)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 获取所有缺少 EndDate 的任务
	var tasks []model.Task
	if err := db.Where("end_date IS NULL").Find(&tasks).Error; err != nil {
		log.Fatalf("查询任务失败: %v", err)
	}

	log.Printf("找到 %d 个缺少结束日期的任务", len(tasks))

	fixedCount := 0
	for _, task := range tasks {
		var endDate *time.Time

		// 策略1: 如果有 DueDate，使用 DueDate 作为 EndDate
		if task.DueDate != nil {
			endDate = task.DueDate
		} else if task.StartDate != nil && task.EstimatedHours != nil && *task.EstimatedHours > 0 {
			// 策略2: 如果有开始日期和预估工时，计算结束日期（假设1天=8小时）
			days := int(*task.EstimatedHours / 8)
			if days < 1 {
				days = 1 // 至少1天
			}
			end := task.StartDate.AddDate(0, 0, days)
			endDate = &end
		} else if task.StartDate != nil {
			// 策略3: 如果只有开始日期，设置结束日期为开始日期后7天
			end := task.StartDate.AddDate(0, 0, 7)
			endDate = &end
		}

		if endDate != nil {
			if err := db.Model(&task).Update("end_date", endDate).Error; err != nil {
				log.Printf("更新任务 %d (%s) 失败: %v", task.ID, task.Title, err)
				continue
			}
			fixedCount++
			log.Printf("修复任务: %s (ID: %d), 结束日期: %s", task.Title, task.ID, endDate.Format("2006-01-02"))
		} else {
			log.Printf("无法修复任务: %s (ID: %d), 缺少必要信息", task.Title, task.ID)
		}
	}

	log.Printf("修复完成，共修复 %d 个任务", fixedCount)
}

