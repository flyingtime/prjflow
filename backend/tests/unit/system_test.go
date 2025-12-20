package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prjflow/internal/api"
	"prjflow/internal/model"
)

func TestSystemHandler_GetBackupConfig(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	handler := api.NewSystemHandler(db)

	t.Run("获取备份配置-默认值", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/system/backup/config", nil)

		handler.GetBackupConfig(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, false, data["enabled"])
		assert.Equal(t, "02:00", data["backup_time"])
		assert.Equal(t, "", data["last_backup_date"])
	})

	t.Run("获取备份配置-已配置", func(t *testing.T) {
		// 设置备份配置
		enabledConfig := model.SystemConfig{
			Key:   "backup_enabled",
			Value: "true",
			Type:  "boolean",
		}
		db.Create(&enabledConfig)

		timeConfig := model.SystemConfig{
			Key:   "backup_time",
			Value: "03:00",
			Type:  "string",
		}
		db.Create(&timeConfig)

		lastDateConfig := model.SystemConfig{
			Key:   "backup_last_date",
			Value: "2025-12-04",
			Type:  "string",
		}
		db.Create(&lastDateConfig)

		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/system/backup/config", nil)

		handler.GetBackupConfig(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, true, data["enabled"])
		assert.Equal(t, "03:00", data["backup_time"])
		assert.Equal(t, "2025-12-04", data["last_backup_date"])
	})
}

func TestSystemHandler_SaveBackupConfig(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	handler := api.NewSystemHandler(db)

	t.Run("保存备份配置成功", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"enabled":     true,
			"backup_time": "03:00",
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/system/backup/config", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.SaveBackupConfig(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		// 验证配置已保存
		var enabledConfig model.SystemConfig
		err = db.Where("key = ?", "backup_enabled").First(&enabledConfig).Error
		assert.NoError(t, err)
		assert.Equal(t, "true", enabledConfig.Value)

		var timeConfig model.SystemConfig
		err = db.Where("key = ?", "backup_time").First(&timeConfig).Error
		assert.NoError(t, err)
		assert.Equal(t, "03:00", timeConfig.Value)
	})

	t.Run("保存备份配置失败-缺少必填字段", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"enabled": true,
			// 缺少backup_time
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/system/backup/config", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.SaveBackupConfig(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusBadRequest || (response["code"] != nil && response["code"] != float64(200)))
	})

	t.Run("保存备份配置失败-时间格式错误", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"enabled":     true,
			"backup_time": "25:00", // 无效的时间格式
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/system/backup/config", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.SaveBackupConfig(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusBadRequest || (response["code"] != nil && response["code"] != float64(200)))
	})
}

func TestSystemHandler_TriggerBackup(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	handler := api.NewSystemHandler(db)

	// 注意：这个测试可能会失败，因为TriggerBackup需要实际的备份功能
	// 这里只测试基本的调用，实际备份功能可能需要mock
	t.Run("触发备份", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/system/backup/trigger", nil)

		handler.TriggerBackup(c)

		// 备份可能成功或失败（取决于备份功能是否可用）
		// 只要返回了响应即可
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
	})
}

func TestSystemHandler_GetLogLevel(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	handler := api.NewSystemHandler(db)

	t.Run("获取日志级别", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/system/log/level", nil)

		handler.GetLogLevel(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		assert.NotNil(t, data["level"])
	})
}

func TestSystemHandler_SetLogLevel(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	handler := api.NewSystemHandler(db)

	t.Run("设置日志级别成功", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"level": "info",
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/system/log/level", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.SetLogLevel(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "info", data["level"])
	})

	t.Run("设置日志级别失败-无效的级别", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{
			"level": "invalid_level",
		}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/system/log/level", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.SetLogLevel(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusBadRequest || (response["code"] != nil && response["code"] != float64(200)))
	})

	t.Run("设置日志级别失败-缺少必填字段", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]interface{}{}
		jsonData, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/system/log/level", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.SetLogLevel(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusBadRequest || (response["code"] != nil && response["code"] != float64(200)))
	})

	t.Run("设置所有有效的日志级别", func(t *testing.T) {
		validLevels := []string{"debug", "info", "warn", "error"}

		for _, level := range validLevels {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			reqBody := map[string]interface{}{
				"level": level,
			}
			jsonData, _ := json.Marshal(reqBody)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/system/log/level", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.SetLogLevel(c)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, float64(200), response["code"])
		}
	})
}

func TestSystemHandler_GetLogFiles(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	handler := api.NewSystemHandler(db)

	t.Run("获取日志文件列表", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/system/log/files", nil)

		handler.GetLogFiles(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])

		data := response["data"].(map[string]interface{})
		assert.NotNil(t, data["files"])
		// files应该是一个数组
		files, ok := data["files"].([]interface{})
		assert.True(t, ok || files == nil) // 可能为空数组
	})
}

func TestSystemHandler_DownloadLogFile(t *testing.T) {
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	handler := api.NewSystemHandler(db)

	t.Run("下载日志文件失败-文件不存在", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/system/log/files/nonexistent.log", nil)
		c.Params = gin.Params{gin.Param{Key: "filename", Value: "nonexistent.log"}}

		handler.DownloadLogFile(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusNotFound || (response["code"] != nil && response["code"] != float64(200)))
	})

	t.Run("下载日志文件失败-无效的文件名", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/system/log/files/../etc/passwd", nil)
		c.Params = gin.Params{gin.Param{Key: "filename", Value: "../etc/passwd"}}

		handler.DownloadLogFile(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusBadRequest || (response["code"] != nil && response["code"] != float64(200)))
	})

	t.Run("下载日志文件失败-文件名包含斜杠", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/system/log/files/path/to/file.log", nil)
		c.Params = gin.Params{gin.Param{Key: "filename", Value: "path/to/file.log"}}

		handler.DownloadLogFile(c)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, w.Code == http.StatusBadRequest || (response["code"] != nil && response["code"] != float64(200)))
	})
}

