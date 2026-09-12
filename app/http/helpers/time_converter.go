package helpers

import (
	"encoding/json"
	"goravel/app/utils"
	"reflect"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/carbon"
)

// ConvertTimesInData 递归转换数据中的时间字段到对应时区
// 使用 JSON 序列化和反序列化来确保正确处理所有类型
func ConvertTimesInData(ctx http.Context, data any) any {
	if data == nil {
		return nil
	}

	// 仅在请求明确指定时区时执行转换，避免默认时区影响审计字段语义
	if !hasTimezoneRequest(ctx) {
		return data
	}

	// 获取请求的时区
	timezone := GetCurrentTimezone(ctx)
	timeFields := getTimeFieldWhitelist()

	// 先序列化为 JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		// 如果序列化失败，尝试使用反射方法
		return convertTimesInValue(reflect.ValueOf(data), timezone, timeFields)
	}

	// 反序列化为 map[string]any
	var result any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		// 如果反序列化失败，返回原数据
		return data
	}

	// 转换时间字段
	converted := convertTimesInMap(result, timezone, timeFields)

	return converted
}

// convertTimesInMap 递归处理 map 或 slice 中的时间字段
func convertTimesInMap(data any, timezone string, timeFields map[string]struct{}) any {
	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any)
		for key, value := range v {
			// 检查是否是时间字段
			if isTimeField(key, timeFields) {
				// 尝试解析时间字符串并转换
				if timeStr, ok := value.(string); ok && timeStr != "" {
					// 统一走转换逻辑（包括目标时区为 UTC 的场景）
					converted := convertTimeString(timeStr, timezone)
					if converted != nil && converted != "" {
						result[key] = converted
						continue
					}
					// 如果转换失败，保留原值
					result[key] = timeStr
					continue
				}
			}
			// 递归处理嵌套数据
			result[key] = convertTimesInMap(value, timezone, timeFields)
		}
		return result

	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = convertTimesInMap(item, timezone, timeFields)
		}
		return result

	default:
		return data
	}
}

// convertTimeString 转换时间字符串到指定时区
// 对于无时区信息的字符串（如 2025-11-22 06:21:25），
// 先按“存储时区”解释，再转换到目标时区。
func convertTimeString(timeStr string, timezone string) any {
	if timeStr == "" || timeStr == "null" {
		return nil
	}

	targetLoc, err := time.LoadLocation(timezone)
	if err != nil {
		return timeStr
	}

	// 无时区时间字符串统一按 UTC 作为源时区解释。
	// 若需兼容历史非 UTC 数据，可通过 app.display_source_timezone 覆盖。
	sourceTimezone := facades.Config().GetString("app.display_source_timezone", carbon.UTC)
	sourceTimezone = NormalizeTimezone(sourceTimezone)
	sourceLoc, sourceErr := time.LoadLocation(sourceTimezone)
	if sourceErr != nil {
		sourceLoc, _ = time.LoadLocation("Asia/Shanghai")
	}

	if t, parseErr := time.ParseInLocation(utils.DateTimeFormat, timeStr, sourceLoc); parseErr == nil {
		return t.In(targetLoc).Format(utils.DateTimeFormat)
	}

	// 带时区信息（RFC3339）按其自身时区转换
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}

	return t.In(targetLoc).Format(utils.DateTimeFormat)
}

// convertTimesInValue 使用反射方法处理值
func convertTimesInValue(v reflect.Value, timezone string, timeFields map[string]struct{}) any {
	if !v.IsValid() {
		return nil
	}

	// 处理指针
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		return convertTimesInValue(v.Elem(), timezone, timeFields)
	}

	// 处理时间类型
	if v.Type() == reflect.TypeOf((*carbon.DateTime)(nil)).Elem() {
		dt := v.Interface().(carbon.DateTime)
		return dt.SetTimezone(timezone).ToDateTimeString()
	}

	// 处理 *carbon.DateTime
	if v.Type() == reflect.TypeOf((*carbon.DateTime)(nil)) {
		if v.IsNil() {
			return nil
		}
		dt := v.Interface().(*carbon.DateTime)
		if dt == nil {
			return nil
		}
		return dt.SetTimezone(timezone).ToDateTimeString()
	}

	// 处理 time.Time
	if v.Type() == reflect.TypeOf(time.Time{}) {
		t := v.Interface().(time.Time)
		dt := carbon.NewDateTime(carbon.Parse(t.Format(utils.DateTimeFormat)))
		return dt.SetTimezone(timezone).ToDateTimeString()
	}

	// 处理 *time.Time
	if v.Type() == reflect.TypeOf((*time.Time)(nil)) {
		if v.IsNil() {
			return nil
		}
		t := v.Interface().(*time.Time)
		if t == nil {
			return nil
		}
		dt := carbon.NewDateTime(carbon.Parse(t.Format(utils.DateTimeFormat)))
		return dt.SetTimezone(timezone).ToDateTimeString()
	}

	// 处理切片
	if v.Kind() == reflect.Slice {
		if v.IsNil() {
			return nil
		}
		result := make([]any, v.Len())
		for i := range result {
			result[i] = convertTimesInValue(v.Index(i), timezone, timeFields)
		}
		return result
	}

	// 处理数组
	if v.Kind() == reflect.Array {
		result := make([]any, v.Len())
		for i := range result {
			result[i] = convertTimesInValue(v.Index(i), timezone, timeFields)
		}
		return result
	}

	// 处理 map
	if v.Kind() == reflect.Map {
		if v.IsNil() {
			return nil
		}
		result := make(map[string]any)
		for _, key := range v.MapKeys() {
			keyStr := key.String()
			if key.Kind() == reflect.Interface {
				keyStr = reflect.ValueOf(key.Interface()).String()
			}
			result[keyStr] = convertTimesInValue(v.MapIndex(key), timezone, timeFields)
		}
		return result
	}

	// 处理结构体
	if v.Kind() == reflect.Struct {
		result := make(map[string]any)
		t := v.Type()
		for i := range make([]int, v.NumField()) {
			field := t.Field(i)
			fieldValue := v.Field(i)

			// 跳过未导出字段
			if !fieldValue.CanInterface() {
				continue
			}

			fieldName := field.Name
			// 检查 json tag
			if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
				// 解析 json tag（处理 "name,omitempty" 格式）
				parts := strings.Split(jsonTag, ",")
				if len(parts) > 0 && parts[0] != "" {
					fieldName = parts[0]
				}
			}

			// 只处理时间相关字段
			if isTimeField(fieldName, timeFields) || isTimeType(fieldValue.Type()) {
				result[fieldName] = convertTimesInValue(fieldValue, timezone, timeFields)
			} else {
				// 递归处理嵌套结构
				if fieldValue.Kind() == reflect.Struct || fieldValue.Kind() == reflect.Ptr || fieldValue.Kind() == reflect.Slice || fieldValue.Kind() == reflect.Map {
					result[fieldName] = convertTimesInValue(fieldValue, timezone, timeFields)
				} else {
					result[fieldName] = fieldValue.Interface()
				}
			}
		}
		return result
	}

	// 其他类型直接返回
	return v.Interface()
}

// isTimeField 检查字段名是否是时间字段
func isTimeField(fieldName string, timeFields map[string]struct{}) bool {
	_, exists := timeFields[strings.ToLower(fieldName)]
	return exists
}

// isTimeType 检查类型是否是时间类型
func isTimeType(t reflect.Type) bool {
	if t == reflect.TypeOf((*carbon.DateTime)(nil)).Elem() ||
		t == reflect.TypeOf((*carbon.DateTime)(nil)) ||
		t == reflect.TypeOf(time.Time{}) ||
		t == reflect.TypeOf((*time.Time)(nil)) {
		return true
	}
	return false
}

func hasTimezoneRequest(ctx http.Context) bool {
	return ctx.Request().Header("X-Timezone", "") != "" ||
		ctx.Request().Header("Timezone", "") != "" ||
		ctx.Request().Input("timezone") != ""
}

func getTimeFieldWhitelist() map[string]struct{} {
	defaultFields := []string{
		"created_at", "updated_at", "deleted_at",
		"pay_time",
		"createdat", "updatedat", "deletedat", "paytime",
	}
	whitelist := make(map[string]struct{}, len(defaultFields))
	for _, field := range defaultFields {
		whitelist[field] = struct{}{}
	}

	// 支持通过配置扩展字段：app.response_time_fields=created_at,updated_at,deleted_at,pay_time
	configFields := facades.Config().GetString("app.response_time_fields", "")
	if strings.TrimSpace(configFields) == "" {
		return whitelist
	}

	for _, field := range strings.Split(configFields, ",") {
		normalized := strings.ToLower(strings.TrimSpace(field))
		if normalized == "" {
			continue
		}
		whitelist[normalized] = struct{}{}
	}

	return whitelist
}
