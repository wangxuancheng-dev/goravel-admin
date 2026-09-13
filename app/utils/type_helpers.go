package utils

import (
	"fmt"
	"reflect"
	"strings"
)

// GetValue ? map[string]any ????????????
// ???????????????????? false
func GetValue[T any](m map[string]any, key string) (T, bool) {
	var zero T
	val, ok := m[key]
	if !ok {
		return zero, false
	}

	// ????????
	if v, ok := val.(T); ok {
		return v, true
	}

	// ??????????????????
	return convertNumeric[T](val)
}

// GetUint ? map[string]any ??? uint ?????????????
func GetUint(m map[string]any, key string) (uint, bool) {
	return GetValue[uint](m, key)
}

// GetFloat64 ? map[string]any ??? float64 ?????????????
func GetFloat64(m map[string]any, key string) (float64, bool) {
	return GetValue[float64](m, key)
}

// GetString ? map[string]any ??? string ?
func GetString(m map[string]any, key string) (string, bool) {
	val, ok := m[key]
	if !ok {
		return "", false
	}
	if v, ok := val.(string); ok {
		return v, true
	}
	return "", false
}

// GetMap ? map[string]any ??? map[string]any ?
func GetMap(m map[string]any, key string) (map[string]any, bool) {
	val, ok := m[key]
	if !ok {
		return nil, false
	}
	if v, ok := val.(map[string]any); ok {
		return v, true
	}
	return nil, false
}

// convertNumeric ???????????????????
func convertNumeric[T any](val any) (T, bool) {
	var zero T
	switch v := val.(type) {
	case float64:
		return convertFromFloat64[T](v)
	case int:
		return convertFromInt[T](v)
	case uint:
		return convertFromUint[T](v)
	case int64:
		return convertFromInt64[T](v)
	case uint64:
		return convertFromUint64[T](v)
	default:
		return zero, false
	}
}

// convertFromFloat64 ? float64 ??
func convertFromFloat64[T any](v float64) (T, bool) {
	var zero T
	switch any(zero).(type) {
	case uint:
		return any(uint(v)).(T), true
	case int:
		return any(int(v)).(T), true
	case float64:
		return any(v).(T), true
	default:
		return zero, false
	}
}

// convertFromInt ? int ??
func convertFromInt[T any](v int) (T, bool) {
	var zero T
	switch any(zero).(type) {
	case uint:
		return any(uint(v)).(T), true
	case int:
		return any(v).(T), true
	case float64:
		return any(float64(v)).(T), true
	default:
		return zero, false
	}
}

// convertFromUint ? uint ??
func convertFromUint[T any](v uint) (T, bool) {
	var zero T
	switch any(zero).(type) {
	case uint:
		return any(v).(T), true
	case int:
		return any(int(v)).(T), true
	case float64:
		return any(float64(v)).(T), true
	default:
		return zero, false
	}
}

// convertFromInt64 ? int64 ??
func convertFromInt64[T any](v int64) (T, bool) {
	var zero T
	switch any(zero).(type) {
	case uint:
		return any(uint(v)).(T), true
	case int:
		return any(int(v)).(T), true
	case float64:
		return any(float64(v)).(T), true
	default:
		return zero, false
	}
}

// convertFromUint64 ? uint64 ??
func convertFromUint64[T any](v uint64) (T, bool) {
	var zero T
	switch any(zero).(type) {
	case uint:
		return any(uint(v)).(T), true
	case int:
		return any(int(v)).(T), true
	case float64:
		return any(float64(v)).(T), true
	default:
		return zero, false
	}
}

// MustGetValue ? map[string]any ????????????????? panic
// ???????????????
func MustGetValue[T any](m map[string]any, key string) T {
	val, ok := GetValue[T](m, key)
	if !ok {
		panic(fmt.Sprintf("key %s not found or type mismatch in map", key))
	}
	return val
}

// FillFiltersFromMap ? map[string]any ?? Filters ???
// ?? string, uint, float64 ????????? snake_case ?? map ? key
// ???
//
//	filters := SomeFilters{}
//	utils.FillFiltersFromMap(m, &filters)
func FillFiltersFromMap(m map[string]any, filtersPtr any) {
	v := reflect.ValueOf(filtersPtr)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return
	}
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}

		structField := t.Field(i)

		// ?? json tag ??? snake_case ???
		key := structField.Tag.Get("json")
		if key == "" || key == "-" {
			key = toSnakeCase(structField.Name)
		}

		switch field.Kind() {
		case reflect.String:
			if val, ok := GetString(m, key); ok {
				field.SetString(val)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if val, ok := GetUint(m, key); ok {
				field.SetUint(uint64(val))
			}
		case reflect.Float64, reflect.Float32:
			if val, ok := GetFloat64(m, key); ok {
				field.SetFloat(val)
			}
		}
	}
}

// ExportFiltersToMap ???????? map???????????????? FillFiltersFromMap ???
// ????/?????? Filters ?????? ExportFiltersToMap(filters) ???Job ? FillFiltersFromMap ???? BuildXxxQuery?
func ExportFiltersToMap(filters any) map[string]any {
	v := reflect.ValueOf(filters)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return map[string]any{}
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return map[string]any{}
	}

	t := v.Type()
	out := make(map[string]any)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := t.Field(i)

		key := structField.Tag.Get("json")
		if key == "" || key == "-" {
			key = toSnakeCase(structField.Name)
		}

		if !field.CanInterface() {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			s := field.String()
			if strings.TrimSpace(s) != "" {
				out[key] = s
			}
		case reflect.Bool:
			// ??? true?false ???????
			if field.Bool() {
				out[key] = true
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() != 0 {
				out[key] = field.Int()
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if field.Uint() != 0 {
				out[key] = field.Uint()
			}
		case reflect.Float32, reflect.Float64:
			if field.Float() != 0 {
				out[key] = field.Float()
			}
		default:
			// ?????? time.Time??????????????????
		}
	}

	return out
}

// toSnakeCase ? PascalCase/camelCase ??? snake_case
func toSnakeCase(s string) string {
	var buf []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				prevLower := s[i-1] >= 'a' && s[i-1] <= 'z'
				nextLower := i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z'
				if prevLower || nextLower {
					buf = append(buf, '_')
				}
			}
			buf = append(buf, c+32)
		} else {
			buf = append(buf, c)
		}
	}
	return string(buf)
}
