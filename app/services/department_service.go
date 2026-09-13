package services

import (
	"context"

	"github.com/dromara/carbon/v2"
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	admin "goravel/app/http/requests/admin"
	"goravel/app/models"
	"goravel/app/utils"
)

type DepartmentService interface {
	GetByID(id uint) (*models.Department, error)
	GetList(filters DepartmentFilters, page, pageSize int) ([]models.Department, int64, error)
	GetIndex(filters DepartmentFilters) (any, error)
	HasAdmins(departmentID uint) (bool, error)
	TransferAdmins(fromDeptID, toDeptID uint) (int64, error)
	Create(req *admin.DepartmentCreate) (*models.Department, error)
	Update(id uint, req *admin.DepartmentUpdate) (*models.Department, error)
	Delete(id uint) error
}

// DepartmentFilters 部门查询过滤器
type DepartmentFilters struct {
	Name      string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

func BuildDepartmentFiltersFromHTTP(ctx http.Context) DepartmentFilters {
	return DepartmentFilters{
		Name:      ctx.Request().Query("name", ""),
		Status:    ctx.Request().Query("status", ""),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Query("order_by", ""),
	}
}

type DepartmentServiceImpl struct {
	ctx context.Context
}

func NewDepartmentService(ctx context.Context) DepartmentService {
	return &DepartmentServiceImpl{ctx: ctx}
}

func (s *DepartmentServiceImpl) treeService() TreeService {
	return NewTreeServiceImpl(s.ctx)
}

func (s *DepartmentServiceImpl) GetByID(id uint) (*models.Department, error) {
	var department models.Department
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&department); err != nil {
		return nil, apperrors.ErrDepartmentNotFound.WithError(err)
	}
	return &department, nil
}

func (s *DepartmentServiceImpl) GetList(filters DepartmentFilters, page, pageSize int) ([]models.Department, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Department{})

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

	var departments []models.Department
	var total int64
	if err := query.Paginate(page, pageSize, &departments, &total); err != nil {
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
	}

	return departments, total, nil
}

func (s *DepartmentServiceImpl) GetIndex(filters DepartmentFilters) (any, error) {
	if filters.Name != "" || filters.Status != "" || filters.StartTime != "" || filters.EndTime != "" {
		departments, _, err := s.GetList(filters, 1, 10000)
		if err != nil {
			return nil, err
		}
		return departments, nil
	}

	departments, err := s.treeService().BuildDepartmentTree(0)
	if err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}
	return utils.ConvertDepartmentTree(departments), nil
}

func (s *DepartmentServiceImpl) HasAdmins(departmentID uint) (bool, error) {
	count, err := appfacades.OrmQuery(s.ctx).Model(&models.Admin{}).Where("department_id", departmentID).Count()
	if err != nil {
		return false, apperrors.ErrQueryFailed.WithError(err)
	}
	return count > 0, nil
}

// TransferAdmins moves all admins from fromDeptID to toDeptID.
func (s *DepartmentServiceImpl) TransferAdmins(fromDeptID, toDeptID uint) (int64, error) {
	if fromDeptID == 0 || toDeptID == 0 {
		return 0, apperrors.ErrParamsError
	}
	if fromDeptID == toDeptID {
		return 0, apperrors.ErrParamsError.WithMessage("source and target department must differ")
	}
	if _, err := s.GetByID(fromDeptID); err != nil {
		return 0, err
	}
	if _, err := s.GetByID(toDeptID); err != nil {
		return 0, err
	}

	result, err := appfacades.OrmQuery(s.ctx).Model(&models.Admin{}).
		Where("department_id", fromDeptID).
		Update("department_id", toDeptID)
	if err != nil {
		return 0, apperrors.ErrUpdateFailed.WithError(err)
	}
	if result == nil {
		return 0, nil
	}
	return result.RowsAffected, nil
}

func (s *DepartmentServiceImpl) Create(req *admin.DepartmentCreate) (*models.Department, error) {
	department := &models.Department{}
	createData := map[string]any{
		"parent_id":  req.ParentID,
		"name":       req.Name,
		"code":       req.Code,
		"leader":     req.Leader,
		"phone":      req.Phone,
		"email":      req.Email,
		"remark":     req.Remark,
		"status":     req.Status,
		"sort":       req.Sort,
		"created_at": carbon.Now(),
		"updated_at": carbon.Now(),
	}

	if err := appfacades.OrmQuery(s.ctx).Model(department).Create(createData); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	return department, nil
}

func (s *DepartmentServiceImpl) Update(id uint, req *admin.DepartmentUpdate) (*models.Department, error) {
	department, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		department.Name = *req.Name
	}
	if req.ParentID != nil {
		department.ParentID = *req.ParentID
	}
	if req.Code != nil {
		department.Code = *req.Code
	}
	if req.Leader != nil {
		department.Leader = *req.Leader
	}
	if req.Phone != nil {
		department.Phone = *req.Phone
	}
	if req.Email != nil {
		department.Email = *req.Email
	}
	if req.Status != nil {
		department.Status = *req.Status
	}
	if req.Sort != nil {
		department.Sort = *req.Sort
	}
	if req.Remark != nil {
		department.Remark = *req.Remark
	}

	if err := appfacades.OrmQuery(s.ctx).Save(department); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	return department, nil
}

func (s *DepartmentServiceImpl) Delete(id uint) error {
	department, err := s.GetByID(id)
	if err != nil {
		return err
	}

	hasChildren, err := s.treeService().HasDepartmentChildren(id)
	if err != nil {
		return apperrors.ErrQueryFailed.WithError(err)
	}
	if hasChildren {
		return apperrors.ErrDepartmentHasChildren
	}

	hasAdmins, err := s.HasAdmins(id)
	if err != nil {
		return err
	}
	if hasAdmins {
		return apperrors.ErrDepartmentHasAdmins
	}

	if _, err := appfacades.OrmQuery(s.ctx).Delete(department); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}
