package orders

import "goravel/app/models"

// ExportData holds an order row for export field extension.
type ExportData struct {
	Order models.Order
}

// WithDetails is an order plus its line items.
type WithDetails struct {
	models.Order
	Details []models.OrderDetail `json:"details"`
}

// ToJSON returns base order display fields.
func ToJSON(order models.Order) map[string]any {
	return map[string]any{
		"id":         order.ID,
		"order_no":   order.OrderNo,
		"user_id":    order.UserID,
		"amount":     order.Amount,
		"status":     order.Status,
		"remark":     order.Remark,
		"created_at": order.CreatedAt,
		"updated_at": order.UpdatedAt,
	}
}

// DetailToJSON returns order detail display fields.
func DetailToJSON(detail models.OrderDetail) map[string]any {
	return map[string]any{
		"id":           detail.ID,
		"order_id":     detail.OrderID,
		"product_id":   detail.ProductID,
		"product_name": detail.ProductName,
		"price":        detail.Price,
		"quantity":     detail.Quantity,
		"subtotal":     detail.Subtotal,
		"created_at":   detail.CreatedAt,
		"updated_at":   detail.UpdatedAt,
	}
}

// WithDetailsToJSON returns a list item payload including details.
func WithDetailsToJSON(item *WithDetails) map[string]any {
	if item == nil {
		return nil
	}
	payload := ToJSON(item.Order)
	detailsList := make([]map[string]any, len(item.Details))
	for i, detail := range item.Details {
		detailsList[i] = DetailToJSON(detail)
	}
	payload["details"] = detailsList
	return payload
}
