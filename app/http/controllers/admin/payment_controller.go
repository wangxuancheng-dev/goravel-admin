package admin

import (
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/apidoc"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/http/trans"
	"goravel/app/jobs"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
)

type PaymentController struct{}

type PaymentMethodSimple struct {
	ID   uint   `json:"id" example:"1"`
	Name string `json:"name" example:"name"`
	Code string `json:"code" example:"wechat"`
	Type string `json:"type" example:"wechat"`
}

type PaymentResponse struct {
	ID              uint                `json:"id" example:"1"`
	PaymentNo       string              `json:"payment_no" example:"PAY202604090001"`
	OrderNo         string              `json:"order_no" example:"ORD202604090001"`
	PaymentMethodID uint                `json:"payment_method_id" example:"1"`
	UserID          uint                `json:"user_id" example:"1001"`
	Amount          float64             `json:"amount" example:"99.99"`
	Status          string              `json:"status" example:"paid"`
	ThirdPartyNo    string              `json:"third_party_no" example:"WX202604090001"`
	PayTime         string              `json:"pay_time" example:"2024-01-01 00:00:00"`
	FailReason      string              `json:"fail_reason" example:""`
	Remark          string              `json:"remark" example:"remark"`
	CreatedAt       string              `json:"created_at" example:"2024-01-01 00:00:00"`
	UpdatedAt       string              `json:"updated_at" example:"2024-01-01 00:00:00"`
	PaymentMethod   PaymentMethodSimple `json:"payment_method,omitempty"`
}

type PaymentListData struct {
	List []PaymentResponse `json:"list"`
	apidoc.Pagination
}

type PaymentListResponse struct {
	apidoc.Success
	Data PaymentListData `json:"data"`
}

type PaymentDetailResponse struct {
	apidoc.Success
	Data PaymentResponse `json:"data"`
}

func NewPaymentController() *PaymentController {
	return &PaymentController{}
}

func (c *PaymentController) PaymentService(ctx http.Context) services.PaymentService {
	return services.NewPaymentService(ctx)
}

func (c *PaymentController) buildFilters(ctx http.Context) (services.PaymentFilters, http.Response) {
	paymentNo := ctx.Request().Input("payment_no", ctx.Request().Query("payment_no", ""))
	orderNo := ctx.Request().Input("order_no", ctx.Request().Query("order_no", ""))
	paymentMethodID := cast.ToUint(ctx.Request().Input("payment_method_id", ctx.Request().Query("payment_method_id", "0")))
	userID := cast.ToUint(ctx.Request().Input("user_id", ctx.Request().Query("user_id", "0")))
	status := ctx.Request().Input("status", ctx.Request().Query("status", ""))
	orderBy := ctx.Request().Input("order_by", ctx.Request().Query("order_by", ""))

	var startTime, endTime time.Time
	if parsedStartTime, resp := parseOptionalTimeFromInputOrQuery(ctx, "start_time", "invalid_start_time"); resp != nil {
		return services.PaymentFilters{}, resp
	} else {
		startTime = parsedStartTime
	}
	if parsedEndTime, resp := parseOptionalTimeFromInputOrQuery(ctx, "end_time", "invalid_end_time"); resp != nil {
		return services.PaymentFilters{}, resp
	} else {
		endTime = parsedEndTime
	}

	if startTime.IsZero() {
		startTime = time.Now().UTC().AddDate(0, 0, -7)
	}
	if endTime.IsZero() {
		endTime = time.Now().UTC()
	}

	if resp := validateTimeRangeResponse(ctx, startTime, endTime); resp != nil {
		return services.PaymentFilters{}, resp
	}

	return services.PaymentFilters{
		PaymentNo:       paymentNo,
		OrderNo:         orderNo,
		PaymentMethodID: paymentMethodID,
		UserID:          userID,
		Status:          status,
		StartTime:       startTime,
		EndTime:         endTime,
		OrderBy:         orderBy,
	}, nil
}

func (c *PaymentController) Index(ctx http.Context) http.Response {
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)

	filters, resp := c.buildFilters(ctx)
	if resp != nil {
		return resp
	}

	payments, total, err := c.PaymentService(ctx).GetPayments(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "payment", http.StatusInternalServerError, err, map[string]any{
			"filters": filters,
		})
	}

	svc := c.PaymentService(ctx)
	paymentList := make([]http.Json, len(payments))
	for i := range payments {
		paymentList[i] = svc.PaymentToJSON(&payments[i])
	}

	return response.Success(ctx, http.Json{
		"list":      paymentList,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *PaymentController) Show(ctx http.Context) http.Response {
	paymentNo := ctx.Request().Route("id")
	if paymentNo == "" {
		return response.Error(ctx, http.StatusBadRequest, "payment_no_required")
	}
	payment, err := c.PaymentService(ctx).GetPaymentByPaymentNo(paymentNo)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "payment", http.StatusNotFound, err, map[string]any{
			"payment_no": paymentNo,
		})
	}

	return response.Success(ctx, c.PaymentService(ctx).PaymentToJSON(payment))
}

func (c *PaymentController) Export(ctx http.Context) http.Response {
	filters, resp := c.buildFilters(ctx)
	if resp != nil {
		return resp
	}

	filtersMap := map[string]any{
		"payment_no":        filters.PaymentNo,
		"order_no":          filters.OrderNo,
		"payment_method_id": filters.PaymentMethodID,
		"user_id":           filters.UserID,
		"status":            filters.Status,
		"order_by":          filters.OrderBy,
	}
	if !filters.StartTime.IsZero() {
		filtersMap["start_time"] = utils.FormatDateTime(filters.StartTime)
	}
	if !filters.EndTime.IsZero() {
		filtersMap["end_time"] = utils.FormatDateTime(filters.EndTime)
	}

	result := EnqueueAsyncExport(ctx, EnqueueAsyncExportInput{
		LockResource: "payments",
		ExportType:   models.ExportTypePayments,
		Filters:      filtersMap,
		Job:          &jobs.ExportPayments{},
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

func (c *PaymentController) GetExportStatus(ctx http.Context) http.Response {
	exportID := helpers.GetUintRoute(ctx, "id")
	if exportID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "id_required")
	}

	var exportRecord models.Export
	if err := appfacades.OrmQuery(ctx).Where("id", exportID).FirstOrFail(&exportRecord); err != nil {
		return response.Error(ctx, http.StatusNotFound, "record_not_found")
	}

	result := http.Json{
		"id":          exportRecord.ID,
		"status":      exportRecord.Status,
		"status_text": c.getExportStatusText(ctx, exportRecord.Status),
		"path":        exportRecord.Path,
		"filename":    exportRecord.Filename,
		"size":        exportRecord.Size,
		"error_msg":   exportRecord.ErrorMsg,
		"created_at":  exportRecord.CreatedAt,
		"updated_at":  exportRecord.UpdatedAt,
	}

	if exportRecord.Status == models.ExportStatusSuccess && exportRecord.Path != "" {
		result["download_url"] = fmt.Sprintf("/api/admin/exports/%d/download", exportRecord.ID)
	}

	return response.Success(ctx, result)
}

func (c *PaymentController) getExportStatusText(ctx http.Context, status uint8) string {
	switch status {
	case models.ExportStatusProcessing:
		return trans.Get(ctx, "processing")
	case models.ExportStatusSuccess:
		return trans.Get(ctx, "success")
	case models.ExportStatusFailed:
		return trans.Get(ctx, "failed")
	default:
		return trans.Get(ctx, "unknown")
	}
}
