package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyHealth(t *testing.T) {
	assert.Equal(t, TenantHealthOK, classifyHealth(nil))
	assert.Equal(t, TenantHealthOK, classifyHealth([]string{}))
	assert.Equal(t, TenantHealthFail, classifyHealth([]string{HealthIssuePingFail}))
	assert.Equal(t, TenantHealthFail, classifyHealth([]string{HealthIssueSchemaBehind, HealthIssueMigrateFail}))
	assert.Equal(t, TenantHealthWarn, classifyHealth([]string{HealthIssueSchemaBehind}))
	assert.Equal(t, TenantHealthWarn, classifyHealth([]string{HealthIssueQuotaHigh, HealthIssueDomainFail}))
}

func TestEncodeParseHealthIssues(t *testing.T) {
	raw := encodeHealthIssues([]string{HealthIssuePingFail, HealthIssueQuotaHigh})
	assert.Equal(t, []string{HealthIssuePingFail, HealthIssueQuotaHigh}, parseHealthIssues(raw))
	assert.Equal(t, []string{}, parseHealthIssues(""))
	assert.Equal(t, "[]", encodeHealthIssues(nil))
}
