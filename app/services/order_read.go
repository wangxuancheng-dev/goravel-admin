package services

import (
	"context"
	"time"

	appfacades "goravel/app/facades"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/utils"
)

// FindOrderByID 通过订单 ID 查找主表记录。
// 写路径与精确定位请务必传 orderNo；仅 ID 时只扫描最近 id_lookup_scan_months 个月，且跨月 id 可能撞号。
func FindOrderByID(ctx context.Context, orderID uint, orderNo ...string) (*models.Order, error) {
	if len(orderNo) > 0 && orderNo[0] != "" {
		order, err := FindOrderByOrderNo(ctx, orderNo[0])
		if err == nil {
			if orderID == 0 || order.ID == orderID {
				return order, nil
			}
		}
	}
	if orderID == 0 {
		return nil, apperrors.ErrOrderIDRequired
	}
	now := time.Now().UTC()
	startTime := now.AddDate(0, -utils.GetIDLookupScanMonths(), 0)
	tableNames := utils.GetShardingTableNames("orders", startTime, now)
	for i := len(tableNames) - 1; i >= 0; i-- {
		var order models.Order
		if err := appfacades.OrmQuery(ctx).Model(&models.Order{}).Table(tableNames[i]).Where("id", orderID).First(&order); err == nil {
			return &order, nil
		}
	}
	return nil, apperrors.ErrOrderNotFound
}

// FindOrderByOrderNo 通过订单号查找主表记录（直接定位或扫描分表）。
func FindOrderByOrderNo(ctx context.Context, orderNo string) (*models.Order, error) {
	yearMonth, ok := utils.ParseShardingNoYearMonth(orderNo, utils.OrderNoConfig)
	if !ok {
		return findOrderByOrderNoScan(ctx, orderNo)
	}
	parsedTime, err := time.Parse(utils.GetTimeShardingSuffixLayout(), yearMonth)
	if err != nil {
		return findOrderByOrderNoScan(ctx, orderNo)
	}
	tableName := utils.GetShardingTableName("orders", parsedTime)
	var order models.Order
	if err := appfacades.OrmQuery(ctx).Model(&models.Order{}).Table(tableName).Where("order_no", orderNo).First(&order); err == nil {
		return &order, nil
	}
	return nil, apperrors.ErrOrderNotFound
}

func findOrderByOrderNoScan(ctx context.Context, orderNo string) (*models.Order, error) {
	now := time.Now().UTC()
	startTime := now.AddDate(0, -utils.GetIDLookupScanMonths(), 0)
	tableNames := utils.GetShardingTableNames("orders", startTime, now)
	for i := len(tableNames) - 1; i >= 0; i-- {
		var order models.Order
		if err := appfacades.OrmQuery(ctx).Model(&models.Order{}).Table(tableNames[i]).Where("order_no", orderNo).First(&order); err == nil {
			return &order, nil
		}
	}
	return nil, apperrors.ErrOrderNotFound
}

// FindOrderWithDetails 加载订单及明细（供 GetOrderByID / ES 等只读场景复用）。
func FindOrderWithDetails(ctx context.Context, orderID uint, orderNoHint string) (*models.Order, []models.OrderDetail, error) {
	var order *models.Order
	var err error
	if orderNoHint != "" {
		order, err = FindOrderByID(ctx, orderID, orderNoHint)
	} else {
		order, err = FindOrderByID(ctx, orderID)
	}
	if err != nil {
		return nil, nil, err
	}
	if order == nil {
		return nil, nil, apperrors.ErrOrderNotFound
	}
	timeStr := order.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)
	detailTableName := utils.GetShardingTableName("order_details", createdAt)
	var details []models.OrderDetail
	if err := appfacades.OrmQuery(ctx).Table(detailTableName).Where("order_id", order.ID).Find(&details); err != nil {
		return nil, nil, apperrors.ErrQueryOrderDetailFailed.WithError(err)
	}
	return order, details, nil
}
