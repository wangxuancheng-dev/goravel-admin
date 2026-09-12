package utils

import "time"

// 时间格式常量
const (
	// DateTimeFormat 标准日期时间格式 (YYYY-MM-DD HH:mm:ss)
	DateTimeFormat = "2006-01-02 15:04:05"

	// DateFormat 标准日期格式 (YYYY-MM-DD)
	DateFormat = "2006-01-02"

	// DateTimeFormatT 带T的日期时间格式 (YYYY-MM-DDTHH:mm:ss)
	DateTimeFormatT = "2006-01-02T15:04:05"

	// DateTimeFormatMs 带毫秒的日期时间格式
	DateTimeFormatMs = "2006-01-02 15:04:05.000"

	// DateTimeFormatTZ 带时区的日期时间格式
	DateTimeFormatTZ = "2006-01-02T15:04:05.000Z07:00"

	// YearMonthFormat 年月格式 (YYYYMM)
	YearMonthFormat = "200601"
)

// ParseDateTime 解析标准格式的日期时间字符串 (2006-01-02 15:04:05)
func ParseDateTime(s string) (time.Time, error) {
	return time.Parse(DateTimeFormat, s)
}

// ParseDateTimeUTC 解析标准格式的日期时间字符串并转换为 UTC
func ParseDateTimeUTC(s string) (time.Time, error) {
	t, err := time.Parse(DateTimeFormat, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// ParseDate 解析标准日期格式字符串 (2006-01-02)
func ParseDate(s string) (time.Time, error) {
	return time.Parse(DateFormat, s)
}

// ParseDateUTC 解析标准日期格式字符串并转换为 UTC
func ParseDateUTC(s string) (time.Time, error) {
	t, err := time.Parse(DateFormat, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// FormatDateTime 格式化时间为标准日期时间字符串 (2006-01-02 15:04:05)
func FormatDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(DateTimeFormat)
}

// FormatDate 格式化时间为标准日期字符串 (2006-01-02)
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(DateFormat)
}

// FormatDateTimePtr 格式化时间指针为标准日期时间字符串（统一按 UTC，配合响应时区转换）
func FormatDateTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(DateTimeFormat)
}

// ParseDateTimeInLocation 在指定时区解析日期时间字符串
func ParseDateTimeInLocation(s string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(DateTimeFormat, s, loc)
}

// FormatYearMonth 格式化时间为年月字符串 (200601)
func FormatYearMonth(t time.Time) string {
	return t.Format(YearMonthFormat)
}

// ParseYearMonth 解析年月格式字符串 (200601)
func ParseYearMonth(s string) (time.Time, error) {
	return time.Parse(YearMonthFormat, s)
}
