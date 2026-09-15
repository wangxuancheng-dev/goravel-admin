package services

import (
	"context"
<<if .HasImport>>
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/spf13/cast"
<<end>>

	appfacades "goravel/app/facades"
	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/requests/admin"
	"goravel/app/models"
)

type <<.ServiceName>> interface {
	GetByID(id uint) (*models.<<.ModelName>>, error)
	GetList(filters <<.ModelName>>Filters, page, pageSize int) ([]models.<<.ModelName>>, int64, error)
<<if .IsTreeList>>
	GetTree() ([]models.<<.ModelName>>, error)
<<end>>
<<if .HasExport>>
	GetAll<<.ModelName>>ForExport(filters <<.ModelName>>Filters) ([]models.<<.ModelName>>, error)
<<end>>
<<if .HasImport>>
	ImportFromCSV(csvContent string) (*ImportResult, error)
<<end>>
<<if .HasCreate>>
	Create(req *admin.<<.RequestCreateName>>) (*models.<<.ModelName>>, error)
<<end>>
<<if .HasEdit>>
	Update(id uint, req *admin.<<.RequestUpdateName>>) (*models.<<.ModelName>>, error)
<<end>>
<<if .HasDelete>>
	Delete(id uint) error
<<end>>
}

type <<.ModelName>>Filters struct {
<<range .SearchableFields>>
	<<.PascalName>> string
	<<- if or (eq .SearchUIType "daterange") (eq .SearchUIType "datetimerange")>>
	<<.PascalName>>Start string
	<<.PascalName>>End   string
	<<- end>>
<<- end>>
}

// Build<<.ModelName>>FiltersFromHTTP is shared by list/export and reads filters from GET query or POST body.
func Build<<.ModelName>>FiltersFromHTTP(ctx http.Context) <<.ModelName>>Filters {
	return <<.ModelName>>Filters{
<<- range .SearchableFields>>
		<<.PascalName>>: ctx.Request().Input("<<.Name>>", ctx.Request().Query("<<.Name>>", "")),
		<<- if or (eq .SearchUIType "daterange") (eq .SearchUIType "datetimerange")>>
		<<.PascalName>>Start: helpers.GetTimeInputOrQueryParam(ctx, "<<.Name>>_start"),
		<<.PascalName>>End: helpers.GetTimeInputOrQueryParam(ctx, "<<.Name>>_end"),
		<<- end>>
<<- end>>
	}
}

type <<.ServiceName>>Impl struct {
	ctx context.Context
}

func New<<.ServiceName>>(ctx context.Context) <<.ServiceName>> {
	return &<<.ServiceName>>Impl{ctx: ctx}
}

func (s *<<.ServiceName>>Impl) withRelations(query orm.Query) orm.Query {
<<- range .FormFields>>
<<- if .Relation>>
	query = query.With("<<.Relation.Name>>")
<<- end>>
<<- end>>
	return query
}

// Build<<.ModelName>>Query builds the <<.ModelName>> query shared by list/export.
func Build<<.ModelName>>Query(ctx context.Context, filters <<.ModelName>>Filters) orm.Query {
	query := appfacades.OrmQuery(ctx).Model(&models.<<.ModelName>>{})
<<- range .SearchableFields>>
	<<- if or (eq .SearchUIType "daterange") (eq .SearchUIType "datetimerange")>>
	if filters.<<.PascalName>>Start != "" {
		query = query.Where("<<.Name>> >= ?", filters.<<.PascalName>>Start)
	}
	if filters.<<.PascalName>>End != "" {
		query = query.Where("<<.Name>> <= ?", filters.<<.PascalName>>End)
	}
	<<- end>>
	if filters.<<.PascalName>> != "" {
		<<- if eq .SearchType "like">>
		query = query.Where("<<.Name>> LIKE ?", "%"+filters.<<.PascalName>>+"%")
		<<- else if eq .SearchType "=">>
		query = query.Where("<<.Name>> = ?", filters.<<.PascalName>>)
		<<- else if eq .SearchType ">" >>
		query = query.Where("<<.Name>> > ?", filters.<<.PascalName>>)
		<<- else if eq .SearchType ">=" >>
		query = query.Where("<<.Name>> >= ?", filters.<<.PascalName>>)
		<<- else if eq .SearchType "<" >>
		query = query.Where("<<.Name>> < ?", filters.<<.PascalName>>)
		<<- else if eq .SearchType "<=" >>
		query = query.Where("<<.Name>> <= ?", filters.<<.PascalName>>)
		<<- else if eq .SearchType "!=" >>
		query = query.Where("<<.Name>> != ?", filters.<<.PascalName>>)
		<<- else if eq .SearchType "in">>
		query = query.Where("<<.Name>> IN ?", filters.<<.PascalName>>)
		<<- else>>
		query = query.Where("<<.Name>> LIKE ?", "%"+filters.<<.PascalName>>+"%")
		<<- end>>
	}
<<- end>>

	return query
}

func (s *<<.ServiceName>>Impl) GetByID(id uint) (*models.<<.ModelName>>, error) {
	var item models.<<.ModelName>>
	query := s.withRelations(appfacades.OrmQuery(s.ctx).Model(&models.<<.ModelName>>{})).Where("id", id)
	if err := query.FirstOrFail(&item); err != nil {
		return nil, apperrors.ErrRecordNotFound.WithError(err)
	}
	return &item, nil
}

func (s *<<.ServiceName>>Impl) GetList(filters <<.ModelName>>Filters, page, pageSize int) ([]models.<<.ModelName>>, int64, error) {
	query := s.withRelations(Build<<.ModelName>>Query(s.ctx, filters))

	var list []models.<<.ModelName>>
	var total int64
	if err := query.Order("id desc").Paginate(page, pageSize, &list, &total); err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

<<if .IsTreeList>>
func (s *<<.ServiceName>>Impl) GetTree() ([]models.<<.ModelName>>, error) {
	var all []models.<<.ModelName>>
	query := s.withRelations(appfacades.OrmQuery(s.ctx).Model(&models.<<.ModelName>>{}))
	if err := query.Order("sort asc, id asc").Get(&all); err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}

	byParent := make(map[uint][]models.<<.ModelName>>)
	for _, item := range all {
		byParent[item.<<.ParentIDFieldName>>] = append(byParent[item.<<.ParentIDFieldName>>], item)
	}

	return s.build<<.ModelName>>Tree(byParent, 0), nil
}

func (s *<<.ServiceName>>Impl) build<<.ModelName>>Tree(byParent map[uint][]models.<<.ModelName>>, parentID uint) []models.<<.ModelName>> {
	children := byParent[parentID]
	for i := range children {
		children[i].Children = s.build<<.ModelName>>Tree(byParent, children[i].ID)
	}
	return children
}

<<end>>

<<if .HasExport>>
func (s *<<.ServiceName>>Impl) GetAll<<.ModelName>>ForExport(filters <<.ModelName>>Filters) ([]models.<<.ModelName>>, error) {
	query := s.withRelations(Build<<.ModelName>>Query(s.ctx, filters))

	var list []models.<<.ModelName>>
	if err := query.Order("id desc").Find(&list); err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}

	return list, nil
}
<<end>>

<<if .HasImport>>
// ImportFromCSV imports <<.ModelName>> rows from CSV content.
// Header names should match field names (case-insensitive). Customize as needed.
func (s *<<.ServiceName>>Impl) ImportFromCSV(csvContent string) (*ImportResult, error) {
	reader := csv.NewReader(strings.NewReader(csvContent))
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, apperrors.ErrInvalidCSVFormat.WithError(err)
	}
	if len(records) < 2 {
		return nil, apperrors.ErrInvalidCSVFormat.WithMessage("CSV文件至少需要表头和数据行")
	}

	headerMap := make(map[string]int)
	for i, header := range records[0] {
		headerMap[strings.TrimSpace(strings.ToLower(header))] = i
	}

	result := &ImportResult{
		TotalRows: len(records) - 1,
		Errors:    []string{},
	}

	for rowIndex, row := range records[1:] {
		lineNo := rowIndex + 2
		if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
			result.TotalRows--
			continue
		}

		item := &models.<<.ModelName>>{}
<<- range .FormFields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at") .ShowInForm>>
		if idx, ok := headerMap["<<.Name>>"]; ok && idx < len(row) {
			val := strings.TrimSpace(row[idx])
			<<- if and .Relation (eq .Relation.RelationType "belongsTo")>>
			item.<<.FieldName>> = cast.ToUint(val)
			<<- else if eq .GoType "string">>
			item.<<.FieldName>> = val
			<<- else if eq .GoType "uint8">>
			item.<<.FieldName>> = uint8(cast.ToUint(val))
			<<- else if eq .GoType "uint64">>
			item.<<.FieldName>> = cast.ToUint64(val)
			<<- else if eq .GoType "uint">>
			item.<<.FieldName>> = cast.ToUint(val)
			<<- else if eq .GoType "int64">>
			item.<<.FieldName>> = cast.ToInt64(val)
			<<- else if eq .GoType "int">>
			item.<<.FieldName>> = cast.ToInt(val)
			<<- else if eq .GoType "float64">>
			item.<<.FieldName>> = cast.ToFloat64(val)
			<<- else if eq .GoType "bool">>
			item.<<.FieldName>> = cast.ToBool(val)
			<<- else>>
			_ = val // unsupported type <<.GoType>> for <<.Name>>; set manually if needed
			<<- end>>
		}
<<- end>>
<<- end>>

		if err := appfacades.OrmQuery(s.ctx).Create(item); err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行：%v", lineNo, err))
			continue
		}
		result.SuccessCount++
	}

	return result, nil
}
<<end>>

<<if .HasCreate>>
func (s *<<.ServiceName>>Impl) Create(req *admin.<<.RequestCreateName>>) (*models.<<.ModelName>>, error) {
	item := &models.<<.ModelName>>{
<<- range .FormFields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
	<<- if and .Relation (eq .Relation.RelationType "belongsTo")>>
		<<.FieldName>>: uint(req.<<.FieldName>>),
	<<- else>>
		<<.FieldName>>: req.<<.FieldName>>,
	<<- end>>
<<- end>>
<<- end>>
	}

	if err := appfacades.OrmQuery(s.ctx).Create(item); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	return item, nil
}
<<end>>

<<if .HasEdit>>
func (s *<<.ServiceName>>Impl) Update(id uint, req *admin.<<.RequestUpdateName>>) (*models.<<.ModelName>>, error) {
	item, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

<<- range .FormFields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
	if req.<<.FieldName>> != nil {
	<<- if and .Relation (eq .Relation.RelationType "belongsTo")>>
		item.<<.FieldName>> = uint(*req.<<.FieldName>>)
	<<- else>>
		item.<<.FieldName>> = *req.<<.FieldName>>
	<<- end>>
	}
<<- end>>
<<- end>>

	if err := appfacades.OrmQuery(s.ctx).Save(item); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}

	return item, nil
}
<<end>>

<<if .HasDelete>>
func (s *<<.ServiceName>>Impl) Delete(id uint) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}

	if _, err := appfacades.OrmQuery(s.ctx).Where("id", id).Delete(&models.<<.ModelName>>{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}
<<end>>
