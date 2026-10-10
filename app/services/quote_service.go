package services

import (
	"context"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"
	appfacades "goravel/app/facades"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/requests/admin"
	"goravel/app/models"
)

type QuoteService interface {
	GetByID(id uint) (*models.Quote, error)
	GetList(filters QuoteFilters, page, pageSize int) ([]models.Quote, int64, error)

	GetAllQuoteForExport(filters QuoteFilters) ([]models.Quote, error)

	Create(req *admin.QuoteCreate) (*models.Quote, error)

	Update(id uint, req *admin.QuoteUpdate) (*models.Quote, error)

	Delete(id uint) error
}

type QuoteFilters struct {
	QuoteNo        string
	CustomerName   string
	Status         string
	Remark         string
	CreatedAt      string
	CreatedAtStart string
	CreatedAtEnd   string
	UpdatedAt      string
	UpdatedAtStart string
	UpdatedAtEnd   string
}

// BuildQuoteFiltersFromHTTP is shared by list/export and reads filters from GET query or POST body.
func BuildQuoteFiltersFromHTTP(ctx http.Context) QuoteFilters {
	return QuoteFilters{
		QuoteNo:        ctx.Request().Input("quote_no", ctx.Request().Query("quote_no", "")),
		CustomerName:   ctx.Request().Input("customer_name", ctx.Request().Query("customer_name", "")),
		Status:         ctx.Request().Input("status", ctx.Request().Query("status", "")),
		Remark:         ctx.Request().Input("remark", ctx.Request().Query("remark", "")),
		CreatedAt:      ctx.Request().Input("created_at", ctx.Request().Query("created_at", "")),
		CreatedAtStart: helpers.GetTimeInputOrQueryParam(ctx, "created_at_start"),
		CreatedAtEnd:   helpers.GetTimeInputOrQueryParam(ctx, "created_at_end"),
		UpdatedAt:      ctx.Request().Input("updated_at", ctx.Request().Query("updated_at", "")),
		UpdatedAtStart: helpers.GetTimeInputOrQueryParam(ctx, "updated_at_start"),
		UpdatedAtEnd:   helpers.GetTimeInputOrQueryParam(ctx, "updated_at_end"),
	}
}

type QuoteServiceImpl struct {
	ctx context.Context
}

func NewQuoteService(ctx context.Context) QuoteService {
	return &QuoteServiceImpl{ctx: ctx}
}

func (s *QuoteServiceImpl) withRelations(query orm.Query) orm.Query {

	query = query.With("Details")

	return query
}

// BuildQuoteQuery builds the Quote query shared by list/export.
func BuildQuoteQuery(ctx context.Context, filters QuoteFilters) orm.Query {
	query := appfacades.OrmQuery(ctx).Model(&models.Quote{})
	if filters.QuoteNo != "" {
		query = query.Where("quote_no LIKE ?", "%"+filters.QuoteNo+"%")
	}
	if filters.CustomerName != "" {
		query = query.Where("customer_name LIKE ?", "%"+filters.CustomerName+"%")
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.Remark != "" {
		query = query.Where("remark LIKE ?", "%"+filters.Remark+"%")
	}
	if filters.CreatedAtStart != "" {
		query = query.Where("created_at >= ?", filters.CreatedAtStart)
	}
	if filters.CreatedAtEnd != "" {
		query = query.Where("created_at <= ?", filters.CreatedAtEnd)
	}
	if filters.CreatedAt != "" {
		query = query.Where("created_at = ?", filters.CreatedAt)
	}
	if filters.UpdatedAtStart != "" {
		query = query.Where("updated_at >= ?", filters.UpdatedAtStart)
	}
	if filters.UpdatedAtEnd != "" {
		query = query.Where("updated_at <= ?", filters.UpdatedAtEnd)
	}
	if filters.UpdatedAt != "" {
		query = query.Where("updated_at = ?", filters.UpdatedAt)
	}

	return query
}

func (s *QuoteServiceImpl) GetByID(id uint) (*models.Quote, error) {
	var item models.Quote
	query := s.withRelations(appfacades.OrmQuery(s.ctx).Model(&models.Quote{})).Where("id", id)
	if err := query.FirstOrFail(&item); err != nil {
		return nil, apperrors.ErrRecordNotFound.WithError(err)
	}
	return &item, nil
}

func (s *QuoteServiceImpl) GetList(filters QuoteFilters, page, pageSize int) ([]models.Quote, int64, error) {
	query := s.withRelations(BuildQuoteQuery(s.ctx, filters))

	var list []models.Quote
	var total int64
	if err := query.Order("id desc").Paginate(page, pageSize, &list, &total); err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (s *QuoteServiceImpl) GetAllQuoteForExport(filters QuoteFilters) ([]models.Quote, error) {
	query := s.withRelations(BuildQuoteQuery(s.ctx, filters))

	var list []models.Quote
	if err := query.Order("id desc").Find(&list); err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}

	return list, nil
}

func (s *QuoteServiceImpl) Create(req *admin.QuoteCreate) (*models.Quote, error) {
	item := &models.Quote{
		QuoteNo:      req.QuoteNo,
		CustomerName: req.CustomerName,
		Status:       req.Status,
		Remark:       req.Remark,
	}

	err := appfacades.OrmTransaction(s.ctx, func(tx orm.Query) error {
		if err := tx.Create(item); err != nil {
			return err
		}
		return s.syncQuoteDetails(tx, item.ID, req.Details)
	})
	if err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	return s.GetByID(item.ID)

}

func (s *QuoteServiceImpl) Update(id uint, req *admin.QuoteUpdate) (*models.Quote, error) {
	item, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if req.QuoteNo != nil {
		item.QuoteNo = *req.QuoteNo
	}
	if req.CustomerName != nil {
		item.CustomerName = *req.CustomerName
	}
	if req.Status != nil {
		item.Status = *req.Status
	}
	if req.Remark != nil {
		item.Remark = *req.Remark
	}

	err = appfacades.OrmTransaction(s.ctx, func(tx orm.Query) error {
		if err := tx.Save(item); err != nil {
			return err
		}
		if req.Details != nil {
			return s.syncQuoteDetails(tx, item.ID, *req.Details)
		}
		return nil
	})
	if err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	return s.GetByID(item.ID)

}

func (s *QuoteServiceImpl) Delete(id uint) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}

	err := appfacades.OrmTransaction(s.ctx, func(tx orm.Query) error {
		if _, err := tx.Where("quote_id", id).Delete(&models.QuoteItem{}); err != nil {
			return err
		}
		if _, err := tx.Where("id", id).Delete(&models.Quote{}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil

}

func isEmptyQuoteItemInput(d admin.QuoteItemInput) bool {
	{
		var zero string
		if d.ProductName != zero {
			return false
		}
	}
	{
		var zero int
		if d.Quantity != zero {
			return false
		}
	}
	{
		var zero float64
		if d.UnitPrice != zero {
			return false
		}
	}
	return true
}

func (s *QuoteServiceImpl) syncQuoteDetails(tx orm.Query, masterID uint, details []admin.QuoteItemInput) error {
	if _, err := tx.Where("quote_id", masterID).Delete(&models.QuoteItem{}); err != nil {
		return err
	}
	for i := range details {
		if isEmptyQuoteItemInput(details[i]) {
			continue
		}
		row := &models.QuoteItem{
			QuoteId:     masterID,
			ProductName: details[i].ProductName,
			Quantity:    details[i].Quantity,
			UnitPrice:   details[i].UnitPrice,
		}
		if err := tx.Create(row); err != nil {
			return err
		}
	}
	return nil
}
