package unit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prjflow/internal/api"
	"prjflow/internal/model"
)

func TestResourceAllocationHandler_GetResourceAllocations(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "资源分配测试项目")
	user := CreateTestUser(t, db, "allocuser", "资源分配用户")

	// 创建资源
	resource := &model.Resource{
		UserID:    user.ID,
		ProjectID: project.ID,
		Role:      "developer",
	}
	db.Create(&resource)

	// 创建资源分配
	projectID := project.ID
	allocation1 := &model.ResourceAllocation{
		ResourceID: resource.ID,
		ProjectID:  &projectID,
		Date:       time.Now(),
		Hours:      8.0,
	}
	db.Create(allocation1)

	handler := api.NewResourceAllocationHandler(db)

	t.Run("获取所有资源分配", func(t *testing.T) {
		// 添加用户到项目（作为项目成员）
		AddUserToProject(t, db, user.ID, project.ID, "member")

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", user.ID)
		c.Set("roles", []string{"developer"})
		c.Request = httptest.NewRequest(http.MethodGet, "/api/resource-allocations", nil)

		handler.GetResourceAllocations(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.GreaterOrEqual(t, len(list), 1)
	})

	t.Run("按资源筛选", func(t *testing.T) {
		// 添加用户到项目（作为项目成员）
		AddUserToProject(t, db, user.ID, project.ID, "member")

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", user.ID)
		c.Set("roles", []string{"developer"})
		c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/resource-allocations?resource_id=%d", resource.ID), nil)

		handler.GetResourceAllocations(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.GreaterOrEqual(t, len(list), 1)
	})
}

func TestResourceAllocationHandler_GetResourceAllocation(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "资源分配详情项目")
	user := CreateTestUser(t, db, "allocdetail", "资源分配详情用户")

	resource := &model.Resource{
		UserID:    user.ID,
		ProjectID: project.ID,
		Role:      "developer",
	}
	db.Create(&resource)

	projectID := project.ID
	allocation := &model.ResourceAllocation{
		ResourceID: resource.ID,
		ProjectID:  &projectID,
		Date:       time.Now(),
		Hours:      8.0,
	}
	db.Create(&allocation)

	handler := api.NewResourceAllocationHandler(db)

	t.Run("获取存在的资源分配", func(t *testing.T) {
		// 添加用户到项目（作为项目成员）
		AddUserToProject(t, db, user.ID, project.ID, "member")

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", user.ID)
		c.Set("roles", []string{"developer"})
		c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/resource-allocations/%d", allocation.ID), nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: fmt.Sprintf("%d", allocation.ID)}}

		handler.GetResourceAllocation(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, 8.0, data["hours"])
	})

	t.Run("获取不存在的资源分配", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/resource-allocations/999", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "999"}}

		handler.GetResourceAllocation(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusNotFound || (response["code"] != nil && response["code"] != float64(200)))
	})
}

func TestResourceAllocationHandler_CreateResourceAllocation(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "创建资源分配项目")
	user := CreateTestUser(t, db, "createalloc", "创建资源分配用户")

	resource := &model.Resource{
		UserID:    user.ID,
		ProjectID: project.ID,
		Role:      "developer",
	}
	db.Create(&resource)

	handler := api.NewResourceAllocationHandler(db)

	t.Run("创建资源分配成功", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"resource_id": resource.ID,
			"project_id":  project.ID,
			"date":        time.Now().Format("2006-01-02"),
			"hours":       8.0,
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/resource-allocations", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateResourceAllocation(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		// 验证资源分配已创建
		var allocation model.ResourceAllocation
		err = db.Where("resource_id = ?", resource.ID).First(&allocation).Error
		assert.NoError(t, err)
		assert.Equal(t, 8.0, allocation.Hours)
	})

	t.Run("创建资源分配失败-缺少必填字段", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"hours": 8.0,
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/resource-allocations", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateResourceAllocation(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusBadRequest || (response["code"] != nil && response["code"] != float64(200)))
	})
}

func TestResourceAllocationHandler_UpdateResourceAllocation(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "更新资源分配项目")
	user := CreateTestUser(t, db, "updatealloc", "更新资源分配用户")

	resource := &model.Resource{
		UserID:    user.ID,
		ProjectID: project.ID,
		Role:      "developer",
	}
	db.Create(&resource)

	projectID := project.ID
	allocation := &model.ResourceAllocation{
		ResourceID: resource.ID,
		ProjectID:  &projectID,
		Date:       time.Now(),
		Hours:      8.0,
	}
	db.Create(&allocation)

	handler := api.NewResourceAllocationHandler(db)

	t.Run("更新资源分配成功", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"hours": 6.0,
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/resource-allocations/1", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

		handler.UpdateResourceAllocation(c)

		assert.Equal(t, http.StatusOK, w.Code)

		// 验证资源分配已更新
		var updatedAllocation model.ResourceAllocation
		err := db.First(&updatedAllocation, allocation.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, 6.0, updatedAllocation.Hours)
	})

	t.Run("更新不存在的资源分配", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"hours": 6.0,
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/resource-allocations/999", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: "999"}}

		handler.UpdateResourceAllocation(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusNotFound || (response["code"] != nil && response["code"] != float64(200)))
	})
}

func TestResourceAllocationHandler_DeleteResourceAllocation(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "删除资源分配项目")
	user := CreateTestUser(t, db, "deletealloc", "删除资源分配用户")

	resource := &model.Resource{
		UserID:    user.ID,
		ProjectID: project.ID,
		Role:      "developer",
	}
	db.Create(&resource)

	projectID := project.ID
	allocation := &model.ResourceAllocation{
		ResourceID: resource.ID,
		ProjectID:  &projectID,
		Date:       time.Now(),
		Hours:      8.0,
	}
	db.Create(&allocation)

	handler := api.NewResourceAllocationHandler(db)

	t.Run("删除资源分配成功", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/api/resource-allocations/1", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

		handler.DeleteResourceAllocation(c)

		assert.Equal(t, http.StatusOK, w.Code)

		// 验证资源分配已软删除
		var deletedAllocation model.ResourceAllocation
		err := db.First(&deletedAllocation, allocation.ID).Error
		assert.Error(t, err) // 应该找不到（软删除）

		// 验证软删除后仍可通过Unscoped查询
		err = db.Unscoped().First(&deletedAllocation, allocation.ID).Error
		assert.NoError(t, err)
		assert.NotNil(t, deletedAllocation.DeletedAt)
	})
}

func TestResourceAllocationHandler_GetResourceCalendar(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "资源日历项目")
	user := CreateTestUser(t, db, "caluser", "日历用户")
	resource := &model.Resource{
		UserID:    user.ID,
		ProjectID: project.ID,
		Role:      "developer",
	}
	db.Create(resource)

	// 创建资源分配
	allocation := &model.ResourceAllocation{
		ResourceID: resource.ID,
		Date:       time.Now(),
		Hours:      8.0,
		ProjectID:  &project.ID,
	}
	db.Create(allocation)

	handler := api.NewResourceAllocationHandler(db)

	t.Run("获取资源日历成功", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/resource-allocations/calendar", nil)

		handler.GetResourceCalendar(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"]
		assert.NotNil(t, data)
	})

	t.Run("获取资源日历-按用户筛选", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/resource-allocations/calendar?user_id=%d", user.ID), nil)

		handler.GetResourceCalendar(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])
	})
}

func TestResourceAllocationHandler_CheckResourceConflict(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "资源冲突项目")
	user := CreateTestUser(t, db, "conflictuser", "冲突用户")
	resource := &model.Resource{
		UserID:    user.ID,
		ProjectID: project.ID,
		Role:      "developer",
	}
	db.Create(resource)

	handler := api.NewResourceAllocationHandler(db)

	t.Run("检查资源冲突成功-无冲突", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		date := time.Now().Format("2006-01-02")
		c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/resource-allocations/check-conflict?resource_id=%d&date=%s", resource.ID, date), nil)

		handler.CheckResourceConflict(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		assert.NotNil(t, data["total_hours"])
		assert.NotNil(t, data["has_conflict"])
	})

	t.Run("检查资源冲突失败-缺少参数", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/resource-allocations/check-conflict", nil)

		handler.CheckResourceConflict(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusBadRequest || (response["code"] != nil && response["code"] != float64(200)))
	})
}

// TestResourceAllocationHandler_GetResourceAllocations_WithUserIDAndDateRange 测试带user_id和日期范围的查询
func TestResourceAllocationHandler_GetResourceAllocations_WithUserIDAndDateRange(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	project := CreateTestProject(t, db, "日期范围测试项目")
	user := CreateTestUser(t, db, "daterangeuser", "日期范围用户")

	// 添加用户到项目（作为项目成员）
	AddUserToProject(t, db, user.ID, project.ID, "member")

	// 创建资源
	resource := &model.Resource{
		UserID:    user.ID,
		ProjectID: project.ID,
		Role:      "developer",
	}
	db.Create(&resource)

	// 创建资源分配（本周的数据）
	now := time.Now()
	weekStart := now
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	projectID := project.ID
	allocation1 := &model.ResourceAllocation{
		ResourceID: resource.ID,
		ProjectID:  &projectID,
		Date:       weekStart,
		Hours:      8.0,
		Description: "周一工作",
	}
	db.Create(allocation1)

	allocation2 := &model.ResourceAllocation{
		ResourceID: resource.ID,
		ProjectID:  &projectID,
		Date:       weekStart.AddDate(0, 0, 1),
		Hours:      6.0,
		Description: "周二工作",
	}
	db.Create(allocation2)

	handler := api.NewResourceAllocationHandler(db)

	t.Run("按user_id和日期范围查询", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", user.ID)
		c.Set("roles", []string{"developer"})

		// 构建查询URL，模拟前端请求
		startDate := weekStart.Format("2006-01-02")
		endDate := weekStart.AddDate(0, 0, 6).Format("2006-01-02")
		url := fmt.Sprintf("/api/resource-allocations?page=1&size=20&start_date=%s&end_date=%s&user_id=%d",
			startDate, endDate, user.ID)
		c.Request = httptest.NewRequest(http.MethodGet, url, nil)

		handler.GetResourceAllocations(c)

		// 打印响应以便调试
		if w.Code != http.StatusOK {
			t.Logf("Response code: %d", w.Code)
			t.Logf("Response body: %s", w.Body.String())
		}

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Response should be valid JSON")

		if response["code"] != float64(200) {
			t.Logf("Error response: %+v", response)
		}
		if response["code"] != float64(200) {
			t.Fatalf("Expected code 200, got %v. Message: %v", response["code"], response["message"])
		}
		assert.Equal(t, float64(200), response["code"], "Response code should be 200")

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Response data is not a map: %+v", response)
		}
		list := data["list"].([]interface{})
		total := data["total"].(float64)

		t.Logf("Found %d allocations, total: %.0f", len(list), total)
		assert.GreaterOrEqual(t, len(list), 2, "Should find at least 2 allocations")
		assert.GreaterOrEqual(t, total, float64(2), "Total should be at least 2")
	})

	t.Run("按user_id和日期范围查询-管理员", func(t *testing.T) {
		// 创建管理员用户
		adminUser := CreateTestAdminUser(t, db, "adminuser", "管理员用户")

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", adminUser.ID)
		c.Set("roles", []string{"admin"})

		startDate := weekStart.Format("2006-01-02")
		endDate := weekStart.AddDate(0, 0, 6).Format("2006-01-02")
		url := fmt.Sprintf("/api/resource-allocations?page=1&size=20&start_date=%s&end_date=%s&user_id=%d",
			startDate, endDate, user.ID)
		c.Request = httptest.NewRequest(http.MethodGet, url, nil)

		handler.GetResourceAllocations(c)

		if w.Code != http.StatusOK {
			t.Logf("Response code: %d", w.Code)
			t.Logf("Response body: %s", w.Body.String())
		}

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})
		assert.GreaterOrEqual(t, len(list), 2, "Admin should see all allocations")
	})
}

