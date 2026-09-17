/**
 * Translate platform operation log title keys (stored as platform.*).
 * Uses platform_op.* i18n namespace to avoid clashing with platform.* UI strings.
 */
export function translatePlatformOpTitle(t, te, title) {
  if (!title) return '-'
  if (typeof title !== 'string') return String(title)

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
