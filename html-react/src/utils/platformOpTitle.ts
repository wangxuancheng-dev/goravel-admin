/**
 * Translate platform operation log title keys (stored as platform.*).
 * Uses platform_op.* i18n namespace to avoid clashing with platform.* UI strings.
 */
export function translatePlatformOpTitle(
  t: (key: string) => string,
  te: ((key: string) => boolean) | undefined,
  title?: string | null,
): string {
  if (!title) return '-'

  if (title.startsWith('platform.')) {
    const key = `platform_op.${title.slice('platform.'.length)}`
    if (typeof te === 'function') {
      if (te(key)) return t(key)
    } else {
      const translated = t(key)
      if (translated && translated !== key) return translated
    }
  }

  return title
}
