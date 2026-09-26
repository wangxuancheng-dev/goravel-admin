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

type BlacklistService interface {
	GetByID(id uint) (*models.Blacklist, error)
	GetList(filters BlacklistFilters, page, pageSize int) ([]models.Blacklist, int64, error)
	Create(req *admin.BlacklistCreate, clientIP string) (*models.Blacklist, error)
	Update(id uint, req *admin.BlacklistUpdate, clientIP string) (*models.Blacklist, error)
	Delete(id uint) error
}

type BlacklistFilters struct {
	IP        string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

// BuildBlacklistFiltersFromHTTP reads list filters from query or body.
func BuildBlacklistFiltersFromHTTP(ctx http.Context) BlacklistFilters {
	return BlacklistFilters{
		IP:        ctx.Request().Input("ip", ctx.Request().Query("ip", "")),
		Status:    ctx.Request().Input("status", ctx.Request().Query("status", "")),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Input("order_by", ctx.Request().Query("order_by", "")),
	}
}

type BlacklistServiceImpl struct {
	ctx context.Context
}

func NewBlacklistService(ctx context.Context) BlacklistService {
	return &BlacklistServiceImpl{ctx: ctx}
}

func (s *BlacklistServiceImpl) validateIP(ip string) error {
	if err := utils.ValidateBlacklistIP(ip); err != nil {
		if businessErr, ok := apperrors.GetBusinessError(err); ok {
			return businessErr
		}
		return apperrors.ErrInvalidIPFormat
	}
	return nil
}

func (s *BlacklistServiceImpl) GetByID(id uint) (*models.Blacklist, error) {
	var blacklist models.Blacklist
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&blacklist); err != nil {
		return nil, apperrors.ErrBlacklistNotFound.WithError(err)
	}
	return &blacklist, nil
}

func (s *BlacklistServiceImpl) GetList(filters BlacklistFilters, page, pageSize int) ([]models.Blacklist, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Blacklist{})

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

	var blacklists []models.Blacklist
	var total int64
	if err := query.Paginate(page, pageSize, &blacklists, &total); err != nil {
		return nil, 0, err
	}

	return blacklists, total, nil
}

func (s *BlacklistServiceImpl) enabledPatternsExcept(excludeID uint) ([]string, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Blacklist{}).Where("status", 1)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var rows []models.Blacklist
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

func ensureBlacklistKeepsClient(clientIP string, patterns []string) error {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return nil
	}
	if clientIPMatchesPatterns(clientIP, patterns) {
		return apperrors.ErrBlacklistLocksSelf.WithParams(map[string]any{"ip": clientIP})
	}
	return nil
}

func (s *BlacklistServiceImpl) Create(req *admin.BlacklistCreate, clientIP string) (*models.Blacklist, error) {
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
	if err := ensureBlacklistKeepsClient(clientIP, patterns); err != nil {
		return nil, err
	}

	blacklist := &models.Blacklist{}
	createData := map[string]any{
		"ip":         req.IP,
		"remark":     req.Remark,
		"status":     req.Status,
		"created_at": carbon.Now(),
		"updated_at": carbon.Now(),
	}

	if err := appfacades.OrmQuery(s.ctx).Model(blacklist).Create(createData); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	InvalidateBlacklistCache(s.ctx)

	var created models.Blacklist
	if err := appfacades.OrmQuery(s.ctx).Where("ip", req.IP).OrderByDesc("id").First(&created); err != nil || created.ID == 0 {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	return &created, nil
}

func (s *BlacklistServiceImpl) Update(id uint, req *admin.BlacklistUpdate, clientIP string) (*models.Blacklist, error) {
	blacklist, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	next := *blacklist
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
	if err := ensureBlacklistKeepsClient(clientIP, patterns); err != nil {
		return nil, err
	}

	*blacklist = next
	if err := appfacades.OrmQuery(s.ctx).Save(blacklist); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	InvalidateBlacklistCache(s.ctx)
	return blacklist, nil
}

func (s *BlacklistServiceImpl) Delete(id uint) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	if _, err := appfacades.OrmQuery(s.ctx).Where("id", id).Delete(&models.Blacklist{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	InvalidateBlacklistCache(s.ctx)
	return nil
}
