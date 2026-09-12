package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectExportJobRegistration(t *testing.T) {
	base := `func (receiver *QueueServiceProvider) Jobs() []queue.Job {
	return []queue.Job{
		&jobs.ExportOrders{},
		&jobs.ExportArticles{},
		// 搜索引擎同步任务（可切换 driver）
		&queuejobs.SyncOrderSearch{},
	}
}
`

	updated, ok := injectExportJobRegistration(base, "Product")
	require.True(t, ok)
	assert.Contains(t, updated, "&jobs.ExportProducts{},")
	assert.Contains(t, updated, "// 搜索引擎同步任务")

	again, ok := injectExportJobRegistration(updated, "Product")
	assert.False(t, ok)
	assert.Equal(t, updated, again)
}

func TestContainsGeneratedExportJob(t *testing.T) {
	s := &CodeGeneratorServiceImpl{}
	assert.True(t, s.containsGeneratedExportJob([]GeneratedFile{{Path: "app/jobs/export_products.go"}}))
	assert.False(t, s.containsGeneratedExportJob([]GeneratedFile{{Path: "app/jobs/send_email.go"}}))
	assert.False(t, s.containsGeneratedExportJob([]GeneratedFile{{Path: "app/services/product_service.go"}}))
}
