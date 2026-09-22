import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Input, Modal } from 'antd'
import { SearchOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useUserStore } from '@/stores/user'
import { resolveMenuIcon } from '@/utils/menuIcons'
import { resolveMenuTitle } from '@/utils/menuTitle'
import type { MenuNode } from '@/types'
import './MenuSearch.scss'

type SearchableMenu = {
  key: string
  title: string
  path: string
  slug: string
  icon?: string
  linkType: number
  openType: number
}

function isApplePlatform(): boolean {
  if (typeof navigator === 'undefined') return false
  const platform = navigator.platform || ''
  const ua = navigator.userAgent || ''
  return /Mac|iPhone|iPad|iPod/i.test(platform) || /Mac OS X/i.test(ua)
}

function flattenMenus(nodes: MenuNode[], result: MenuNode[] = []): MenuNode[] {
  nodes.forEach((menu) => {
    const menuType = menu.type ?? menu.Type ?? 1
    const path = String(menu.path || menu.Path || '').trim()
    if (menuType === 2 && path) {
      result.push(menu)
    }
    const children = menu.children || menu.Children
    if (children?.length) {
      flattenMenus(children, result)
    }
  })
  return result
}

export default function MenuSearch() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const menus = useUserStore((s) => s.menus)
  const [open, setOpen] = useState(false)
  const [keyword, setKeyword] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<any>(null)

  const shortcutLabel = useMemo(() => (isApplePlatform() ? '\u2318K' : 'Ctrl+K'), [])

  const allMenus = useMemo<SearchableMenu[]>(() => {
    const fromTree = flattenMenus(menus || []).map((menu, index) => {
      const path = String(menu.path || menu.Path || '').trim()
      const slug = String(menu.slug || menu.Slug || '')
      const title = resolveMenuTitle(t, {
        slug,
        fallback: menu.Title || menu.title || menu.Name || menu.name || slug || path,
      })
      return {
        key: `${path}-${menu.id ?? menu.ID ?? index}`,
        title,
        path,
        slug,
        icon: menu.Icon || menu.icon,
        linkType: menu.link_type ?? menu.LinkType ?? 1,
        openType: (menu as { open_type?: number; OpenType?: number }).open_type
          ?? (menu as { OpenType?: number }).OpenType
          ?? 1,
      }
    })

    const dashboard: SearchableMenu = {
      key: 'dashboard',
      title: t('menu.dashboard'),
      path: '/dashboard',
      slug: 'dashboard',
      linkType: 1,
      openType: 1,
    }

    const seen = new Set<string>()
    const merged: SearchableMenu[] = []
    for (const item of [dashboard, ...fromTree]) {
      const id = item.path.toLowerCase()
      if (!id || seen.has(id)) continue
      seen.add(id)
      merged.push(item)
    }
    return merged
  }, [menus, t])

  const filteredMenus = useMemo(() => {
    const q = keyword.trim().toLowerCase()
    if (!q) return allMenus.slice(0, 10)
    return allMenus
      .filter((menu) => {
        return (
          menu.title.toLowerCase().includes(q) ||
          menu.path.toLowerCase().includes(q) ||
          menu.slug.toLowerCase().includes(q)
        )
      })
      .slice(0, 10)
  }, [allMenus, keyword])

  useEffect(() => {
    setSelectedIndex(0)
  }, [keyword, open])

  const close = useCallback(() => {
    setOpen(false)
    setKeyword('')
    setSelectedIndex(0)
  }, [])

  const openPalette = useCallback(() => {
    setOpen(true)
  }, [])

  const handleSelect = useCallback(
    (menu: SearchableMenu) => {
      if (!menu.path) return
      if (menu.linkType === 2) {
        if (menu.openType === 2) {
          window.open(menu.path, '_blank')
        } else {
          navigate(`/iframe?url=${encodeURIComponent(menu.path)}&title=${encodeURIComponent(menu.title)}`)
        }
      } else {
        navigate(menu.path)
      }
      close()
    },
    [close, navigate],
  )

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const key = String(event.key || '').toLowerCase()
      if (key === 'k' && (event.metaKey || event.ctrlKey) && !event.altKey) {
        event.preventDefault()
        setOpen((prev) => {
          if (prev) {
            setKeyword('')
            setSelectedIndex(0)
            return false
          }
          return true
        })
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  return (
    <>
      <button
        type="button"
        className="layout-header__icon-btn menu-search-trigger"
        title={`${t('header.menu_search')} (${shortcutLabel})`}
        onClick={openPalette}
      >
        <SearchOutlined />
        <kbd className="menu-search-kbd">{shortcutLabel}</kbd>
      </button>

      <Modal
        open={open}
        onCancel={close}
        footer={null}
        closable={false}
        destroyOnHidden
        width={480}
        centered
        className="menu-search-modal"
        afterOpenChange={(visible) => {
          if (visible) {
            window.setTimeout(() => inputRef.current?.focus?.(), 0)
          }
        }}
      >
        <div className="menu-search-panel">
          <Input
            ref={inputRef}
            allowClear
            size="large"
            prefix={<SearchOutlined />}
            placeholder={t('header.menu_search_placeholder')}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'ArrowDown') {
                event.preventDefault()
                setSelectedIndex((prev) => (prev + 1) % Math.max(filteredMenus.length, 1))
              } else if (event.key === 'ArrowUp') {
                event.preventDefault()
                setSelectedIndex((prev) =>
                  prev <= 0 ? Math.max(filteredMenus.length - 1, 0) : prev - 1,
                )
              } else if (event.key === 'Enter') {
                event.preventDefault()
                const target = filteredMenus[selectedIndex] || filteredMenus[0]
                if (target) handleSelect(target)
              } else if (event.key === 'Escape') {
                event.preventDefault()
                close()
              }
            }}
          />

          <div className="menu-search-results">
            {filteredMenus.length > 0 ? (
              filteredMenus.map((menu, index) => (
                <button
                  key={menu.key}
                  type="button"
                  className={`menu-search-item${index === selectedIndex ? ' is-active' : ''}`}
                  onMouseEnter={() => setSelectedIndex(index)}
                  onClick={() => handleSelect(menu)}
                >
                  <span className="menu-search-item__icon">{resolveMenuIcon(menu.icon)}</span>
                  <span className="menu-search-item__title">{menu.title}</span>
                  <span className="menu-search-item__path">{menu.path}</span>
                </button>
              ))
            ) : (
              <div className="menu-search-empty">
                {keyword.trim()
                  ? t('header.menu_search_no_results')
                  : t('header.menu_search_hint')}
              </div>
            )}
          </div>

          <div className="menu-search-footer">
            <span>{t('header.menu_search_shortcut_hint', { shortcut: shortcutLabel })}</span>
          </div>
        </div>
      </Modal>
    </>
  )
}
