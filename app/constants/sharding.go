package constants

// UserBalanceLogsShards 用户余额变动记录分表数量（已迁移至 config sharding.user_balance_logs_shards）。
// 保留常量供迁移/文档引用，运行时应使用 utils.GetUserBalanceLogsShards()。
// 警告：上线后勿热改分片数，变更需数据迁移。
const UserBalanceLogsShards = 4
