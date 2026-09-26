package services

import (
	"context"
	"strings"

	"github.com/goravel/framework/contracts/http"

	appfacades "goravel/app/facades"

	"github.com/dromara/carbon/v2"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/requests/admin"
	"goravel/app/models"
)

type DictionaryService interface {
	GetByID(id uint) (*models.Dictionary, error)
	GetList(filters DictionaryFilters, page, pageSize int) ([]models.Dictionary, int64, error)
	GetByType(dictType string) ([]models.Dictionary, error)
	GetAllTypes() ([]string, error)
	Create(req *admin.DictionaryCreate) (*models.Dictionary, error)
	Update(id uint, req *admin.DictionaryUpdate) (*models.Dictionary, error)
	Delete(id uint) error
}

type DictionaryFilters struct {
	Type      string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

// BuildDictionaryFiltersFromHTTP reads list filters from query or body.
func BuildDictionaryFiltersFromHTTP(ctx http.Context) DictionaryFilters {
	return DictionaryFilters{
		Type:      ctx.Request().Input("type", ctx.Request().Query("type", "")),
		Status:    ctx.Request().Input("status", ctx.Request().Query("status", "")),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Input("order_by", ctx.Request().Query("order_by", "")),
	}
}

type DictionaryServiceImpl struct {
	ctx context.Context
}

func NewDictionaryService(ctx context.Context) DictionaryService {
	return &DictionaryServiceImpl{ctx: ctx}
}

func (s *DictionaryServiceImpl) GetByID(id uint) (*models.Dictionary, error) {
	var dictionary models.Dictionary
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&dictionary); err != nil {
		return nil, apperrors.ErrDictionaryNotFound.WithError(err)
	}
	return &dictionary, nil
}

func (s *DictionaryServiceImpl) GetList(filters DictionaryFilters, page, pageSize int) ([]models.Dictionary, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Dictionary{})

	if filters.Type != "" {
		query = query.Where("type", filters.Type)
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
		orderBy = "is_system:desc,sort:asc,id:desc"
	}
	query = helpers.ApplySort(query, orderBy, "is_system:desc,sort:asc,id:desc")

	var dictionaries []models.Dictionary
	var total int64
	if err := query.Paginate(page, pageSize, &dictionaries, &total); err != nil {
		return nil, 0, err
	}

	return dictionaries, total, nil
}

func (s *DictionaryServiceImpl) GetByType(dictType string) ([]models.Dictionary, error) {
	var dictionaries []models.Dictionary
	if err := appfacades.OrmQuery(s.ctx).
		Where("type", dictType).
		Where("status", 1).
		Order("sort asc, id asc").
		Find(&dictionaries); err != nil {
		return nil, err
	}
	return dictionaries, nil
}

func (s *DictionaryServiceImpl) GetAllTypes() ([]string, error) {
	var types []string
	if err := appfacades.OrmQuery(s.ctx).Model(&models.Dictionary{}).Distinct("type").Order("type asc").Pluck("type", &types); err != nil {
		return []string{}, nil
	}
	return types, nil
}

func (s *DictionaryServiceImpl) assertTypeValueUnique(dictType, value string, excludeID uint) error {
	dictType = strings.TrimSpace(dictType)
	value = strings.TrimSpace(value)
	query := appfacades.OrmQuery(s.ctx).Model(&models.Dictionary{}).Where("type", dictType).Where("value", value)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	count, err := query.Count()
	if err != nil {
		return apperrors.ErrCreateFailed.WithError(err)
	}
	if count > 0 {
		return apperrors.ErrDictionaryAlreadyExists
	}
	return nil
}

func (s *DictionaryServiceImpl) Create(req *admin.DictionaryCreate) (*models.Dictionary, error) {
	dictType := strings.TrimSpace(req.Type)
	value := strings.TrimSpace(req.Value)
	if err := s.assertTypeValueUnique(dictType, value, 0); err != nil {
		return nil, err
	}

	dictionary := &models.Dictionary{}
	createData := map[string]any{
		"type":            dictType,
		"label":           strings.TrimSpace(req.Label),
		"value":           value,
		"translation_key": req.TranslationKey,
		"description":     req.Description,
		"remark":          req.Remark,
		"status":          req.Status,
		"is_system":       0,
		"sort":            req.Sort,
		"created_at":      carbon.Now(),
		"updated_at":      carbon.Now(),
	}

	if err := appfacades.OrmQuery(s.ctx).Model(dictionary).Create(createData); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	// map Create may not populate primary key — reload by type+value.
	var created models.Dictionary
	if err := appfacades.OrmQuery(s.ctx).Where("type", dictType).Where("value", value).OrderByDesc("id").First(&created); err != nil || created.ID == 0 {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	return &created, nil
}

func (s *DictionaryServiceImpl) Update(id uint, req *admin.DictionaryUpdate) (*models.Dictionary, error) {
	dictionary, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	nextType := dictionary.Type
	nextValue := dictionary.Value
	if req.Type != nil {
		nextType = strings.TrimSpace(*req.Type)
	}
	if req.Value != nil {
		nextValue = strings.TrimSpace(*req.Value)
	}

	if dictionary.IsSystem == 1 {
		if nextType != dictionary.Type || nextValue != dictionary.Value {
			return nil, apperrors.ErrDictionaryProtectedCannotModifyKey
		}
		if req.Status != nil && *req.Status != 1 {
			return nil, apperrors.ErrDictionaryProtectedCannotDisable
		}
	}

	if nextType != dictionary.Type || nextValue != dictionary.Value {
		if err := s.assertTypeValueUnique(nextType, nextValue, dictionary.ID); err != nil {
			return nil, err
		}
	}

	if req.Type != nil {
		dictionary.Type = nextType
	}
	if req.Label != nil {
		dictionary.Label = strings.TrimSpace(*req.Label)
	}
	if req.Value != nil {
		dictionary.Value = nextValue
	}
	if req.TranslationKey != nil {
		dictionary.TranslationKey = *req.TranslationKey
	}
	if req.Description != nil {
		dictionary.Description = *req.Description
	}
	if req.Status != nil {
		dictionary.Status = *req.Status
	}
	if req.Sort != nil {
		dictionary.Sort = *req.Sort
	}
	if req.Remark != nil {
		dictionary.Remark = *req.Remark
	}
	if dictionary.IsSystem == 1 {
		dictionary.IsSystem = 1
		dictionary.Status = 1
	}
	if err := appfacades.OrmQuery(s.ctx).Save(dictionary); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	return dictionary, nil
}

func (s *DictionaryServiceImpl) Delete(id uint) error {
	dictionary, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if dictionary.IsSystem == 1 {
		return apperrors.ErrDictionaryProtectedCannotDelete
	}
	if _, err := appfacades.OrmQuery(s.ctx).Where("id", id).Delete(&models.Dictionary{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}
