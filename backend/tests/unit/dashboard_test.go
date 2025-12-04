package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"project-management/internal/api"
	"project-management/internal/model"
)

func TestDashboardHandler_GetDashboard(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "仪表板测试项目")
	user := CreateTestUser(t, db, "dashboarduser", "仪表板用户")

	// 创建一些测试数据
	requirement := &model.Requirement{
		Title:     "测试需求",
		ProjectID: project.ID,
		CreatorID: user.ID,
		Status:    "in_progress",
	}
	db.Create(&requirement)

	bug := &model.Bug{
		Title:     "测试Bug",
		ProjectID: project.ID,
		CreatorID: user.ID,
		Status:    "open",
	}
	db.Create(&bug)
	// 关联用户到Bug
	db.Exec("INSERT INTO bug_assignees (bug_id, user_id) VALUES (?, ?)", bug.ID, user.ID)

	task := &model.Task{
		Title:      "测试任务",
		ProjectID:  project.ID,
		CreatorID:  user.ID,
		AssigneeID: &user.ID,
		Status:     "todo",
	}
	db.Create(&task)

	handler := api.NewDashboardHandler(db)

	t.Run("获取仪表板数据", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)

		// 设置user_id（GetDashboard需要）
		c.Set("user_id", user.ID)

		handler.GetDashboard(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		assert.NotNil(t, data["tasks"])
		assert.NotNil(t, data["bugs"])
		assert.NotNil(t, data["requirements"])
		assert.NotNil(t, data["projects"])
		assert.NotNil(t, data["statistics"])
	})

	t.Run("获取仪表板数据-未授权", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)

		// 不设置user_id

		handler.GetDashboard(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusUnauthorized || (response["code"] != nil && response["code"] != float64(200)))
	})

	t.Run("测试任务统计准确性", func(t *testing.T) {
		// 创建不同状态的任务
		task1 := &model.Task{
			Title:      "待办任务",
			ProjectID:  project.ID,
			CreatorID:  user.ID,
			AssigneeID: &user.ID,
			Status:     "wait",
		}
		db.Create(task1)

		task2 := &model.Task{
			Title:      "进行中任务",
			ProjectID:  project.ID,
			CreatorID:  user.ID,
			AssigneeID: &user.ID,
			Status:     "doing",
		}
		db.Create(task2)

		task3 := &model.Task{
			Title:      "已完成任务",
			ProjectID:  project.ID,
			CreatorID:  user.ID,
			AssigneeID: &user.ID,
			Status:     "done",
		}
		db.Create(task3)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
		c.Set("user_id", user.ID)

		handler.GetDashboard(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		tasks := data["tasks"].(map[string]interface{})
		assert.GreaterOrEqual(t, int(tasks["todo"].(float64)), 1)
		assert.GreaterOrEqual(t, int(tasks["in_progress"].(float64)), 1)
		assert.GreaterOrEqual(t, int(tasks["done"].(float64)), 1)
	})

	t.Run("测试Bug统计准确性", func(t *testing.T) {
		// 创建不同状态的Bug
		bug1 := &model.Bug{
			Title:     "活跃Bug",
			ProjectID: project.ID,
			CreatorID: user.ID,
			Status:    "active",
		}
		db.Create(bug1)
		db.Exec("INSERT INTO bug_assignees (bug_id, user_id) VALUES (?, ?)", bug1.ID, user.ID)

		bug2 := &model.Bug{
			Title:     "已解决Bug",
			ProjectID: project.ID,
			CreatorID: user.ID,
			Status:    "resolved",
		}
		db.Create(bug2)
		db.Exec("INSERT INTO bug_assignees (bug_id, user_id) VALUES (?, ?)", bug2.ID, user.ID)

		bug3 := &model.Bug{
			Title:     "已关闭Bug",
			ProjectID: project.ID,
			CreatorID: user.ID,
			Status:    "closed",
		}
		db.Create(bug3)
		db.Exec("INSERT INTO bug_assignees (bug_id, user_id) VALUES (?, ?)", bug3.ID, user.ID)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
		c.Set("user_id", user.ID)

		handler.GetDashboard(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		bugs := data["bugs"].(map[string]interface{})
		assert.GreaterOrEqual(t, int(bugs["active"].(float64)), 1)
		assert.GreaterOrEqual(t, int(bugs["resolved"].(float64)), 1)
		assert.GreaterOrEqual(t, int(bugs["closed"].(float64)), 1)
	})

	t.Run("测试需求统计准确性", func(t *testing.T) {
		// 创建不同状态的需求
		req1 := &model.Requirement{
			Title:     "进行中需求",
			ProjectID: project.ID,
			CreatorID: user.ID,
			AssigneeID: &user.ID,
			Status:    "active",
		}
		db.Create(req1)

		req2 := &model.Requirement{
			Title:     "已完成需求",
			ProjectID: project.ID,
			CreatorID: user.ID,
			AssigneeID: &user.ID,
			Status:    "completed",
		}
		db.Create(req2)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
		c.Set("user_id", user.ID)

		handler.GetDashboard(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		requirements := data["requirements"].(map[string]interface{})
		assert.GreaterOrEqual(t, int(requirements["in_progress"].(float64)), 1)
		assert.GreaterOrEqual(t, int(requirements["completed"].(float64)), 1)
	})

	t.Run("测试项目列表", func(t *testing.T) {
		// 添加用户到项目
		AddUserToProject(t, db, user.ID, project.ID, "member")

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
		c.Set("user_id", user.ID)

		handler.GetDashboard(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		projects := data["projects"].([]interface{})
		assert.GreaterOrEqual(t, len(projects), 1)
	})

	t.Run("测试工作报告统计", func(t *testing.T) {
		// 创建日报
		date, _ := time.Parse("2006-01-02", "2024-01-01")
		dailyReport := &model.DailyReport{
			UserID: user.ID,
			Date:   date,
			Status: "draft",
		}
		db.Create(dailyReport)

		// 创建周报
		weekStart, _ := time.Parse("2006-01-02", "2024-01-01")
		weekEnd, _ := time.Parse("2006-01-02", "2024-01-07")
		weeklyReport := &model.WeeklyReport{
			UserID:    user.ID,
			WeekStart: weekStart,
			WeekEnd:   weekEnd,
			Status:    "submitted",
		}
		db.Create(weeklyReport)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
		c.Set("user_id", user.ID)

		handler.GetDashboard(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		reports := data["reports"].(map[string]interface{})
		assert.GreaterOrEqual(t, int(reports["pending"].(float64)), 1)
		assert.GreaterOrEqual(t, int(reports["submitted"].(float64)), 1)
	})

	t.Run("测试资源分配统计", func(t *testing.T) {
		// 创建资源和资源分配
		resource := &model.Resource{
			UserID: user.ID,
		}
		db.Create(resource)

		date, _ := time.Parse("2006-01-02", "2024-01-01")
		allocation := &model.ResourceAllocation{
			ResourceID: resource.ID,
			Date:       date,
			Hours:      8.0,
		}
		db.Create(allocation)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
		c.Set("user_id", user.ID)

		handler.GetDashboard(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		statistics := data["statistics"].(map[string]interface{})
		assert.NotNil(t, statistics["week_hours"])
		assert.NotNil(t, statistics["month_hours"])
	})
}

