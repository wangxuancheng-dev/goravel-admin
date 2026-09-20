package admin

import (
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/http/trans"
	"goravel/app/jobs"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
)

type OrderController struct{}

func NewOrderController() *OrderController {
	return &OrderController{}
}

func (r *OrderController) orderService(ctx http.Context) services.OrderService {
	return services.NewOrderService(ctx)
}

// resolveOrderNo 分表场景以 order_no 为业务主键：优先 query/body，其次 Resource 路由 {id}。
func (r *OrderController) resolveOrderNo(ctx http.Context) string {
	orderNo := strings.TrimSpace(ctx.Request().Input("order_no", ctx.Request().Query("order_no", "")))
	if orderNo != "" {
		return orderNo
	}
	routeID := strings.TrimSpace(ctx.Request().Route("id"))
	if routeID == "" || routeID == "0" {
		return ""
	}
	return routeID
}

// buildFilters builds filters shared by list/export endpoints.
func (r *OrderController) buildFilters(ctx http.Context) (services.OrderFilters, http.Response) {
	filters, err := services.BuildOrderFiltersFromHTTP(ctx)
	if err != nil {
		return services.OrderFilters{}, response.Error(ctx, http.StatusBadRequest, err.Error())
	}
	if resp := validateTimeRangeResponse(ctx, filters.StartTime, filters.EndTime, 3); resp != nil {
		return services.OrderFilters{}, resp
	}
	return filters, nil
}

// Index returns paginated order list.
// @Summary      Get order list
// @Description  Returns paginated orders with filters; time range is limited. Lookup key for show/update/delete is order_no.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Param        page       query    int     false "Page number" default(1)
// @Param        page_size  query    int     false "Page size" default(10)
// @Param        user_id    query    int     false "User ID"
// @Param        order_no   query    string  false "Order number (exact)"
// @Param        status     query    string  false "Order status"
// @Param        min_amount query    float64 false "Minimum amount"
// @Param        max_amount query    float64 false "Maximum amount"
// @Param        start_time query    string  false "Start time (2006-01-02 15:04:05)"
// @Param        end_time   query    string  false "End time (2006-01-02 15:04:05)"
// @Param        order_by   query    string  false "Sort field:direction"
// @Success      200        {object} apidoc.Success
// @Failure      400        {object} apidoc.Error "Bad request"
// @Failure      500        {object} apidoc.Error "Server error"
// @Router       /api/admin/orders [get]
// @Security     BearerAuth
func (r *OrderController) Index(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})

	filters, resp := r.buildFilters(ctx)
	if resp != nil {
		return resp
	}

	ordersWithDetails, total, err := r.orderService(ctx).GetOrdersWithDetails(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "order", http.StatusBadRequest, err, map[string]any{
			"filters": filters,
		})
	}

	svc := r.orderService(ctx)
	orderList := make([]http.Json, len(ordersWithDetails))
	for i := range ordersWithDetails {
		orderList[i] = svc.OrderWithDetailsToJSON(&ordersWithDetails[i])
	}

	return response.Success(ctx, http.Json{
		"list":      orderList,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Show returns one order by order_no (query preferred; Resource {id} also treated as order_no).
// @Summary      Get order detail
// @Description  Sharded lookup by order_no (not table-local numeric id).
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Param        id        path   string true  "order_no (Resource path)"
// @Param        order_no  query  string false "order_no (preferred)"
// @Success      200       {object} apidoc.Success
// @Failure      400       {object} apidoc.Error
// @Failure      404       {object} apidoc.Error
// @Router       /api/admin/orders/{id} [get]
// @Security     BearerAuth
func (r *OrderController) Show(ctx http.Context) http.Response {
	orderNo := r.resolveOrderNo(ctx)
	if orderNo == "" {
		return response.Error(ctx, http.StatusBadRequest, "order_no_required")
	}

	order, details, err := r.orderService(ctx).GetOrderByOrderNo(orderNo)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "order", http.StatusNotFound, err, map[string]any{
			"order_no": orderNo,
		})
	}
	return r.buildOrderDetailResponse(ctx, order, details)
}

func (r *OrderController) buildOrderDetailResponse(ctx http.Context, order *models.Order, details []models.OrderDetail) http.Response {
	svc := r.orderService(ctx)
	orderJson := svc.OrderToJSON(*order)

	detailList := make([]http.Json, len(details))
	for i, detail := range details {
		detailList[i] = svc.OrderDetailToJSON(detail)
	}

	return response.Success(ctx, http.Json{
		"order":   orderJson,
		"details": detailList,
	})
}

func (r *OrderController) Store(ctx http.Context) http.Response {
	var req adminrequests.OrderCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}
	if len(req.Products) == 0 {
		return response.Error(ctx, http.StatusBadRequest, "empty_products")
	}

	products := make([]services.OrderProduct, len(req.Products))
	for i, p := range req.Products {
		products[i] = services.OrderProduct{
			ProductID:   p.ProductID,
			ProductName: p.ProductName,
			Price:       p.Price,
			Quantity:    p.Quantity,
		}
	}

	var expireArgs []time.Duration
	if req.ExpireInSeconds > 0 {
		expireArgs = append(expireArgs, time.Duration(req.ExpireInSeconds)*time.Second)
	}
	order, details, err := r.orderService(ctx).CreateOrder(req.UserID, req.Amount, products, req.RequestID, req.Remark, expireArgs...)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "order", http.StatusBadRequest, err, map[string]any{
			"user_id": req.UserID,
		})
	}

	svc := r.orderService(ctx)
	orderJson := svc.OrderToJSON(*order)
	delete(orderJson, "created_at")
	delete(orderJson, "updated_at")

	detailList := make([]http.Json, len(details))
	for i, detail := range details {
		item := svc.OrderDetailToJSON(detail)
		delete(item, "created_at")
		delete(item, "updated_at")
		detailList[i] = item
	}

	return response.Success(ctx, http.Json{
		"order":   orderJson,
		"details": detailList,
	})
}

func (r *OrderController) Update(ctx http.Context) http.Response {
	var req adminrequests.OrderUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	orderNo := strings.TrimSpace(req.OrderNo)
	if orderNo == "" {
		orderNo = r.resolveOrderNo(ctx)
	}
	if orderNo == "" {
		return response.Error(ctx, http.StatusBadRequest, "order_no_required")
	}

	if err := r.orderService(ctx).UpdateOrderByOrderNo(orderNo, req.Status, req.Remark); err != nil {
		return HandleGeneratedServiceError(ctx, "order", http.StatusInternalServerError, err, map[string]any{
			"order_no": orderNo,
			"status":   req.Status,
			"remark":   req.Remark,
		})
	}

	return response.Success(ctx)
}

// Destroy deletes an order by order_no.
// @Summary      Delete order
// @Description  Soft-delete order and details by order_no (query/body preferred; path {id} also treated as order_no).
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Param        id        path   string true  "order_no (Resource path)"
// @Param        order_no  query  string false "order_no (preferred)"
// @Success      200       {object} apidoc.Success
// @Failure      400       {object} apidoc.Error
// @Failure      500       {object} apidoc.Error
// @Router       /api/admin/orders/{id} [delete]
// @Security     BearerAuth
func (r *OrderController) Destroy(ctx http.Context) http.Response {
	orderNo := r.resolveOrderNo(ctx)
	if orderNo == "" {
		return response.Error(ctx, http.StatusBadRequest, "order_no_required")
	}

	if err := r.orderService(ctx).DeleteOrderByOrderNo(orderNo); err != nil {
		return HandleGeneratedServiceError(ctx, "order", http.StatusInternalServerError, err, map[string]any{
			"order_no": orderNo,
		})
	}

	return response.Success(ctx, "delete_success", http.Json{})
}

// Export creates async export task for order list.
// @Summary      Export order list
// @Description  Create export task by current filters and return export_id.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Param        user_id    query    int     false "User ID"
// @Param        order_no   query    string  false "Order number"
// @Param        status     query    string  false "Order status"
// @Param        min_amount query    float64 false "Minimum amount"
// @Param        max_amount query    float64 false "Maximum amount"
// @Param        start_time query    string  false "Start time"
// @Param        end_time   query    string  false "End time"
// @Param        order_by   query    string  false "Sort field:direction"
// @Success      200        {object} apidoc.Success "Task queued with export_id"
// @Failure      400        {object} apidoc.Error "Bad request"
// @Failure      401        {object} apidoc.Error "Unauthorized"
// @Failure      403        {object} apidoc.Error "Forbidden"
// @Failure      500        {object} apidoc.Error "Server error"
// @Router       /api/admin/orders/export [post]
// @Security     BearerAuth
func (r *OrderController) Export(ctx http.Context) http.Response {
	filters, resp := r.buildFilters(ctx)
	if resp != nil {
		return resp
	}

	filtersMap := map[string]any{
		"user_id":    filters.UserID,
		"order_no":   filters.OrderNo,
		"status":     filters.Status,
		"min_amount": filters.MinAmount,
		"max_amount": filters.MaxAmount,
		"order_by":   filters.OrderBy,
	}
	if !filters.StartTime.IsZero() {
		filtersMap["start_time"] = utils.FormatDateTime(filters.StartTime)
	}
	if !filters.EndTime.IsZero() {
		filtersMap["end_time"] = utils.FormatDateTime(filters.EndTime)
	}

	result := EnqueueAsyncExport(ctx, EnqueueAsyncExportInput{
		LockResource: "orders",
		ExportType:   models.ExportTypeOrders,
		Filters:      filtersMap,
		Job:          &jobs.ExportOrders{},
	})
	if result.Unauthorized {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUnauthorized.Code)
	}
	if result.Blocked {
		return response.Error(ctx, http.StatusTooManyRequests, apperrors.ErrGetLockFailed.Code)
	}
	if result.Err != nil {
		return HandleGeneratedServiceError(ctx, "export", http.StatusInternalServerError, result.Err, nil)
	}

	return response.Success(ctx, http.Json{
		"export_id": result.ExportID,
		"message":   trans.Get(ctx, "queued"),
	})
}

// GetExportStatus returns export task status.
// @Summary      Get export status
// @Description  Query export status by export record ID.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Export record ID"
// @Success      200  {object}  apidoc.Success
// @Failure      400  {object}  apidoc.Error  "Bad request"
// @Failure      401  {object}  apidoc.Error  "Unauthorized"
// @Failure      403  {object}  apidoc.Error  "Forbidden"
// @Failure      500  {object}  apidoc.Error  "Server error"
// @Router       /api/admin/orders/export/status/{id} [get]
// @Security     BearerAuth
func (r *OrderController) GetExportStatus(ctx http.Context) http.Response {
	return OwnedExportStatusResponse(ctx)
}

func (r *OrderController) Import(ctx http.Context) http.Response {
	adminID, err := helpers.GetAdminIDFromContext(ctx)
	if err != nil {
		return response.Error(ctx, http.StatusUnauthorized, "unauthorized")
	}

	file, err := ctx.Request().File("file")
	if err != nil {
		return response.Error(ctx, http.StatusBadRequest, "file_required")
	}

	importSvc := services.NewImportOrderService(ctx)
	asyncFlag := strings.TrimSpace(ctx.Request().Query("async", ctx.Request().Input("async", "")))
	forceAsync := asyncFlag == "1" || strings.EqualFold(asyncFlag, "true")

	// Peek row count via temp save when async requested or for threshold check.
	// Always save-then-count when forceAsync; otherwise sync path keeps previous behavior unless large.
	if forceAsync {
		disk, path, filename, dataRows, saveErr := importSvc.SaveUploadedCSVForAsync(file)
		if saveErr != nil {
			attrs := map[string]any{"filename": filename, "admin_id": adminID}
			fallback := http.StatusInternalServerError
			if _, ok := apperrors.GetBusinessError(saveErr); ok {
				fallback = http.StatusBadRequest
			}
			return HandleGeneratedServiceError(ctx, "import", fallback, saveErr, attrs)
		}
		return r.enqueueOrderImport(ctx, disk, path, dataRows)
	}

	// Sync path: save briefly to count; if over threshold, keep file and enqueue.
	disk, path, filename, dataRows, saveErr := importSvc.SaveUploadedCSVForAsync(file)
	if saveErr != nil {
		attrs := map[string]any{"filename": filename, "admin_id": adminID}
		fallback := http.StatusInternalServerError
		if _, ok := apperrors.GetBusinessError(saveErr); ok {
			fallback = http.StatusBadRequest
		}
		return HandleGeneratedServiceError(ctx, "import", fallback, saveErr, attrs)
	}
	if dataRows >= services.AsyncImportRowThreshold {
		return r.enqueueOrderImport(ctx, disk, path, dataRows)
	}

	// Small file: import sync then delete source.
	storage, storErr := utils.StorageDisk(disk)
	if storErr != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, storErr, nil)
	}
	csvContent, getErr := storage.Get(path)
	if getErr != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, getErr, nil)
	}
	defer func() { _ = storage.Delete(path) }()

	result, err := importSvc.ImportOrders(csvContent)
	if err != nil {
		attrs := map[string]any{"filename": filename, "admin_id": adminID}
		fallback := http.StatusInternalServerError
		if _, ok := apperrors.GetBusinessError(err); ok {
			fallback = http.StatusBadRequest
		}
		return HandleGeneratedServiceError(ctx, "import", fallback, err, attrs)
	}

	return response.Success(ctx, http.Json{
		"async":         false,
		"total_rows":    result.TotalRows,
		"success_count": result.SuccessCount,
		"failed_count":  result.FailedCount,
		"errors":        result.Errors,
		"message":       trans.Get(ctx, "import_success"),
	})
}

func (r *OrderController) enqueueOrderImport(ctx http.Context, disk, path string, totalRows int) http.Response {
	result := EnqueueAsyncImport(ctx, EnqueueAsyncImportInput{
		LockResource: "orders_import",
		ImportType:   models.ImportTypeOrders,
		Disk:         disk,
		Path:         path,
		TotalRows:    totalRows,
		Job:          &jobs.ImportOrders{},
	})
	if result.Unauthorized {
		return response.Error(ctx, http.StatusUnauthorized, "unauthorized")
	}
	if result.Blocked {
		return response.Error(ctx, http.StatusTooManyRequests, "too_many_requests")
	}
	if result.Err != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, result.Err, map[string]any{
			"import_id": result.ImportID,
		})
	}
	return response.Success(ctx, http.Json{
		"async":     true,
		"import_id": result.ImportID,
		"message":   trans.Get(ctx, "queued"),
	})
}
