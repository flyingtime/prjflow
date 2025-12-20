package main

import (
	"flag"
	"log"
	"os"

	"prjflow/internal/config"
	"prjflow/internal/utils"
)

func main() {
	// 解析命令行参数
	var configPath string
	flag.StringVar(&configPath, "config", "", "配置文件路径（可选，默认为 config.yaml）")
	var reset bool
	flag.BoolVar(&reset, "reset", false, "是否重置数据库（删除所有数据后重新生成）")
	flag.Parse()

	// 加载配置
	if err := config.LoadConfig(configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	db, err := utils.InitDB()
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 创建 Seeder 实例
	seeder := NewSeeder(db)

	// 如果设置了 reset 标志，先清空数据库
	if reset {
		log.Println("警告: 将删除所有现有数据！")
		log.Println("如果确认继续，请按 Enter...")
		_, err := os.Stdin.Read(make([]byte, 1))
		if err != nil {
			log.Fatalf("读取输入失败: %v", err)
		}
		_, _ = os.Stdin.Read(make([]byte, 1)) // 读取换行符

		if err := seeder.ResetDatabase(); err != nil {
			log.Fatalf("重置数据库失败: %v", err)
		}
		log.Println("数据库已清空")
	}

	// 生成演示数据
	if err := seeder.SeedAll(); err != nil {
		log.Fatalf("生成演示数据失败: %v", err)
	}

	log.Println("演示数据生成完成！")
}

