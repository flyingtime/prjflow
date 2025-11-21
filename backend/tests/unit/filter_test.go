package unit

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"project-management/internal/model"
	"project-management/internal/utils"
)

func TestIsAdmin(t *testing.T) {
	t.Run("管理员用户", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("roles", []string{"admin"})

		assert.True(t, utils.IsAdmin(c))
	})

	t.Run("非管理员用户", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("roles", []string{"developer"})

		assert.False(t, utils.IsAdmin(c))
	})

	t.Run("无角色用户", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("roles", []string{})

		assert.False(t, utils.IsAdmin(c))
	})

	t.Run("未设置角色", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		assert.False(t, utils.IsAdmin(c))
	})

	t.Run("多个角色包含admin", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("roles", []string{"developer", "admin", "manager"})

		assert.True(t, utils.IsAdmin(c))
	})
}

func TestGetUserID(t *testing.T) {
	t.Run("获取用户ID成功", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", uint(123))

		assert.Equal(t, uint(123), utils.GetUserID(c))
	})

	t.Run("未设置用户ID", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		assert.Equal(t, uint(0), utils.GetUserID(c))
	})

	t.Run("用户ID类型错误", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", "invalid")

		assert.Equal(t, uint(0), utils.GetUserID(c))
	})
}

func TestGetUserProjectIDs(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	user := CreateTestUser(t, db, "projectuser", "项目用户")
	project1 := CreateTestProject(t, db, "项目1")
	project2 := CreateTestProject(t, db, "项目2")
	project3 := CreateTestProject(t, db, "项目3")

	// 添加用户到项目1和项目2
	member1 := &model.ProjectMember{
		ProjectID: project1.ID,
		UserID:    user.ID,
		Role:      "member",
	}
	db.Create(member1)

	member2 := &model.ProjectMember{
		ProjectID: project2.ID,
		UserID:    user.ID,
		Role:      "owner",
	}
	db.Create(member2)

	t.Run("获取用户参与的项目ID列表", func(t *testing.T) {
		projectIDs := utils.GetUserProjectIDs(db, user.ID)

		assert.Equal(t, 2, len(projectIDs))
		assert.Contains(t, projectIDs, project1.ID)
		assert.Contains(t, projectIDs, project2.ID)
		assert.NotContains(t, projectIDs, project3.ID)
	})

	t.Run("用户未参与任何项目", func(t *testing.T) {
		otherUser := CreateTestUser(t, db, "otheruser", "其他用户")
		projectIDs := utils.GetUserProjectIDs(db, otherUser.ID)

		assert.Equal(t, 0, len(projectIDs))
	})
}

func TestCheckProjectAccess(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	user := CreateTestUser(t, db, "accessuser", "访问用户")
	adminUser := CreateTestUser(t, db, "adminuser", "管理员用户")
	otherUser := CreateTestUser(t, db, "otheruser", "其他用户")

	project := CreateTestProject(t, db, "访问测试项目")

	// 添加用户到项目
	member := &model.ProjectMember{
		ProjectID: project.ID,
		UserID:    user.ID,
		Role:      "member",
	}
	db.Create(member)

	// 创建管理员角色并分配给adminUser
	adminRole := &model.Role{
		Name:        "管理员",
		Code:        "admin",
		Description: "系统管理员",
		Status:      1,
	}
	db.Create(adminRole)
	db.Model(&adminUser).Association("Roles").Append(adminRole)

	t.Run("管理员可以访问所有项目", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", adminUser.ID)
		c.Set("roles", []string{"admin"})

		assert.True(t, utils.CheckProjectAccess(db, c, project.ID))
	})

	t.Run("项目成员可以访问项目", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", user.ID)
		c.Set("roles", []string{"developer"})

		assert.True(t, utils.CheckProjectAccess(db, c, project.ID))
	})

	t.Run("非项目成员不能访问项目", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", otherUser.ID)
		c.Set("roles", []string{"developer"})

		assert.False(t, utils.CheckProjectAccess(db, c, project.ID))
	})

	t.Run("未登录用户不能访问项目", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		assert.False(t, utils.CheckProjectAccess(db, c, project.ID))
	})
}

func TestCheckRequirementAccess(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	creator := CreateTestUser(t, db, "creator", "创建者")
	assignee := CreateTestUser(t, db, "assignee", "负责人")
	member := CreateTestUser(t, db, "member", "项目成员")
	otherUser := CreateTestUser(t, db, "other", "其他用户")
	adminUser := CreateTestUser(t, db, "adminuser2", "管理员用户2")

	project := CreateTestProject(t, db, "需求访问测试项目")

	// 添加成员到项目
	projectMember := &model.ProjectMember{
		ProjectID: project.ID,
		UserID:    member.ID,
		Role:      "member",
	}
	db.Create(projectMember)

	// 创建管理员角色
	adminRole := &model.Role{
		Name:        "管理员",
		Code:        "admin",
		Description: "系统管理员",
		Status:      1,
	}
	db.Create(adminRole)
	db.Model(&adminUser).Association("Roles").Append(adminRole)

	// 创建需求
	requirement := &model.Requirement{
		Title:     "测试需求",
		ProjectID: project.ID,
		CreatorID: creator.ID,
		Status:    "pending",
	}
	db.Create(requirement)

	t.Run("管理员可以访问所有需求", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", adminUser.ID)
		c.Set("roles", []string{"admin"})

		assert.True(t, utils.CheckRequirementAccess(db, c, requirement.ID))
	})

	t.Run("创建者可以访问需求", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", creator.ID)
		c.Set("roles", []string{"developer"})

		assert.True(t, utils.CheckRequirementAccess(db, c, requirement.ID))
	})

	t.Run("项目成员可以访问需求", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", member.ID)
		c.Set("roles", []string{"developer"})

		assert.True(t, utils.CheckRequirementAccess(db, c, requirement.ID))
	})

	t.Run("非项目成员不能访问需求", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", otherUser.ID)
		c.Set("roles", []string{"developer"})

		assert.False(t, utils.CheckRequirementAccess(db, c, requirement.ID))
	})

	t.Run("负责人可以访问需求", func(t *testing.T) {
		// 设置需求负责人
		requirement.AssigneeID = &assignee.ID
		db.Save(requirement)

		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", assignee.ID)
		c.Set("roles", []string{"developer"})

		assert.True(t, utils.CheckRequirementAccess(db, c, requirement.ID))
	})
}

func TestCheckTaskAccess(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	creator := CreateTestUser(t, db, "taskcreator", "任务创建者")
	assignee := CreateTestUser(t, db, "taskassignee", "任务负责人")
	member := CreateTestUser(t, db, "taskmember", "任务项目成员")
	otherUser := CreateTestUser(t, db, "taskother", "任务其他用户")
	adminUser := CreateTestUser(t, db, "adminuser3", "管理员用户3")

	project := CreateTestProject(t, db, "任务访问测试项目")

	// 添加成员到项目
	projectMember := &model.ProjectMember{
		ProjectID: project.ID,
		UserID:    member.ID,
		Role:      "member",
	}
	db.Create(projectMember)

	// 创建管理员角色
	adminRole := &model.Role{
		Name:        "管理员",
		Code:        "admin",
		Description: "系统管理员",
		Status:      1,
	}
	db.Create(adminRole)
	db.Model(&adminUser).Association("Roles").Append(adminRole)

	// 创建任务
	task := &model.Task{
		Title:     "测试任务",
		ProjectID: project.ID,
		CreatorID: creator.ID,
		Status:    "todo",
	}
	db.Create(task)

	t.Run("管理员可以访问所有任务", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", adminUser.ID)
		c.Set("roles", []string{"admin"})

		assert.True(t, utils.CheckTaskAccess(db, c, task.ID))
	})

	t.Run("创建者可以访问任务", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", creator.ID)
		c.Set("roles", []string{"developer"})

		assert.True(t, utils.CheckTaskAccess(db, c, task.ID))
	})

	t.Run("项目成员可以访问任务", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", member.ID)
		c.Set("roles", []string{"developer"})

		assert.True(t, utils.CheckTaskAccess(db, c, task.ID))
	})

	t.Run("非项目成员不能访问任务", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", otherUser.ID)
		c.Set("roles", []string{"developer"})

		assert.False(t, utils.CheckTaskAccess(db, c, task.ID))
	})

	t.Run("负责人可以访问任务", func(t *testing.T) {
		// 设置任务负责人
		task.AssigneeID = &assignee.ID
		db.Save(task)

		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", assignee.ID)
		c.Set("roles", []string{"developer"})

		assert.True(t, utils.CheckTaskAccess(db, c, task.ID))
	})
}

func TestCheckBugAccess(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	creator := CreateTestUser(t, db, "bugcreator", "Bug创建者")
	assignee := CreateTestUser(t, db, "bugassignee", "Bug分配人")
	member := CreateTestUser(t, db, "bugmember", "Bug项目成员")
	otherUser := CreateTestUser(t, db, "bugother", "Bug其他用户")
	adminUser := CreateTestUser(t, db, "adminuser4", "管理员用户4")

	project := CreateTestProject(t, db, "Bug访问测试项目")

	// 添加成员到项目
	projectMember := &model.ProjectMember{
		ProjectID: project.ID,
		UserID:    member.ID,
		Role:      "member",
	}
	db.Create(projectMember)

	// 创建管理员角色
	adminRole := &model.Role{
		Name:        "管理员",
		Code:        "admin",
		Description: "系统管理员",
		Status:      1,
	}
	db.Create(adminRole)
	db.Model(&adminUser).Association("Roles").Append(adminRole)

	// 创建Bug
	bug := &model.Bug{
		Title:     "测试Bug",
		ProjectID: project.ID,
		CreatorID: creator.ID,
		Status:    "open",
	}
	db.Create(bug)

	t.Run("管理员可以访问所有Bug", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", adminUser.ID)
		c.Set("roles", []string{"admin"})

		assert.True(t, utils.CheckBugAccess(db, c, bug.ID))
	})

	t.Run("创建者可以访问Bug", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", creator.ID)
		c.Set("roles", []string{"developer"})

		assert.True(t, utils.CheckBugAccess(db, c, bug.ID))
	})

	t.Run("项目成员可以访问Bug", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", member.ID)
		c.Set("roles", []string{"developer"})

		assert.True(t, utils.CheckBugAccess(db, c, bug.ID))
	})

	t.Run("非项目成员不能访问Bug", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", otherUser.ID)
		c.Set("roles", []string{"developer"})

		assert.False(t, utils.CheckBugAccess(db, c, bug.ID))
	})

	t.Run("分配人可以访问Bug", func(t *testing.T) {
		// 分配Bug给用户
		db.Model(&bug).Association("Assignees").Append(assignee)

		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", assignee.ID)
		c.Set("roles", []string{"developer"})

		assert.True(t, utils.CheckBugAccess(db, c, bug.ID))
	})
}

