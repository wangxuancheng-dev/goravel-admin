package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("jwt", map[string]any{
		// JWT Authentication Secret
		//
		// Don't forget to set this in your .env file, as it will be used to sign
		// your tokens. A helper command is provided for this:
		// `go run . artisan jwt:secret`
		"secret": config.Env("JWT_SECRET", ""),

		// JWT time to live
		//
		// Specify the length of time (in minutes) that the token will be valid for.
		// Defaults to 1 hour.
		//
		// Note: The system implements sliding expiration, which means the token
		// expiration time will be automatically extended on each request.
		// This ensures that active users won't be logged out unexpectedly.
		//
		// You can also set this to 0, to yield a never expiring token.
		// Some people may want this behaviour for e.g. a mobile app.
		// This is not particularly recommended, so make sure you have appropriate
		// systems in place to revoke the token if necessary.
		"ttl": config.Env("JWT_TTL", 60),

		// Refresh time to live (minutes)
		//
		// How long after expiry a token can still be refreshed. Since framework v1.18.1
		// a logged-out token also stays blacklisted in cache for exp + refresh_ttl, so
		// leaving this unset (falls back to 100 years) keeps refresh windows unbounded
		// and logout cache entries effectively forever. Defaults to 2 weeks.
		"refresh_ttl": config.Env("JWT_REFRESH_TTL", 20160),
	})
}
