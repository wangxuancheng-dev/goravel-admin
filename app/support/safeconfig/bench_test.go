package safeconfig

import (
	"testing"

	"github.com/goravel/framework/facades"
)

// Config reads on the request path go through the RWMutex. These measure what that
// costs against the unwrapped framework config under parallel load.
func BenchmarkGetStringWrapped(b *testing.B) {
	cfg := facades.Config()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cfg.GetString("tenancy.driver")
		}
	})
}

func BenchmarkGetStringUnwrapped(b *testing.B) {
	cfg := facades.Config().(*Config).inner
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cfg.GetString("tenancy.driver")
		}
	})
}
