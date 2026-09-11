package helpers

import (
	"fmt"
	"strings"
	"time"

	"goravel/app/dto"
)

const orderIndexDateLayout = "2006-01-02"
const orderIndexDateTimeLayout = "2006-01-02 15:04:05"

// ParseOrderSearchCreatedRange 解析 created_from / created_to。
// 索引引擎：两参数皆空则 IndexGTE/IndexLTE 为 nil（不按时间过滤）；否则为格式化后的边界字符串。
// DB：两参数皆空则默认 [now-3个月, now]；否则按解析结果构造窗口（单侧缺失时与「近 3 个月」规则组合）。
// 解析失败时返回 errField + errMsgKey（i18n 键，如 validation_start_time_invalid）。
func ParseOrderSearchCreatedRange(createdFrom, createdTo string) (dto.OrderSearchCreatedRange, string, string) {
	var out dto.OrderSearchCreatedRange
	fromS := strings.TrimSpace(createdFrom)
	toS := strings.TrimSpace(createdTo)
	now := time.Now()

	var fromT, toT time.Time
	var hasFrom, hasTo bool

	if fromS != "" {
		t, err := parseOrderCreatedBound(fromS, true)
		if err != nil {
			return out, "created_from", "validation.datetime.start_time_invalid"
		}
		fromT, hasFrom = t, true
		s := t.Format(orderIndexDateTimeLayout)
		out.IndexGTE = &s
	}
	if toS != "" {
		t, err := parseOrderCreatedBound(toS, false)
		if err != nil {
			return out, "created_to", "validation.datetime.end_time_invalid"
		}
		toT, hasTo = t, true
		s := t.Format(orderIndexDateTimeLayout)
		out.IndexLTE = &s
	}
	if hasFrom && hasTo && fromT.After(toT) {
		return out, "created_to", "validation.range.time_inverted"
	}

	switch {
	case !hasFrom && !hasTo:
		out.DBEnd = now
		out.DBStart = now.AddDate(0, -3, 0)
	case hasFrom && !hasTo:
		out.DBStart = fromT
		out.DBEnd = now
	case !hasFrom && hasTo:
		out.DBEnd = toT
		out.DBStart = toT.AddDate(0, -3, 0)
	default:
		out.DBStart = fromT
		out.DBEnd = toT
	}

	return out, "", ""
}

func parseOrderCreatedBound(s string, isStartBound bool) (time.Time, error) {
	s = strings.TrimSpace(s)
	loc := time.Local
	if len(s) == len(orderIndexDateLayout) {
		t, err := time.ParseInLocation(orderIndexDateLayout, s, loc)
		if err != nil {
			return time.Time{}, err
		}
		if isStartBound {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc), nil
		}
		return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, loc), nil
	}
	if len(s) > len(orderIndexDateLayout) {
		return time.ParseInLocation(orderIndexDateTimeLayout, s, loc)
	}
	return time.Time{}, fmt.Errorf("invalid datetime length")
}
