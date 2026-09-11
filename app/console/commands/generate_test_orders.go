package commands

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/support/carbon"
	"github.com/oklog/ulid/v2"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
)

// GenerateTestOrders 生成测试订单数据
type GenerateTestOrders struct {
	shardingService services.ShardingService
	cancel          context.CancelFunc
}

// Signature The name and signature of the console command.
func (receiver *GenerateTestOrders) Signature() string {
	return "order:generate-test-data"
}

// # 使用默认10个并发协程
// go run . artisan order:generate-test-data --count=1000000

// # 使用20个并发协程（更快）
// go run . artisan order:generate-test-data --count=1000000 --workers=20

// # 根据服务器性能调整并发数
// go run . artisan order:generate-test-data --count=1000000 --workers=50 --batch-size=2000

// Description The console command description.
func (receiver *GenerateTestOrders) Description() string {
	return "生成订单测试数据（tenancy 开启时必须 --tenant）"
}

// Extend The console command extend.
func (receiver *GenerateTestOrders) Extend() command.Extend {
	return command.Extend{
		Category: "order",
		Flags: []command.Flag{
			&command.IntFlag{
				Name:    "count",
				Aliases: []string{"c"},
				Value:   1000000,
				Usage:   "要生成的订单数量（默认：1000000）",
			},
			&command.IntFlag{
				Name:    "batch-size",
				Aliases: []string{"b"},
				Value:   1000,
				Usage:   "批量插入的大小（默认：1000）",
			},
			&command.StringFlag{
				Name:    "start-date",
				Aliases: []string{"s"},
				Usage:   "开始日期（格式：YYYY-MM-DD，默认：当前月份的第一天）",
			},
			&command.StringFlag{
				Name:    "end-date",
				Aliases: []string{"e"},
				Usage:   "结束日期（格式：YYYY-MM-DD，默认：当前日期）",
			},
			&command.IntFlag{
				Name:  "min-user-id",
				Value: 1,
				Usage: "最小用户ID（默认：1）",
			},
			&command.IntFlag{
				Name:  "max-user-id",
				Value: 1000,
				Usage: "最大用户ID（默认：1000）",
			},
			&command.IntFlag{
				Name:    "workers",
				Aliases: []string{"w"},
				Value:   10,
				Usage:   "并发工作协程数量（默认：10）",
			},
			TenantScopeFlag(),
		},
	}
}

// Handle Execute the console command.
func (receiver *GenerateTestOrders) Handle(ctx console.Context) error {
	return RunTenantScopedRequire(ctx, func(_ *models.Tenant, bound context.Context) error {
		return receiver.handleScoped(ctx, bound)
	})
}

func (receiver *GenerateTestOrders) handleScoped(ctx console.Context, bound context.Context) error {
	receiver.shardingService = services.NewShardingService(bound)

	runCtx, cancel := context.WithCancel(ctx)
	receiver.cancel = cancel
	defer cancel()

	// 获取参数
	count := ctx.OptionInt("count")
	batchSize := ctx.OptionInt("batch-size")
	startDateStr := ctx.Option("start-date")
	endDateStr := ctx.Option("end-date")
	minUserID := ctx.OptionInt("min-user-id")
	maxUserID := ctx.OptionInt("max-user-id")
	workers := ctx.OptionInt("workers")

	// 解析日期
	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = utils.ParseDate(startDateStr)
		if err != nil {
			return fmt.Errorf("开始日期格式错误，请使用 YYYY-MM-DD 格式: %v", err)
		}
	} else {
		// 默认：当前月份的第一天
		now := time.Now().UTC()
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	if endDateStr != "" {
		endDate, err = utils.ParseDate(endDateStr)
		if err != nil {
			return fmt.Errorf("结束日期格式错误，请使用 YYYY-MM-DD 格式: %v", err)
		}
		// 设置为当天的最后一秒
		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, time.UTC)
	} else {
		// 默认：当前时间
		endDate = time.Now().UTC()
	}

	if startDate.After(endDate) {
		return fmt.Errorf("开始日期不能晚于结束日期")
	}

	// 验证用户ID范围
	if minUserID <= 0 || maxUserID <= 0 || minUserID > maxUserID {
		return fmt.Errorf("用户ID范围无效")
	}

	ctx.Info("开始生成测试订单数据...")
	ctx.Line("")
	ctx.Info(fmt.Sprintf("订单数量: %s", formatNumber(count)))
	ctx.Info(fmt.Sprintf("批量大小: %s", formatNumber(batchSize)))
	ctx.Info(fmt.Sprintf("并发协程: %d", workers))
	ctx.Info(fmt.Sprintf("开始日期: %s", utils.FormatDate(startDate)))
	ctx.Info(fmt.Sprintf("结束日期: %s", utils.FormatDate(endDate)))
	ctx.Info(fmt.Sprintf("用户ID范围: %d - %d", minUserID, maxUserID))
	ctx.Line("")

	// 计算时间范围（秒）
	timeRange := endDate.Sub(startDate).Seconds()
	if timeRange <= 0 {
		return fmt.Errorf("时间范围无效")
	}

	// 生成订单状态列表
	statuses := []string{"pending", "paid", "cancelled"}

	// 生成商品列表（用于订单详情）
	products := []struct {
		ID    uint
		Name  string
		Price float64
	}{
		{1, "商品A", 99.99},
		{2, "商品B", 199.99},
		{3, "商品C", 299.99},
		{4, "商品D", 49.99},
		{5, "商品E", 149.99},
		{6, "商品F", 79.99},
		{7, "商品G", 249.99},
		{8, "商品H", 89.99},
	}

	// 开始生成
	startTime := time.Now()
	totalInserted := 0
	batches := (count + batchSize - 1) / batchSize // 向上取整

	// 使用随机种子
	rand.Seed(time.Now().UnixNano())

	for range batches {
		if runCtx.Err() != nil {
			ctx.Warning("收到停止信号，正在退出...")
			return runCtx.Err()
		}

		// 计算本批次要插入的数量
		remaining := count - totalInserted
		currentBatchSize := min(remaining, batchSize)

		// 准备批量数据
		orders := make([]models.Order, 0, currentBatchSize)

		for i := range currentBatchSize {
			// 随机生成订单时间（在指定时间范围内）
			randomSeconds := rand.Float64() * timeRange
			orderTime := startDate.Add(time.Duration(randomSeconds) * time.Second)

			// 生成订单号
			yearMonth := orderTime.Format("200601")
			ulidStr := ulid.MustNew(ulid.Timestamp(orderTime), ulid.DefaultEntropy()).String()
			orderNo := fmt.Sprintf("ORD%s%s", yearMonth, ulidStr)

			// 随机生成用户ID
			userID := uint(minUserID + rand.Intn(maxUserID-minUserID+1))

			// 随机生成订单金额（10.00 - 9999.99）
			amount := 10.0 + rand.Float64()*(9999.99-10.0)

			// 随机选择订单状态
			status := statuses[rand.Intn(len(statuses))]

			// 将time.Time转换为时间字符串
			timeStr := utils.FormatDateTime(orderTime)

			// 创建订单对象用于后续处理
			order := models.Order{
				OrderNo: orderNo,
				UserID:  userID,
				Amount:  amount,
				Status:  status,
				Remark:  fmt.Sprintf("测试订单-%d", totalInserted+i+1),
			}
			// 设置CreatedAt用于分表计算（使用carbon.NewDateTime）
			order.CreatedAt = carbon.NewDateTime(carbon.Parse(timeStr))
			order.UpdatedAt = carbon.NewDateTime(carbon.Parse(timeStr))

			// 存储订单对象
			orders = append(orders, order)
		}

		// 按分表分组插入订单，同时准备订单数据map
		ordersByTable := make(map[string][]models.Order)
		orderDataByTable := make(map[string][]map[string]any)
		for _, order := range orders {
			// 从carbon.DateTime获取time.Time用于分表
			timeStr := order.CreatedAt.ToDateTimeString()
			orderTime, _ := utils.ParseDateTimeUTC(timeStr)
			tableName := utils.GetShardingTableName("orders", orderTime)
			ordersByTable[tableName] = append(ordersByTable[tableName], order)

			// 准备订单数据map
			orderData := map[string]any{
				"order_no":   order.OrderNo,
				"user_id":    order.UserID,
				"amount":     order.Amount,
				"status":     order.Status,
				"remark":     order.Remark,
				"created_at": timeStr,
				"updated_at": timeStr,
			}
			orderDataByTable[tableName] = append(orderDataByTable[tableName], orderData)
		}

		// 并发插入订单
		var wg sync.WaitGroup
		var mu sync.Mutex
		workerChan := make(chan struct{}, workers) // 控制并发数量
		errChan := make(chan error, len(ordersByTable))
		orderIDMap := make(map[string]uint) // 订单号到ID的映射（线程安全）

		for tableName, tableOrders := range ordersByTable {
			wg.Add(1)
			workerChan <- struct{}{} // 获取工作协程

			go func(tn string, tos []models.Order, odl []map[string]any) {
				defer wg.Done()
				defer func() { <-workerChan }() // 释放工作协程

				// 确保分表存在
				if err := receiver.shardingService.EnsureShardingTable(tn, "orders"); err != nil {
					errChan <- fmt.Errorf("确保分表 %s 存在失败: %v", tn, err)
					return
				}

				// 分批插入订单
				for i := 0; i < len(odl); i += 100 {
					end := min(i+100, len(odl))
					batchData := odl[i:end]

					// 批量插入
					for j := range batchData {
						if err := appfacades.OrmQuery(bound).Table(tn).Create(batchData[j]); err != nil {
							errChan <- fmt.Errorf("插入订单失败: %v", err)
							return
						}
					}

					// 批量查询ID（优化：减少查询次数）
					orderNos := make([]string, 0, len(batchData))
					for j := range batchData {
						if orderNo, ok := batchData[j]["order_no"].(string); ok {
							orderNos = append(orderNos, orderNo)
						}
					}

					// 批量查询订单ID
					if len(orderNos) > 0 {
						var insertedOrders []models.Order
						if err := appfacades.OrmQuery(bound).Table(tn).Where("order_no IN ?", orderNos).Find(&insertedOrders); err == nil {
							// 更新全局订单ID映射
							mu.Lock()
							for k := range insertedOrders {
								orderIDMap[insertedOrders[k].OrderNo] = insertedOrders[k].ID
							}
							mu.Unlock()
						}
					}
				}
			}(tableName, tableOrders, orderDataByTable[tableName])
		}

		// 等待所有goroutine完成
		go func() {
			wg.Wait()
			close(errChan)
		}()

		// 检查错误
		for err := range errChan {
			if err != nil {
				return err
			}
		}

		// 更新所有订单的ID
		mu.Lock()
		for tableName := range ordersByTable {
			for i := range ordersByTable[tableName] {
				if id, exists := orderIDMap[ordersByTable[tableName][i].OrderNo]; exists {
					ordersByTable[tableName][i].ID = id
				}
			}
		}
		mu.Unlock()

		// 为每个订单生成订单详情
		orderDetailsMap := make(map[string][]models.OrderDetail) // key: tableName, value: details
		for _, tableOrders := range ordersByTable {
			for i := range tableOrders {
				order := &tableOrders[i]
				// 从carbon.DateTime获取time.Time用于分表
				timeStr := order.CreatedAt.ToDateTimeString()
				orderTime, _ := utils.ParseDateTimeUTC(timeStr)
				detailTableName := utils.GetShardingTableName("order_details", orderTime)

				// 确保订单详情分表存在
				if err := receiver.shardingService.EnsureShardingTable(detailTableName, "order_details"); err != nil {
					return fmt.Errorf("确保分表 %s 存在失败: %v", detailTableName, err)
				}

				// 随机生成1-3个商品
				productCount := 1 + rand.Intn(3)
				details := make([]models.OrderDetail, 0, productCount)

				for j := 0; j < productCount; j++ {
					product := products[rand.Intn(len(products))]
					quantity := 1 + rand.Intn(5) // 1-5个
					subtotal := product.Price * float64(quantity)

					detail := models.OrderDetail{
						OrderID:     order.ID,
						ProductID:   product.ID,
						ProductName: product.Name,
						Price:       product.Price,
						Quantity:    quantity,
						Subtotal:    subtotal,
					}
					// 设置CreatedAt和UpdatedAt（使用订单的创建时间）
					detail.CreatedAt = order.CreatedAt
					detail.UpdatedAt = order.CreatedAt

					details = append(details, detail)
				}

				// 将详情添加到对应分表的列表中
				if _, exists := orderDetailsMap[detailTableName]; !exists {
					orderDetailsMap[detailTableName] = make([]models.OrderDetail, 0)
				}
				orderDetailsMap[detailTableName] = append(orderDetailsMap[detailTableName], details...)
			}
		}

		// 并发插入订单详情
		detailWg := sync.WaitGroup{}
		detailErrChan := make(chan error, len(orderDetailsMap))

		for detailTableName, details := range orderDetailsMap {
			if len(details) == 0 {
				continue
			}

			detailWg.Add(1)
			workerChan <- struct{}{} // 获取工作协程

			go func(dtn string, dets []models.OrderDetail) {
				defer detailWg.Done()
				defer func() { <-workerChan }() // 释放工作协程

				// 确保订单详情分表存在
				if err := receiver.shardingService.EnsureShardingTable(dtn, "order_details"); err != nil {
					detailErrChan <- fmt.Errorf("确保分表 %s 存在失败: %v", dtn, err)
					return
				}

				// 分批插入订单详情
				for i := 0; i < len(dets); i += 100 {
					end := min(i+100, len(dets))
					batchDetails := dets[i:end]

					// 批量插入
					for j := range batchDetails {
						// 准备详情数据map
						// 安全处理CreatedAt，如果为nil则使用当前时间
						var createdAtStr string
						if batchDetails[j].CreatedAt != nil && !batchDetails[j].CreatedAt.IsZero() {
							createdAtStr = batchDetails[j].CreatedAt.ToDateTimeString()
						} else {
							// 如果CreatedAt为nil，使用当前时间
							createdAtStr = utils.FormatDateTime(time.Now().UTC())
						}

						detailData := map[string]any{
							"order_id":     batchDetails[j].OrderID,
							"product_id":   batchDetails[j].ProductID,
							"product_name": batchDetails[j].ProductName,
							"price":        batchDetails[j].Price,
							"quantity":     batchDetails[j].Quantity,
							"subtotal":     batchDetails[j].Subtotal,
							"created_at":   createdAtStr,
							"updated_at":   createdAtStr,
						}
						if err := appfacades.OrmQuery(bound).Table(dtn).Create(detailData); err != nil {
							detailErrChan <- fmt.Errorf("插入订单详情失败: %v", err)
							return
						}
					}
				}
			}(detailTableName, details)
		}

		// 等待所有详情插入完成
		go func() {
			detailWg.Wait()
			close(detailErrChan)
		}()

		// 检查错误
		for err := range detailErrChan {
			if err != nil {
				return err
			}
		}

		totalInserted += currentBatchSize

		// 显示进度
		progress := float64(totalInserted) / float64(count) * 100
		elapsed := time.Since(startTime)
		rate := float64(totalInserted) / elapsed.Seconds()
		remainingCount := count - totalInserted
		eta := time.Duration(float64(remainingCount)/rate) * time.Second

		ctx.Info(fmt.Sprintf("进度: %s/%s (%.2f%%) | 已用时间: %s | 速度: %.0f 条/秒 | 预计剩余: %s",
			formatNumber(totalInserted),
			formatNumber(count),
			progress,
			formatDuration(elapsed),
			rate,
			formatDuration(eta),
		))
	}

	// 完成
	elapsed := time.Since(startTime)
	ctx.Line("")
	ctx.Info(fmt.Sprintf("✅ 成功生成 %s 条订单数据！", formatNumber(totalInserted)))
	ctx.Info(fmt.Sprintf("总耗时: %s", formatDuration(elapsed)))
	ctx.Info(fmt.Sprintf("平均速度: %.0f 条/秒", float64(totalInserted)/elapsed.Seconds()))

	return nil
}

func (receiver *GenerateTestOrders) Shutdown(ctx console.Context) error {
	if receiver.cancel != nil {
		receiver.cancel()
	}
	return nil
}

// formatNumber 格式化数字（添加千位分隔符）
func formatNumber(n int) string {
	str := strconv.Itoa(n)
	if len(str) <= 3 {
		return str
	}

	result := ""
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f秒", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0f分钟", d.Minutes())
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	}
}
