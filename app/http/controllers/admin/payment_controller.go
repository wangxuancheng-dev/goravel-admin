package admin

import (
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/apidoc"
	"goravel/app/http/helpers"
	adminreq "goravel/app/http/requests/admin"
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
	filters, err := services.BuildPaymentFiltersFromHTTP(ctx)
	if err != nil {
		return services.PaymentFilters{}, response.Error(ctx, http.StatusBadRequest, err.Error())
	}
	if resp := validateTimeRangeResponse(ctx, filters.StartTime, filters.EndTime); resp != nil {
		return services.PaymentFilters{}, resp
	}
	return filters, nil
}

func (c *PaymentController) Index(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})

	filters, resp := c.buildFilters(ctx)
	if resp != nil {
		return resp
	}

	payments, total, err := c.PaymentService(ctx).GetPayments(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "payment", http.StatusBadRequest, err, map[string]any{
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

// Store creates a payment for a pending order; optional initiate calls the gateway (mock returns notify hint).
func (c *PaymentController) Store(ctx http.Context) http.Response {
	var req adminreq.PaymentCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	payment, err := c.PaymentService(ctx).CreatePayment(req.OrderNo, req.PaymentMethodID, 0, req.Amount, req.Remark)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "payment", http.StatusBadRequest, err, map[string]any{
			"order_no":          req.OrderNo,
			"payment_method_id": req.PaymentMethodID,
		})
	}

	out := c.PaymentService(ctx).PaymentToJSON(payment)
	if req.Initiate {
		gateway, err := services.NewPaymentGatewayService(ctx).CreatePaymentOrder(payment, ctx.Request().Ip())
		if err != nil {
			return HandleGeneratedServiceError(ctx, "payment", http.StatusNotImplemented, err, map[string]any{
				"payment_no": payment.PaymentNo,
			})
		}
		out["gateway"] = gateway
	}
	return response.Success(ctx, out)
}

// Query asks the payment gateway for third-party status (mock reads local DB; wechat/alipay still stub).
func (c *PaymentController) Query(ctx http.Context) http.Response {
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

	result, err := services.NewPaymentGatewayService(ctx).QueryPaymentOrder(payment)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "payment", http.StatusNotImplemented, err, map[string]any{
			"payment_no": paymentNo,
		})
	}
	return response.Success(ctx, result)
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
		"time_field":        filters.TimeField,
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
	return OwnedExportStatusResponse(ctx)
}
