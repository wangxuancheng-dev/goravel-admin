package services

import (
	"context"

	"github.com/goravel/framework/contracts/http"

	appfacades "goravel/app/facades"

	"github.com/dromara/carbon/v2"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/requests/admin"
	"goravel/app/models"
)

type PositionService interface {
	GetByID(id uint) (*models.Position, error)
	GetList(filters PositionFilters, page, pageSize int) ([]models.Position, int64, error)
	Create(req *admin.PositionCreate) (*models.Position, error)
	Update(id uint, req *admin.PositionUpdate) (*models.Position, error)
	Delete(id uint) error
}

type PositionFilters struct {
	Name      string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

// BuildPositionFiltersFromHTTP reads list/export filters from query or body.
func BuildPositionFiltersFromHTTP(ctx http.Context) PositionFilters {
	return PositionFilters{
		Name:      ctx.Request().Input("name", ctx.Request().Query("name", "")),
		Status:    ctx.Request().Input("status", ctx.Request().Query("status", "")),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Input("order_by", ctx.Request().Query("order_by", "")),
	}
}

type PositionServiceImpl struct {
	ctx context.Context
}

func NewPositionService(ctx context.Context) PositionService {
	return &PositionServiceImpl{ctx: ctx}
}

func (s *PositionServiceImpl) GetByID(id uint) (*models.Position, error) {
	var position models.Position
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&position); err != nil {
		return nil, apperrors.ErrPositionNotFound.WithError(err)
	}
	return &position, nil
}

func (s *PositionServiceImpl) GetList(filters PositionFilters, page, pageSize int) ([]models.Position, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Position{})
	if filters.Name != "" {
		query = query.Where("name LIKE ?", "%"+filters.Name+"%")
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
		orderBy = "sort:asc,id:asc"
	}
	query = helpers.ApplySort(query, orderBy, "sort:asc,id:asc")
	var list []models.Position
	var total int64
	if err := query.Paginate(page, pageSize, &list, &total); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *PositionServiceImpl) hasAdmins(positionID uint) (bool, error) {
	count, err := appfacades.OrmQuery(s.ctx).Model(&models.Admin{}).Where("position_id", positionID).Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *PositionServiceImpl) Create(req *admin.PositionCreate) (*models.Position, error) {
	position := &models.Position{}
	createData := map[string]any{
		"name":       req.Name,
		"code":       req.Code,
		"remark":     req.Remark,
		"status":     req.Status,
		"sort":       req.Sort,
		"created_at": carbon.Now(),
		"updated_at": carbon.Now(),
	}
	if err := appfacades.OrmQuery(s.ctx).Model(position).Create(createData); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	// map Create may not populate primary key — reload by unique code or name.
	var created models.Position
	q := appfacades.OrmQuery(s.ctx)
	if req.Code != "" {
		q = q.Where("code", req.Code)
	} else {
		q = q.Where("name", req.Name)
	}
	if err := q.OrderByDesc("id").First(&created); err != nil || created.ID == 0 {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	return &created, nil
}

func (s *PositionServiceImpl) Update(id uint, req *admin.PositionUpdate) (*models.Position, error) {
	position, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		position.Name = *req.Name
	}
	if req.Code != nil {
		position.Code = *req.Code
	}
	if req.Status != nil {
		position.Status = *req.Status
	}
	if req.Sort != nil {
		position.Sort = *req.Sort
	}
	if req.Remark != nil {
		position.Remark = *req.Remark
	}
	if err := appfacades.OrmQuery(s.ctx).Save(position); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	return position, nil
}

func (s *PositionServiceImpl) Delete(id uint) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	hasAdmins, err := s.hasAdmins(id)
	if err != nil {
		return apperrors.ErrQueryFailed.WithError(err)
	}
	if hasAdmins {
		return apperrors.ErrPositionHasAdmins
	}
	if _, err := appfacades.OrmQuery(s.ctx).Where("id", id).Delete(&models.Position{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}
