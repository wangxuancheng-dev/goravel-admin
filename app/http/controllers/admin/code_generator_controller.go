package admin

import (
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/services"
	"goravel/app/utils"
)

type CodeGeneratorController struct {}


type GenerateRequest struct {
	ModuleName      string                 `json:"module_name"`
	TableName       string                 `json:"table_name"`
	Fields          []services.FieldConfig `json:"fields"`
	Files           []string               `json:"files"`
	Options         map[string]bool        `json:"options"`
	DetailTableName string                 `json:"detail_table_name"`
	DetailFields    []services.FieldConfig `json:"detail_fields"`
}

type PreviewRequest struct {
	ModuleName      string                 `json:"module_name"`
	TableName       string                 `json:"table_name"`
	Fields          []services.FieldConfig `json:"fields"`
	FileType        string                 `json:"file_type"`
	Options         map[string]bool        `json:"options"`
	DetailTableName string                 `json:"detail_table_name"`
	DetailFields    []services.FieldConfig `json:"detail_fields"`
}

type SaveRequest struct {
	ModuleName      string                        `json:"module_name"`
	TableName       string                        `json:"table_name"`
	Fields          []services.FieldConfig        `json:"fields"`
	Force           bool                          `json:"force"`
	Files           []string                      `json:"files"`
	Options         map[string]bool               `json:"options"`
	Install         *services.ModuleInstallConfig `json:"install"`
	DetailTableName string                        `json:"detail_table_name"`
	DetailFields    []services.FieldConfig        `json:"detail_fields"`
}

type InstallModuleRequest struct {
	ModuleName string                        `json:"module_name"`
	TableName  string                        `json:"table_name"`
	Options    map[string]bool               `json:"options"`
	Install    *services.ModuleInstallConfig `json:"install"`
}

type GenerateWithAIRequest struct {
	Description string `json:"description"`
}

func NewCodeGeneratorController() *CodeGeneratorController {
	return &CodeGeneratorController{}
}

func (c *CodeGeneratorController) codeGeneratorService(ctx http.Context) services.CodeGeneratorService {
	return services.NewCodeGeneratorService(ctx)
}

func (c *CodeGeneratorController) applyMasterDetail(svc services.CodeGeneratorService, options map[string]bool, detailTable string, detailFields []services.FieldConfig) (services.CodeGeneratorService, string) {
	if err := services.ValidateMasterDetailRequest(options, detailTable, detailFields); err != nil {
		return svc, err.Error()
	}
	if options != nil && options["is_master_detail"] {
		return svc.WithMasterDetail(detailTable, detailFields), ""
	}
	return svc, ""
}

// Generate 生成CRUD代码
func (c *CodeGeneratorController) Generate(ctx http.Context) http.Response {
	var req GenerateRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, "invalid_fields")
	}

	if req.ModuleName == "" {
		return response.Error(ctx, http.StatusBadRequest, "module_name_required")
	}
	if req.TableName == "" {
		return response.Error(ctx, http.StatusBadRequest, "table_name_required")
	}
	if services.IsCodeGeneratorReservedTable(req.TableName) {
		return response.Error(ctx, http.StatusBadRequest, "system_table_not_allowed")
	}

	svc, errCode := c.applyMasterDetail(c.codeGeneratorService(ctx), req.Options, req.DetailTableName, req.DetailFields)
	if errCode != "" {
		return response.Error(ctx, http.StatusBadRequest, errCode)
	}

	files, err := svc.Generate(req.ModuleName, req.TableName, req.Fields, req.Files, req.Options)
	if err != nil {
		if businessErr, ok := apperrors.GetBusinessError(err); ok {
			return response.Error(ctx, http.StatusInternalServerError, businessErr.Code)
		}
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}

	return response.Success(ctx, http.Json{
		"files": files,
	})
}

// Preview 预览生成的代码
func (c *CodeGeneratorController) Preview(ctx http.Context) http.Response {
	var req PreviewRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, "invalid_fields")
	}

	if req.ModuleName == "" {
		return response.Error(ctx, http.StatusBadRequest, "module_name_required")
	}
	if req.TableName == "" {
		return response.Error(ctx, http.StatusBadRequest, "table_name_required")
	}
	if services.IsCodeGeneratorReservedTable(req.TableName) {
		return response.Error(ctx, http.StatusBadRequest, "system_table_not_allowed")
	}
	if req.FileType == "" {
		return response.Error(ctx, http.StatusBadRequest, "file_type_required")
	}

	svc, errCode := c.applyMasterDetail(c.codeGeneratorService(ctx), req.Options, req.DetailTableName, req.DetailFields)
	if errCode != "" {
		return response.Error(ctx, http.StatusBadRequest, errCode)
	}

	code, err := svc.Preview(req.ModuleName, req.TableName, req.Fields, req.FileType, req.Options)
	if err != nil {
		if businessErr, ok := apperrors.GetBusinessError(err); ok {
			return response.Error(ctx, http.StatusInternalServerError, businessErr.Code)
		}
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}

	return response.Success(ctx, http.Json{
		"code": code,
	})
}

// Save 保存生成的代码到文件系统
func (c *CodeGeneratorController) Save(ctx http.Context) http.Response {
	var req SaveRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, "invalid_fields")
	}

	if req.ModuleName == "" {
		return response.Error(ctx, http.StatusBadRequest, "module_name_required")
	}
	if req.TableName == "" {
		return response.Error(ctx, http.StatusBadRequest, "table_name_required")
	}
	if services.IsCodeGeneratorReservedTable(req.TableName) {
		return response.Error(ctx, http.StatusBadRequest, "system_table_not_allowed")
	}

	svc, errCode := c.applyMasterDetail(c.codeGeneratorService(ctx), req.Options, req.DetailTableName, req.DetailFields)
	if errCode != "" {
		return response.Error(ctx, http.StatusBadRequest, errCode)
	}

	var savedFiles []string
	var err error

	if req.Force {
		savedFiles, err = svc.ForceSave(req.ModuleName, req.TableName, req.Fields, req.Files, req.Options)
	} else {
		savedFiles, err = svc.Save(req.ModuleName, req.TableName, req.Fields, req.Files, req.Options)
	}

	if err != nil {
		if filesExistErr, ok := err.(*services.FilesExistError); ok {
			return ctx.Response().Json(409, http.Json{
				"code":       409,
				"message":    facades.Lang(ctx).Get("files_exist"),
				"error_code": "files_exist",
				"files":      filesExistErr.Files,
			})
		}
		if businessErr, ok := apperrors.GetBusinessError(err); ok {
			return response.Error(ctx, http.StatusInternalServerError, businessErr.Code)
		}
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}

	payload := http.Json{
		"saved_files": savedFiles,
	}
	if req.Install != nil && req.Install.Enabled {
		installResult, installErr := c.codeGeneratorService(ctx).InstallModule(req.ModuleName, req.TableName, req.Options, req.Install)
		if installErr != nil {
			if businessErr, ok := apperrors.GetBusinessError(installErr); ok {
				return response.Error(ctx, http.StatusInternalServerError, businessErr.Code)
			}
			return response.Error(ctx, http.StatusInternalServerError, installErr.Error())
		}
		payload["install"] = installResult
	}

	return response.Success(ctx, payload)
}

// InstallModule registers menu and permissions for a generated module.
func (c *CodeGeneratorController) InstallModule(ctx http.Context) http.Response {
	var req InstallModuleRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, "invalid_fields")
	}

	if req.ModuleName == "" {
		return response.Error(ctx, http.StatusBadRequest, "module_name_required")
	}
	if req.TableName == "" {
		return response.Error(ctx, http.StatusBadRequest, "table_name_required")
	}

	install := req.Install
	if install == nil {
		install = &services.ModuleInstallConfig{Enabled: true}
	}
	install.Enabled = true

	result, err := c.codeGeneratorService(ctx).InstallModule(req.ModuleName, req.TableName, req.Options, install)
	if err != nil {
		if businessErr, ok := apperrors.GetBusinessError(err); ok {
			return response.Error(ctx, http.StatusInternalServerError, businessErr.Code)
		}
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}

	return response.Success(ctx, http.Json{
		"install": result,
	})
}

// GetFieldTypes 获取支持的字段类型
func (c *CodeGeneratorController) GetFieldTypes(ctx http.Context) http.Response {
	fieldTypes := c.codeGeneratorService(ctx).GetFieldTypes()
	return response.Success(ctx, http.Json{
		"field_types": fieldTypes,
		"ai_enabled":  utils.AIEnabled(),
		"frontends":   utils.CodeGeneratorFrontends(),
	})
}

// GetTables 获取数据库表列表
func (c *CodeGeneratorController) GetTables(ctx http.Context) http.Response {
	tables, err := c.codeGeneratorService(ctx).GetTables()
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}
	return response.Success(ctx, http.Json{
		"tables": tables,
	})
}

// GetTableColumns 获取表字段
func (c *CodeGeneratorController) GetTableColumns(ctx http.Context) http.Response {
	tableName := ctx.Request().Query("table_name")
	if tableName == "" {
		return response.Error(ctx, http.StatusBadRequest, "table_name_required")
	}

	fields, err := c.codeGeneratorService(ctx).GetTableColumns(tableName)
	if err != nil {
		if err.Error() == "system_table_not_allowed" || err.Error() == "table_name_required" {
			return response.Error(ctx, http.StatusBadRequest, err.Error())
		}
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}
	return response.Success(ctx, http.Json{
		"fields": fields,
	})
}

// GenerateWithAI 使用 AI 生成模块配置
func (c *CodeGeneratorController) GenerateWithAI(ctx http.Context) http.Response {
	var req GenerateWithAIRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, "invalid_fields")
	}

	if req.Description == "" {
		return response.Error(ctx, http.StatusBadRequest, "description_required")
	}

	if !utils.AIEnabled() {
		return response.Error(ctx, http.StatusBadRequest, "ai_not_configured")
	}

	config, err := c.codeGeneratorService(ctx).GenerateWithAI(ctx, req.Description)
	if err != nil {
		// 检查是否是 JSON 解析错误
		errMsg := err.Error()
		if strings.Contains(errMsg, "parse AI response") || strings.Contains(errMsg, "parse") || strings.Contains(errMsg, "JSON") {
			// JSON 解析错误返回 400，并返回友好的错误信息
			return ctx.Response().Json(400, http.Json{
				"code":       400,
				"message":    errMsg,
				"error_code": "ai_response_parse_error",
			})
		}
		if businessErr, ok := apperrors.GetBusinessError(err); ok {
			return response.Error(ctx, http.StatusInternalServerError, businessErr.Code)
		}
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}

	return response.Success(ctx, http.Json{
		"config": config,
	})
}
