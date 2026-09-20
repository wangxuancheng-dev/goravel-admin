package utils

import (
	"context"
	"fmt"
	"strings"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

func configCtx(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// GetConfigGroupMap loads all key/value pairs for a configs group.
// Missing group or ORM errors yield an empty map (caller applies defaults).
func GetConfigGroupMap(ctx context.Context, group string) map[string]string {
	out := map[string]string{}
	defer func() {
		if recover() != nil {
		}
	}()
	if group == "" || appfacades.Orm() == nil {
		return out
	}
	var configs []models.Config
	if err := appfacades.OrmQuery(configCtx(ctx)).Where("group", group).Get(&configs); err != nil {
		return out
	}
	for _, c := range configs {
		if c.Key == "" {
			continue
		}
		out[c.Key] = c.Value
	}
	return out
}

// ParseConfigBool parses stored config switch values ("0"/"1"/"true"/...).
func ParseConfigBool(value string) bool {
	v := strings.TrimSpace(strings.ToLower(value))
	switch v {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// GetConfigValue 从数据库获取配置值
func GetConfigValue(ctx context.Context, group, key string, defaultValue string) (result string) {
	result = defaultValue
	defer func() {
		if recover() != nil {
			result = defaultValue
		}
	}()

	if appfacades.Orm() == nil {
		return defaultValue
	}

	var config models.Config
	err := appfacades.OrmQuery(configCtx(ctx)).Where("group", group).Where("key", key).First(&config)
	if err != nil {
		return defaultValue
	}
	// Goravel First returns nil error when record is missing; detect via ID.
	if config.ID == 0 {
		return defaultValue
	}
	if config.Value == "" {
		return defaultValue
	}
	return config.Value
}

// GetConfigValueInt 从数据库获取配置值（整数类型）
func GetConfigValueInt(ctx context.Context, group, key string, defaultValue int) (result int) {
	result = defaultValue
	defer func() {
		if recover() != nil {
			result = defaultValue
		}
	}()

	if appfacades.Orm() == nil {
		return defaultValue
	}

	var config models.Config
	err := appfacades.OrmQuery(configCtx(ctx)).Where("group", group).Where("key", key).First(&config)
	if err != nil {
		return defaultValue
	}
	if config.ID == 0 {
		return defaultValue
	}
	if config.Value == "" {
		return defaultValue
	}
	value := 0
	_, err = fmt.Sscanf(strings.TrimSpace(config.Value), "%d", &value)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetConfigValueBool 从数据库获取配置值（布尔类型）
// 记录存在时以库内值为准（含 "0"）；仅在记录不存在时回退 defaultValue。
func GetConfigValueBool(ctx context.Context, group, key string, defaultValue bool) (result bool) {
	result = defaultValue
	defer func() {
		if recover() != nil {
			result = defaultValue
		}
	}()

	if appfacades.Orm() == nil {
		return defaultValue
	}

	var config models.Config
	err := appfacades.OrmQuery(configCtx(ctx)).Where("group", group).Where("key", key).First(&config)
	if err != nil {
		return defaultValue
	}
	// Goravel First returns nil error when record is missing; detect via ID.
	if config.ID == 0 {
		return defaultValue
	}
	// Key exists in DB: honor stored value (do not fall back to env default on "").
	return ParseConfigBool(config.Value)
}
