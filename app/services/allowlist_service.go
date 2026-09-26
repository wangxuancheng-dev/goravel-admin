package services

import (
	"context"
	"strings"

	"github.com/dromara/carbon/v2"
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/http/requests/admin"
	"goravel/app/models"
	"goravel/app/utils"
)

type AllowlistService interface {
	GetByID(id uint) (*models.Allowlist, error)
	GetList(filters AllowlistFilters, page, pageSize int) ([]models.Allowlist, int64, error)
	Create(req *admin.AllowlistCreate, clientIP string) (*models.Allowlist, error)
	Update(id uint, req *admin.AllowlistUpdate, clientIP string) (*models.Allowlist, error)
	Delete(id uint, clientIP string) error
}

type AllowlistFilters struct {
	IP        string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

func BuildAllowlistFiltersFromHTTP(ctx http.Context) AllowlistFilters {
	return AllowlistFilters{
		IP:        ctx.Request().Input("ip", ctx.Request().Query("ip", "")),
		Status:    ctx.Request().Input("status", ctx.Request().Query("status", "")),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Input("order_by", ctx.Request().Query("order_by", "")),
	}
}

type AllowlistServiceImpl struct {
	ctx context.Context
}

func NewAllowlistService(ctx context.Context) AllowlistService {
	return &AllowlistServiceImpl{ctx: ctx}
}

func (s *AllowlistServiceImpl) validateIP(ip string) error {
	if err := utils.ValidateBlacklistIP(ip); err != nil {
		if businessErr, ok := apperrors.GetBusinessError(err); ok {
			return businessErr
		}
		return apperrors.ErrInvalidIPFormat
	}
	return nil
}

func (s *AllowlistServiceImpl) GetByID(id uint) (*models.Allowlist, error) {
	var row models.Allowlist
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&row); err != nil {
		return nil, apperrors.ErrAllowlistNotFound.WithError(err)
	}
	return &row, nil
}

func (s *AllowlistServiceImpl) GetList(filters AllowlistFilters, page, pageSize int) ([]models.Allowlist, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Allowlist{})
	if filters.IP != "" {
		query = query.Where("ip LIKE ?", "%"+filters.IP+"%")
	}
	if filters.Status != "" {
		query = query.Where("status", filters.Status)
	}
	if filters.StartTime != "" {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime != "" {
		query = query.Where("created_at <= ?", filters.EndTime)
	}
	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "id:desc"
	}
	query = helpers.ApplySort(query, orderBy, "id:desc")
	var list []models.Allowlist
	var total int64
	if err := query.Paginate(page, pageSize, &list, &total); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *AllowlistServiceImpl) enabledPatternsExcept(excludeID uint) ([]string, error) {
	if !appfacades.SchemaHasTable(s.ctx, "allowlists") {
		return nil, nil
	}
	query := appfacades.OrmQuery(s.ctx).Model(&models.Allowlist{}).Where("status", 1)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var rows []models.Allowlist
	if err := query.Get(&rows); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.IP != "" {
			out = append(out, row.IP)
		}
	}
	return out, nil
}

func ensureAllowlistKeepsClient(clientIP string, patterns []string) error {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return nil
	}
	if len(patterns) == 0 {
		return nil // unrestricted
	}
	if clientIPMatchesPatterns(clientIP, patterns) {
		return nil
	}
	return apperrors.ErrAllowlistLocksSelf.WithParams(map[string]any{"ip": clientIP})
}

func clientIPMatchesPatterns(clientIP string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern != "" && utils.IsIPInBlacklist(clientIP, pattern) {
			return true
		}
	}
	return false
}

func (s *AllowlistServiceImpl) Create(req *admin.AllowlistCreate, clientIP string) (*models.Allowlist, error) {
	if err := s.validateIP(req.IP); err != nil {
		return nil, err
	}
	patterns, err := s.enabledPatternsExcept(0)
	if err != nil {
		return nil, err
	}
	if req.Status == 1 {
		patterns = append(patterns, req.IP)
	}
	if err := ensureAllowlistKeepsClient(clientIP, patterns); err != nil {
		return nil, err
	}

	row := &models.Allowlist{}
	createData := map[string]any{
		"ip":         req.IP,
		"remark":     req.Remark,
		"status":     req.Status,
		"created_at": carbon.Now(),
		"updated_at": carbon.Now(),
	}
	if err := appfacades.OrmQuery(s.ctx).Model(row).Create(createData); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	InvalidateAllowlistCache(s.ctx)
	var created models.Allowlist
	if err := appfacades.OrmQuery(s.ctx).Where("ip", req.IP).OrderByDesc("id").First(&created); err != nil || created.ID == 0 {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	return &created, nil
}

func (s *AllowlistServiceImpl) Update(id uint, req *admin.AllowlistUpdate, clientIP string) (*models.Allowlist, error) {
	row, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	next := *row
	if req.IP != nil {
		if err := s.validateIP(*req.IP); err != nil {
			return nil, err
		}
		next.IP = *req.IP
	}
	if req.Remark != nil {
		next.Remark = *req.Remark
	}
	if req.Status != nil {
		next.Status = *req.Status
	}

	patterns, err := s.enabledPatternsExcept(id)
	if err != nil {
		return nil, err
	}
	if next.Status == 1 && next.IP != "" {
		patterns = append(patterns, next.IP)
	}
	if err := ensureAllowlistKeepsClient(clientIP, patterns); err != nil {
		return nil, err
	}

	*row = next
	if err := appfacades.OrmQuery(s.ctx).Save(row); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	InvalidateAllowlistCache(s.ctx)
	return row, nil
}

func (s *AllowlistServiceImpl) Delete(id uint, clientIP string) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	patterns, err := s.enabledPatternsExcept(id)
	if err != nil {
		return err
	}
	if err := ensureAllowlistKeepsClient(clientIP, patterns); err != nil {
		return err
	}
	if _, err := appfacades.OrmQuery(s.ctx).Where("id", id).Delete(&models.Allowlist{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	InvalidateAllowlistCache(s.ctx)
	return nil
}
