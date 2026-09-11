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
	"github.com/goravel/framework/facades"
	"github.com/oklog/ulid/v2"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

// GenerateTestPayments 生成测试支付记录数据
type GenerateTestPayments struct {
	paymentService  services.PaymentService
	shardingService services.ShardingService
}

// Signature The name and signature of the console command.
func (receiver *GenerateTestPayments) Signature() string {
	return "payment:generate-test-data"
}

// # 使用默认10个并发协程
// go run . artisan payment:generate-test-data --count=1000000

// # 使用20个并发协程（更快）
// go run . artisan payment:generate-test-data --count=1000000 --workers=20

// # 根据服务器性能调整并发数
// go run . artisan payment:generate-test-data --count=1000000 --workers=50 --batch-size=2000

// Description The console command description.
func (receiver *GenerateTestPayments) Description() string {
	return "生成支付记录测试数据（用于测试支付记录导出等功能）"
}

// Extend The console command extend.
func (receiver *GenerateTestPayments) Extend() command.Extend {
	return command.Extend{
		Category: "payment",
		Flags: []command.Flag{
			&command.IntFlag{
				Name:    "count",
				Aliases: []string{"c"},
				Value:   10000,
				Usage:   "生成的记录数量",
			},
			&command.IntFlag{
				Name:    "workers",
				Aliases: []string{"w"},
				Value:   10,
				Usage:   "并发协程数",
			},
			&command.IntFlag{
				Name:    "batch-size",
				Aliases: []string{"b"},
				Value:   1000,
				Usage:   "每个批次插入的记录数",
			},
			&command.StringFlag{
				Name:    "start-date",
				Aliases: []string{"s"},
				Value:   "",
				Usage:   "开始日期（格式：2006-01-02），默认为3个月前",
			},
			&command.StringFlag{
				Name:    "end-date",
				Aliases: []string{"e"},
				Value:   "",
				Usage:   "结束日期（格式：2006-01-02），默认为当前时间",
			},
		},
	}
}

// Handle Execute the console command.
func (receiver *GenerateTestPayments) Handle(ctx console.Context) error {
	count := ctx.OptionInt("count")
	workers := ctx.OptionInt("workers")
	batchSize := ctx.OptionInt("batch-size")
	startDateStr := ctx.Option("start-date")
	endDateStr := ctx.Option("end-date")

	// 参数验证
	if count <= 0 {
		return fmt.Errorf("记录数量必须大于0")
	}
	if workers <= 0 {
		return fmt.Errorf("并发协程数必须大于0")
	}
	if batchSize <= 0 {
		return fmt.Errorf("批次大小必须大于0")
	}

	// 解析日期范围
	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = time.ParseInLocation("2006-01-02", startDateStr, time.UTC)
		if err != nil {
			return fmt.Errorf("开始日期格式错误，应为 YYYY-MM-DD 格式: %v", err)
		}
	} else {
		startDate = time.Now().UTC().AddDate(0, -3, 0) // 默认3个月前
	}

	if endDateStr != "" {
		endDate, err = time.ParseInLocation("2006-01-02", endDateStr, time.UTC)
		if err != nil {
			return fmt.Errorf("结束日期格式错误，应为 YYYY-MM-DD 格式: %v", err)
		}
	} else {
		endDate = time.Now().UTC() // 默认当前时间
	}

	if startDate.After(endDate) {
		return fmt.Errorf("开始日期不能晚于结束日期")
	}

	ctx.Info(fmt.Sprintf("开始生成 %d 条支付记录测试数据", count))
	ctx.Info(fmt.Sprintf("时间范围: %s 至 %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")))
	ctx.Info(fmt.Sprintf("并发协程数: %d, 批次大小: %d", workers, batchSize))

	// 获取支付方式列表
	paymentMethods, err := receiver.getPaymentMethods()
	if err != nil {
		return fmt.Errorf("获取支付方式失败: %v", err)
	}

	if len(paymentMethods) == 0 {
		return fmt.Errorf("没有找到支付方式，请先创建支付方式")
	}

	// 计算每个协程的工作量
	totalBatches := (count + batchSize - 1) / batchSize
	batchesPerWorker := totalBatches / workers
	remainder := totalBatches % workers

	startTime := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount int64
	var errorCount int64

	// 启动工作协程
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// 计算当前协程的批次范围
			batchStart := i * batchesPerWorker
			batchEnd := batchStart + batchesPerWorker
			if i < remainder {
				batchEnd++
			}

			for batchID := batchStart; batchID < batchEnd; batchID++ {
				// 计算当前批次的记录数
				remainingCount := count - batchID*batchSize
				currentBatchSize := batchSize
				if remainingCount < batchSize {
					currentBatchSize = remainingCount
				}

				if currentBatchSize <= 0 {
					break
				}

				// 生成批次数据
				err := receiver.generateBatch(currentBatchSize, startDate, endDate, paymentMethods)
				mu.Lock()
				if err != nil {
					errorCount++
					ctx.Error(fmt.Sprintf("Worker %d 批次 %d 失败: %v", workerID, batchID, err))
				} else {
					successCount += int64(currentBatchSize)
					ctx.Info(fmt.Sprintf("Worker %d 批次 %d 完成，生成 %d 条记录", workerID, batchID, currentBatchSize))
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// 输出结果
	duration := time.Since(startTime)
	ctx.Info(fmt.Sprintf("\n✅ 完成！"))
	ctx.Info(fmt.Sprintf("总耗时: %v", duration))
	ctx.Info(fmt.Sprintf("成功生成: %d 条记录", successCount))
	ctx.Info(fmt.Sprintf("失败批次: %d", errorCount))
	if successCount > 0 {
		ctx.Info(fmt.Sprintf("平均速度: %.0f 条/秒", float64(successCount)/duration.Seconds()))
	}

	return nil
}

// getPaymentMethods 获取支付方式列表
func (receiver *GenerateTestPayments) getPaymentMethods() ([]models.PaymentMethod, error) {
	var paymentMethods []models.PaymentMethod
	err := facades.Orm().Query().Model(&models.PaymentMethod{}).Where("is_active", true).Find(&paymentMethods)
	return paymentMethods, err
}

// generateBatch 生成一批支付记录
func (receiver *GenerateTestPayments) generateBatch(count int, startDate, endDate time.Time, paymentMethods []models.PaymentMethod) error {
	payments := make([]models.Payment, 0, count)

	for i := 0; i < count; i++ {
		// 随机生成时间
		randomTime := randomTimeInRange(startDate, endDate)

		// 随机选择支付方式
		paymentMethod := paymentMethods[rand.Intn(len(paymentMethods))]

		// 随机生成订单号（模拟已存在的订单）
		orderNo := fmt.Sprintf("ORD%s%s", randomTime.Format("20060102"), strconv.Itoa(rand.Intn(999999)))

		// 随机生成用户ID
		userID := uint(rand.Intn(10000) + 1)

		// 随机生成金额（0.01-10000.00）
		amount := float64(rand.Intn(1000000)+1) / 100

		// 随机生成状态
		statuses := []string{"pending", "paid", "failed", "cancelled"}
		status := statuses[rand.Intn(len(statuses))]

		// 生成支付单号
		paymentNo := fmt.Sprintf("PAY%s%s", randomTime.Format("20060102"), ulid.Make().String())

		payment := models.Payment{
			PaymentNo:       paymentNo,
			OrderNo:         orderNo,
			PaymentMethodID: paymentMethod.ID,
			UserID:          userID,
			Amount:          amount,
			Status:          status,
		}

		// 根据状态设置其他字段
		switch status {
		case "paid":
			payment.PayTime = &randomTime
			payment.ThirdPartyNo = fmt.Sprintf("TXN%s%d", randomTime.Format("20060102150405"), rand.Intn(999999))
		case "failed":
			payment.FailReason = "支付超时"
		}

		payments = append(payments, payment)
	}

	// 确保分表存在（使用最早的时间）
	tableName := utils.GetShardingTableName("payments", startDate)
	if !utils.ShardingTableExists(tableName) {
		if err := receiver.ensurePaymentShardingTableExists(startDate); err != nil {
			return fmt.Errorf("创建分表失败: %v", err)
		}
	}

	// 批量插入（按分表分组）
	tableGroups := make(map[string][]models.Payment)
	for _, payment := range payments {
		// 从支付单号提取日期
		if len(payment.PaymentNo) >= 11 {
			dateStr := payment.PaymentNo[3:11] // 提取日期部分 YYYYMMDD
			if parsedTime, err := time.Parse("20060102", dateStr); err == nil {
				tableName := utils.GetShardingTableName("payments", parsedTime)
				tableGroups[tableName] = append(tableGroups[tableName], payment)
			}
		}
	}

	// 分表插入
	for tableName, tablePayments := range tableGroups {
		// 确保分表存在
		if !utils.ShardingTableExists(tableName) {
			// 从表名解析月份
			if len(tableName) > 8 {
				monthStr := tableName[len(tableName)-6:] // 提取 YYYYMM
				if parsedTime, err := time.Parse("200601", monthStr); err == nil {
					if err := receiver.ensurePaymentShardingTableExists(parsedTime); err != nil {
						return fmt.Errorf("创建分表 %s 失败: %v", tableName, err)
					}
				}
			}
		}

		// 转换为 map 以便批量插入
		paymentMaps := make([]map[string]any, len(tablePayments))
		for i, payment := range tablePayments {
			// 从支付单号解析创建时间
			var createdAt time.Time
			if len(payment.PaymentNo) >= 11 {
				dateStr := payment.PaymentNo[3:11] // 提取日期部分 YYYYMMDD
				if parsedTime, err := time.Parse("20060102", dateStr); err == nil {
					// 添加随机时分秒
					createdAt = parsedTime.Add(time.Duration(rand.Intn(86400)) * time.Second)
				}
			}
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}

			paymentMaps[i] = map[string]any{
				"payment_no":        payment.PaymentNo,
				"order_no":          payment.OrderNo,
				"payment_method_id": payment.PaymentMethodID,
				"user_id":           payment.UserID,
				"amount":            payment.Amount,
				"status":            payment.Status,
				"third_party_no":    payment.ThirdPartyNo,
				"pay_time":          payment.PayTime,
				"fail_reason":       payment.FailReason,
				"remark":            payment.Remark,
				"created_at":        createdAt,
				"updated_at":        createdAt,
			}
		}

		if err := facades.Orm().Query().Table(tableName).Create(paymentMaps); err != nil {
			errorlog.Record(context.Background(), "payment", "批量插入支付记录失败", map[string]any{
				"table_name": tableName,
				"count":      len(paymentMaps),
				"error":      err.Error(),
			}, "批量插入支付记录失败: %v", err)
			return fmt.Errorf("插入分表 %s 失败: %v", tableName, err)
		}
	}

	return nil
}

// randomTimeInRange 生成指定时间范围内的随机时间
func randomTimeInRange(start, end time.Time) time.Time {
	delta := end.Sub(start)
	sec := rand.Int63n(int64(delta.Seconds()))
	return start.Add(time.Duration(sec) * time.Second)
}

// ensurePaymentShardingTableExists 确保支付记录分表存在
func (receiver *GenerateTestPayments) ensurePaymentShardingTableExists(paymentTime time.Time) error {
	if receiver.shardingService == nil {
		receiver.shardingService = services.NewShardingService(context.Background())
	}
	tableName := utils.GetShardingTableName("payments", paymentTime)
	return receiver.shardingService.EnsureShardingTable(tableName, "payments")
}
