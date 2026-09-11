package services

import (
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/filesystem"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/samber/lo"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

// ImportOrderService 订单导入服务
type ImportOrderService struct {
	ctx http.Context
}

func NewImportOrderService(ctx http.Context) *ImportOrderService {
	return &ImportOrderService{ctx: ctx}
}

// ImportUploadedCSV 校验并读取上传的 CSV，再导入订单（临时文件自动清理）。
func (s *ImportOrderService) ImportUploadedCSV(file filesystem.File) (*ImportResult, string, error) {
	if file == nil {
		return nil, "", apperrors.ErrFileRequired
	}

	filename := file.GetClientOriginalName()
	if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
		return nil, filename, apperrors.ErrInvalidFileType
	}

	storage := facades.Storage().Disk("local")
	savedPath, err := storage.PutFile("", file)
	if err != nil {
		return nil, filename, err
	}
	defer func() {
		_ = storage.Delete(savedPath)
	}()

	csvContent, err := storage.Get(savedPath)
	if err != nil {
		return nil, filename, err
	}

	result, err := s.ImportOrders(csvContent)
	return result, filename, err
}

// ImportOrderRow 导入订单行数据
type ImportOrderRow struct {
	OrderID     string // 订单ID（可选，如果为空则自动生成订单号）
	OrderNo     string // 订单号（可选）
	UserID      string // 用户ID
	Amount      string // 订单金额
	Status      string // 订单状态
	ItemIndex   string // 商品序号（如：1/2）
	ProductID   string // 商品ID
	ProductName string // 商品名称
	Price       string // 单价
	Quantity    string // 数量
	Subtotal    string // 小计
	Remark      string // 备注
	CreatedAt   string // 创建时间（可选）
}

// ImportResult 导入结果
type ImportResult struct {
	TotalRows    int      // 总行数
	SuccessCount int      // 成功数量
	FailedCount  int      // 失败数量
	Errors       []string // 错误信息列表
}

// ImportOrders 从CSV内容导入订单
func (s *ImportOrderService) ImportOrders(csvContent string) (*ImportResult, error) {
	// 解析CSV内容
	reader := csv.NewReader(strings.NewReader(csvContent))
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		errorlog.RecordHTTP(s.ctx, "import", "解析CSV失败", map[string]any{
			"error": err.Error(),
		}, "解析CSV失败: %v", err)
		return nil, apperrors.ErrInvalidCSVFormat.WithError(err)
	}

	if len(records) < 2 {
		return nil, apperrors.ErrInvalidCSVFormat.WithMessage("CSV文件至少需要表头和数据行")
	}

	// 解析表头，确定列索引
	headers := records[0]
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.TrimSpace(strings.ToLower(header))] = i
	}

	// 定义列索引（支持中英文表头）
	colIndices := s.getColumnIndices(headerMap)

	// 解析数据行
	dataRows := records[1:]
	result := &ImportResult{
		TotalRows: len(dataRows),
		Errors:    []string{},
	}

	// 解析并过滤有效的数据行，同时保留索引信息用于分组
	type RowWithIndex struct {
		Row   ImportOrderRow
		Index int
	}
	validRows := lo.FilterMap(lo.Range(len(dataRows)), func(rowIndex int, _ int) (RowWithIndex, bool) {
		row := dataRows[rowIndex]
		// 跳过空行
		if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
			return RowWithIndex{}, false
		}

		// 解析行数据
		orderRow := s.parseRow(row, colIndices, rowIndex+2) // +2 因为表头是第1行，数据从第2行开始
		if orderRow == nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行：数据格式错误", rowIndex+2))
			return RowWithIndex{}, false
		}

		return RowWithIndex{Row: *orderRow, Index: rowIndex}, true
	})

	// 按订单分组数据（同一订单可能有多行，每行一个商品）
	orderMap := lo.GroupBy(validRows, func(item RowWithIndex) string {
		orderRow := item.Row
		// 确定订单标识（优先使用订单号，其次使用订单ID）
		orderKey := orderRow.OrderNo
		if orderKey == "" {
			orderKey = orderRow.OrderID
		}
		if orderKey == "" {
			// 如果都没有，使用行号作为临时标识
			orderKey = fmt.Sprintf("temp_%d", item.Index)
		}
		return orderKey
	})

	// 将分组结果转换为 []ImportOrderRow
	orderMapRows := lo.MapValues(orderMap, func(items []RowWithIndex, _ string) []ImportOrderRow {
		return lo.Map(items, func(item RowWithIndex, _ int) ImportOrderRow {
			return item.Row
		})
	})

	// 导入订单
	orderService := NewOrderService(s.ctx)
	for orderKey, rows := range orderMapRows {
		if err := s.importOrderGroup(orderService, orderKey, rows, result); err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("订单 %s: %v", orderKey, err))
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

// getColumnIndices 获取列索引映射（支持中英文表头）
func (s *ImportOrderService) getColumnIndices(headerMap map[string]int) map[string]int {
	indices := make(map[string]int)

	// 定义可能的表头名称（中英文）
	headerMappings := map[string][]string{
		"order_id":     {"id", "订单id", "order id", "订单编号"},
		"order_no":     {"order_no", "订单号", "order no", "订单编号", "order number"},
		"user_id":      {"user_id", "用户id", "user id", "用户编号"},
		"amount":       {"amount", "金额", "订单金额", "total amount"},
		"status":       {"status", "状态", "订单状态", "order status"},
		"item_index":   {"item_index", "商品序号", "item index", "序号"},
		"product_id":   {"product_id", "商品id", "product id", "商品编号"},
		"product_name": {"product_name", "商品名称", "product name", "商品名"},
		"price":        {"price", "单价", "商品单价", "unit price"},
		"quantity":     {"quantity", "数量", "商品数量", "qty"},
		"subtotal":     {"subtotal", "小计", "商品小计", "item total"},
		"remark":       {"remark", "备注", "订单备注", "note"},
		"created_at":   {"created_at", "创建时间", "created at", "创建日期"},
	}

	// 查找每个字段的列索引
	for field, possibleHeaders := range headerMappings {
		for _, header := range possibleHeaders {
			if idx, exists := headerMap[header]; exists {
				indices[field] = idx
				break
			}
		}
	}

	return indices
}

// parseRow 解析单行数据
func (s *ImportOrderService) parseRow(row []string, colIndices map[string]int, rowNum int) *ImportOrderRow {
	orderRow := &ImportOrderRow{}

	// 安全获取列值
	getValue := func(field string) string {
		if idx, exists := colIndices[field]; exists && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	orderRow.OrderID = getValue("order_id")
	orderRow.OrderNo = getValue("order_no")
	orderRow.UserID = getValue("user_id")
	orderRow.Amount = getValue("amount")
	orderRow.Status = getValue("status")
	orderRow.ItemIndex = getValue("item_index")
	orderRow.ProductID = getValue("product_id")
	orderRow.ProductName = getValue("product_name")
	orderRow.Price = getValue("price")
	orderRow.Quantity = getValue("quantity")
	orderRow.Subtotal = getValue("subtotal")
	orderRow.Remark = getValue("remark")
	orderRow.CreatedAt = getValue("created_at")

	return orderRow
}

// importOrderGroup 导入一组订单数据（一个订单可能有多行，每行一个商品）
func (s *ImportOrderService) importOrderGroup(orderService OrderService, orderKey string, rows []ImportOrderRow, result *ImportResult) error {
	if len(rows) == 0 {
		return fmt.Errorf("订单数据为空")
	}

	// 使用第一行作为订单主信息
	firstRow := rows[0]

	// 验证必填字段
	if firstRow.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	userID := cast.ToUint(firstRow.UserID)
	if userID == 0 {
		return fmt.Errorf("用户ID格式错误: %s", firstRow.UserID)
	}

	// 解析订单金额（如果为空，则从商品计算）
	amount := cast.ToFloat64(firstRow.Amount)
	if amount == 0 {
		// 从商品明细计算总金额
		for _, row := range rows {
			if row.Subtotal != "" {
				amount += cast.ToFloat64(row.Subtotal)
			} else if row.Price != "" && row.Quantity != "" {
				amount += cast.ToFloat64(row.Price) * cast.ToFloat64(row.Quantity)
			}
		}
	}

	// 解析订单状态（默认为pending）
	status := strings.ToLower(strings.TrimSpace(firstRow.Status))
	if status == "" {
		status = "pending"
	}
	// 支持中英文状态转换
	status = s.normalizeStatus(status)

	// 解析商品列表
	products := []OrderProduct{}
	for _, row := range rows {
		// 跳过没有商品信息的行
		if row.ProductID == "" && row.ProductName == "" {
			continue
		}

		productID := cast.ToUint(row.ProductID)
		productName := strings.TrimSpace(row.ProductName)
		if productName == "" {
			productName = fmt.Sprintf("商品%d", productID)
		}

		price := cast.ToFloat64(row.Price)
		if price == 0 && row.Subtotal != "" && row.Quantity != "" {
			// 如果单价为空，从小计和数量计算
			subtotal := cast.ToFloat64(row.Subtotal)
			quantity := cast.ToInt(row.Quantity)
			if quantity > 0 {
				price = subtotal / float64(quantity)
			}
		}

		quantity := cast.ToInt(row.Quantity)
		if quantity == 0 {
			quantity = 1
		}

		products = append(products, OrderProduct{
			ProductID:   productID,
			ProductName: productName,
			Price:       price,
			Quantity:    quantity,
		})
	}

	if len(products) == 0 {
		return fmt.Errorf("订单没有商品信息")
	}

	// 如果金额仍为0，从商品计算
	if amount == 0 {
		for _, product := range products {
			amount += product.Price * float64(product.Quantity)
		}
	}

	// 解析备注
	remark := strings.TrimSpace(firstRow.Remark)

	// 创建订单（使用空字符串作为requestID，让服务自动生成）
	order, _, err := orderService.CreateOrder(userID, amount, products, "", remark)
	if err != nil {
		errorlog.RecordHTTP(s.ctx, "import", "创建订单失败", map[string]any{
			"order_key": orderKey,
			"user_id":   userID,
			"amount":    amount,
			"error":     err.Error(),
		}, "创建订单失败: %v", err)
		return fmt.Errorf("创建订单失败: %v", err)
	}

	// 如果导入的订单有状态且不是pending，更新状态
	if status != "pending" && status != order.Status {
		// 解析创建时间（如果提供）
		orderTime := time.Time{}
		if firstRow.CreatedAt != "" {
			if t, err := utils.ParseDateTime(firstRow.CreatedAt); err == nil {
				orderTime = t
			} else if t, err := utils.ParseDate(firstRow.CreatedAt); err == nil {
				orderTime = t
			}
		}

		if err := orderService.UpdateOrder(order.ID, orderTime, status, remark, order.OrderNo); err != nil {
			// 状态更新失败不影响导入，只记录日志
			facades.Log().Warningf("导入订单后更新状态失败: order_id=%d, status=%s, error=%v", order.ID, status, err)
		}
	}

	facades.Log().Infof("成功导入订单: order_id=%d, order_no=%s, user_id=%d, amount=%.2f", order.ID, order.OrderNo, userID, amount)

	return nil
}

// normalizeStatus 规范化订单状态（支持中英文）
func (s *ImportOrderService) normalizeStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))

	// 中文状态映射
	statusMap := map[string]string{
		"待支付":       "pending",
		"已支付":       "paid",
		"已取消":       "cancelled",
		"pending":   "pending",
		"paid":      "paid",
		"cancelled": "cancelled",
	}

	if normalized, exists := statusMap[status]; exists {
		return normalized
	}

	// 默认返回pending
	return "pending"
}
