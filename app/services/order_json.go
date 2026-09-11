package services

import "goravel/app/models"

// OrderExportData 订单导出数据结构（用于扩展导出字段）
type OrderExportData struct {
	Order models.Order
}

// OrderWithDetails 订单及详情
type OrderWithDetails struct {
	models.Order
	Details []models.OrderDetail `json:"details"`
}

func (s *OrderServiceImpl) OrderToJSON(order models.Order) map[string]any {
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

func (s *OrderServiceImpl) OrderDetailToJSON(detail models.OrderDetail) map[string]any {
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

func (s *OrderServiceImpl) OrderWithDetailsToJSON(item *OrderWithDetails) map[string]any {
	payload := s.OrderToJSON(item.Order)
	detailsList := make([]map[string]any, len(item.Details))
	for i, detail := range item.Details {
		detailsList[i] = s.OrderDetailToJSON(detail)
	}
	payload["details"] = detailsList
	return payload
}
