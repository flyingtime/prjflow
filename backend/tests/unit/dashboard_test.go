package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
}

