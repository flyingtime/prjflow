package main

import (
	"fmt"
	"log"
	"time"

	"prjflow/internal/model"
	"prjflow/internal/utils"

	"gorm.io/gorm"
)

// Seeder 数据生成器
type Seeder struct {
	db *gorm.DB
}

// NewSeeder 创建新的 Seeder 实例
func NewSeeder(db *gorm.DB) *Seeder {
	return &Seeder{db: db}
}

// ResetDatabase 重置数据库（删除所有数据）
func (s *Seeder) ResetDatabase() error {
	// 按依赖关系逆序删除（避免外键约束）
	// 使用 Unscoped 进行硬删除

	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.DailyReportApproval{}).Error; err != nil {
		log.Printf("警告: 删除日报审批记录失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.WeeklyReportApproval{}).Error; err != nil {
		log.Printf("警告: 删除周报审批记录失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.DailyReport{}).Error; err != nil {
		log.Printf("警告: 删除日报失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.WeeklyReport{}).Error; err != nil {
		log.Printf("警告: 删除周报失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.ResourceAllocation{}).Error; err != nil {
		log.Printf("警告: 删除资源分配失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.Resource{}).Error; err != nil {
		log.Printf("警告: 删除资源失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.TestCaseBug{}).Error; err != nil {
		log.Printf("警告: 删除测试用例Bug关联失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.TestCase{}).Error; err != nil {
		log.Printf("警告: 删除测试用例失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.Version{}).Error; err != nil {
		log.Printf("警告: 删除版本失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.BugAssignee{}).Error; err != nil {
		log.Printf("警告: 删除Bug分配失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.Bug{}).Error; err != nil {
		log.Printf("警告: 删除Bug失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.TaskDependency{}).Error; err != nil {
		log.Printf("警告: 删除任务依赖失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.Task{}).Error; err != nil {
		log.Printf("警告: 删除任务失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.Requirement{}).Error; err != nil {
		log.Printf("警告: 删除需求失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.BoardColumn{}).Error; err != nil {
		log.Printf("警告: 删除看板列失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.Board{}).Error; err != nil {
		log.Printf("警告: 删除看板失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.ProjectMember{}).Error; err != nil {
		log.Printf("警告: 删除项目成员失败: %v", err)
	}
	// 删除项目标签关联
	if err := s.db.Exec("DELETE FROM project_tags").Error; err != nil {
		log.Printf("警告: 删除项目标签关联失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.Project{}).Error; err != nil {
		log.Printf("警告: 删除项目失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.Module{}).Error; err != nil {
		log.Printf("警告: 删除模块失败: %v", err)
	}
	// 删除用户角色关联
	if err := s.db.Exec("DELETE FROM user_roles").Error; err != nil {
		log.Printf("警告: 删除用户角色关联失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.User{}).Error; err != nil {
		log.Printf("警告: 删除用户失败: %v", err)
	}
	if err := s.db.Unscoped().Where("1 = 1").Delete(&model.Tag{}).Error; err != nil {
		log.Printf("警告: 删除标签失败: %v", err)
	}
	// 注意：不删除部门和角色，因为它们是系统基础数据
	// 如果需要重置，可以取消注释下面的代码
	// if err := s.db.Unscoped().Where("1 = 1").Delete(&model.Department{}).Error; err != nil {
	// 	log.Printf("警告: 删除部门失败: %v", err)
	// }

	return nil
}

// SeedAll 生成所有演示数据
func (s *Seeder) SeedAll() error {
	log.Println("开始生成演示数据...")

	// 确保权限和角色已初始化
	if err := utils.AutoMigrate(s.db); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 1. 生成部门
	departments, err := s.seedDepartments()
	if err != nil {
		return fmt.Errorf("生成部门失败: %w", err)
	}

	// 2. 生成用户
	users, err := s.seedUsers(departments)
	if err != nil {
		return fmt.Errorf("生成用户失败: %w", err)
	}

	// 3. 生成标签
	tags, err := s.seedTags()
	if err != nil {
		return fmt.Errorf("生成标签失败: %w", err)
	}

	// 4. 生成项目
	projects, err := s.seedProjects(tags, users)
	if err != nil {
		return fmt.Errorf("生成项目失败: %w", err)
	}

	// 5. 生成模块
	modules, err := s.seedModules()
	if err != nil {
		return fmt.Errorf("生成模块失败: %w", err)
	}

	// 6. 生成需求
	requirements, err := s.seedRequirements(projects, users)
	if err != nil {
		return fmt.Errorf("生成需求失败: %w", err)
	}

	// 7. 生成任务
	tasks, err := s.seedTasks(projects, requirements, users)
	if err != nil {
		return fmt.Errorf("生成任务失败: %w", err)
	}

	// 8. 生成Bug
	bugs, err := s.seedBugs(projects, requirements, modules, users)
	if err != nil {
		return fmt.Errorf("生成Bug失败: %w", err)
	}

	// 9. 生成测试用例
	_, err = s.seedTestCases(projects, users)
	if err != nil {
		return fmt.Errorf("生成测试用例失败: %w", err)
	}

	// 10. 生成版本
	_, err = s.seedVersions(projects, requirements, bugs)
	if err != nil {
		return fmt.Errorf("生成版本失败: %w", err)
	}

	// 11. 生成资源分配
	err = s.seedResourceAllocations(projects, tasks, bugs, requirements, users)
	if err != nil {
		return fmt.Errorf("生成资源分配失败: %w", err)
	}

	// 12. 生成工作报告
	err = s.seedReports(users)
	if err != nil {
		return fmt.Errorf("生成工作报告失败: %w", err)
	}

	// 13. 设置系统初始化状态
	err = s.setInitialized()
	if err != nil {
		return fmt.Errorf("设置初始化状态失败: %w", err)
	}

	log.Println("演示数据生成完成！")
	return nil
}

// seedDepartments 生成部门数据
func (s *Seeder) seedDepartments() ([]model.Department, error) {
	departments := []model.Department{
		{Name: "研发部", Code: "RD", Level: 1, Sort: 1, Status: 1},
		{Name: "测试部", Code: "QA", Level: 1, Sort: 2, Status: 1},
		{Name: "产品部", Code: "PM", Level: 1, Sort: 3, Status: 1},
		{Name: "运营部", Code: "OP", Level: 1, Sort: 4, Status: 1},
	}

	result := []model.Department{}
	for _, dept := range departments {
		var existing model.Department
		if err := s.db.Where("code = ?", dept.Code).First(&existing).Error; err == nil {
			result = append(result, existing)
			continue
		}
		if err := s.db.Create(&dept).Error; err != nil {
			return nil, err
		}
		result = append(result, dept)
	}

	log.Printf("生成部门: %d 个", len(result))
	return result, nil
}

// seedUsers 生成用户数据
func (s *Seeder) seedUsers(departments []model.Department) ([]model.User, error) {
	// 获取角色
	var adminRole, devRole, qaRole, pmRole model.Role
	if err := s.db.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
		return nil, fmt.Errorf("管理员角色不存在，请先运行数据库迁移: %w", err)
	}
	if err := s.db.Where("code = ?", "developer").First(&devRole).Error; err != nil {
		// 如果开发者角色不存在，创建它
		devRole = model.Role{
			Name:        "开发者",
			Code:        "developer",
			Description: "开发人员角色",
			Status:      1,
		}
		if err := s.db.Create(&devRole).Error; err != nil {
			return nil, err
		}
	}
	if err := s.db.Where("code = ?", "qa").First(&qaRole).Error; err != nil {
		qaRole = model.Role{
			Name:        "测试人员",
			Code:        "qa",
			Description: "测试人员角色",
			Status:      1,
		}
		if err := s.db.Create(&qaRole).Error; err != nil {
			return nil, err
		}
	}
	if err := s.db.Where("code = ?", "pm").First(&pmRole).Error; err != nil {
		pmRole = model.Role{
			Name:        "产品经理",
			Code:        "pm",
			Description: "产品经理角色",
			Status:      1,
		}
		if err := s.db.Create(&pmRole).Error; err != nil {
			return nil, err
		}
	}

	rdDeptID := departments[0].ID
	qaDeptID := departments[1].ID
	pmDeptID := departments[2].ID

	hashedPassword, _ := utils.HashPassword("Demo123") // 所有演示用户使用相同密码

	users := []struct {
		model.User
		Role model.Role
	}{
		{
			User: model.User{
				Username:     "admin",
				Nickname:     "系统管理员",
				Password:     hashedPassword,
				Email:        "admin@example.com",
				Phone:        "13800000001",
				Status:       1,
				DepartmentID: &rdDeptID,
			},
			Role: adminRole,
		},
		{
			User: model.User{
				Username:     "zhangsan",
				Nickname:     "张三",
				Password:     hashedPassword,
				Email:        "zhangsan@example.com",
				Phone:        "13800000002",
				Status:       1,
				DepartmentID: &rdDeptID,
			},
			Role: devRole,
		},
		{
			User: model.User{
				Username:     "lisi",
				Nickname:     "李四",
				Password:     hashedPassword,
				Email:        "lisi@example.com",
				Phone:        "13800000003",
				Status:       1,
				DepartmentID: &rdDeptID,
			},
			Role: devRole,
		},
		{
			User: model.User{
				Username:     "wangwu",
				Nickname:     "王五",
				Password:     hashedPassword,
				Email:        "wangwu@example.com",
				Phone:        "13800000004",
				Status:       1,
				DepartmentID: &qaDeptID,
			},
			Role: qaRole,
		},
		{
			User: model.User{
				Username:     "zhaoliu",
				Nickname:     "赵六",
				Password:     hashedPassword,
				Email:        "zhaoliu@example.com",
				Phone:        "13800000005",
				Status:       1,
				DepartmentID: &pmDeptID,
			},
			Role: pmRole,
		},
	}

	result := []model.User{}
	for _, u := range users {
		var existing model.User
		if err := s.db.Where("username = ?", u.Username).First(&existing).Error; err == nil {
			result = append(result, existing)
			continue
		}
		if err := s.db.Create(&u.User).Error; err != nil {
			return nil, err
		}
		// 分配角色
		if err := s.db.Model(&u.User).Association("Roles").Append(&u.Role); err != nil {
			return nil, err
		}
		result = append(result, u.User)
	}

	log.Printf("生成用户: %d 个", len(result))
	return result, nil
}

// seedTags 生成标签数据
func (s *Seeder) seedTags() ([]model.Tag, error) {
	tags := []model.Tag{
		{Name: "前端", Description: "前端相关", Color: "blue"},
		{Name: "后端", Description: "后端相关", Color: "green"},
		{Name: "移动端", Description: "移动端相关", Color: "orange"},
		{Name: "重要", Description: "重要项目", Color: "red"},
		{Name: "紧急", Description: "紧急项目", Color: "volcano"},
		{Name: "新功能", Description: "新功能开发", Color: "cyan"},
	}

	result := []model.Tag{}
	for _, tag := range tags {
		var existing model.Tag
		if err := s.db.Where("name = ?", tag.Name).First(&existing).Error; err == nil {
			result = append(result, existing)
			continue
		}
		if err := s.db.Create(&tag).Error; err != nil {
			return nil, err
		}
		result = append(result, tag)
	}

	log.Printf("生成标签: %d 个", len(result))
	return result, nil
}

// seedProjects 生成项目数据
func (s *Seeder) seedProjects(tags []model.Tag, users []model.User) ([]model.Project, error) {
	now := time.Now()
	startDate1 := now.AddDate(0, -3, 0)
	endDate1 := now.AddDate(0, 3, 0)
	startDate2 := now.AddDate(0, -1, 0)
	endDate2 := now.AddDate(0, 5, 0)

	projects := []model.Project{
		{
			Name:        "电商平台系统",
			Code:        "ECOM",
			Description: "一个完整的电商平台系统，包括用户管理、商品管理、订单管理、支付等功能。",
			Status:      "doing",
			StartDate:   &startDate1,
			EndDate:     &endDate1,
		},
		{
			Name:        "移动办公APP",
			Code:        "MOBILE",
			Description: "企业移动办公应用，支持考勤、审批、通讯录等功能。",
			Status:      "doing",
			StartDate:   &startDate2,
			EndDate:     &endDate2,
		},
		{
			Name:        "数据可视化平台",
			Code:        "DASHBOARD",
			Description: "企业级数据可视化分析平台，支持多种图表类型和数据分析功能。",
			Status:      "wait",
		},
	}

	result := []model.Project{}
	for i, project := range projects {
		var existing model.Project
		if err := s.db.Where("code = ?", project.Code).First(&existing).Error; err == nil {
			result = append(result, existing)
			continue
		}
		if err := s.db.Create(&project).Error; err != nil {
			return nil, err
		}
		// 关联标签（每个项目关联2-3个标签）
		projectTags := tags[:2+i%3]
		if err := s.db.Model(&project).Association("Tags").Append(projectTags); err != nil {
			return nil, err
		}
		// 添加项目成员
		if len(users) > 0 {
			members := []model.ProjectMember{
				{ProjectID: project.ID, UserID: users[0].ID, Role: "owner"},
			}
			if len(users) > 1 {
				members = append(members, model.ProjectMember{
					ProjectID: project.ID,
					UserID:    users[1].ID,
					Role:      "member",
				})
			}
			for _, member := range members {
				if err := s.db.Create(&member).Error; err != nil {
					return nil, err
				}
			}
		}
		result = append(result, project)
	}

	log.Printf("生成项目: %d 个", len(result))
	return result, nil
}

// seedModules 生成模块数据
func (s *Seeder) seedModules() ([]model.Module, error) {
	modules := []model.Module{
		{Name: "用户模块", Code: "USER", Description: "用户注册、登录、个人信息管理", Status: 1, Sort: 1},
		{Name: "商品模块", Code: "PRODUCT", Description: "商品管理、分类、库存管理", Status: 1, Sort: 2},
		{Name: "订单模块", Code: "ORDER", Description: "订单创建、支付、退款", Status: 1, Sort: 3},
		{Name: "支付模块", Code: "PAYMENT", Description: "支付接口、支付记录", Status: 1, Sort: 4},
	}

	result := []model.Module{}
	for _, module := range modules {
		var existing model.Module
		if err := s.db.Where("code = ?", module.Code).First(&existing).Error; err == nil {
			result = append(result, existing)
			continue
		}
		if err := s.db.Create(&module).Error; err != nil {
			return nil, err
		}
		result = append(result, module)
	}

	log.Printf("生成模块: %d 个", len(result))
	return result, nil
}

// seedRequirements 生成需求数据
func (s *Seeder) seedRequirements(projects []model.Project, users []model.User) ([]model.Requirement, error) {
	if len(projects) == 0 || len(users) == 0 {
		return []model.Requirement{}, nil
	}

	requirements := []model.Requirement{
		{
			Title:          "用户注册登录功能",
			Description:    "实现用户注册、登录、忘记密码等基础功能。\n\n## 功能点\n- 手机号注册\n- 邮箱注册\n- 短信验证码登录\n- 密码找回",
			Status:         "active",
			Priority:       "high",
			ProjectID:      projects[0].ID,
			CreatorID:      users[0].ID,
			AssigneeID:     &users[1].ID,
			EstimatedHours: floatPtr(40),
		},
		{
			Title:          "商品详情页优化",
			Description:    "优化商品详情页的用户体验，包括图片展示、规格选择、价格计算等。",
			Status:         "reviewing",
			Priority:       "medium",
			ProjectID:      projects[0].ID,
			CreatorID:      users[4].ID, // PM
			AssigneeID:     &users[1].ID,
			EstimatedHours: floatPtr(24),
		},
		{
			Title:          "移动端首页改版",
			Description:    "重新设计移动端首页，提升用户体验和转化率。",
			Status:         "active",
			Priority:       "high",
			ProjectID:      projects[1].ID,
			CreatorID:      users[4].ID,
			AssigneeID:     &users[2].ID,
			EstimatedHours: floatPtr(60),
		},
	}

	result := []model.Requirement{}
	for _, req := range requirements {
		if err := s.db.Create(&req).Error; err != nil {
			return nil, err
		}
		result = append(result, req)
	}

	log.Printf("生成需求: %d 个", len(result))
	return result, nil
}

// seedTasks 生成任务数据
func (s *Seeder) seedTasks(projects []model.Project, requirements []model.Requirement, users []model.User) ([]model.Task, error) {
	if len(projects) == 0 || len(users) == 0 {
		return []model.Task{}, nil
	}

	now := time.Now()
	startDate1 := now.AddDate(0, 0, -5)
	endDate1 := now.AddDate(0, 0, 5)
	startDate2 := now.AddDate(0, 0, -2)
	endDate2 := now.AddDate(0, 0, 10)
	dueDate1 := now.AddDate(0, 0, 3)

	var requirementID *uint
	if len(requirements) > 0 {
		requirementID = &requirements[0].ID
	}

	tasks := []model.Task{
		{
			Title:          "实现用户注册接口",
			Description:    "开发用户注册的API接口，包括参数验证、密码加密、数据入库等。",
			Status:         "doing",
			Priority:       "high",
			ProjectID:      projects[0].ID,
			RequirementID:  requirementID,
			CreatorID:      users[0].ID,
			AssigneeID:     &users[1].ID,
			StartDate:      &startDate1,
			EndDate:        &endDate1,
			DueDate:        &dueDate1,
			Progress:       60,
			EstimatedHours: floatPtr(8),
		},
		{
			Title:          "设计商品详情页UI",
			Description:    "使用设计工具设计商品详情页的UI界面。",
			Status:         "done",
			Priority:       "medium",
			ProjectID:      projects[0].ID,
			CreatorID:      users[4].ID,
			AssigneeID:     &users[4].ID,
			StartDate:      &startDate2,
			EndDate:        &endDate2,
			Progress:       100,
			EstimatedHours: floatPtr(4),
			ActualHours:    floatPtr(4),
		},
		{
			Title:          "开发移动端首页组件",
			Description:    "使用React Native开发移动端首页的各个组件。",
			Status:         "doing",
			Priority:       "high",
			ProjectID:      projects[1].ID,
			CreatorID:      users[0].ID,
			AssigneeID:     &users[2].ID,
			Progress:       40,
			EstimatedHours: floatPtr(16),
		},
		{
			Title:          "编写API文档",
			Description:    "为用户注册接口编写详细的API文档。",
			Status:         "wait",
			Priority:       "low",
			ProjectID:      projects[0].ID,
			RequirementID:  requirementID,
			CreatorID:      users[1].ID,
			Progress:       0,
			EstimatedHours: floatPtr(2),
		},
	}

	result := []model.Task{}
	for _, task := range tasks {
		if err := s.db.Create(&task).Error; err != nil {
			return nil, err
		}
		result = append(result, task)
	}

	log.Printf("生成任务: %d 个", len(result))
	return result, nil
}

// seedBugs 生成Bug数据
func (s *Seeder) seedBugs(projects []model.Project, requirements []model.Requirement, modules []model.Module, users []model.User) ([]model.Bug, error) {
	if len(projects) == 0 || len(users) == 0 {
		return []model.Bug{}, nil
	}

	var requirementID *uint
	if len(requirements) > 0 {
		requirementID = &requirements[0].ID
	}
	var moduleID *uint
	if len(modules) > 0 {
		moduleID = &modules[0].ID
	}

	bugs := []model.Bug{
		{
			Title:          "用户登录时密码验证失败",
			Description:    "使用正确密码登录时，系统提示密码错误。\n\n## 复现步骤\n1. 输入正确的用户名和密码\n2. 点击登录按钮\n3. 系统提示密码错误\n\n## 预期结果\n应该成功登录\n\n## 实际结果\n提示密码错误",
			Status:         "active",
			Priority:       "high",
			Severity:       "high",
			Confirmed:      true,
			ProjectID:      projects[0].ID,
			RequirementID:  requirementID,
			ModuleID:       moduleID,
			CreatorID:      users[3].ID, // QA
			EstimatedHours: floatPtr(4),
		},
		{
			Title:          "商品详情页图片加载缓慢",
			Description:    "商品详情页的图片加载速度很慢，影响用户体验。",
			Status:         "active",
			Priority:       "medium",
			Severity:       "medium",
			Confirmed:      true,
			ProjectID:      projects[0].ID,
			CreatorID:      users[3].ID,
			EstimatedHours: floatPtr(8),
		},
		{
			Title:          "移动端首页在某些机型上显示异常",
			Description:    "在某些Android低版本手机上，首页布局出现错乱。",
			Status:         "resolved",
			Priority:       "medium",
			Severity:       "medium",
			Confirmed:      true,
			ProjectID:      projects[1].ID,
			Solution:       "已解决",
			SolutionNote:   "已修复CSS兼容性问题",
			CreatorID:      users[3].ID,
			EstimatedHours: floatPtr(6),
			ActualHours:    floatPtr(6),
		},
	}

	result := []model.Bug{}
	for _, bug := range bugs {
		if err := s.db.Create(&bug).Error; err != nil {
			return nil, err
		}
		// 分配Bug给处理人
		if bug.Status != "closed" && len(users) > 1 {
			if err := s.db.Model(&bug).Association("Assignees").Append(&users[1]); err != nil {
				return nil, err
			}
		}
		result = append(result, bug)
	}

	log.Printf("生成Bug: %d 个", len(result))
	return result, nil
}

// seedTestCases 生成测试用例数据
func (s *Seeder) seedTestCases(projects []model.Project, users []model.User) ([]model.TestCase, error) {
	if len(projects) == 0 || len(users) == 0 {
		return []model.TestCase{}, nil
	}

	testCases := []model.TestCase{
		{
			Name:        "用户登录功能测试",
			Description: "测试用户登录的各个场景",
			TestSteps:   "1. 输入正确的用户名和密码，点击登录\n2. 输入错误的密码，点击登录\n3. 输入不存在的用户名，点击登录\n4. 不输入密码，点击登录",
			Types:       model.StringArray{"functional"},
			Status:      "normal",
			Result:      "passed",
			Summary:     "所有测试场景均通过",
			ProjectID:   projects[0].ID,
			CreatorID:   users[3].ID, // QA
		},
		{
			Name:        "商品详情页性能测试",
			Description: "测试商品详情页的加载性能",
			TestSteps:   "1. 打开商品详情页\n2. 记录页面加载时间\n3. 检查图片加载速度\n4. 检查接口响应时间",
			Types:       model.StringArray{"performance"},
			Status:      "normal",
			Result:      "failed",
			Summary:     "图片加载时间超过3秒，需要优化",
			ProjectID:   projects[0].ID,
			CreatorID:   users[3].ID,
		},
	}

	result := []model.TestCase{}
	for _, tc := range testCases {
		if err := s.db.Create(&tc).Error; err != nil {
			return nil, err
		}
		result = append(result, tc)
	}

	log.Printf("生成测试用例: %d 个", len(result))
	return result, nil
}

// seedVersions 生成版本数据
func (s *Seeder) seedVersions(projects []model.Project, requirements []model.Requirement, bugs []model.Bug) ([]model.Version, error) {
	if len(projects) == 0 {
		return []model.Version{}, nil
	}

	now := time.Now()
	releaseDate1 := now.AddDate(0, 0, -10)
	releaseDate2 := now.AddDate(0, 0, 30)

	versions := []model.Version{
		{
			VersionNumber: "v1.0.0",
			ReleaseNotes:  "## 版本说明\n\n### 新增功能\n- 用户注册登录功能\n- 商品详情页\n\n### 修复问题\n- 修复登录密码验证问题\n- 修复图片加载缓慢问题",
			Status:        "normal",
			ProjectID:     projects[0].ID,
			ReleaseDate:   &releaseDate1,
		},
		{
			VersionNumber: "v2.0.0",
			ReleaseNotes:  "## 版本说明\n\n### 新增功能\n- 移动端首页改版\n- 新增移动办公功能\n\n### 优化\n- 优化页面加载速度\n- 优化用户体验",
			Status:        "wait",
			ProjectID:     projects[1].ID,
			ReleaseDate:   &releaseDate2,
		},
	}

	result := []model.Version{}
	for _, version := range versions {
		if err := s.db.Create(&version).Error; err != nil {
			return nil, err
		}
		// 关联需求
		if len(requirements) > 0 && version.ProjectID == requirements[0].ProjectID {
			if err := s.db.Model(&version).Association("Requirements").Append(&requirements[0]); err != nil {
				return nil, err
			}
		}
		// 关联Bug
		if len(bugs) > 0 && version.ProjectID == bugs[0].ProjectID {
			if err := s.db.Model(&version).Association("Bugs").Append(&bugs[0]); err != nil {
				return nil, err
			}
		}
		result = append(result, version)
	}

	log.Printf("生成版本: %d 个", len(result))
	return result, nil
}

// seedResourceAllocations 生成资源分配数据
func (s *Seeder) seedResourceAllocations(projects []model.Project, tasks []model.Task, bugs []model.Bug, requirements []model.Requirement, users []model.User) error {
	if len(projects) == 0 || len(users) == 0 {
		return nil
	}

	// 先创建资源
	var resources []model.Resource
	for i := 0; i < len(users) && i < len(projects); i++ {
		var resource model.Resource
		if err := s.db.Where("user_id = ? AND project_id = ?", users[i].ID, projects[i%len(projects)].ID).First(&resource).Error; err != nil {
			resource = model.Resource{
				UserID:    users[i].ID,
				ProjectID: projects[i%len(projects)].ID,
				Role:      "developer",
			}
			if err := s.db.Create(&resource).Error; err != nil {
				return err
			}
		}
		resources = append(resources, resource)
	}

	// 生成最近7天的资源分配记录
	now := time.Now()
	allocations := []model.ResourceAllocation{}
	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, -i)
		for j, resource := range resources {
			var taskID *uint
			var bugID *uint
			var requirementID *uint

			if j < len(tasks) {
				taskID = &tasks[j%len(tasks)].ID
			}
			if j < len(bugs) {
				bugID = &bugs[j%len(bugs)].ID
			}
			if j < len(requirements) {
				requirementID = &requirements[j%len(requirements)].ID
			}

			hours := 2.0 + float64(j%4) // 2-6小时
			allocation := model.ResourceAllocation{
				ResourceID:    resource.ID,
				Date:          date,
				Hours:         hours,
				TaskID:        taskID,
				BugID:         bugID,
				RequirementID: requirementID,
				ProjectID:     &resource.ProjectID,
				Description:   fmt.Sprintf("完成相关开发工作"),
			}
			allocations = append(allocations, allocation)
		}
	}

	for _, allocation := range allocations {
		if err := s.db.Create(&allocation).Error; err != nil {
			return err
		}
	}

	log.Printf("生成资源分配记录: %d 条", len(allocations))
	return nil
}

// seedReports 生成工作报告数据
func (s *Seeder) seedReports(users []model.User) error {
	if len(users) == 0 {
		return nil
	}

	now := time.Now()

	// 生成最近7天的日报
	dailyReports := []model.DailyReport{}
	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, -i)
		for _, user := range users {
			content := fmt.Sprintf("## 今日工作总结\n\n1. 完成了相关功能的开发和测试\n2. 修复了一些已知问题\n3. 参与了项目评审会议\n\n## 遇到的问题\n\n暂无\n\n## 明日计划\n\n继续推进项目进度")
			status := "submitted"
			if i == 0 {
				status = "draft" // 今天的日报为草稿
			}
			report := model.DailyReport{
				Date:    date,
				Content: content,
				Status:  status,
				UserID:  user.ID,
			}
			if err := s.db.Create(&report).Error; err != nil {
				// 如果已存在，跳过
				continue
			}
			dailyReports = append(dailyReports, report)
		}
	}

	// 生成最近4周的周报
	weeklyReports := []model.WeeklyReport{}
	for i := 0; i < 4; i++ {
		weekEnd := now.AddDate(0, 0, -i*7-1)
		weekStart := weekEnd.AddDate(0, 0, -6)
		for _, user := range users {
			summary := fmt.Sprintf("## 本周工作总结\n\n1. 完成了项目相关功能的开发\n2. 修复了多个Bug\n3. 参与了团队技术分享\n\n## 完成的工作\n\n- 开发了用户注册登录功能\n- 优化了商品详情页性能\n- 修复了移动端显示问题")
			nextWeekPlan := fmt.Sprintf("## 下周工作计划\n\n1. 继续开发新功能\n2. 优化系统性能\n3. 编写技术文档")
			status := "submitted"
			if i == 0 {
				status = "draft" // 本周的周报为草稿
			}
			report := model.WeeklyReport{
				WeekStart:    weekStart,
				WeekEnd:      weekEnd,
				Summary:      summary,
				NextWeekPlan: nextWeekPlan,
				Status:       status,
				UserID:       user.ID,
			}
			if err := s.db.Create(&report).Error; err != nil {
				// 如果已存在，跳过
				continue
			}
			weeklyReports = append(weeklyReports, report)
		}
	}

	log.Printf("生成日报: %d 条", len(dailyReports))
	log.Printf("生成周报: %d 条", len(weeklyReports))
	return nil
}

// setInitialized 设置系统初始化状态为 true
func (s *Seeder) setInitialized() error {
	initConfig := model.SystemConfig{
		Key:   "initialized",
		Value: "true",
		Type:  "boolean",
	}

	// 使用 FirstOrCreate 确保如果已存在则更新，不存在则创建
	if err := s.db.Where("key = ?", "initialized").
		Assign(model.SystemConfig{Value: "true", Type: "boolean"}).
		FirstOrCreate(&initConfig).Error; err != nil {
		return err
	}

	log.Println("系统初始化状态已设置为 true")
	return nil
}

// floatPtr 返回 float64 的指针
func floatPtr(f float64) *float64 {
	return &f
}
