package dto

import "time"

// OrderSearchCreatedRange C 端订单搜索时间：索引引擎用字符串边界（可全空表示不按时间过滤）；DB 分表用 DBStart/DBEnd（无参数时由解析逻辑填入默认窗口）。
type OrderSearchCreatedRange struct {
	IndexGTE *string
	IndexLTE *string
	DBStart  time.Time
	DBEnd    time.Time
}

// OrderSearchListItem C 端「我的订单」检索行（搜索引擎与数据库路径共用 JSON 形态）。
type OrderSearchListItem struct {
	ID           uint     `json:"id"`
	OrderNo      string   `json:"order_no"`
	Amount       float64  `json:"amount"`
	Status       string   `json:"status"`
	Remark       string   `json:"remark"`
	CreatedAt    string   `json:"created_at"`
	ProductNames []string `json:"product_names"`
}
