package admin

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
)

type AttachmentController struct{}

func NewAttachmentController() *AttachmentController {
	return &AttachmentController{}
}

func attachmentTempDir(ctx http.Context) string {
	prefix := strings.TrimSuffix(helpers.TenantStoragePrefix(ctx), "/")
	if prefix == "" {
		return "tmp"
	}
	return prefix + "/tmp"
}

func (r *AttachmentController) AttachmentService(ctx http.Context) services.AttachmentService {
	return services.NewAttachmentService(ctx)
}

// Index 附件列表
func (r *AttachmentController) Index(ctx http.Context) http.Response {
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)

	filters := services.BuildAttachmentFiltersFromHTTP(ctx)
	attachmentService := r.AttachmentService(ctx)
	attachments, total, err := attachmentService.GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      attachmentService.AttachmentListToJSON(attachments),
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Upload 普通文件上传（小文件）
func (r *AttachmentController) Upload(ctx http.Context) http.Response {
	file, err := ctx.Request().File("file")
	if err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrFileRequired.Code)
	}

	filename := file.GetClientOriginalName()
	if filename == "" {
		filename = "uploaded_file"
	}

	// 读取文件内容：先将文件保存到临时位置，然后读取
	storage, resp, ok := response.OpenStorageDisk(ctx, "attachment", "local", map[string]any{"filename": filename})
	if !ok {
		return resp
	}

	// 保存文件到临时位置，PutFile 返回保存后的路径
	tmpDir := attachmentTempDir(ctx)
	savedPath, err := storage.PutFile(tmpDir, file)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
			"filename": filename,
		})
	}

	// 读取文件内容
	fileDataStr, err := storage.Get(savedPath)
	if err != nil {
		// 清理临时文件
		_ = storage.Delete(savedPath)
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
			"filename": filename,
		})
	}

	// 清理临时文件
	_ = storage.Delete(savedPath)

	// 转换为字节数组
	fileData := []byte(fileDataStr)

	attachmentService := r.AttachmentService(ctx)
	isPublicRaw := ctx.Request().Input("is_public", "")
	attachment, err := attachmentService.UploadFile(fileData, filename, "", isPublicRaw)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
			"filename": filename,
		})
	}

	return response.Success(ctx, "upload_success", r.buildUploadResponse(attachmentService, attachment))
}

// ChunkUpload 大文件分片上传统一接口
// 通过 action 参数区分不同操作：init（初始化）、upload（上传分片）、merge（合并分片）、progress（获取进度）
func (r *AttachmentController) ChunkUpload(ctx http.Context) http.Response {
	action := ctx.Request().Input("action", "")
	if action == "" {
		// 兼容 GET 请求获取进度
		action = ctx.Request().Query("action", "progress")
	}

	attachmentService := r.AttachmentService(ctx)

	switch action {
	case "init":
		// 初始化分片上传
		filename := ctx.Request().Input("filename", "")
		if filename == "" {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrFilenameRequired.Code)
		}

		totalSizeStr := ctx.Request().Input("total_size", "0")
		totalSize, err := strconv.ParseInt(totalSizeStr, 10, 64)
		if err != nil {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidTotalSize.Code)
		}
		if totalSize <= 0 {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidTotalSize.Code)
		}

		chunkSizeStr := ctx.Request().Input("chunk_size", "0")
		chunkSize, err := strconv.ParseInt(chunkSizeStr, 10, 64)
		if err != nil {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidChunkSize.Code)
		}
		if chunkSize <= 0 {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidChunkSize.Code)
		}

		totalChunksStr := ctx.Request().Input("total_chunks", "0")
		totalChunks, err := strconv.Atoi(totalChunksStr)
		if err != nil {
			// 尝试作为浮点数解析（以防传入 "5.0" 格式）
			if floatVal, floatErr := strconv.ParseFloat(totalChunksStr, 64); floatErr == nil {
				totalChunks = int(floatVal)
			} else {
				return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidTotalChunks.Code)
			}
		}
		if totalChunks <= 0 {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidTotalChunks.Code)
		}

		// 验证分片数量计算的合理性
		expectedChunks := int((totalSize + chunkSize - 1) / chunkSize) // 向上取整
		if totalChunks != expectedChunks {
			// 不返回错误，使用客户端提供的值（可能是由于浮点数计算差异）
		}

		chunkID, err := attachmentService.InitChunkUpload(filename, totalSize, chunkSize, totalChunks)
		if err != nil {
			return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
				"filename":     filename,
				"total_size":   totalSize,
				"chunk_size":   chunkSize,
				"total_chunks": totalChunks,
			})
		}

		return response.Success(ctx, "init_chunk_upload_success", http.Json{
			"chunk_id": chunkID,
		})

	case "upload":
		// 上传分片
		chunkID := ctx.Request().Input("chunk_id", "")
		if chunkID == "" {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrChunkIDRequired.Code)
		}

		chunkIndex, err := strconv.Atoi(ctx.Request().Input("chunk_index", "-1"))
		if err != nil || chunkIndex < 0 {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidChunkIndex.Code)
		}

		file, err := ctx.Request().File("chunk")
		if err != nil {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrChunkFileRequired.Code)
		}

		// 读取分片数据：先将文件保存到临时位置，然后读取
		storage, resp, ok := response.OpenStorageDisk(ctx, "attachment", "local", map[string]any{
			"chunk_id":    chunkID,
			"chunk_index": chunkIndex,
		})
		if !ok {
			return resp
		}

		// 保存文件到临时位置
		tmpDir := attachmentTempDir(ctx)
		savedPath, err := storage.PutFile(tmpDir, file)
		if err != nil {
			return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
				"chunk_id":    chunkID,
				"chunk_index": chunkIndex,
			})
		}

		// 读取文件内容
		chunkDataStr, err := storage.Get(savedPath)
		if err != nil {
			// 清理临时文件
			_ = storage.Delete(savedPath)
			return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
				"chunk_id":    chunkID,
				"chunk_index": chunkIndex,
			})
		}

		// 清理临时文件
		_ = storage.Delete(savedPath)

		// 转换为字节数组
		chunkData := []byte(chunkDataStr)

		if err := attachmentService.UploadChunk(chunkID, chunkIndex, chunkData); err != nil {
			return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
				"chunk_id":    chunkID,
				"chunk_index": chunkIndex,
			})
		}

		return response.Success(ctx, "upload_chunk_success")

	case "merge":
		// 合并分片
		chunkID := ctx.Request().Input("chunk_id", "")
		if chunkID == "" {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrChunkIDRequired.Code)
		}

		filename := ctx.Request().Input("filename", "")
		if filename == "" {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrFilenameRequired.Code)
		}

		totalChunksStr := ctx.Request().Input("total_chunks", "0")
		totalChunks, err := strconv.Atoi(totalChunksStr)
		if err != nil {
			// 尝试作为浮点数解析（以防传入 "5.0" 格式）
			if floatVal, floatErr := strconv.ParseFloat(totalChunksStr, 64); floatErr == nil {
				totalChunks = int(floatVal)
			} else {
				return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidTotalChunks.Code)
			}
		}
		if totalChunks <= 0 {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidTotalChunks.Code)
		}

		attachment, err := attachmentService.MergeChunks(chunkID, filename, "", totalChunks, ctx.Request().Input("is_public", ""))
		if err != nil {
			return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
				"chunk_id":     chunkID,
				"filename":     filename,
				"total_chunks": totalChunks,
			})
		}

		// 合并成功后，记录日志（仅 Debug 模式）
		facades.Log().Debugf("Successfully merged chunks for chunkID %s, filename: %s, total_chunks: %d", chunkID, filename, totalChunks)

		return response.Success(ctx, "merge_chunks_success", r.buildUploadResponse(attachmentService, attachment))

	case "progress":
		// 获取分片上传进度
		chunkID := ctx.Request().Query("chunk_id", "")
		if chunkID == "" {
			chunkID = ctx.Request().Input("chunk_id", "")
		}
		if chunkID == "" {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrChunkIDRequired.Code)
		}

		totalChunks, err := strconv.Atoi(ctx.Request().Query("total_chunks", "0"))
		if totalChunks == 0 {
			totalChunks, err = strconv.Atoi(ctx.Request().Input("total_chunks", "0"))
		}
		if err != nil || totalChunks <= 0 {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidTotalChunks.Code)
		}

		progress, err := attachmentService.GetChunkProgress(chunkID, totalChunks)
		if err != nil {
			return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
				"chunk_id": chunkID,
			})
		}

		return response.Success(ctx, progress)

	default:
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidAction.Code)
	}
}

// Download 下载附件文件
func (r *AttachmentController) Download(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	attachmentService := r.AttachmentService(ctx)
	attachment, err := attachmentService.GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusNotFound, err, map[string]any{"id": id})
	}
	if resp := ForbidUnlessAttachmentReadable(ctx, attachment); resp != nil {
		return resp
	}

	if attachment.Path == "" || attachment.Disk == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrFilePathRequired.Code)
	}

	storage, errResp, ok := response.OpenStorageDisk(ctx, "attachment", attachment.Disk, map[string]any{
		"path": attachment.Path,
	})
	if !ok {
		return errResp
	}

	// 对于云存储，尝试生成临时URL并重定向，避免通过服务器中转
	if attachment.Disk != "local" && attachment.Disk != "public" {
		if url, err := storage.TemporaryUrl(attachment.Path, time.Now().Add(24*time.Hour)); err == nil {
			return ctx.Response().Redirect(http.StatusFound, url)
		}
	}

	// 读取文件内容
	content, err := storage.Get(attachment.Path)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
			"disk": attachment.Disk,
			"path": attachment.Path,
		})
	}

	// 设置响应头
	filename := attachment.Filename
	if filename == "" {
		filename = attachment.Path
	}

	// 根据MIME类型设置 Content-Type
	contentType := attachment.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 设置响应头，使用链式调用确保顺序正确
	httpResp := ctx.Response().
		Header("Content-Type", contentType).
		Header("Content-Length", fmt.Sprintf("%d", len(content))).
		Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename)).
		Header("X-Content-Type-Options", "nosniff").
		Header("Cache-Control", "no-cache, no-store, must-revalidate").
		Header("Pragma", "no-cache").
		Header("Expires", "0")

	return httpResp.String(http.StatusOK, content)
}

// PublicPreview 公开附件预览（无需登录，仅 is_public=1 可访问）
func (r *AttachmentController) PublicPreview(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	attachmentService := r.AttachmentService(ctx)
	attachment, err := attachmentService.GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusNotFound, err, map[string]any{"id": id})
	}

	if attachment.IsPublic != 1 {
		return response.Error(ctx, http.StatusForbidden, apperrors.ErrAttachmentNotPublic.Code)
	}

	return r.serveAttachmentContent(ctx, attachment, "inline")
}

// Preview 附件预览（需登录，公开/私有均可）
func (r *AttachmentController) Preview(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	attachmentService := r.AttachmentService(ctx)
	attachment, err := attachmentService.GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusNotFound, err, map[string]any{"id": id})
	}
	if resp := ForbidUnlessAttachmentReadable(ctx, attachment); resp != nil {
		return resp
	}

	return r.serveAttachmentContent(ctx, attachment, "inline")
}

// Destroy 删除附件
func (r *AttachmentController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	attachmentService := r.AttachmentService(ctx)
	attachment, err := attachmentService.GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusNotFound, err, map[string]any{"id": id})
	}
	if resp := ForbidUnlessAttachmentMutable(ctx, attachment); resp != nil {
		return resp
	}

	if err := attachmentService.DeleteFile(attachment); err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
			"attachId": attachment.ID,
		})
	}

	return response.Success(ctx)
}

type AttachmentBatchDestroyRequest struct {
	IDs []uint `json:"ids"`
}

// BatchDestroy 批量删除附件
func (r *AttachmentController) BatchDestroy(ctx http.Context) http.Response {
	var req AttachmentBatchDestroyRequest

	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}

	if len(req.IDs) == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDsRequired.Code)
	}

	ids := req.IDs

	// 查询要删除的附件
	attachmentService := r.AttachmentService(ctx)
	attachments, err := attachmentService.GetByIDs(ids)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
			"ids": ids,
		})
	}
	for i := range attachments {
		if resp := ForbidUnlessAttachmentMutable(ctx, &attachments[i]); resp != nil {
			return resp
		}
	}

	// 删除文件和记录
	for _, attachment := range attachments {
		if err := attachmentService.DeleteFile(&attachment); err != nil {
			// 批量删除中单个文件删除失败只记录日志，不影响主流程
			errorlog.RecordHTTP(ctx, "attachment", "Failed to delete attachment in batch delete", map[string]any{
				"error":    err.Error(),
				"attachId": attachment.ID,
			}, "Delete attachment in batch delete error: %v", err)
		}
	}

	return response.Success(ctx)
}

// UpdateDisplayName 更新显示名称
func (r *AttachmentController) UpdateDisplayName(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	attachmentService := r.AttachmentService(ctx)
	displayName := ctx.Request().Input("display_name", "")

	attachment, err := attachmentService.GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusNotFound, err, map[string]any{"id": id})
	}
	if resp := ForbidUnlessAttachmentMutable(ctx, attachment); resp != nil {
		return resp
	}

	if err := attachmentService.UpdateDisplayName(id, displayName); err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
			"attachId": id,
		})
	}

	// 重新获取更新后的附件
	attachment, err = attachmentService.GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"attachment": attachment,
	})
}

// UpdateCategory 更新附件分类
func (r *AttachmentController) UpdateCategory(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	categoryIDStr := ctx.Request().Input("category_id", "")
	categoryID, _ := strconv.ParseUint(categoryIDStr, 10, 64)
	if categoryID == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}

	attachmentService := r.AttachmentService(ctx)
	attachment, err := attachmentService.GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusNotFound, err, map[string]any{"id": id})
	}
	if resp := ForbidUnlessAttachmentMutable(ctx, attachment); resp != nil {
		return resp
	}

	if err := attachmentService.UpdateCategory(id, uint(categoryID)); err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
			"attachId":    id,
			"category_id": categoryID,
		})
	}

	attachment, err = attachmentService.GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"attachment": attachment,
	})
}

// UpdateVisibility 更新附件公开/私有状态
func (r *AttachmentController) UpdateVisibility(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	isPublicStr := strings.TrimSpace(ctx.Request().Input("is_public", ""))
	if isPublicStr == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}

	isPublic := isPublicStr == "1" || strings.EqualFold(isPublicStr, "true")
	attachmentService := r.AttachmentService(ctx)
	attachment, err := attachmentService.GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusNotFound, err, map[string]any{"id": id})
	}
	if resp := ForbidUnlessAttachmentMutable(ctx, attachment); resp != nil {
		return resp
	}

	if err := attachmentService.UpdateVisibility(id, isPublic); err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
			"attachId":  id,
			"is_public": isPublicStr,
		})
	}

	attachment, err = attachmentService.GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"attachment": attachment,
		"file_url":   attachmentService.GetFileURL(attachment),
	})
}

func (r *AttachmentController) buildUploadResponse(attachmentService services.AttachmentService, attachment *models.Attachment) http.Json {
	fileURL := attachmentService.GetFileURL(attachment)
	return http.Json{
		"id":        attachment.ID,
		"filename":  attachment.Filename,
		"size":      attachment.Size,
		"mime_type": attachment.MimeType,
		"file_type": attachment.FileType,
		"is_public": attachment.IsPublic,
		"file_url":  fileURL,
	}
}

func (r *AttachmentController) serveAttachmentContent(ctx http.Context, attachment *models.Attachment, disposition string) http.Response {
	if attachment.Path == "" || attachment.Disk == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrFilePathRequired.Code)
	}

	storage, errResp, ok := response.OpenStorageDisk(ctx, "attachment", attachment.Disk, map[string]any{
		"path": attachment.Path,
	})
	if !ok {
		return errResp
	}

	if attachment.Disk != "local" && attachment.Disk != "public" {
		if url, err := storage.TemporaryUrl(attachment.Path, time.Now().Add(24*time.Hour)); err == nil {
			return ctx.Response().Redirect(http.StatusFound, url)
		}
	}

	content, err := storage.Get(attachment.Path)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "attachment", http.StatusInternalServerError, err, map[string]any{
			"disk": attachment.Disk,
			"path": attachment.Path,
		})
	}

	mimeType := attachment.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	effectiveDisposition := utils.AttachmentPreviewDisposition(mimeType, attachment.FileType, disposition)
	cacheControl := "public, max-age=3600"
	if effectiveDisposition == "attachment" {
		cacheControl = "no-cache, no-store, must-revalidate"
	}

	httpResp := ctx.Response().
		Header("Content-Type", mimeType).
		Header("Content-Length", fmt.Sprintf("%d", len(content))).
		Header("X-Content-Type-Options", "nosniff").
		Header("Cache-Control", cacheControl)

	if effectiveDisposition == "attachment" {
		filename := attachment.Filename
		if filename == "" {
			filename = attachment.Path
		}
		httpResp = httpResp.
			Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename)).
			Header("Pragma", "no-cache").
			Header("Expires", "0")
	} else if utils.AttachmentInlinePreviewSafe(mimeType, attachment.FileType) {
		httpResp = httpResp.Header("Content-Disposition", "inline")
	}

	if attachment.FileType == "image" || attachment.FileType == "video" {
		if utils.AttachmentInlinePreviewSafe(mimeType, attachment.FileType) {
			httpResp = httpResp.Header("Accept-Ranges", "bytes")
		}
	}

	return httpResp.String(http.StatusOK, content)
}
