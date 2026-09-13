package services

import (
	"goravel/app/models"
	"goravel/app/orders"
)

// OrderExportData 订单导出数据结构（用于扩展导出字段）
type OrderExportData = orders.ExportData

// OrderWithDetails 订单及详情
type OrderWithDetails = orders.WithDetails

func (s *OrderServiceImpl) OrderToJSON(order models.Order) map[string]any {
	return orders.ToJSON(order)
}

func (s *OrderServiceImpl) OrderDetailToJSON(detail models.OrderDetail) map[string]any {
	return orders.DetailToJSON(detail)
}

func (s *OrderServiceImpl) OrderWithDetailsToJSON(item *OrderWithDetails) map[string]any {
	return orders.WithDetailsToJSON(item)
}
