package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // 纯Go SQLite驱动，支持静态编译，必须在 gorm.io/driver/sqlite 之前导入
	
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"prjflow/internal/config"
)

// gormLogrusWriter 实现GORM的logger.Writer接口，将GORM日志输出到logrus
type gormLogrusWriter struct {
	logger *logrus.Logger
}

// Printf 实现logger.Writer接口
func (w *gormLogrusWriter) Printf(format string, args ...interface{}) {
	if w.logger != nil {
		// 将GORM的日志格式转换为logrus日志
		message := fmt.Sprintf(format, args...)
		w.logger.Info(message)
	}
}

func InitDB() (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch config.AppConfig.Database.Type {
	case "sqlite":
		// 使用纯Go实现的SQLite驱动（modernc.org/sqlite）
		// 支持静态编译（CGO_ENABLED=0），无需CGO和系统库
		// modernc.org/sqlite 注册为 "sqlite" 驱动（不是 "sqlite3"）
		// 使用 sqlite.New() 并指定 DriverName 为 "sqlite"
		// 添加 busy_timeout 参数，避免数据库锁定（5秒超时）
		dsn := config.AppConfig.Database.DSN
		if !strings.Contains(dsn, "?") {
			dsn += "?_busy_timeout=5000&_journal_mode=WAL"
		} else if !strings.Contains(dsn, "_busy_timeout") {
			dsn += "&_busy_timeout=5000&_journal_mode=WAL"
		}
		dialector = sqlite.New(sqlite.Config{
			DriverName: "sqlite", // 使用 modernc.org/sqlite 注册的驱动名
			DSN:        dsn,
		})
	case "mysql":
		dsn := config.AppConfig.Database.DSN
		if dsn == "" {
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				config.AppConfig.Database.User,
				config.AppConfig.Database.Password,
				config.AppConfig.Database.Host,
				config.AppConfig.Database.Port,
				config.AppConfig.Database.DBName,
			)
		}
		dialector = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.AppConfig.Database.Type)
	}

	// 配置GORM logger
	var gormLogger logger.Interface
	if Logger != nil {
		// 创建logrus适配器，实现GORM的logger.Writer接口
		gormWriter := &gormLogrusWriter{logger: Logger}
		gormLogger = logger.New(
			gormWriter,
			logger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logger.Info,
				Colorful:      false,
			},
		)
	} else {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	return db, nil
}

// InitAuditDB 初始化审计日志数据库（默认使用独立的审计数据库）
// 如果配置了 audit_database，使用配置的值；否则根据主数据库类型自动创建默认的独立数据库
func InitAuditDB() (*gorm.DB, error) {
	auditDBConfig := config.AppConfig.AuditDatabase
	mainDBConfig := config.AppConfig.Database

	var dialector gorm.Dialector

	// 确定数据库类型（默认使用主数据库类型）
	dbType := auditDBConfig.Type
	if dbType == "" {
		dbType = mainDBConfig.Type
	}

	switch dbType {
	case "sqlite":
		dsn := auditDBConfig.DSN
		if dsn == "" {
			// 如果未配置 DSN，使用与主数据库同一目录下的 audit.db
			mainDSN := mainDBConfig.DSN
			if mainDSN != "" {
				// 获取主数据库的目录
				mainDir := filepath.Dir(mainDSN)
				// 如果主数据库路径是相对路径，需要处理
				if !filepath.IsAbs(mainDSN) {
					// 尝试从当前工作目录或可执行文件目录查找
					if _, err := os.Stat(mainDSN); os.IsNotExist(err) {
						// 尝试从可执行文件目录查找
						exePath, err := os.Executable()
						if err == nil {
							exeDir := filepath.Dir(exePath)
							absPath := filepath.Join(exeDir, mainDSN)
							if _, err := os.Stat(absPath); err == nil {
								mainDSN = absPath
							}
						}
					}
					// 获取绝对路径
					if absPath, err := filepath.Abs(mainDSN); err == nil {
						mainDSN = absPath
					}
				}
				mainDir = filepath.Dir(mainDSN)
				dsn = filepath.Join(mainDir, "audit.db")
			} else {
				// 如果主数据库也没有配置，使用默认的 audit.db
				dsn = "audit.db"
			}
		}
		if !strings.Contains(dsn, "?") {
			dsn += "?_busy_timeout=5000&_journal_mode=WAL"
		} else if !strings.Contains(dsn, "_busy_timeout") {
			dsn += "&_busy_timeout=5000&_journal_mode=WAL"
		}
		dialector = sqlite.New(sqlite.Config{
			DriverName: "sqlite",
			DSN:        dsn,
		})
	case "mysql":
		dsn := auditDBConfig.DSN
		if dsn == "" {
			// 确定数据库名
			dbName := auditDBConfig.DBName
			if dbName == "" {
				if mainDBConfig.DBName != "" {
					dbName = mainDBConfig.DBName + "_audit" // 默认在主数据库名后加 _audit
				} else {
					dbName = "audit" // 如果主数据库也没有名称，使用 "audit"
				}
			}
			
			// 确定连接信息（优先使用审计数据库配置，否则使用主数据库配置）
			user := auditDBConfig.User
			if user == "" {
				user = mainDBConfig.User
			}
			password := auditDBConfig.Password
			if password == "" {
				password = mainDBConfig.Password
			}
			host := auditDBConfig.Host
			if host == "" {
				host = mainDBConfig.Host
				if host == "" {
					host = "localhost"
				}
			}
			port := auditDBConfig.Port
			if port == 0 {
				port = mainDBConfig.Port
				if port == 0 {
					port = 3306
				}
			}
			
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				user,
				password,
				host,
				port,
				dbName,
			)
		}
		dialector = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported audit database type: %s", dbType)
	}

	// 配置GORM logger
	var gormLogger logger.Interface
	if Logger != nil {
		gormWriter := &gormLogrusWriter{logger: Logger}
		gormLogger = logger.New(
			gormWriter,
			logger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logger.Info,
				Colorful:      false,
			},
		)
	} else {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect audit database: %w", err)
	}

	return db, nil
}

// IsUniqueConstraintError 检查是否是唯一约束错误
func IsUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "UNIQUE constraint failed") ||
		strings.Contains(errStr, "Duplicate entry") ||
		strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "UNIQUE constraint")
}

// IsUniqueConstraintOnField 检查是否是特定字段的唯一约束错误
func IsUniqueConstraintOnField(err error, fieldName string) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// SQLite: UNIQUE constraint failed: modules.name
	// MySQL: Duplicate entry 'xxx' for key 'modules.name'
	return strings.Contains(errStr, fieldName) && IsUniqueConstraintError(err)
}

// IsRecordNotFound 检查是否是记录不存在错误
func IsRecordNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err == gorm.ErrRecordNotFound
}
