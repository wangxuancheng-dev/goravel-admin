package admin

import (
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/http/trans"
	"goravel/app/services"
	"goravel/app/services/option_providers"
)

type OptionController struct{}

func NewOptionController() *OptionController {
	return &OptionController{}
}

func (r *OptionController) dictionaryService(ctx http.Context) services.DictionaryService {
	return services.NewDictionaryService(ctx)
}

func (r *OptionController) provider(ctx http.Context, optionType string) (services.OptionProvider, bool) {
	if p, ok := services.LookupOptionProvider(ctx, optionType); ok {
		return p, true
	}

	tree := services.NewTreeServiceImpl(ctx)
	providers := map[string]services.OptionProvider{
		"role":                option_providers.NewRoleOptionProvider(ctx),
		"department":          option_providers.NewDepartmentOptionProvider(ctx),
		"position":            option_providers.NewPositionOptionProvider(ctx),
		"attachment_category": option_providers.NewAttachmentCategoryOptionProvider(ctx),
		"menu":                option_providers.NewMenuOptionProvider(ctx, tree),
		"status":              option_providers.NewStatusOptionProvider(ctx),
		"method":              option_providers.NewMethodOptionProvider(ctx),
		"yes_no":              option_providers.NewYesNoOptionProvider(ctx),
		"admin":               option_providers.NewAdminOptionProvider(ctx),
		"payment_method":      option_providers.NewPaymentMethodOptionProvider(ctx),
		"form_demo":           option_providers.NewFormDemoOptionProvider(ctx),
	}
	provider, ok := providers[optionType]
	return provider, ok
}

// Index 获取选项列表
// 通过 type 参数指定选项类型，例如: /options?type=role
func (r *OptionController) Index(ctx http.Context) http.Response {
	optionType := ctx.Request().Query("type", "")

	if optionType == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrOptionTypeRequired.Code)
	}

	// 如果 type 是 "dictionary"，则需要进一步检查 dictionary_type 参数
	if optionType == "dictionary" {
		dictType := ctx.Request().Query("dictionary_type", "")
		if dictType == "" {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrOptionTypeRequired.Code)
		}

		dictionaries, err := r.dictionaryService(ctx).GetByType(dictType)
		if err != nil {
			return response.Error(ctx, http.StatusInternalServerError, apperrors.ErrQueryFailed.Code)
		}

		// 转换为选项格式，并处理多语言
		var options []map[string]any
		for _, dict := range dictionaries {
			label := dict.Label
			// 如果有 translation_key，优先使用翻译键
			if dict.TranslationKey != "" {
				translated := trans.Get(ctx, dict.TranslationKey)
				// 只有当翻译结果不同于 key 本身（表示找到了翻译）且不为空时才使用
				if translated != "" && translated != dict.TranslationKey {
					label = translated
				}
			}

			options = append(options, map[string]any{
				"label": label,
				"value": dict.Value,
			})
		}
		return response.Success(ctx, options)
	}

	provider, exists := r.provider(ctx, optionType)
	if !exists {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidOptionType.Code)
	}

	data, err := provider.GetOptions(ctx)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, apperrors.ErrQueryFailed.Code)
	}

	return response.Success(ctx, data)
}

// RegisterProvider registers a custom option type via services.RegisterOptionProvider.
// Prefer calling services.RegisterOptionProvider from init() in secondary-dev code.
func RegisterProvider(optionType string, factory services.OptionProviderFactory) {
	services.RegisterOptionProvider(optionType, factory)
}