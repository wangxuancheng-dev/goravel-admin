package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/support/path"

	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/utils"
)

type ClearChunks struct {
}

func (r *ClearChunks) Signature() string {
	return "app:clear-chunks"
}

func (r *ClearChunks) Description() string {
	return "清理3天前的分片文件（tenancy 开启时按租户目录执行）"
}

func (r *ClearChunks) Extend() command.Extend {
	return command.Extend{
		Category: "app",
		Flags:    []command.Flag{TenantScopeFlag()},
	}
}

func (r *ClearChunks) Handle(ctx console.Context) error {
	threeDaysAgo := time.Now().AddDate(0, 0, -3)

	return RunTenantScoped(ctx, func(_ *models.Tenant, bound context.Context) error {
		disk := utils.GetConfigValue(bound, "storage", "file_disk", "")
		if disk == "" {
			disk = "local"
		}
		if disk != "local" && disk != "public" {
			ctx.Info(fmt.Sprintf("当前存储驱动为 %s，清理分片文件功能仅支持本地存储，跳过清理", disk))
			return nil
		}

		storage, err := utils.StorageDisk(disk)
		if err != nil {
			return fmt.Errorf("打开存储驱动失败: %w", err)
		}

		var storageRoot string
		if disk == "public" {
			storageRoot = path.Storage("app/public")
		} else {
			storageRoot = path.Storage("app")
		}

		prefix := helpers.TenantStoragePrefix(bound)
		relChunks := strings.TrimSuffix(prefix, "/") + "/chunks"
		if prefix == "" {
			relChunks = "chunks"
		}
		relChunks = strings.TrimPrefix(relChunks, "/")
		chunksDir := filepath.Join(storageRoot, filepath.FromSlash(relChunks))

		ctx.Info(fmt.Sprintf("开始清理3天前的分片文件（%s，目录 %s）...", threeDaysAgo.Format(utils.DateTimeFormat), relChunks))

		if _, err := os.Stat(chunksDir); os.IsNotExist(err) {
			ctx.Info("分片目录不存在，无需清理")
			return nil
		}

		cleanedCount := 0
		cleanedSize := int64(0)
		errorCount := 0

		walkErr := filepath.Walk(chunksDir, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				ctx.Info(fmt.Sprintf("无法访问路径 %s: %v", p, err))
				errorCount++
				return nil
			}
			if p == chunksDir {
				return nil
			}
			if !info.IsDir() {
				if info.Name() == ".gitignore" {
					return nil
				}
				if info.ModTime().Before(threeDaysAgo) {
					relativePath := strings.TrimPrefix(p, storageRoot+string(filepath.Separator))
					relativePath = strings.ReplaceAll(relativePath, string(filepath.Separator), "/")
					if err := storage.Delete(relativePath); err != nil {
						ctx.Info(fmt.Sprintf("删除分片文件失败 %s: %v", relativePath, err))
						errorCount++
					} else {
						cleanedCount++
						cleanedSize += info.Size()
					}
				}
			}
			return nil
		})
		if walkErr != nil {
			return fmt.Errorf("遍历分片目录失败: %w", walkErr)
		}

		emptyDirCount := r.cleanupEmptyDirs(chunksDir)
		ctx.Info(fmt.Sprintf("分片文件清理完成！已清理 %d 个文件，释放空间 %s，删除 %d 个空目录，错误数: %d",
			cleanedCount, r.formatFileSize(cleanedSize), emptyDirCount, errorCount))
		return nil
	})
}

func (r *ClearChunks) cleanupEmptyDirs(rootDir string) int {
	var dirs []string
	_ = filepath.Walk(rootDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && p != rootDir {
			dirs = append(dirs, p)
		}
		return nil
	})

	removedCount := 0
	for i := len(dirs) - 1; i >= 0; i-- {
		dir := dirs[i]
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		if len(entries) == 0 {
			if err := os.Remove(dir); err == nil {
				removedCount++
			}
		}
	}
	return removedCount
}

func (r *ClearChunks) formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
