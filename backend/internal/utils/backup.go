package utils

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"project-management/internal/config"
	"gorm.io/gorm"
)

// BackupDatabase 备份数据库（改进版，使用 VACUUM INTO 确保数据一致性）
// 同时备份审计数据库，并在备份前清理过期的审计日志
func BackupDatabase(db *gorm.DB) error {
	// 加载配置
	if err := config.LoadConfig(""); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 检查数据库类型，只支持 SQLite 备份
	if config.AppConfig.Database.Type != "sqlite" {
		return fmt.Errorf("backup only supports SQLite database, current type: %s", config.AppConfig.Database.Type)
	}

	// 备份前清理过期的审计日志（保留30天）
	// 使用审计日志数据库（如果配置了）或主数据库
	auditDB := db
	if AuditDB != nil {
		auditDB = AuditDB
	}
	if Logger != nil {
		Logger.Info("[Backup] Cleaning up old audit logs before backup (retention: 30 days)")
	}
	if err := CleanupOldAuditLogs(auditDB, 30); err != nil {
		if Logger != nil {
			Logger.Warnf("[Backup] Warning: Failed to cleanup old audit logs: %v", err)
		}
		// 清理失败不影响备份，继续执行
	}

	// 获取数据库文件路径
	dbPath := config.AppConfig.Database.DSN
	if !filepath.IsAbs(dbPath) {
		// 相对路径，从当前工作目录或可执行文件目录查找
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			// 尝试从可执行文件目录查找
			exePath, err := os.Executable()
			if err == nil {
				exeDir := filepath.Dir(exePath)
				absPath := filepath.Join(exeDir, dbPath)
				if _, err := os.Stat(absPath); err == nil {
					dbPath = absPath
				}
			}
		}
	}

	// 检查数据库文件是否存在
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database file not found: %s", dbPath)
	}

	// 创建备份目录（在数据库文件同目录下）
	dbDir := filepath.Dir(dbPath)
	backupDir := filepath.Join(dbDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 生成备份文件名（带时间戳）
	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("data_%s.db", timestamp)
	backupPath := filepath.Join(backupDir, backupFileName)

	// 获取绝对路径（VACUUM INTO 需要绝对路径）
	absBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 转义路径中的单引号
	escapedPath := strings.ReplaceAll(absBackupPath, "'", "''")

	startTime := time.Now()
	if Logger != nil {
		Logger.Infof("[Backup] Starting database backup at %s: %s -> %s", 
			startTime.Format("2006-01-02 15:04:05"), dbPath, absBackupPath)
	}

	// 尝试使用 VACUUM INTO 命令创建备份（SQLite 3.27.0+）
	// 这会创建一个一致的数据库快照，支持在线备份
	sql := fmt.Sprintf("VACUUM INTO '%s'", escapedPath)
	if err := db.Exec(sql).Error; err != nil {
		// 如果 VACUUM INTO 失败，回退到文件复制方式（带警告）
		if Logger != nil {
			Logger.Warnf("[Backup] Warning: VACUUM INTO failed: %v, falling back to file copy", err)
		}
		if err := copyDatabaseFile(dbPath, absBackupPath); err != nil {
			return fmt.Errorf("failed to backup database: %w", err)
		}
		if Logger != nil {
			Logger.Info("[Backup] Backup created using file copy method")
		}
	} else {
		if Logger != nil {
			Logger.Info("[Backup] Backup created using VACUUM INTO method")
		}
	}

	// 验证备份文件是否存在
	if _, err := os.Stat(absBackupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file was not created: %s", absBackupPath)
	}

	// 获取文件大小
	fileInfo, err := os.Stat(absBackupPath)
	if err != nil {
		return fmt.Errorf("failed to get backup file info: %w", err)
	}

	// 验证备份文件大小（不应该为0）
	if fileInfo.Size() == 0 {
		os.Remove(absBackupPath)
		return fmt.Errorf("backup file is empty: %s", absBackupPath)
	}

	// 验证备份文件完整性（尝试打开SQLite文件）
	if err := verifyBackupFile(absBackupPath); err != nil {
		if Logger != nil {
			Logger.Warnf("[Backup] Warning: Backup file verification failed: %v", err)
		}
		// 不返回错误，但记录警告（因为某些情况下验证可能失败但不影响备份可用性）
	} else {
		if Logger != nil {
			Logger.Info("[Backup] ✓ Backup file verified successfully")
		}
	}

	fileSize := float64(fileInfo.Size())
	unit := "B"
	if fileSize >= 1024*1024 {
		fileSize /= 1024 * 1024
		unit = "MB"
	} else if fileSize >= 1024 {
		fileSize /= 1024
		unit = "KB"
	}

	duration := time.Since(startTime)
	if Logger != nil {
		Logger.Infof("[Backup] ✓ Main database backup created: %s (%.2f %s) in %v", absBackupPath, fileSize, unit, duration)
	}

	// 备份审计数据库（如果配置了独立审计数据库）
	var auditBackupPath string
	if AuditDB != nil && config.AppConfig.AuditDatabase.Type == "sqlite" {
		auditBackupPath, err = backupAuditDatabaseToFile(AuditDB, backupDir, timestamp)
		if err != nil {
			if Logger != nil {
				Logger.Warnf("[Backup] Warning: Failed to backup audit database: %v", err)
			}
			// 审计数据库备份失败不影响主数据库备份，继续执行
			auditBackupPath = "" // 标记为失败
		}
	}

	// 计算总原始大小（在删除文件前）
	totalOriginalSize := float64(fileInfo.Size())
	var auditFileSize int64
	if auditBackupPath != "" {
		if auditInfo, err := os.Stat(auditBackupPath); err == nil {
			auditFileSize = auditInfo.Size()
			totalOriginalSize += float64(auditFileSize)
		}
	}

	// 将两个数据库文件打包成一个ZIP文件
	zipFileName := fmt.Sprintf("backup_%s.zip", timestamp)
	zipPath := filepath.Join(backupDir, zipFileName)
	if err := createBackupZip(absBackupPath, auditBackupPath, zipPath); err != nil {
		if Logger != nil {
			Logger.Errorf("[Backup] Failed to create backup ZIP: %v", err)
		}
		return fmt.Errorf("failed to create backup ZIP: %w", err)
	}

	// 删除临时的.db文件
	os.Remove(absBackupPath)
	if auditBackupPath != "" {
		os.Remove(auditBackupPath)
	}

	// 获取ZIP文件大小并计算压缩比例
	zipInfo, err := os.Stat(zipPath)
	if err == nil {
		zipSize := float64(zipInfo.Size())
		zipUnit := "B"
		if zipSize >= 1024*1024 {
			zipSize /= 1024 * 1024
			zipUnit = "MB"
		} else if zipSize >= 1024 {
			zipSize /= 1024
			zipUnit = "KB"
		}
		compressionRatio := float64(zipInfo.Size()) / totalOriginalSize * 100
		if Logger != nil {
			Logger.Infof("[Backup] ✓ Backup ZIP created: %s (%.2f %s, %.1f%% of original size)", 
				zipPath, zipSize, zipUnit, compressionRatio)
		}
	}

	// 清理旧备份（保留最近7天）
	if err := cleanupOldBackups(backupDir, 7); err != nil {
		if Logger != nil {
			Logger.Warnf("[Backup] Warning: Failed to cleanup old backups: %v", err)
		}
	}

	// 显示备份统计
	backupCount, totalSize, err := getBackupStats(backupDir)
	if err == nil {
		if Logger != nil {
			Logger.Infof("[Backup] Statistics: %d backups, total size: %s", backupCount, formatSize(totalSize))
		}
	}

	totalDuration := time.Since(startTime)
	if Logger != nil {
		Logger.Infof("[Backup] ✓ Backup completed successfully in %v", totalDuration)
	}

	return nil
}

// backupAuditDatabaseToFile 备份审计数据库到文件，返回备份文件路径
func backupAuditDatabaseToFile(auditDB *gorm.DB, backupDir string, timestamp string) (string, error) {
	// 获取审计数据库文件路径
	auditDBPath := config.AppConfig.AuditDatabase.DSN
	if auditDBPath == "" {
		// 如果未配置，使用与主数据库同一目录下的 audit.db
		mainDBPath := config.AppConfig.Database.DSN
		if mainDBPath != "" {
			// 获取主数据库的目录
			mainDir := filepath.Dir(mainDBPath)
			// 如果主数据库路径是相对路径，需要处理
			if !filepath.IsAbs(mainDBPath) {
				// 尝试从当前工作目录或可执行文件目录查找
				if _, err := os.Stat(mainDBPath); os.IsNotExist(err) {
					// 尝试从可执行文件目录查找
					exePath, err := os.Executable()
					if err == nil {
						exeDir := filepath.Dir(exePath)
						absPath := filepath.Join(exeDir, mainDBPath)
						if _, err := os.Stat(absPath); err == nil {
							mainDBPath = absPath
						}
					}
				}
				// 获取绝对路径
				if absPath, err := filepath.Abs(mainDBPath); err == nil {
					mainDBPath = absPath
				}
			}
			mainDir = filepath.Dir(mainDBPath)
			auditDBPath = filepath.Join(mainDir, "audit.db")
		} else {
			// 如果主数据库也没有配置，使用默认的 audit.db
			auditDBPath = "audit.db"
		}
	}

	// 处理相对路径
	if !filepath.IsAbs(auditDBPath) {
		if _, err := os.Stat(auditDBPath); os.IsNotExist(err) {
			// 尝试从可执行文件目录查找
			exePath, err := os.Executable()
			if err == nil {
				exeDir := filepath.Dir(exePath)
				absPath := filepath.Join(exeDir, auditDBPath)
				if _, err := os.Stat(absPath); err == nil {
					auditDBPath = absPath
				}
			}
		}
		// 获取绝对路径
		if absPath, err := filepath.Abs(auditDBPath); err == nil {
			auditDBPath = absPath
		}
	}

	// 检查审计数据库文件是否存在
	if _, err := os.Stat(auditDBPath); os.IsNotExist(err) {
		if Logger != nil {
			Logger.Warnf("[Backup] Audit database file not found: %s, skipping audit backup", auditDBPath)
		}
		return "", nil // 文件不存在不算错误，返回空路径
	}

	// 生成备份文件名
	backupFileName := fmt.Sprintf("audit_%s.db", timestamp)
	backupPath := filepath.Join(backupDir, backupFileName)

	// 获取绝对路径（VACUUM INTO 需要绝对路径）
	absBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 转义路径中的单引号
	escapedPath := strings.ReplaceAll(absBackupPath, "'", "''")

	auditStartTime := time.Now()
	if Logger != nil {
		Logger.Infof("[Backup] Starting audit database backup at %s: %s -> %s",
			auditStartTime.Format("2006-01-02 15:04:05"), auditDBPath, absBackupPath)
	}

	// 尝试使用 VACUUM INTO 命令创建备份
	sql := fmt.Sprintf("VACUUM INTO '%s'", escapedPath)
	if err := auditDB.Exec(sql).Error; err != nil {
		// 如果 VACUUM INTO 失败，回退到文件复制方式
		if Logger != nil {
			Logger.Warnf("[Backup] Warning: VACUUM INTO failed for audit database: %v, falling back to file copy", err)
		}
		if err := copyDatabaseFile(auditDBPath, absBackupPath); err != nil {
			return "", fmt.Errorf("failed to backup audit database: %w", err)
		}
		if Logger != nil {
			Logger.Info("[Backup] Audit database backup created using file copy method")
		}
	} else {
		if Logger != nil {
			Logger.Info("[Backup] Audit database backup created using VACUUM INTO method")
		}
	}

	// 验证备份文件是否存在
	if _, err := os.Stat(absBackupPath); os.IsNotExist(err) {
		return "", fmt.Errorf("audit database backup file was not created: %s", absBackupPath)
	}

	// 获取文件大小
	fileInfo, err := os.Stat(absBackupPath)
	if err != nil {
		return "", fmt.Errorf("failed to get audit backup file info: %w", err)
	}

	// 验证备份文件大小
	if fileInfo.Size() == 0 {
		os.Remove(absBackupPath)
		return "", fmt.Errorf("audit database backup file is empty: %s", absBackupPath)
	}

	// 验证备份文件完整性
	if err := verifyBackupFile(absBackupPath); err != nil {
		if Logger != nil {
			Logger.Warnf("[Backup] Warning: Audit database backup file verification failed: %v", err)
		}
	}

	fileSize := float64(fileInfo.Size())
	unit := "B"
	if fileSize >= 1024*1024 {
		fileSize /= 1024 * 1024
		unit = "MB"
	} else if fileSize >= 1024 {
		fileSize /= 1024
		unit = "KB"
	}

	auditDuration := time.Since(auditStartTime)
	if Logger != nil {
		Logger.Infof("[Backup] ✓ Audit database backup created: %s (%.2f %s) in %v",
			absBackupPath, fileSize, unit, auditDuration)
	}

	return absBackupPath, nil
}

// createBackupZip 将主数据库和审计数据库备份文件打包成一个ZIP文件
func createBackupZip(mainDBPath, auditDBPath, zipPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create ZIP file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 添加主数据库文件
	if err := addFileToZip(zipWriter, mainDBPath, "data.db"); err != nil {
		return fmt.Errorf("failed to add main database to ZIP: %w", err)
	}

	// 添加审计数据库文件（如果存在）
	if auditDBPath != "" {
		if err := addFileToZip(zipWriter, auditDBPath, "audit.db"); err != nil {
			return fmt.Errorf("failed to add audit database to ZIP: %w", err)
		}
	}

	if Logger != nil {
		Logger.Info("[Backup] ✓ Backup ZIP file created successfully")
	}

	return nil
}

// addFileToZip 将文件添加到ZIP压缩包中
func addFileToZip(zipWriter *zip.Writer, filePath, zipName string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = zipName
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// verifyBackupFile 验证备份文件完整性
func verifyBackupFile(backupPath string) error {
	// 尝试打开SQLite文件，验证文件头
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// 读取SQLite文件头（前16字节）
	header := make([]byte, 16)
	if _, err := file.Read(header); err != nil {
		return fmt.Errorf("failed to read backup file header: %w", err)
	}

	// SQLite文件头应该是 "SQLite format 3\000"
	sqliteHeader := []byte("SQLite format 3\x00")
	if len(header) < len(sqliteHeader) {
		return fmt.Errorf("backup file header too short")
	}

	// 检查文件头是否匹配
	match := true
	for i := 0; i < len(sqliteHeader); i++ {
		if header[i] != sqliteHeader[i] {
			match = false
			break
		}
	}

	if !match {
		return fmt.Errorf("backup file is not a valid SQLite database")
	}

	return nil
}

// copyDatabaseFile 复制数据库文件（备选方案）
// 注意：此方法在数据库正在写入时可能复制到不一致的数据
// 建议使用 VACUUM INTO 命令，此方法仅作为回退方案
func copyDatabaseFile(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dstFile.Close()

	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		dstFile.Close()
		os.Remove(dstPath)
		return fmt.Errorf("failed to copy database file: %w", err)
	}

	if written == 0 {
		os.Remove(dstPath)
		return fmt.Errorf("backup file is empty")
	}

	// 验证复制的文件大小
	srcInfo, err := os.Stat(srcPath)
	if err == nil && srcInfo.Size() != written {
		if Logger != nil {
			Logger.Warnf("[Backup] Warning: Source file size (%d) != copied size (%d)", srcInfo.Size(), written)
		}
	}

	// 验证备份文件完整性
	if err := verifyBackupFile(dstPath); err != nil {
		if Logger != nil {
			Logger.Warnf("[Backup] Warning: Backup file verification failed after copy: %v", err)
		}
		// 不返回错误，但记录警告
	}

	return nil
}

// compressFile 压缩文件
func compressFile(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	_, err = io.Copy(gzWriter, srcFile)
	return err
}

// cleanupOldBackups 清理旧备份
func cleanupOldBackups(backupDir string, keepDays int) error {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}

	cutoffTime := time.Now().AddDate(0, 0, -keepDays)
	removedCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只处理备份文件（ZIP格式）
		if !strings.HasPrefix(entry.Name(), "backup_") || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 如果文件修改时间早于保留期限，删除
		if info.ModTime().Before(cutoffTime) {
			filePath := filepath.Join(backupDir, entry.Name())
			if err := os.Remove(filePath); err == nil {
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		if Logger != nil {
			Logger.Infof("[Backup] Cleaned up %d old backup(s) (older than %d days)", removedCount, keepDays)
		}
	}

	return nil
}

// getBackupStats 获取备份统计信息
func getBackupStats(backupDir string) (int, int64, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return 0, 0, err
	}

	count := 0
	totalSize := int64(0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 统计ZIP备份文件
		if !strings.HasPrefix(entry.Name(), "backup_") || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		count++
		totalSize += info.Size()
	}

	return count, totalSize, nil
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	unit := "B"
	fsize := float64(size)
	if fsize >= 1024*1024*1024 {
		fsize /= 1024 * 1024 * 1024
		unit = "GB"
	} else if fsize >= 1024*1024 {
		fsize /= 1024 * 1024
		unit = "MB"
	} else if fsize >= 1024 {
		fsize /= 1024
		unit = "KB"
	}
	return fmt.Sprintf("%.2f %s", fsize, unit)
}

