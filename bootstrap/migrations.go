package bootstrap

import (
	"goravel/database/migrations"

	"github.com/goravel/framework/contracts/database/schema"
)

func Migrations() []schema.Migration {
	return []schema.Migration{
		&migrations.M20210101000002CreateJobsTable{},
		// 后台管理系统相关表
		&migrations.M20250101000001CreateDepartmentsTable{},
		&migrations.M20250101000002CreateAdminsTable{},
		&migrations.M20250101000003CreateRolesTable{},
		&migrations.M20250101000004CreatePermissionsTable{},
		&migrations.M20250101000005CreateMenusTable{},
		&migrations.M20250101000006CreateDictionariesTable{},
		&migrations.M20250101000015CreateConfigsTable{},
		&migrations.M20250101000016CreateBlacklistsTable{},
		&migrations.M20250101000007CreateAdminRoleTable{},
		&migrations.M20250101000008CreateRolePermissionTable{},
		&migrations.M20250101000009CreateRoleMenuTable{},
		&migrations.M20250101000010CreateOperationLogsTable{},
		&migrations.M20250101000018AddTitleToOperationLogs{},
		&migrations.M20250101000011CreateLoginLogsTable{},
		&migrations.M20250101000019AddRequestToLoginLogsTable{},
		&migrations.M20250101000012CreateSystemLogsTable{},
		&migrations.M20250201000016AddTraceIdToSystemLogsTable{},
		&migrations.M20250101000014CreatePersonalAccessTokensTable{},
		&migrations.M20250101000017AddOnlineAdminFieldsToPersonalAccessTokens{},
		&migrations.M20250201000003CreateNotificationsTable{},
		&migrations.M20250301000021CreateExportsTable{},
		&migrations.M20250130000006AddErrorMsgToExportsTable{}, // 添加 error_msg 字段到 exports 表
		&migrations.M20250301000024AddTypeToExportsTable{},     // 添加 type 字段到 exports 表
		&migrations.M20250301000022CreateAttachmentsTable{},
		&migrations.M20250301000023AddDisplayNameToAttachments{},
		&migrations.M20250101000024AddGoogleSecretToAdmins{},
		&migrations.M20250101000025AddLinkTypeToMenus{},
		&migrations.M20250101000026ModifyMenusPathLength{},
		&migrations.M20251227063517AddFulltextIndexToOperationLogsRequest{},
		&migrations.M20250128000001CreateOrdersTable{},
		&migrations.M20251228004525AddPaymentMethodToOrdersShardingTables{},
		&migrations.M20250105000001AddCompositeIndexesToOrders{}, // 为订单分表添加复合索引
		// 货币表（需要在用户表之前创建）
		&migrations.M20250130000003CreateCurrenciesTable{},
		&migrations.M20250130000005AddDecimalPlacesToCurrenciesTable{}, // 添加小数位数字段
		// 用户相关表
		&migrations.M20250130000001CreateUsersTable{},
		&migrations.M20250130000004AddCurrencyIdToUsersTable{}, // 添加货币字段（如果用户表已存在）
		&migrations.M20250130000002CreateUserBalanceLogsTable{},
		&migrations.M20250131000003AddTransactionHashToUserBalanceLogsShardingTables{}, // 用户余额变动记录分表新增字段
		// 支付相关表
		&migrations.M20250131000001CreatePaymentMethodsTable{},
		&migrations.M20250131000002CreatePaymentsTable{},
		&migrations.M20250110000001CreatePaymentsShardingTable{}, // 支付记录分表
		&migrations.M20250301000025AddTranslationKeyToDictionaries{},
		// 添加是否缓存字段
		&migrations.M20250131000020AddNoCacheToMenus{},
		// 操作日志增加变更详情字段
		&migrations.M20260328000001AddChangesToOperationLogs{},
		&migrations.M20260404000001CreatePositionsTable{},
		&migrations.M20260404000002AddPositionIdToAdminsTable{},
		&migrations.M20260423000100AddTraceIdToOperationLogsTable{},
		&migrations.M20260423000200CreateSlowQueryLogsTable{},
		&migrations.M20260426021000CreateApiEndpointMetricsTable{},
		&migrations.M20260115152848ArticleTable{},
		&migrations.M20260710000001AdjustSoftDeleteUniqueIndexes{},
		&migrations.M20260711000001CreateSearchSyncOutboxTable{},
		&migrations.M20260716000001CreateAttachmentCategoriesTable{},
		&migrations.M20260717000001AddIsPublicToAttachmentsTable{},
		&migrations.M20260911000001CreateTenantsTable{},
		&migrations.M20260911000002CreatePlatformAdminsTable{},
		&migrations.M20260912000001AddProvisionStatusToTenants{},
		&migrations.M20260912000002AddMigrateMetaToTenants{},
		&migrations.M20260912000003AddDataScopeToRoles{},
		&migrations.M20260913000001AddTenantOpsMeta{},
		&migrations.M20260913000002AddMustChangePasswordToAdmins{},
		&migrations.M20260913000003CreateImportsTable{},
		&migrations.M20260913210000CreateTenantOpLogsTable{},
		&migrations.M20260913223000AddTenantOpLogMeta{},
		&migrations.M20260913240000AddTenantQuotaMeta{},
		&migrations.M20260913250000DropTenantTrafficLimit{},
	}
}
