package unit

import (
	"testing"
	"time"

	"prjflow/internal/api"
	"prjflow/internal/model"

	"github.com/stretchr/testify/assert"
)

// TestGenerateUniqueUsername_OpenIDWithDate 测试使用OpenID后8位+日期生成用户名
func TestGenerateUniqueUsername_OpenIDWithDate(t *testing.T) {
	// 使用测试辅助函数创建数据库
	db := SetupTestDB(t)
	defer TeardownTestDB(t, db)

	t.Run("空昵称时使用OpenID后8位+日期", func(t *testing.T) {
		openID := "oUpF8uMuAJO_M2pxb1Q9zNjWeS6o" // 28位OpenID
		username := api.GenerateUniqueUsername(db, "", openID)

		// 验证格式：{OpenID后8位}_{YYYYMMDD}
		// OpenID后8位应该是：zNjWeS6o（从后往前数8位）
		expectedSuffix := "zNjWeS6o"
		expectedDate := time.Now().Format("20060102")
		expectedUsername := expectedSuffix + "_" + expectedDate

		assert.Equal(t, expectedUsername, username, "用户名应该是OpenID后8位+日期格式")
		assert.LessOrEqual(t, len(username), 50, "用户名长度应该小于等于50字符")
	})

	t.Run("OpenID长度不足8位时补0", func(t *testing.T) {
		openID := "abc" // 只有3位
		username := api.GenerateUniqueUsername(db, "", openID)

		// 应该补0到8位：00000abc
		expectedSuffix := "00000abc"
		expectedDate := time.Now().Format("20060102")
		expectedUsername := expectedSuffix + "_" + expectedDate

		assert.Equal(t, expectedUsername, username, "OpenID长度不足8位时应该前面补0")
	})

	t.Run("OpenID长度正好8位", func(t *testing.T) {
		openID := "12345678" // 正好8位
		username := api.GenerateUniqueUsername(db, "", openID)

		expectedSuffix := "12345678"
		expectedDate := time.Now().Format("20060102")
		expectedUsername := expectedSuffix + "_" + expectedDate

		assert.Equal(t, expectedUsername, username, "OpenID正好8位时应该直接使用")
	})

	t.Run("用户名冲突时不添加后缀", func(t *testing.T) {
		openID := "oUpF8uMuAJO_M2pxb1Q9zNjWeS6o"
		expectedSuffix := "zNjWeS6o"
		expectedDate := time.Now().Format("20060102")
		baseUsername := expectedSuffix + "_" + expectedDate

		// 先创建一个用户使用这个用户名
		existingUser := model.User{
			Username: baseUsername,
			Nickname: "测试用户",
			Status:   1,
		}
		db.Create(&existingUser)

		// 再次生成用户名，应该仍然返回相同的格式（不检查冲突，由创建时处理）
		username := api.GenerateUniqueUsername(db, "", openID)

		// 应该仍然是 baseUsername（不添加后缀，冲突由创建时处理）
		assert.Equal(t, baseUsername, username, "用户名冲突时不添加后缀，直接返回格式化的用户名")
	})

	t.Run("有昵称时仍然使用OpenID后8位+日期", func(t *testing.T) {
		// 使用不同的OpenID，避免与之前的测试冲突
		openID := "oUpF8uMuAJO_M2pxb1Q9zNjWeS7p" // 不同的OpenID
		nickname := "张三" // 有昵称
		username := api.GenerateUniqueUsername(db, nickname, openID)

		// 即使有昵称，也应该使用OpenID后8位+日期
		expectedSuffix := "zNjWeS7p" // 新OpenID的后8位
		expectedDate := time.Now().Format("20060102")
		expectedUsername := expectedSuffix + "_" + expectedDate

		assert.Equal(t, expectedUsername, username, "即使有昵称，也应该使用OpenID后8位+日期格式")
	})
}

