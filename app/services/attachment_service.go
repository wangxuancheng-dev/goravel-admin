package services

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	appfacades "goravel/app/facades"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

type AttachmentService interface {
	// GetByID 根据ID获取附件
	GetByID(id uint) (*models.Attachment, error)
	// GetByIDs 根据ID列表获取附件
	GetByIDs(ids []uint) ([]models.Attachment, error)
	// GetList 获取附件列表
	GetList(filters AttachmentFilters, page, pageSize int) ([]models.Attachment, int64, error)
	// InitChunkUpload 初始化分片上传（不再使用服务端缓存）
	InitChunkUpload(filename string, totalSize int64, chunkSize int64, totalChunks int) (string, error)

	// UploadChunk 上传分片（不再使用服务端缓存）
	UploadChunk(chunkID string, chunkIndex int, chunkData []byte) error

	// MergeChunks 合并分片（不再使用服务端缓存，需要传入 totalChunks）
	MergeChunks(chunkID string, filename string, mimeType string, totalChunks int, isPublicRaw string) (*models.Attachment, error)

	// GetChunkProgress 获取分片上传进度（不再使用服务端缓存，需要传入 totalChunks）
	GetChunkProgress(chunkID string, totalChunks int) (map[string]any, error)

	// UploadFile 普通文件上传（小文件）
	UploadFile(fileData []byte, filename string, mimeType string, isPublicRaw string) (*models.Attachment, error)

	// GetFileURL 获取文件访问URL
	GetFileURL(attachment *models.Attachment) string

	// GetFileType 根据MIME类型判断文件类型
	GetFileType(mimeType string) string

	// DeleteFile 删除文件
	DeleteFile(attachment *models.Attachment) error
	// UpdateDisplayName 更新显示名称
	UpdateDisplayName(id uint, displayName string) error
	// UpdateCategory 更新附件分类
	UpdateCategory(id uint, categoryID uint) error
	// UpdateVisibility 更新附件公开/私有状态
	UpdateVisibility(id uint, isPublic bool) error
	// AttachmentToJSON 附件列表/详情展示字段（含 file_url）
	AttachmentToJSON(attachment *models.Attachment) map[string]any
	// AttachmentListToJSON 附件列表展示字段
	AttachmentListToJSON(items []models.Attachment) []map[string]any
}

// AttachmentFilters 附件查询过滤器
type AttachmentFilters struct {
	AdminID     string
	Filename    string
	DisplayName string
	Keyword     string // filename OR display_name
	CategoryID  string
	IsPublic    string
	FileType    string
	Extension   string
	StartTime   string
	EndTime     string
	OrderBy     string
}

// BuildAttachmentFiltersFromHTTP reads list filters from query or body.
func BuildAttachmentFiltersFromHTTP(ctx http.Context) AttachmentFilters {
	return AttachmentFilters{
		AdminID:     ctx.Request().Query("admin_id", ""),
		Filename:    ctx.Request().Query("filename", ""),
		DisplayName: ctx.Request().Query("display_name", ""),
		Keyword:     ctx.Request().Query("keyword", ""),
		CategoryID:  ctx.Request().Query("category_id", ""),
		IsPublic:    ctx.Request().Query("is_public", ""),
		FileType:    ctx.Request().Query("file_type", ""),
		Extension:   ctx.Request().Query("extension", ""),
		StartTime:   helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:     helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:     ctx.Request().Query("order_by", ""),
	}
}

type AttachmentServiceImpl struct {
	ctx              http.Context
	disk             string
	systemLogService SystemLogService
}

func NewAttachmentService(ctx http.Context) AttachmentService {
	// 从数据库读取文件存储配置
	// 优先使用 file_disk，如果没有则使用 storage_disk（向后兼容），最后使用默认值 local
	disk := utils.GetConfigValue(ctx, "storage", "file_disk", "")
	if disk == "" {
		disk = utils.GetConfigValue(ctx, "storage", "storage_disk", "")
	}
	if disk == "" {
		disk = "local"
	}

	return &AttachmentServiceImpl{
		ctx:              ctx,
		disk:             disk,
		systemLogService: NewSystemLogService(ctx),
	}
}

const (
	AttachmentPublicURLTemplate  = "/api/admin/public/images/%d"
	AttachmentPrivateURLTemplate = "/api/admin/attachments/%d/preview"
)

// AttachmentPublicURL 公开附件的稳定访问路径（可写入富文本/配置）
func AttachmentPublicURL(id uint) string {
	return fmt.Sprintf(AttachmentPublicURLTemplate, id)
}

// AttachmentPrivatePreviewURL 私有附件的鉴权预览路径
func AttachmentPrivatePreviewURL(id uint) string {
	return fmt.Sprintf(AttachmentPrivateURLTemplate, id)
}

func defaultAttachmentIsPublic(fileType string) uint8 {
	switch fileType {
	case "image", "video":
		return 1
	default:
		return 0
	}
}

func parseAttachmentIsPublicInput(raw string, fileType string) uint8 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultAttachmentIsPublic(fileType)
	}
	switch raw {
	case "1", "true", "yes", "on":
		return 1
	case "0", "false", "no", "off":
		return 0
	default:
		return defaultAttachmentIsPublic(fileType)
	}
}

// InitChunkUpload 初始化分片上传
// 注意：分片信息由客户端缓存，服务端只生成 chunkID
func (s *AttachmentServiceImpl) InitChunkUpload(filename string, totalSize int64, chunkSize int64, totalChunks int) (string, error) {
	// 检查存储驱动：大文件分片上传仅支持本地存储
	cloudStorageDrivers := []string{"s3", "oss", "cos", "minio", "qiniu"}
	for _, driver := range cloudStorageDrivers {
		if s.disk == driver {
			return "", apperrors.ErrChunkUploadOnlyLocalStorage.WithMessage(fmt.Sprintf("大文件分片上传仅支持本地存储，当前存储驱动为: %s", driver))
		}
	}

	// 生成唯一的分片ID
	hash := md5.Sum(fmt.Appendf(nil, "%s_%d_%d", filename, totalSize, time.Now().UnixNano()))
	chunkID := hex.EncodeToString(hash[:])

	if err := utils.ValidateAttachmentExtension(filename); err != nil {
		return "", err
	}

	// 不再使用服务端缓存，分片信息由客户端管理
	return chunkID, nil
}

// UploadChunk 上传分片
// 注意：不再使用服务端缓存，直接保存分片文件
func (s *AttachmentServiceImpl) UploadChunk(chunkID string, chunkIndex int, chunkData []byte) error {
	if chunkIndex < 0 {
		return apperrors.ErrInvalidChunkIndex
	}

	// 保存分片到临时目录
	storage, err := utils.StorageDisk(s.disk)
	if err != nil {
		return err
	}
	chunkPath := fmt.Sprintf("chunks/%s/%d", chunkID, chunkIndex)

	if err := storage.Put(chunkPath, string(chunkData)); err != nil {
		if s.ctx != nil {
			errorlog.RecordHTTP(s.ctx, "attachment", "保存分片失败", map[string]any{
				"chunk_id":    chunkID,
				"chunk_index": chunkIndex,
				"error":       err.Error(),
			}, "保存分片失败: %w", err)
		}
		return apperrors.ErrSaveChunkFailed.WithError(err)
	}

	return nil
}

// MergeChunks 合并分片
// 注意：不再使用服务端缓存，通过检查实际文件系统来验证分片
func (s *AttachmentServiceImpl) MergeChunks(chunkID string, filename string, mimeType string, totalChunks int, isPublicRaw string) (*models.Attachment, error) {
	storage, err := utils.StorageDisk(s.disk)
	if err != nil {
		return nil, err
	}

	// 检查所有分片文件是否存在
	indices := make([]int, totalChunks)
	for i := range indices {
		chunkPath := fmt.Sprintf("chunks/%s/%d", chunkID, i)
		if !storage.Exists(chunkPath) {
			return nil, apperrors.ErrChunkNotFound.WithParams(map[string]any{
				"chunk_index": i,
			})
		}
	}

	// 生成最终文件路径
	ext := filepath.Ext(filename)
	if ext == "" {
		exts, _ := mime.ExtensionsByType(mimeType)
		if len(exts) > 0 {
			ext = exts[0]
		}
	}

	// 生成唯一文件名
	hash := md5.Sum([]byte(fmt.Sprintf("%s_%d", filename, time.Now().UnixNano())))
	uniqueName := hex.EncodeToString(hash[:]) + ext
	datePath := time.Now().Format("2006/01/02")
	finalPath := fmt.Sprintf("attachments/%s/%s", datePath, uniqueName)

	// 合并分片（流式写入，避免大文件内存占用过高）
	// 对于本地存储，直接使用文件系统操作以提高性能
	var fileSize int64
	var missingChunks []int // 记录缺失的分片索引

	// 获取存储根目录
	storageRoot := facades.Config().GetString("filesystems.disks." + s.disk + ".root")
	if storageRoot == "" {
		storageRoot = "storage/app"
	}

	// 构建目标文件的完整路径
	finalFullPath := filepath.Join(storageRoot, finalPath)

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(finalFullPath), 0755); err != nil {
		if s.ctx != nil {
			errorlog.RecordHTTP(s.ctx, "attachment", "创建目标目录失败", map[string]any{
				"chunk_id":  chunkID,
				"directory": filepath.Dir(finalFullPath),
				"error":     err.Error(),
			}, "创建目标目录失败: %w", err)
		}
		return nil, apperrors.ErrCreateDirectoryFailed.WithError(err)
	}

	// 创建目标文件（流式写入）
	outFile, err := os.Create(finalFullPath)
	if err != nil {
		if s.ctx != nil {
			errorlog.RecordHTTP(s.ctx, "attachment", "创建目标文件失败", map[string]any{
				"chunk_id":  chunkID,
				"file_path": finalFullPath,
				"error":     err.Error(),
			}, "创建目标文件失败: %w", err)
		}
		return nil, apperrors.ErrCreateFileFailed.WithError(err)
	}
	defer outFile.Close()

	// 按顺序读取并写入每个分片
	for i := range make([]int, totalChunks) {
		chunkPath := fmt.Sprintf("chunks/%s/%d", chunkID, i)
		chunkFullPath := filepath.Join(storageRoot, chunkPath)

		// 检查分片文件是否存在
		if _, err := os.Stat(chunkFullPath); os.IsNotExist(err) {
			missingChunks = append(missingChunks, i)
			continue
		}

		// 打开分片文件
		chunkFile, err := os.Open(chunkFullPath)
		if err != nil {
			// 读取失败，记录到系统日志
			if s.ctx != nil {
				_ = s.systemLogService.RecordHTTP(s.ctx, "warning", "attachment", fmt.Sprintf("Failed to read chunk %d for chunkID %s", i, chunkID), map[string]any{
					"chunk_index": i,
					"chunk_id":    chunkID,
					"error":       err.Error(),
				})
			}
			missingChunks = append(missingChunks, i)
			continue
		}

		// 流式复制分片内容到目标文件
		written, err := io.Copy(outFile, chunkFile)
		if err != nil {
			chunkFile.Close()
			// 如果写入失败，删除已创建的目标文件
			_ = os.Remove(finalFullPath)
			if s.ctx != nil {
				errorlog.RecordHTTP(s.ctx, "attachment", "写入分片失败", map[string]any{
					"chunk_id":    chunkID,
					"chunk_index": i,
					"file_path":   finalFullPath,
					"error":       err.Error(),
				}, "写入分片 %d 失败: %w", i, err)
			}
			return nil, apperrors.ErrWriteChunkFailed.WithError(err).WithParams(map[string]any{
				"chunk_index": i,
			})
		}
		fileSize += written
		chunkFile.Close()
	}

	// 关闭目标文件
	if err := outFile.Close(); err != nil {
		_ = os.Remove(finalFullPath)
		if s.ctx != nil {
			errorlog.RecordHTTP(s.ctx, "attachment", "关闭目标文件失败", map[string]any{
				"chunk_id":  chunkID,
				"file_path": finalFullPath,
				"error":     err.Error(),
			}, "关闭目标文件失败: %w", err)
		}
		return nil, apperrors.ErrCloseFileFailed.WithError(err)
	}

	// 检查是否有缺失的分片
	if len(missingChunks) > 0 {
		_ = os.Remove(finalFullPath)
		return nil, apperrors.ErrChunkMissing.WithParams(map[string]any{
			"missing_chunks": missingChunks,
			"count":          len(missingChunks),
		})
	}

	// 检查是否有数据被合并
	if fileSize == 0 {
		_ = os.Remove(finalFullPath)
		return nil, apperrors.ErrNoChunkDataToMerge
	}

	// magic bytes / 白名单校验（合并完成后读取文件头）
	header := make([]byte, 512)
	if headerFile, err := os.Open(finalFullPath); err == nil {
		if n, readErr := headerFile.Read(header); readErr == nil && n > 0 {
			validatedMIME, validateErr := utils.ValidateAttachmentFile(filename, header[:n])
			if validateErr != nil {
				_ = headerFile.Close()
				_ = os.Remove(finalFullPath)
				return nil, validateErr
			}
			mimeType = validatedMIME
		} else {
			_ = headerFile.Close()
			_ = os.Remove(finalFullPath)
			return nil, apperrors.ErrAttachmentFileContentMismatch
		}
		_ = headerFile.Close()
	} else {
		_ = os.Remove(finalFullPath)
		return nil, apperrors.ErrAttachmentFileContentMismatch
	}

	// 验证文件大小（从文件系统获取实际大小）
	if fileInfo, err := os.Stat(finalFullPath); err == nil {
		fileSize = fileInfo.Size()
	}

	// 创建附件记录
	adminID := uint(0)
	if s.ctx != nil {
		if id, err := helpers.GetAdminIDFromContext(s.ctx); err == nil {
			adminID = id
		}
	}

	fileType := s.GetFileType(mimeType)
	attachment := &models.Attachment{
		AdminID:    adminID,
		CategoryID: s.resolveDefaultCategoryID(),
		Disk:       s.disk,
		Path:       finalPath,
		Filename:   filename,
		Extension:  strings.TrimPrefix(ext, "."),
		MimeType:   mimeType,
		Size:       fileSize,
		Status:     1,
		FileType:   fileType,
		IsPublic:   parseAttachmentIsPublicInput(isPublicRaw, fileType),
		ChunkID:    chunkID,
	}

	if err := appfacades.OrmQuery(s.ctx).Create(attachment); err != nil {
		// 如果创建记录失败，删除已上传的文件
		_ = storage.Delete(finalPath)
		if s.ctx != nil {
			errorlog.RecordHTTP(s.ctx, "attachment", "创建附件记录失败", map[string]any{
				"chunk_id":  chunkID,
				"file_path": finalPath,
				"filename":  filename,
				"error":     err.Error(),
			}, "创建附件记录失败: %w", err)
		}
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	// 合并成功后才清理分片文件（确保数据已保存到数据库）
	// 清理所有分片文件（包括可能缺失的）
	cleanupSuccess := true
	cleanupCount := 0
	for i := range make([]int, totalChunks) {
		chunkPath := fmt.Sprintf("chunks/%s/%d", chunkID, i)
		if storage.Exists(chunkPath) {
			if err := storage.Delete(chunkPath); err != nil {
				// 记录删除失败到系统日志，但不影响整体流程
				if s.ctx != nil {
					_ = s.systemLogService.RecordHTTP(s.ctx, "warning", "attachment", fmt.Sprintf("Failed to delete chunk file %s", chunkPath), map[string]any{
						"chunk_path": chunkPath,
						"chunk_id":   chunkID,
						"error":      err.Error(),
					})
				}
				cleanupSuccess = false
			} else {
				cleanupCount++
			}
		}
	}

	if !cleanupSuccess && s.ctx != nil {
		// 记录部分分片删除失败的警告到系统日志
		_ = s.systemLogService.RecordHTTP(s.ctx, "warning", "attachment", fmt.Sprintf("Some chunk files failed to delete for chunkID %s", chunkID), map[string]any{
			"chunk_id":      chunkID,
			"cleaned_count": cleanupCount,
			"total_chunks":  totalChunks,
		})
	}

	return attachment, nil
}

// GetChunkProgress 获取分片上传进度
// 注意：不再使用服务端缓存，通过检查实际文件系统来获取进度
// 优化：如果分片数量很大，可以考虑限制返回的索引数量或使用并发检查
func (s *AttachmentServiceImpl) GetChunkProgress(chunkID string, totalChunks int) (map[string]any, error) {
	storage, err := utils.StorageDisk(s.disk)
	if err != nil {
		return nil, err
	}

	uploadedCount := 0
	uploadedIndices := []int{} // 已上传的分片索引数组

	// 优化：如果分片数量超过1000，只返回前100个已上传的索引，减少响应大小
	maxIndices := 100
	if totalChunks > 1000 {
		maxIndices = 100
	}

	// 检查每个分片文件是否存在
	indices := make([]int, totalChunks)
	for i := range indices {
		chunkPath := fmt.Sprintf("chunks/%s/%d", chunkID, i)
		if storage.Exists(chunkPath) {
			uploadedCount++
			// 只记录前 maxIndices 个索引，减少响应大小
			if len(uploadedIndices) < maxIndices {
				uploadedIndices = append(uploadedIndices, i)
			}
		}
	}

	progress := float64(uploadedCount) / float64(totalChunks) * 100

	return map[string]any{
		"chunk_id":        chunkID,
		"total_chunks":    totalChunks,
		"uploaded_count":  uploadedCount,
		"uploaded_chunks": uploadedIndices, // 返回已上传的分片索引数组（最多100个）
		"progress":        progress,
		"completed":       uploadedCount == totalChunks,
	}, nil
}

// UploadFile 普通文件上传（小文件）
func (s *AttachmentServiceImpl) UploadFile(fileData []byte, filename string, mimeType string, isPublicRaw string) (*models.Attachment, error) {
	validatedMIME, err := utils.ValidateAttachmentFile(filename, fileData)
	if err != nil {
		return nil, err
	}
	mimeType = validatedMIME

	// 生成文件路径
	ext := filepath.Ext(filename)
	hash := md5.Sum([]byte(fmt.Sprintf("%s_%d", filename, time.Now().UnixNano())))
	uniqueName := hex.EncodeToString(hash[:]) + ext
	datePath := time.Now().Format("2006/01/02")
	finalPath := fmt.Sprintf("attachments/%s/%s", datePath, uniqueName)

	// 保存文件（云盘未配置时返回业务错误，避免 Storage().Disk panic）
	storage, err := utils.StorageDisk(s.disk)
	if err != nil {
		return nil, err
	}
	if err := storage.Put(finalPath, string(fileData)); err != nil {
		if s.ctx != nil {
			errorlog.RecordHTTP(s.ctx, "attachment", "保存文件失败", map[string]any{
				"filename":  filename,
				"file_path": finalPath,
				"error":     err.Error(),
			}, "保存文件失败: %w", err)
		}
		return nil, apperrors.ErrSaveFileFailed.WithError(err)
	}

	// 获取文件大小
	fileSize := int64(len(fileData))
	if size, err := storage.Size(finalPath); err == nil {
		fileSize = size
	}

	// 创建附件记录
	adminID := uint(0)
	if s.ctx != nil {
		if id, err := helpers.GetAdminIDFromContext(s.ctx); err == nil {
			adminID = id
		}
	}

	fileType := s.GetFileType(mimeType)
	attachment := &models.Attachment{
		AdminID:    adminID,
		CategoryID: s.resolveDefaultCategoryID(),
		Disk:       s.disk,
		Path:       finalPath,
		Filename:   filename,
		Extension:  strings.TrimPrefix(ext, "."),
		MimeType:   mimeType,
		Size:       fileSize,
		Status:     1,
		FileType:   fileType,
		IsPublic:   parseAttachmentIsPublicInput(isPublicRaw, fileType),
	}

	if err := appfacades.OrmQuery(s.ctx).Create(attachment); err != nil {
		_ = storage.Delete(finalPath)
		if s.ctx != nil {
			errorlog.RecordHTTP(s.ctx, "attachment", "创建附件记录失败", map[string]any{
				"filename":  filename,
				"file_path": finalPath,
				"error":     err.Error(),
			}, "创建附件记录失败: %w", err)
		}
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	return attachment, nil
}

// GetFileURL 获取文件稳定访问URL（不返回云存储临时链）
func (s *AttachmentServiceImpl) GetFileURL(attachment *models.Attachment) string {
	if attachment == nil || attachment.ID == 0 {
		return ""
	}
	if attachment.IsPublic == 1 {
		return AttachmentPublicURL(attachment.ID)
	}
	return AttachmentPrivatePreviewURL(attachment.ID)
}

// AttachmentToJSON 附件列表/详情展示字段（含 file_url）
func (s *AttachmentServiceImpl) AttachmentToJSON(attachment *models.Attachment) map[string]any {
	if attachment == nil {
		return nil
	}
	payload := map[string]any{
		"id":           attachment.ID,
		"admin_id":     attachment.AdminID,
		"category_id":  attachment.CategoryID,
		"disk":         attachment.Disk,
		"path":         attachment.Path,
		"filename":     attachment.Filename,
		"display_name": attachment.DisplayName,
		"extension":    attachment.Extension,
		"mime_type":    attachment.MimeType,
		"size":         attachment.Size,
		"status":       attachment.Status,
		"file_type":    attachment.FileType,
		"is_public":    attachment.IsPublic,
		"chunk_id":     attachment.ChunkID,
		"created_at":   attachment.CreatedAt,
		"updated_at":   attachment.UpdatedAt,
		"file_url":     s.GetFileURL(attachment),
	}
	if attachment.Admin != nil {
		payload["admin"] = attachment.Admin
	}
	if attachment.Category != nil {
		payload["category"] = attachment.Category
	}
	return payload
}

// AttachmentListToJSON 附件列表展示字段
func (s *AttachmentServiceImpl) AttachmentListToJSON(items []models.Attachment) []map[string]any {
	list := make([]map[string]any, len(items))
	for i := range items {
		list[i] = s.AttachmentToJSON(&items[i])
	}
	return list
}

// GetFileType 根据MIME类型判断文件类型
func (s *AttachmentServiceImpl) GetFileType(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	if strings.HasPrefix(mimeType, "application/pdf") ||
		strings.HasPrefix(mimeType, "application/msword") ||
		strings.HasPrefix(mimeType, "application/vnd.openxmlformats-officedocument") ||
		strings.HasPrefix(mimeType, "application/vnd.ms-excel") ||
		strings.HasPrefix(mimeType, "text/") {
		return "document"
	}
	return "other"
}

// GetByID 根据ID获取附件
func (s *AttachmentServiceImpl) GetByID(id uint) (*models.Attachment, error) {
	var attachment models.Attachment
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&attachment); err != nil {
		return nil, apperrors.ErrAttachmentNotFound.WithError(err)
	}
	return &attachment, nil
}

// GetByIDs 根据ID列表获取附件
func (s *AttachmentServiceImpl) GetByIDs(ids []uint) ([]models.Attachment, error) {
	if len(ids) == 0 {
		return []models.Attachment{}, nil
	}

	idsAny := helpers.ConvertUintSliceToAny(ids)
	var attachments []models.Attachment
	if err := appfacades.OrmQuery(s.ctx).WhereIn("id", idsAny).Get(&attachments); err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}
	return attachments, nil
}

// GetList 获取附件列表
func (s *AttachmentServiceImpl) GetList(filters AttachmentFilters, page, pageSize int) ([]models.Attachment, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Attachment{})

	// 应用筛选条件
	if filters.AdminID != "" {
		query = query.Where("admin_id", filters.AdminID)
	}
	if filters.Keyword != "" {
		kw := "%" + filters.Keyword + "%"
		query = query.Where("(filename LIKE ? OR display_name LIKE ?)", kw, kw)
	} else {
		if filters.Filename != "" {
			query = query.Where("filename LIKE ?", "%"+filters.Filename+"%")
		}
		if filters.DisplayName != "" {
			query = query.Where("display_name LIKE ?", "%"+filters.DisplayName+"%")
		}
	}
	if filters.CategoryID != "" {
		query = query.Where("category_id", filters.CategoryID)
	}
	if filters.IsPublic != "" {
		query = query.Where("is_public", filters.IsPublic)
	}
	if filters.FileType != "" {
		query = query.Where("file_type = ?", filters.FileType)
	}
	if filters.Extension != "" {
		query = query.Where("extension = ?", filters.Extension)
	}
	if filters.StartTime != "" {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime != "" {
		query = query.Where("created_at <= ?", filters.EndTime)
	}

	// 应用排序
	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "id:desc"
	}
	query = helpers.ApplySort(query, orderBy, "id:desc")

	// 分页查询
	var attachments []models.Attachment
	var total int64
	if err := query.With("Admin").With("Category").Paginate(page, pageSize, &attachments, &total); err != nil {
		return nil, 0, err
	}

	return attachments, total, nil
}

// UpdateDisplayName 更新显示名称
func (s *AttachmentServiceImpl) UpdateDisplayName(id uint, displayName string) error {
	var attachment models.Attachment
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&attachment); err != nil {
		return apperrors.ErrAttachmentNotFound.WithError(err)
	}

	attachment.DisplayName = displayName
	if err := appfacades.OrmQuery(s.ctx).Save(&attachment); err != nil {
		if s.ctx != nil {
			errorlog.RecordHTTP(s.ctx, "attachment", "更新附件显示名称失败", map[string]any{
				"attachment_id": id,
				"display_name":  displayName,
				"error":         err.Error(),
			}, "更新附件显示名称失败: %v", err)
		}
		return apperrors.ErrUpdateFailed.WithError(err)
	}

	return nil
}

// UpdateCategory 更新附件分类
func (s *AttachmentServiceImpl) UpdateCategory(id uint, categoryID uint) error {
	var attachment models.Attachment
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&attachment); err != nil {
		return apperrors.ErrAttachmentNotFound.WithError(err)
	}

	categoryService := NewAttachmentCategoryService(s.ctx)
	if _, err := categoryService.GetByID(categoryID); err != nil {
		return err
	}

	attachment.CategoryID = categoryID
	if err := appfacades.OrmQuery(s.ctx).Save(&attachment); err != nil {
		return apperrors.ErrUpdateFailed.WithError(err)
	}
	return nil
}

// UpdateVisibility 更新附件公开/私有状态
func (s *AttachmentServiceImpl) UpdateVisibility(id uint, isPublic bool) error {
	var attachment models.Attachment
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&attachment); err != nil {
		return apperrors.ErrAttachmentNotFound.WithError(err)
	}

	if isPublic {
		attachment.IsPublic = 1
	} else {
		attachment.IsPublic = 0
	}

	if err := appfacades.OrmQuery(s.ctx).Save(&attachment); err != nil {
		return apperrors.ErrUpdateFailed.WithError(err)
	}
	return nil
}

// resolveDefaultCategoryID 新上传附件默认归入「未分类」
func (s *AttachmentServiceImpl) resolveDefaultCategoryID() uint {
	id, err := NewAttachmentCategoryService(s.ctx).GetUncategorizedID()
	if err != nil {
		return 0
	}
	return id
}

// DeleteFile 删除文件
func (s *AttachmentServiceImpl) DeleteFile(attachment *models.Attachment) error {
	// 删除文件
	if attachment.Path != "" && attachment.Disk != "" {
		storage, err := utils.StorageDisk(attachment.Disk)
		if err != nil {
			return err
		}
		if err := storage.Delete(attachment.Path); err != nil {
			if s.ctx != nil {
				errorlog.RecordHTTP(s.ctx, "attachment", "删除文件失败", map[string]any{
					"attachment_id": attachment.ID,
					"file_path":     attachment.Path,
					"disk":          attachment.Disk,
					"error":         err.Error(),
				}, "删除文件失败: %w", err)
			}
			return apperrors.ErrDeleteFileFailed.WithError(err)
		}
	}

	// 删除数据库记录
	if _, err := appfacades.OrmQuery(s.ctx).Delete(attachment); err != nil {
		if s.ctx != nil {
			errorlog.RecordHTTP(s.ctx, "attachment", "删除附件记录失败", map[string]any{
				"attachment_id": attachment.ID,
				"error":         err.Error(),
			}, "删除附件记录失败: %w", err)
		}
		return apperrors.ErrDeleteFailed.WithError(err)
	}

	return nil
}
