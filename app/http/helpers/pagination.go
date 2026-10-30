package helpers

import (
	"github.com/goravel/framework/contracts/http"
)

// PaginationLimits 分页边界；零值表示 DefaultPageSize=10、MaxPageSize=100（与 PaginateQuery 一致）。
type PaginationLimits struct {
	DefaultPageSize int
	MaxPageSize     int
}

// PaginationFromQuery 从 Query 读取 page、page_size 并规范化（GET 列表推荐写法，等同 OnlineAdmin / Export 里两行合并）。
func PaginationFromQuery(ctx http.Context, limits PaginationLimits) (page, pageSize int) {
	page = GetIntQuery(ctx, "page", 1)
	pageSize = GetIntQuery(ctx, "page_size", 10)
	return ValidatePaginationEx(page, pageSize, limits)
}

// ValidatePaginationEx 按给定边界规范化 page、page_size。
func ValidatePaginationEx(page, pageSize int, limits PaginationLimits) (int, int) {
	def := limits.DefaultPageSize
	if def < 1 {
		def = 10
	}
	max := limits.MaxPageSize
	if max < 1 {
		max = 100
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = def
	}
	if pageSize > max {
		pageSize = max
	}
	return page, pageSize
}

// ValidatePagination 验证并规范化分页参数（默认每页 10、最大 100）。
func ValidatePagination(page, pageSize int) (int, int) {
	return ValidatePaginationEx(page, pageSize, PaginationLimits{})
}

// MaxListRows 普通单表列表的分页深度上限（offset+page_size）。OFFSET 需要数据库读出并丢弃
// 之前的所有行，越往后翻越慢；压测中操作日志翻到第 5000 页时 p50 达到 141ms。
const MaxListRows = 10000

// ListTooDeep 判断 page/page_size 是否超出 MaxListRows（参数应已经过 ValidatePagination 规范化）。
func ListTooDeep(page, pageSize int) bool {
	return page*pageSize > MaxListRows
}

// TotalPages 根据总条数与每页条数计算总页数（total 小于 1 或 pageSize 小于 1 时为 0）。
func TotalPages(total int64, pageSize int) int {
	if pageSize < 1 || total < 1 {
		return 0
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}

// PaginateSlice 对切片进行分页处理
// 返回分页后的切片和总数
func PaginateSlice[T any](slice []T, page, pageSize int) ([]T, int64) {
	total := int64(len(slice))
	if total == 0 {
		return []T{}, 0
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	// 如果起始位置超出范围，返回空切片
	if start >= len(slice) {
		return []T{}, total
	}

	// 如果结束位置超出范围，截取到末尾
	if end > len(slice) {
		end = len(slice)
	}

	return slice[start:end], total
}
