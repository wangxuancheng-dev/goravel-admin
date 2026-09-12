package providers

import (
	"github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/facades"

	"goravel/app/models"
	"goravel/app/production"
	"goravel/app/utils"
)

type AppServiceProvider struct {
}

func (receiver *AppServiceProvider) Register(app foundation.Application) {

}

func (receiver *AppServiceProvider) Boot(app foundation.Application) {
	// 从数据库同步文件存储驱动选择到框架配置
	// 框架的其他配置（密钥、bucket等）直接从 .env 读取
	receiver.syncStorageDiskFromDatabase()
	production.WarnInsecureDefaults()
}

// syncStorageDiskFromDatabase 从数据库同步文件存储驱动选择
// 只同步驱动选择（file_disk），其他配置（密钥、bucket等）从 .env 读取
func (receiver *AppServiceProvider) syncStorageDiskFromDatabase() {
	// 使用 recover 来捕获可能的 panic（例如在构建时数据库不可用）
	defer func() {
		if r := recover(); r != nil {
			// 静默处理，使用 .env 的默认值
			facades.Log().Debugf("Failed to sync storage disk from database, using .env defaults: %v", r)
		}
	}()

	// 检查数据库连接是否可用
	orm := facades.Orm()
	if orm == nil {
		return
	}
	// 首次迁移前 configs 表不存在，直接跳过，避免启动阶段打印 SQL 错误
	if !facades.Schema().HasTable("configs") {
		return
	}

	// 获取 storage 分组的 file_disk 配置
	// 优先使用 file_disk，如果没有则使用 storage_disk（向后兼容）
	var config models.Config
	disk := ""
	if err := orm.Query().Where("group", "storage").Where("key", "file_disk").First(&config); err == nil && config.Value != "" {
		disk = config.Value
	} else {
		// 如果 file_disk 不存在，尝试读取 storage_disk（向后兼容）
		if err := orm.Query().Where("group", "storage").Where("key", "storage_disk").First(&config); err == nil && config.Value != "" {
			disk = config.Value
		}
	}

	// 如果数据库中有配置，更新框架的默认磁盘（云盘密钥未配齐时不切换，避免运行时 panic）
	if disk != "" {
		if err := utils.ValidateFilesystemDisk(disk); err != nil {
			facades.Log().Warningf("Skip syncing storage disk %q: %v", disk, err)
			return
		}
		facades.Config().Add("filesystems.default", disk)
		facades.Log().Debugf("Storage disk synced from database: %s", disk)
	}
}
