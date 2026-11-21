package feature_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/models"
)

var errRollbackZeroValue = errors.New("rollback: zero-value create probe")

// Creating a row with a deliberate 0/false must store 0/false. These fields used to carry
// gorm default:1/true, which gorm substitutes for a zero value before INSERT: disabled
// admins came out enabled, private attachments public, failed operations logged as
// successes, private plans public. Everything runs in a rolled-back transaction, so the
// dev database keeps no rows.
func TestCreateKeepsDeliberateZeroValues(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	cases := []struct {
		name   string
		create func(tx orm.Query) (reload func() any, err error)
		want   any
	}{
		{
			name: "admin disabled",
			create: func(tx orm.Query) (func() any, error) {
				row := models.Admin{Username: "zero_" + suffix, Password: "x", Nickname: "zero", Status: 0}
				err := tx.Create(&row)
				return func() any { var got models.Admin; _ = tx.Where("id", row.ID).First(&got); return got.Status }, err
			},
			want: uint8(0),
		},
		{
			name: "attachment private",
			create: func(tx orm.Query) (func() any, error) {
				row := models.Attachment{Disk: "local", Path: "zero/" + suffix + ".pdf", Filename: "zero.pdf", FileType: "document", Status: 1, IsPublic: 0}
				err := tx.Create(&row)
				return func() any { var got models.Attachment; _ = tx.Where("id", row.ID).First(&got); return got.IsPublic }, err
			},
			want: uint8(0),
		},
		{
			name: "operation log failed",
			create: func(tx orm.Query) (func() any, error) {
				row := models.OperationLog{Method: "POST", Path: "/zero/" + suffix, Title: "zero", Status: 0}
				err := tx.Create(&row)
				return func() any { var got models.OperationLog; _ = tx.Where("id", row.ID).First(&got); return got.Status }, err
			},
			want: uint8(0),
		},
		{
			name: "role disabled",
			create: func(tx orm.Query) (func() any, error) {
				row := models.Role{Name: "zero_" + suffix, Slug: "zero_" + suffix, Status: 0, DataScope: models.DataScopeAll}
				err := tx.Create(&row)
				return func() any { var got models.Role; _ = tx.Where("id", row.ID).First(&got); return got.Status }, err
			},
			want: uint8(0),
		},
		{
			name: "plan not public",
			create: func(tx orm.Query) (func() any, error) {
				row := models.PlatformPlan{Code: "zero_" + suffix, Name: "zero", Status: 1, IsPublic: false, Currency: "CNY"}
				err := tx.Create(&row)
				return func() any { var got models.PlatformPlan; _ = tx.Where("id", row.ID).First(&got); return got.IsPublic }, err
			},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := facades.Orm().Transaction(func(tx orm.Query) error {
				reload, err := tc.create(tx)
				require.NoError(t, err)
				assert.Equal(t, tc.want, reload())
				return errRollbackZeroValue
			})
			require.ErrorIs(t, err, errRollbackZeroValue)
		})
	}
}
