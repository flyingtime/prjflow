package unit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"project-management/internal/api"
	"project-management/internal/model"
)

func TestRequirementHandler_GetRequirements(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "需求测试项目")
	user := CreateTestUser(t, db, "requser", "需求用户")
	adminUser := CreateTestAdminUser(t, db, "adminreq", "管理员需求用户")
	otherUser := CreateTestUser(t, db, "otherreq", "其他需求用户")

	// 添加用户到项目
	AddUserToProject(t, db, user.ID, project.ID, "member")

	// 创建测试需求
	projectID := project.ID
	req1 := &model.Requirement{
		Title:     "需求1",
		ProjectID: projectID,
		CreatorID: user.ID,
		Status:    "pending",
		Priority:  "high",
	}
	db.Create(req1)

	// 创建另一个项目的需求
	project2 := CreateTestProject(t, db, "需求测试项目2")
	projectID2 := project2.ID
	req2 := &model.Requirement{
		Title:     "需求2",
		ProjectID: projectID2,
		CreatorID: otherUser.ID,
		Status:    "pending",
		Priority:  "high",
	}
	db.Create(req2)

	handler := api.NewRequirementHandler(db)

	t.Run("管理员可以获取所有需求", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/requirements", nil)
		c.Set("user_id", adminUser.ID)
		c.Set("roles", []string{"admin"})

		handler.GetRequirements(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		// 管理员应该能看到所有需求
		assert.GreaterOrEqual(t, len(list), 2)
	})

	t.Run("普通用户只能看到自己创建或参与的需求", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/requirements", nil)
		c.Set("user_id", user.ID)
		c.Set("roles", []string{"developer"})

		handler.GetRequirements(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		// 用户创建了需求1且是项目成员，应该能看到需求1
		assert.Equal(t, 1, len(list))
	})

	t.Run("搜索需求-管理员", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/requirements?keyword=需求1", nil)
		c.Set("user_id", adminUser.ID)
		c.Set("roles", []string{"admin"})

		handler.GetRequirements(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.GreaterOrEqual(t, len(list), 1)
	})

	t.Run("搜索需求-普通用户", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/requirements?keyword=需求1", nil)
		c.Set("user_id", user.ID)
		c.Set("roles", []string{"developer"})

		handler.GetRequirements(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		// 用户应该能看到自己创建的需求1
		assert.Equal(t, 1, len(list))
	})

	_ = project
	_ = user
}

func TestRequirementHandler_GetRequirement(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "需求详情项目")
	user := CreateTestUser(t, db, "reqdetail", "需求详情用户")
	adminUser := CreateTestAdminUser(t, db, "adminreq2", "管理员需求用户2")
	otherUser := CreateTestUser(t, db, "otherreq2", "其他需求用户2")

	// 添加用户到项目
	AddUserToProject(t, db, user.ID, project.ID, "member")

	projectID := project.ID
	requirement := &model.Requirement{
		Title:     "测试需求",
		ProjectID: projectID,
		CreatorID: user.ID,
		Status:    "pending",
	}
	db.Create(&requirement)

	handler := api.NewRequirementHandler(db)

	t.Run("管理员可以获取任何需求", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/requirements/%d", requirement.ID), nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: fmt.Sprintf("%d", requirement.ID)}}
		c.Set("user_id", adminUser.ID)
		c.Set("roles", []string{"admin"})

		handler.GetRequirement(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "测试需求", data["title"])
	})

	t.Run("创建者可以获取需求", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/requirements/%d", requirement.ID), nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: fmt.Sprintf("%d", requirement.ID)}}
		c.Set("user_id", user.ID)
		c.Set("roles", []string{"developer"})

		handler.GetRequirement(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])
	})

	t.Run("非项目成员不能获取需求", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/requirements/%d", requirement.ID), nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: fmt.Sprintf("%d", requirement.ID)}}
		c.Set("user_id", otherUser.ID)
		c.Set("roles", []string{"developer"})

		handler.GetRequirement(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		// 应该返回403或code不为200
		assert.True(t, w.Code == http.StatusForbidden || (response["code"] != nil && response["code"] != float64(200)))
	})

	t.Run("获取不存在的需求", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/requirements/999", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "999"}}

		handler.GetRequirement(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusNotFound || (response["code"] != nil && response["code"] != float64(200)))
	})
}

func TestRequirementHandler_CreateRequirement(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "创建需求项目")
	user := CreateTestUser(t, db, "createreq", "创建需求用户")
	handler := api.NewRequirementHandler(db)

	t.Run("创建需求成功-项目成员", func(t *testing.T) {
		// 添加用户到项目
		AddUserToProject(t, db, user.ID, project.ID, "member")

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// 设置user_id（CreateRequirement需要）
		c.Set("user_id", user.ID)
		c.Set("roles", []string{"developer"})

		projectID := float64(project.ID)
		reqBody := map[string]interface{}{
			"title":      "新需求",
			"description": "这是一个新需求",
			"status":     "draft", // 使用有效的状态值
			"priority":   "high",
			"project_id": projectID,
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/requirements", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateRequirement(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		// 验证需求已创建
		var requirement model.Requirement
		err = db.Where("title = ?", "新需求").First(&requirement).Error
		assert.NoError(t, err)
		assert.Equal(t, "新需求", requirement.Title)
	})

	t.Run("创建需求失败-非项目成员", func(t *testing.T) {
		otherUser := CreateTestUser(t, db, "othercreatereq", "其他创建需求用户")
		otherProject := CreateTestProject(t, db, "其他项目")

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// 设置user_id
		c.Set("user_id", otherUser.ID)
		c.Set("roles", []string{"developer"})

		projectID := float64(otherProject.ID)
		reqBody := map[string]interface{}{
			"title":      "新需求",
			"description": "这是一个新需求",
			"status":     "pending",
			"priority":   "high",
			"project_id": projectID,
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/requirements", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateRequirement(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		// 应该返回403或code不为200
		assert.True(t, w.Code == http.StatusForbidden || (response["code"] != nil && response["code"] != float64(200)))
	})

	t.Run("创建需求失败-缺少必填字段", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"description": "只有描述",
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/requirements", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateRequirement(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusBadRequest || (response["code"] != nil && response["code"] != float64(200)))
	})

	_ = project
	_ = user
}

func TestRequirementHandler_UpdateRequirement(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "更新需求项目")
	user := CreateTestUser(t, db, "updatereq", "更新需求用户")

	requirement := &model.Requirement{
		Title:     "更新需求",
		ProjectID: project.ID,
		CreatorID: user.ID,
		Status:    "pending",
	}
	db.Create(&requirement)

	handler := api.NewRequirementHandler(db)

	t.Run("更新需求成功", func(t *testing.T) {
		// 添加用户到项目（作为项目成员）
		AddUserToProject(t, db, user.ID, project.ID, "member")

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", user.ID)
		c.Set("roles", []string{"developer"})

		reqBody := map[string]interface{}{
			"title":     "已更新需求",
			"status":    "active", // 使用有效的状态值
			"priority":  "medium",
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/requirements/%d", requirement.ID), bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: fmt.Sprintf("%d", requirement.ID)}}

		handler.UpdateRequirement(c)

		assert.Equal(t, http.StatusOK, w.Code)

		// 验证需求已更新
		var updatedRequirement model.Requirement
		err := db.First(&updatedRequirement, requirement.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "已更新需求", updatedRequirement.Title)
		assert.Equal(t, "active", updatedRequirement.Status)
	})

	t.Run("更新不存在的需求", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"title": "不存在的需求",
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/requirements/999", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: "999"}}

		handler.UpdateRequirement(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusNotFound || (response["code"] != nil && response["code"] != float64(200)))
	})
}

func TestRequirementHandler_UpdateRequirementStatus(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "更新需求状态项目")
	user := CreateTestUser(t, db, "updatereqstatus", "更新需求状态用户")

	requirement := &model.Requirement{
		Title:     "更新状态需求",
		ProjectID: project.ID,
		CreatorID: user.ID,
		Status:    "pending",
	}
	db.Create(&requirement)

	handler := api.NewRequirementHandler(db)

	t.Run("更新需求状态成功", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"status": "in_progress",
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/requirements/1/status", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

		handler.UpdateRequirementStatus(c)

		assert.Equal(t, http.StatusOK, w.Code)

		// 验证状态已更新
		var updatedRequirement model.Requirement
		err := db.First(&updatedRequirement, requirement.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "in_progress", updatedRequirement.Status)
	})

	t.Run("更新需求状态失败-无效状态", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"status": "invalid_status",
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/requirements/1/status", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

		handler.UpdateRequirementStatus(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, response["code"] != nil && response["code"] != float64(200))
	})
}

func TestRequirementHandler_DeleteRequirement(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "删除需求项目")
	user := CreateTestUser(t, db, "deletereq", "删除需求用户")

	requirement := &model.Requirement{
		Title:     "删除需求",
		ProjectID: project.ID,
		CreatorID: user.ID,
		Status:    "pending",
	}
	db.Create(&requirement)

	handler := api.NewRequirementHandler(db)

	t.Run("删除需求成功", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/api/requirements/1", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

		handler.DeleteRequirement(c)

		assert.Equal(t, http.StatusOK, w.Code)

		// 验证需求已软删除
		var deletedRequirement model.Requirement
		err := db.First(&deletedRequirement, requirement.ID).Error
		assert.Error(t, err) // 应该找不到（软删除）

		// 验证软删除后仍可通过Unscoped查询
		err = db.Unscoped().First(&deletedRequirement, requirement.ID).Error
		assert.NoError(t, err)
		assert.NotNil(t, deletedRequirement.DeletedAt)
	})
}

