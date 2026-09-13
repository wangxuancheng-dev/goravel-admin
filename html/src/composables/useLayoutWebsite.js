import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getConfigByGroup } from '@/api/config'
import {
  resolveImageDisplayUrl,
  WEBSITE_CONFIG_UPDATED_EVENT
} from '@/utils/publicImage'

export function useLayoutWebsite() {
  const { t } = useI18n()
  const websiteSiteName = ref('')
  const websiteSiteLogo = ref('')
  const websiteLogoDisplayUrl = ref('')
  const websiteConfigLoaded = ref(false)
  let revokeLogoUrl = null

  const systemTitle = computed(() => {
    if (!websiteConfigLoaded.value) return ''
    const name = websiteSiteName.value?.trim()
    return name || t('header.system')
  })

  const websiteLogoUrl = computed(() => websiteLogoDisplayUrl.value)

  const clearLogoDisplay = () => {
    if (typeof revokeLogoUrl === 'function') {
      revokeLogoUrl()
      revokeLogoUrl = null
    }
    websiteLogoDisplayUrl.value = ''
  }

  const refreshLogoDisplay = async (raw) => {
    clearLogoDisplay()
    const value = String(raw || '').trim()
    if (!value) return

    const result = await resolveImageDisplayUrl(value)
    // Ignore stale responses if logo changed while fetching
    if (String(websiteSiteLogo.value || '').trim() !== value) {
      if (typeof result.revoke === 'function') result.revoke()
      return
    }
    revokeLogoUrl = result.revoke || null
    websiteLogoDisplayUrl.value = result.url || ''
  }

  watch(websiteSiteLogo, (val) => {
    refreshLogoDisplay(val)
  })

  const loadWebsiteTitle = async () => {
    try {
      const res = await getConfigByGroup('website')
      const configs = res?.data?.configs
      if (Array.isArray(configs)) {
        const pick = (k) => {
          const item = configs.find((config) => {
            const key = config?.Key || config?.key
            return key === k
          })
          const value = item?.Value || item?.value || ''
          return typeof value === 'string' ? value : ''
        }
        websiteSiteName.value = pick('site_name')
        websiteSiteLogo.value = pick('site_logo')
      } else {
        websiteSiteName.value = ''
        websiteSiteLogo.value = ''
      }
    } catch {
      websiteSiteName.value = ''
      websiteSiteLogo.value = ''
    } finally {
      websiteConfigLoaded.value = true
    }
  }

  const onWebsiteConfigUpdated = () => {
    loadWebsiteTitle()
  }

  onMounted(() => {
    window.addEventListener(WEBSITE_CONFIG_UPDATED_EVENT, onWebsiteConfigUpdated)
  })

  onUnmounted(() => {
    window.removeEventListener(WEBSITE_CONFIG_UPDATED_EVENT, onWebsiteConfigUpdated)
    clearLogoDisplay()
  })

  return {
    systemTitle,
    websiteLogoUrl,
    loadWebsiteTitle
  }
}
