package admin

import (
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	"goravel/app/http/apidoc"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/http/trans"
	"goravel/app/jobs"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
)

type OrderController struct {}


// OrderResponse swagger response for order summary.
type OrderResponse struct {
	ID        uint    `json:"id" example:"1"`                           // Order ID
	OrderNo   string  `json:"order_no" example:"ORD202604090001"`       // Order number
	UserID    uint    `json:"user_id" example:"1001"`                   // User ID
	Amount    float64 `json:"amount" example:"199.98"`                  // Total amount
	Status    string  `json:"status" example:"pending"`                 // Order status
	Remark    string  `json:"remark" example:"note"`                    // Remark
	CreatedAt string  `json:"created_at" example:"2024-01-01 00:00:00"` // Created at
	UpdatedAt string  `json:"updated_at" example:"2024-01-01 00:00:00"` // Updated at
}

// OrderDetailResponse swagger response for order item details.
type OrderDetailResponse struct {
	ID          uint    `json:"id" example:"1"`                           // Detail ID
	OrderID     uint    `json:"order_id" example:"1"`                     // Order ID
	ProductID   uint    `json:"product_id" example:"101"`                 // Product ID
	ProductName string  `json:"product_name" example:"sample product"`    // Product name
	Price       float64 `json:"price" example:"99.99"`                    // Unit price
	Quantity    int     `json:"quantity" example:"2"`                     // Quantity
	Subtotal    float64 `json:"subtotal" example:"199.98"`                // Subtotal
	CreatedAt   string  `json:"created_at" example:"2024-01-01 00:00:00"` // Created at
	UpdatedAt   string  `json:"updated_at" example:"2024-01-01 00:00:00"` // Updated at
}

// OrderWithDetailsResponse combines order and details.
type OrderWithDetailsResponse struct {
	Order   OrderResponse         `json:"order"`   // Order summary
	Details []OrderDetailResponse `json:"details"` // Detail list
}

// OrderListData list response payload.
type OrderListData struct {
	Data []OrderWithDetailsResponse `json:"data"` // Order list
	apidoc.Pagination
}

type OrderListResponse struct {
	apidoc.Success
	Data OrderListData `json:"data"`
}

type OrderDetailData struct {
	Order   OrderResponse         `json:"order"`   // Order summary
	Details []OrderDetailResponse `json:"details"` // Order details
}

type OrderDetailResponseWrapper struct {
	apidoc.Success
	Data OrderDetailData `json:"data"`
}

type OrderCreateRequest struct {
	UserID    uint               `json:"user_id" example:"1001"`
	Amount    float64            `json:"amount" example:"199.98"`
	Products  []OrderProductItem `json:"products"`
	RequestID string             `json:"request_id" example:"req_20260409_001"`
	Remark    string             `json:"remark" example:"remark"`
}

type OrderUpdateRequest struct {
	Status string `json:"status" example:"paid"`
	Remark string `json:"remark" example:"remark"`
}

type ExportTaskData struct {
	ExportID uint   `json:"export_id" example:"1"`
	Message  string `json:"message" example:"message"`
}

type ExportTaskResponse struct {
	apidoc.Success
	Data ExportTaskData `json:"data"`
}

type ExportStatusData struct {
	ID         uint   `json:"id" example:"1"`
	Status     uint8  `json:"status" example:"1"`
	StatusText string `json:"status_text" example:"status_text"`
	FileURL    string `json:"file_url" example:"/api/admin/exports/1/download"`
	Filename   string `json:"filename" example:"orders_20260409.csv"`
	Size       int64  `json:"size" example:"1024"`
	ErrorMsg   string `json:"error_msg" example:""`
	CreatedAt  string `json:"created_at" example:"2024-01-01 00:00:00"`
	UpdatedAt  string `json:"updated_at" example:"2024-01-01 00:00:00"`
}

type ExportStatusResponse struct {
	apidoc.Success
	Data ExportStatusData `json:"data"`
}

type ImportResultData struct {
	TotalRows    int      `json:"total_rows" example:"10"`
	SuccessCount int      `json:"success_count" example:"8"`
	FailedCount  int      `json:"failed_count" example:"2"`
	Errors       []string `json:"errors"`
	Message      string   `json:"message" example:"message"`
}

type ImportResultResponse struct {
	apidoc.Success
	Data ImportResultData `json:"data"`
}

type OrderProductItem struct {
	ProductID   uint    `json:"product_id" example:"1" binding:"required"`
	ProductName string  `json:"product_name" example:"product_name" binding:"required"`
	Price       float64 `json:"price" example:"99.99" binding:"required"`
	Quantity    int     `json:"quantity" example:"2" binding:"required"`
}

func NewOrderController() *OrderController {
	return &OrderController{}
}

func (r *OrderController) orderService(ctx http.Context) services.OrderService {
	return services.NewOrderService(ctx)
}


// buildFilters builds filters shared by list/export endpoints.
// It accepts both query params (GET) and body params (POST).
func (r *OrderController) buildFilters(ctx http.Context) (services.OrderFilters, http.Response) {
	// Prefer request body, fallback to query params for compatibility.
	userID := cast.ToUint(ctx.Request().Input("user_id", ctx.Request().Query("user_id", "0")))
	orderNo := ctx.Request().Input("order_no", ctx.Request().Query("order_no", ""))
	status := ctx.Request().Input("status", ctx.Request().Query("status", ""))
	minAmount := cast.ToFloat64(ctx.Request().Input("min_amount", ctx.Request().Query("min_amount", "0")))
	maxAmount := cast.ToFloat64(ctx.Request().Input("max_amount", ctx.Request().Query("max_amount", "0")))
	orderBy := ctx.Request().Input("order_by", ctx.Request().Query("order_by", ""))

	// Parse time params and normalize to UTC time strings.
	startTimeStr := getTimeInputOrQueryUTC(ctx, "start_time")
	endTimeStr := getTimeInputOrQueryUTC(ctx, "end_time")

	startTime, endTime, err := r.parseTimeRange(startTimeStr, endTimeStr)
	if err != nil {
		return services.OrderFilters{}, response.Error(ctx, http.StatusBadRequest, err.Error())
	}

	// Validate time range (default limit: 3 months).
	if resp := validateTimeRangeResponse(ctx, startTime, endTime, 3); resp != nil {
		return services.OrderFilters{}, resp
	}

	return services.OrderFilters{
		UserID:    userID,
		OrderNo:   orderNo,
		Status:    status,
		MinAmount: minAmount,
		MaxAmount: maxAmount,
		StartTime: startTime,
		EndTime:   endTime,
		OrderBy:   orderBy,
	}, nil
}

// parseTimeRange parses time range; default start is now - 7 days.
func (r *OrderController) parseTimeRange(startTimeStr, endTimeStr string) (time.Time, time.Time, error) {
	var startTime, endTime time.Time
	var err error

	if startTimeStr == "" {
		// Default to recent 7 days in UTC.
		startTime = time.Now().UTC().AddDate(0, 0, -7)
	} else {
		// Parse UTC datetime string.
		startTime, err = utils.ParseDateTime(startTimeStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid_start_time")
		}
	}

	if endTimeStr == "" {
		// Empty end time means no upper bound.
		endTime = time.Time{}
	} else {
		// Parse UTC datetime string.
		endTime, err = utils.ParseDateTime(endTimeStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid_end_time")
		}
	}

	return startTime, endTime, nil
}

// Index returns paginated order list.
// @Summary      Get order list
// @Description  Returns paginated orders with filters; time range is limited.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Param        page       query    int     false "Page number" default(1)
// @Param        page_size  query    int     false "Page size" default(10)
// @Param        user_id    query    int     false "User ID"
// @Param        order_no   query    string  false "Order number (fuzzy)"
// @Param        status     query    string  false "Order status"
// @Param        min_amount query    float64 false "Minimum amount"
// @Param        max_amount query    float64 false "Maximum amount"
// @Param        start_time query    string  false "Start time (2006-01-02 15:04:05)"
// @Param        end_time   query    string  false "End time (2006-01-02 15:04:05)"
// @Param        order_by   query    string  false "Sort field:direction"
// @Success      200        {object} OrderListResponse
// @Failure      400        {object} apidoc.Error "Bad request"
// @Failure      500        {object} apidoc.Error "Server error"
// @Router       /api/admin/orders [get]
// @Security     BearerAuth
func (r *OrderController) Index(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})

	// Build filters shared with export.
	filters, resp := r.buildFilters(ctx)
	if resp != nil {
		return resp
	}

	// Query order list with details.
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

func (r *OrderController) Show(ctx http.Context) http.Response {
	orderNo := ctx.Request().Query("order_no", "")

	if orderNo != "" {
		order, details, err := r.orderService(ctx).GetOrderByOrderNo(orderNo)
		if err == nil {
			return r.buildOrderDetailResponse(ctx, order, details)
		}
		if routeID := ctx.Request().Route("id"); routeID != "" && orderNo == routeID {
			if orderID := cast.ToUint(routeID); orderID > 0 {
				order, details, err := r.orderService(ctx).GetOrderByID(orderID, time.Time{})
				if err == nil {
					return r.buildOrderDetailResponse(ctx, order, details)
				}
			}
		}
		return HandleGeneratedServiceError(ctx, "order", http.StatusNotFound, apperrors.ErrOrderNotFound, map[string]any{
			"order_no": orderNo,
		})
	}

	return response.Error(ctx, http.StatusBadRequest, "order_no_or_id_required")
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
	var req struct {
		UserID    uint                    `json:"user_id" binding:"required"`
		Amount    float64                 `json:"amount" binding:"required"`
		Products  []services.OrderProduct `json:"products" binding:"required"`
		RequestID string                  `json:"request_id"`
		Remark    string                  `json:"remark"`
	}

	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, "invalid_params")
	}

	if len(req.Products) == 0 {
		return response.Error(ctx, http.StatusBadRequest, "empty_products")
	}

	order, details, err := r.orderService(ctx).CreateOrder(req.UserID, req.Amount, req.Products, req.RequestID, req.Remark)
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
	var req struct {
		OrderNo string `json:"order_no"`
		Status  string `json:"status" binding:"required"`
		Remark  string `json:"remark"`
	}

	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, "invalid_params")
	}

	orderNo := strings.TrimSpace(req.OrderNo)
	if orderNo == "" {
		orderNo = strings.TrimSpace(ctx.Request().Query("order_no", ""))
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

// Destroy 鍒犻櫎璁㈠崟
// @Summary      鍒犻櫎璁㈠崟
// @Description  鍒犻櫎璁㈠崟鍙婂叾璇︽儏銆備娇鐢ㄨ鍗曞彿鏌ヨ锛堝彲鐩存帴瀹氫綅鍒嗚〃锛?
// @Tags         璁㈠崟绠＄悊
// @Accept       json
// @Produce      json
// @Param        id         path     string  true "璁㈠崟鍙?
// @Success      200        {object} apidoc.Success
// @Failure      400        {object} apidoc.Error "鍙傛暟閿欒"
// @Failure      500        {object} apidoc.Error "鏈嶅姟鍣ㄩ敊璇?
// @Router       /api/admin/orders/{id} [delete]
// @Security     BearerAuth
func (r *OrderController) Destroy(ctx http.Context) http.Response {
	orderNo := strings.TrimSpace(ctx.Request().Input("order_no", ctx.Request().Query("order_no", "")))
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
// @Success      200        {object} ExportTaskResponse "Task queued with export_id"
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
// @Success      200  {object}  ExportStatusResponse
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

	filename := file.GetClientOriginalName()
	if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidFileType.Code)
	}

	storage := facades.Storage().Disk("local")
	savedPath, err := storage.PutFile("", file)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, err, map[string]any{
			"filename": filename,
		})
	}

	csvContent, err := storage.Get(savedPath)
	if err != nil {
		_ = storage.Delete(savedPath)
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, err, map[string]any{
			"filename": filename,
		})
	}

	defer func() {
		_ = storage.Delete(savedPath)
	}()

	importService := services.NewImportOrderService(ctx)
	result, err := importService.ImportOrders(csvContent)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "import", http.StatusInternalServerError, err, map[string]any{
			"filename": filename,
			"admin_id": adminID,
		})
	}

	return response.Success(ctx, http.Json{
		"total_rows":    result.TotalRows,
		"success_count": result.SuccessCount,
		"failed_count":  result.FailedCount,
		"errors":        result.Errors,
		"message":       trans.Get(ctx, "import_success"),
	})
}
