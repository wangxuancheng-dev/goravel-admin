package api

import (
	"strconv"
	"time"

	"github.com/goravel/framework/contracts/http"
	contractsqueue "github.com/goravel/framework/contracts/queue"
	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/http/response"
	"goravel/app/jobs"
	"goravel/app/models"
	"goravel/app/services"
)

type QueueTestController struct{}

func NewQueueTestController() *QueueTestController {
	return &QueueTestController{}
}

func uniqueQueueCacheKey(uniqueKey string) string {
	if uniqueKey == "" {
		uniqueKey = "default"
	}

	return "queue-test:unique:" + uniqueKey
}

func (c *QueueTestController) Dispatch(ctx http.Context) http.Response {
	args := []contractsqueue.Arg{
		{Type: "string", Value: "queue-test-dispatch"},
		{Type: "string", Value: time.Now().Format(time.RFC3339)},
	}
	if err := facades.Queue().Job(&jobs.Test{}, args).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	return response.Success(ctx, "success", http.Json{
		"queued":     true,
		"type":       "dispatch",
		"connection": facades.Config().GetString("queue.default", "sync"),
		"queue":      "default",
	})
}

func (c *QueueTestController) Delay(ctx http.Context) http.Response {
	delaySeconds, _ := strconv.Atoi(ctx.Request().Query("seconds", "5"))
	if delaySeconds <= 0 {
		delaySeconds = 5
	}

	args := []contractsqueue.Arg{
		{Type: "string", Value: "queue-test-delay"},
		{Type: "int", Value: delaySeconds},
		{Type: "string", Value: time.Now().Format(time.RFC3339)},
	}
	if err := facades.Queue().Job(&jobs.Test{}, args).Delay(time.Now().Add(time.Duration(delaySeconds) * time.Second)).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	return response.Success(ctx, "success", http.Json{
		"queued":       true,
		"type":         "delay",
		"delay_second": delaySeconds,
		"connection":   facades.Config().GetString("queue.default", "sync"),
		"queue":        "default",
	})
}

func (c *QueueTestController) LongRunning(ctx http.Context) http.Response {
	args := []contractsqueue.Arg{
		{Type: "string", Value: "queue-test-long-running"},
		{Type: "string", Value: time.Now().Format(time.RFC3339)},
	}
	if err := facades.Queue().Job(&jobs.Test{}, args).OnQueue("long-running").Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	return response.Success(ctx, "success", http.Json{
		"queued":     true,
		"type":       "long-running",
		"connection": facades.Config().GetString("queue.default", "sync"),
		"queue":      "long-running",
	})
}

func (c *QueueTestController) Fail(ctx http.Context) http.Response {
	args := []contractsqueue.Arg{
		{Type: "string", Value: "queue-test-fail"},
		{Type: "string", Value: time.Now().Format(time.RFC3339)},
	}
	if err := facades.Queue().Job(&jobs.TestErr{}, args).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	return response.Success(ctx, "success", http.Json{
		"queued":     true,
		"type":       "fail",
		"connection": facades.Config().GetString("queue.default", "sync"),
		"queue":      "default",
	})
}

func (c *QueueTestController) Backoff(ctx http.Context) http.Response {
	marker := "backoff-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	args := []contractsqueue.Arg{
		{Type: "string", Value: marker},
	}
	if err := facades.Queue().Job(&jobs.TestBackoff{}, args).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	return response.Success(ctx, "success", http.Json{
		"queued":      true,
		"type":        "backoff",
		"connection":  facades.Config().GetString("queue.default", "sync"),
		"queue":       "default",
		"marker":      marker,
		"retry_plan":  []int{5, 10, 20},
		"description": "fail first 3 attempts, then succeed on next run",
	})
}

func (c *QueueTestController) Unique(ctx http.Context) http.Response {
	windowSeconds, _ := strconv.Atoi(ctx.Request().Query("window_seconds", "30"))
	if windowSeconds <= 0 {
		windowSeconds = 30
	}

	uniqueKey := ctx.Request().Query("key", "default")
	if uniqueKey == "" {
		uniqueKey = "default"
	}
	cacheKey := uniqueQueueCacheKey(uniqueKey)
	exists := facades.Cache().GetString(cacheKey, "")
	if exists != "" {
		return response.Success(ctx, "success", http.Json{
			"queued":         false,
			"skipped":        true,
			"type":           "unique",
			"key":            uniqueKey,
			"window_seconds": windowSeconds,
			"message":        "already queued within current unique window",
		})
	}

	now := time.Now().Format(time.RFC3339)
	args := []contractsqueue.Arg{
		{Type: "string", Value: "queue-test-unique-" + uniqueKey},
		{Type: "string", Value: now},
	}
	if err := facades.Queue().Job(&jobs.Test{}, args).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	if err := facades.Cache().Put(cacheKey, now, time.Duration(windowSeconds)*time.Second); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	return response.Success(ctx, "success", http.Json{
		"queued":         true,
		"skipped":        false,
		"type":           "unique",
		"key":            uniqueKey,
		"window_seconds": windowSeconds,
		"connection":     facades.Config().GetString("queue.default", "sync"),
		"queue":          "default",
	})
}

func (c *QueueTestController) UniqueStatus(ctx http.Context) http.Response {
	uniqueKey := ctx.Request().Query("key", "default")
	if uniqueKey == "" {
		uniqueKey = "default"
	}
	cacheKey := uniqueQueueCacheKey(uniqueKey)

	startedAt := facades.Cache().GetString(cacheKey, "")

	return response.Success(ctx, "success", http.Json{
		"type":       "unique-status",
		"key":        uniqueKey,
		"active":     startedAt != "",
		"started_at": startedAt,
	})
}

func (c *QueueTestController) AllInOne(ctx http.Context) http.Response {
	delaySeconds, _ := strconv.Atoi(ctx.Request().Query("seconds", "5"))
	if delaySeconds <= 0 {
		delaySeconds = 5
	}

	now := time.Now().Format(time.RFC3339)
	connection := facades.Config().GetString("queue.default", "sync")

	if err := facades.Queue().Job(&jobs.Test{}, []contractsqueue.Arg{
		{Type: "string", Value: "queue-test-all-dispatch"},
		{Type: "string", Value: now},
	}).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	if err := facades.Queue().Job(&jobs.Test{}, []contractsqueue.Arg{
		{Type: "string", Value: "queue-test-all-delay"},
		{Type: "int", Value: delaySeconds},
		{Type: "string", Value: now},
	}).Delay(time.Now().Add(time.Duration(delaySeconds) * time.Second)).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	if err := facades.Queue().Job(&jobs.Test{}, []contractsqueue.Arg{
		{Type: "string", Value: "queue-test-all-long-running"},
		{Type: "string", Value: now},
	}).OnQueue("long-running").Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	if err := facades.Queue().Job(&jobs.TestErr{}, []contractsqueue.Arg{
		{Type: "string", Value: "queue-test-all-fail"},
		{Type: "string", Value: now},
	}).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	return response.Success(ctx, "success", http.Json{
		"queued":       true,
		"type":         "all-in-one",
		"connection":   connection,
		"delay_second": delaySeconds,
		"items": []string{
			"default:dispatch",
			"default:delay",
			"long-running:dispatch",
			"default:fail",
		},
	})
}

func (c *QueueTestController) Reclaim(ctx http.Context) http.Response {
	sleepSeconds, _ := strconv.Atoi(ctx.Request().Query("sleep", "30"))
	if sleepSeconds <= 0 {
		sleepSeconds = 30
	}
	targetQueue := ctx.Request().Query("queue", "default")
	marker := "claim-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	args := []contractsqueue.Arg{
		{Type: "string", Value: marker},
		{Type: "int", Value: sleepSeconds},
	}
	if err := facades.Queue().Job(&jobs.TestClaim{}, args).OnQueue(targetQueue).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	return response.Success(ctx, "success", http.Json{
		"queued":       true,
		"type":         "reclaim",
		"connection":   facades.Config().GetString("queue.default", "sync"),
		"queue":        targetQueue,
		"marker":       marker,
		"sleep_second": sleepSeconds,
		"hint":         "start worker A, then kill before ack, wait retry_after, start worker B",
	})
}

func (c *QueueTestController) AllSpecial(ctx http.Context) http.Response {
	delaySeconds, _ := strconv.Atoi(ctx.Request().Query("delay_seconds", "10"))
	if delaySeconds <= 0 {
		delaySeconds = 10
	}
	reclaimSleep, _ := strconv.Atoi(ctx.Request().Query("reclaim_sleep", "30"))
	if reclaimSleep <= 0 {
		reclaimSleep = 30
	}
	reclaimQueue := ctx.Request().Query("reclaim_queue", "default")
	now := time.Now().Format(time.RFC3339)
	reclaimMarker := "all-special-claim-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	if err := facades.Queue().Job(&jobs.Test{}, []contractsqueue.Arg{
		{Type: "string", Value: "queue-test-all-special-delay"},
		{Type: "int", Value: delaySeconds},
		{Type: "string", Value: now},
	}).Delay(time.Now().Add(time.Duration(delaySeconds) * time.Second)).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	if err := facades.Queue().Job(&jobs.TestErr{}, []contractsqueue.Arg{
		{Type: "string", Value: "queue-test-all-special-fail"},
		{Type: "string", Value: now},
	}).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	if err := facades.Queue().Job(&jobs.TestClaim{}, []contractsqueue.Arg{
		{Type: "string", Value: reclaimMarker},
		{Type: "int", Value: reclaimSleep},
	}).OnQueue(reclaimQueue).Dispatch(); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	return response.Success(ctx, "success", http.Json{
		"queued":           true,
		"type":             "all-special",
		"connection":       facades.Config().GetString("queue.default", "sync"),
		"delay_second":     delaySeconds,
		"reclaim_sleep":    reclaimSleep,
		"reclaim_queue":    reclaimQueue,
		"reclaim_marker":   reclaimMarker,
		"contains":         []string{"delay", "fail", "reclaim"},
		"reclaim_test_tip": "kill worker before ack, wait retry_after, then start another worker",
	})
}

func (c *QueueTestController) Result(ctx http.Context) http.Response {
	return response.Success(ctx, "success", http.Json{
		"test_result":         jobs.TestResult,
		"test_err_result":     jobs.TestErrResult,
		"test_claim_result":   jobs.TestClaimResult,
		"test_backoff_result": jobs.TestBackoffResult,
	})
}

func (c *QueueTestController) Reset(ctx http.Context) http.Response {
	jobs.TestResult = nil
	jobs.TestErrResult = nil
	jobs.TestClaimResult = nil
	jobs.ResetTestBackoff()

	return response.Success(ctx, "success", http.Json{
		"reset": true,
	})
}

// OrderExpireDemo creates a pending order with expire_at and a Delay cancel job (open-source sample).
// Query: user_id (optional), seconds (default 60), amount (default 0.01).
func (c *QueueTestController) OrderExpireDemo(ctx http.Context) http.Response {
	seconds, _ := strconv.Atoi(ctx.Request().Query("seconds", "60"))
	if seconds <= 0 {
		seconds = 60
	}
	if seconds > 3600 {
		seconds = 3600
	}
	userID := uint(0)
	if v := ctx.Request().Query("user_id", ""); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			userID = uint(n)
		}
	}
	if userID == 0 {
		var user models.User
		if err := appfacades.OrmQuery(ctx).Model(&models.User{}).Order("id asc").First(&user); err != nil || user.ID == 0 {
			return response.Error(ctx, http.StatusBadRequest, "order_expire_demo_user_required")
		}
		userID = user.ID
	}
	amount := 0.01
	if v := ctx.Request().Query("amount", ""); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			amount = f
		}
	}

	order, _, err := services.NewOrderService(ctx).CreateOrder(
		userID,
		amount,
		nil,
		"",
		"schedule-demo expire",
		time.Duration(seconds)*time.Second,
	)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}

	expireAt := ""
	if order.ExpireAt != nil {
		expireAt = order.ExpireAt.UTC().Format(time.RFC3339)
	}
	return response.Success(ctx, "success", http.Json{
		"queued":         true,
		"type":           "order-expire-demo",
		"order":          services.OrderToJSONMap(*order),
		"expire_at":      expireAt,
		"delay_seconds":  seconds,
		"connection":     facades.Config().GetString("queue.default", "sync"),
		"hint":           "wait for Delay job or run: go run . artisan order:cancel-expired",
	})
}

