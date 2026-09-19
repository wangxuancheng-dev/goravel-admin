import { useEffect, useMemo, useRef } from 'react'
import { renderContent } from '@/utils/markdown'
import { hydrateContentImages } from '@/utils/publicImage'
import '@/components/MarkdownEditor.scss'

interface MarkdownContentProps {
  content?: string
  className?: string
}

export default function MarkdownContent({ content, className }: MarkdownContentProps) {
  const html = useMemo(() => renderContent(content || '', 'auto'), [content])
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let active = true
    let revoke = () => {}
    const root = rootRef.current
    if (!root || !html) return undefined
    void hydrateContentImages(root).then((fn) => {
      if (!active) {
        fn()
        return
      }
      revoke = fn
    })
    return () => {
      active = false
      revoke()
    }
  }, [html])

  if (!html) return null

  return (
    <div
      ref={rootRef}
      className={['markdown-content', className].filter(Boolean).join(' ')}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
}
