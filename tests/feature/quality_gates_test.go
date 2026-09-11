package feature_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/models"
	"goravel/app/utils"
	"goravel/tests"
)

func TestOrdersModuleDisabledBlocksAPI(t *testing.T) {
	token := loginAdminWithPermission(t, "order_module_actor", "OrderModuleActor123!", "order.index", "GET", "/api/admin/orders")

	prevOrders := facades.Config().GetBool("module.orders_enabled", true)
	prevPayments := facades.Config().GetBool("module.payments_enabled", false)
	prevFrontend := facades.Config().GetString("module.code_generator_frontend", "vue,react")
	facades.Config().Add("module", map[string]any{
		"orders_enabled":          false,
		"payments_enabled":        prevPayments,
		"code_generator_frontend": prevFrontend,
	})
	t.Cleanup(func() {
		facades.Config().Add("module", map[string]any{
			"orders_enabled":          prevOrders,
			"payments_enabled":        prevPayments,
			"code_generator_frontend": prevFrontend,
		})
	})
	require.False(t, utils.OrdersEnabled(), "module toggle should disable orders")

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/orders")
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "module_orders_disabled")
}

func TestRoleIndexAllowedWithPermission(t *testing.T) {
	token := loginAdminWithPermission(t, "role_list_actor", "RoleListActor123!", "role.index", "GET", "/api/admin/roles")

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/roles")
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	assert.Equal(t, 200, payload.Code, "body: %s", content)
}

func TestOrderIndexAllowedWithPermission(t *testing.T) {
	prevOrders := facades.Config().GetBool("module.orders_enabled", true)
	prevPayments := facades.Config().GetBool("module.payments_enabled", false)
	prevFrontend := facades.Config().GetString("module.code_generator_frontend", "vue,react")
	facades.Config().Add("module", map[string]any{
		"orders_enabled":          true,
		"payments_enabled":        prevPayments,
		"code_generator_frontend": prevFrontend,
	})
	t.Cleanup(func() {
		facades.Config().Add("module", map[string]any{
			"orders_enabled":          prevOrders,
			"payments_enabled":        prevPayments,
			"code_generator_frontend": prevFrontend,
		})
	})

	token := loginAdminWithPermission(t, "order_list_actor", "OrderListActor123!", "order.index", "GET", "/api/admin/orders")

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/orders")
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	assert.Equal(t, 200, payload.Code, "body: %s", content)
}

func TestDictionaryIndexAllowedWithPermission(t *testing.T) {
	token := loginAdminWithPermission(t, "dict_list_actor", "DictListActor123!", "dictionary.index", "GET", "/api/admin/dictionaries")

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/dictionaries")
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	assert.Equal(t, 200, payload.Code, "body: %s", content)
}

func TestPaymentNotifyStubReturnsNotImplemented(t *testing.T) {
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		Post("/api/payment/notify/wechat", strings.NewReader(`{"out_trade_no":"demo"}`))
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code      int    `json:"code"`
		ErrorCode string `json:"error_code"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	assert.Equal(t, 501, payload.Code, "body: %s", content)
	assert.Equal(t, "payment_gateway_not_implemented", payload.ErrorCode)
}

func loginAdminWithPermission(t *testing.T, username, password, slug, method, path string) string {
	t.Helper()

	admin := ensureNamedAdmin(t, username, password)
	roleSlug := "perm_" + strings.ReplaceAll(slug, ".", "_")

	var role models.Role
	err := facades.Orm().Query().Where("slug", roleSlug).First(&role)
	if err != nil || role.ID == 0 {
		role = models.Role{
			Name:   roleSlug,
			Slug:   roleSlug,
			Status: 1,
			Sort:   990,
		}
		require.NoError(t, facades.Orm().Query().Create(&role))
	}

	var perm models.Permission
	err = facades.Orm().Query().Where("slug", slug).First(&perm)
	if err != nil || perm.ID == 0 {
		perm = models.Permission{
			Name:   slug,
			Slug:   slug,
			Method: method,
			Path:   path,
			Status: 1,
			Sort:   1,
		}
		require.NoError(t, facades.Orm().Query().Create(&perm))
	} else {
		// Ensure path/method match expected route for this test.
		perm.Method = method
		perm.Path = path
		perm.Status = 1
		require.NoError(t, facades.Orm().Query().Save(&perm))
	}

	require.NoError(t, facades.Orm().Query().Model(&role).Association("Permissions").Replace([]models.Permission{perm}))
	require.NoError(t, facades.Orm().Query().Model(&admin).Association("Roles").Replace([]models.Role{role}))

	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		Post("/api/admin/login", strings.NewReader(body))
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	require.Equal(t, 200, payload.Code, "login body: %s", content)
	require.NotEmpty(t, payload.Data.Token)
	return payload.Data.Token
}
