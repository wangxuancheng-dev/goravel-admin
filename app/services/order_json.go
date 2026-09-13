package services

import "goravel/app/models"

// OrderExportData holds an order row for export field extension.
type OrderExportData struct {
	Order models.Order
}

// OrderWithDetails is an order plus its line items.
type OrderWithDetails struct {
	models.Order
	Details []models.OrderDetail `json:"details"`
}

// OrderToJSONMap returns base order display fields.
func OrderToJSONMap(order models.Order) map[string]any {
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

// OrderDetailToJSONMap returns order detail display fields.
func OrderDetailToJSONMap(detail models.OrderDetail) map[string]any {
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

// OrderWithDetailsToJSONMap returns a list item payload including details.
func OrderWithDetailsToJSONMap(item *OrderWithDetails) map[string]any {
	if item == nil {
		return nil
	}
	payload := OrderToJSONMap(item.Order)
	detailsList := make([]map[string]any, len(item.Details))
	for i, detail := range item.Details {
		detailsList[i] = OrderDetailToJSONMap(detail)
	}
	payload["details"] = detailsList
	return payload
}
