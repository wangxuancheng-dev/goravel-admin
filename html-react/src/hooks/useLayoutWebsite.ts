import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { getConfigByGroup } from '@/api/config'
import {
  resolveImageDisplayUrl,
  WEBSITE_CONFIG_UPDATED_EVENT,
} from '@/utils/publicImage'

function pickConfigValue(configs: Array<Record<string, unknown>> | undefined, key: string): string {
  if (!Array.isArray(configs)) return ''
  const item = configs.find((config) => {
    const k = config?.Key ?? config?.key
    return k === key
  })
  const value = item?.Value ?? item?.value ?? ''
  return typeof value === 'string' ? value : ''
}

export function useLayoutWebsite() {
  const { t } = useTranslation()
  const [websiteSiteName, setWebsiteSiteName] = useState('')
  const [websiteSiteLogo, setWebsiteSiteLogo] = useState('')
  const [websiteLogoDisplayUrl, setWebsiteLogoDisplayUrl] = useState('')
  const [websiteConfigLoaded, setWebsiteConfigLoaded] = useState(false)
  const revokeLogoUrlRef = useRef<(() => void) | null>(null)
  const logoRawRef = useRef('')

  const systemTitle = useMemo(() => {
    if (!websiteConfigLoaded) return ''
    const name = websiteSiteName.trim()
    return name || t('header.system')
  }, [websiteConfigLoaded, websiteSiteName, t])

  const clearLogoDisplay = useCallback(() => {
    if (typeof revokeLogoUrlRef.current === 'function') {
      revokeLogoUrlRef.current()
      revokeLogoUrlRef.current = null
    }
    setWebsiteLogoDisplayUrl('')
  }, [])

  const refreshLogoDisplay = useCallback(
    async (raw: string) => {
      clearLogoDisplay()
      const value = String(raw || '').trim()
      if (!value) return

      const result = await resolveImageDisplayUrl(value)
      if (logoRawRef.current.trim() !== value) {
        if (typeof result.revoke === 'function') result.revoke()
        return
      }
      revokeLogoUrlRef.current = result.revoke || null
      setWebsiteLogoDisplayUrl(result.url || '')
    },
    [clearLogoDisplay],
  )

  useEffect(() => {
    logoRawRef.current = websiteSiteLogo
    void refreshLogoDisplay(websiteSiteLogo)
  }, [websiteSiteLogo, refreshLogoDisplay])

  const loadWebsiteTitle = useCallback(async () => {
    try {
      const res = await getConfigByGroup('website')
      const configs = res?.data?.configs
      setWebsiteSiteName(pickConfigValue(configs, 'site_name'))
      setWebsiteSiteLogo(pickConfigValue(configs, 'site_logo'))
    } catch {
      setWebsiteSiteName('')
      setWebsiteSiteLogo('')
    } finally {
      setWebsiteConfigLoaded(true)
    }
  }, [])

  useEffect(() => {
    void loadWebsiteTitle()
    const onUpdated = () => {
      void loadWebsiteTitle()
    }
    window.addEventListener(WEBSITE_CONFIG_UPDATED_EVENT, onUpdated)
    return () => {
      window.removeEventListener(WEBSITE_CONFIG_UPDATED_EVENT, onUpdated)
      clearLogoDisplay()
    }
  }, [loadWebsiteTitle, clearLogoDisplay])

  return {
    systemTitle,
    websiteLogoUrl: websiteLogoDisplayUrl,
    loadWebsiteTitle,
  }
}
