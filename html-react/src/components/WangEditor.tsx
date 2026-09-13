import '@wangeditor/editor/dist/css/style.css'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Editor, Toolbar } from '@wangeditor/editor-for-react'
import type { IDomEditor, IEditorConfig, IToolbarConfig } from '@wangeditor/editor'
import { App, Button } from 'antd'
import { useTranslation } from 'react-i18next'
import { useAppStore } from '@/stores/app'
import { getApiBaseURL } from '@/utils/env'
import { resolveUploadStorageUrl } from '@/utils/attachmentUrl'
import Storage from '@/utils/storage'
import AttachmentImageField, {
  type AttachmentImageFieldRef,
  type AttachmentSelectPayload,
} from './AttachmentImageField'
import './WangEditor.scss'

interface WangEditorProps {
  value?: string
  onChange?: (value: string) => void
  mode?: 'default' | 'simple'
  width?: string | number
  height?: number
  placeholder?: string
  excludeToolbarKeys?: string[]
}

interface UploadResponse {
  code?: number
  data?: {
    id?: number
    is_public?: number
    file_url?: string
    filename?: string
  }
}

function escapeAttr(value: string) {
  return String(value || '')
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

export default function WangEditor({
  value = '',
  onChange,
  mode = 'default',
  width = '100%',
  height = 300,
  placeholder,
  excludeToolbarKeys = ['group-video'],
}: WangEditorProps) {
  const { t, i18n } = useTranslation()
  const { message } = App.useApp()
  const darkMode = useAppStore((s) => s.darkMode)
  const [editor, setEditor] = useState<IDomEditor | null>(null)
  const [html, setHtml] = useState(value)
  const lastEmittedRef = useRef(value)
  const mediaPickerRef = useRef<AttachmentImageFieldRef>(null)

  const normalizedWidth = typeof width === 'number' ? `${width}px` : width

  useEffect(() => {
    if (value === lastEmittedRef.current) return
    if (value === html) return
    if (editor && value === editor.getHtml()) return
    setHtml(value)
    lastEmittedRef.current = value
  }, [value, html, editor])

  useEffect(() => {
    return () => {
      if (editor == null) return
      editor.destroy()
      setEditor(null)
    }
  }, [editor])

  const uploadAction = getApiBaseURL() + '/attachments/upload'

  const uploadHeaders = useMemo(
    () => {
      const token = String(Storage.getItem('token', '') ?? '').trim()
      return {
        Authorization: `Bearer ${token}`,
        'Accept-Language': i18n.language === 'en-US' ? 'en-US' : 'zh-CN',
      }
    },
    [i18n.language],
  )

  const toolbarConfig = useMemo<Partial<IToolbarConfig>>(
    () => ({
      excludeKeys: excludeToolbarKeys,
    }),
    [excludeToolbarKeys],
  )

  const editorConfig = useMemo<Partial<IEditorConfig>>(
    () => ({
      placeholder,
      MENU_CONF: {
        uploadImage: {
          server: uploadAction,
          fieldName: 'file',
          headers: uploadHeaders,
          maxFileSize: 5 * 1024 * 1024,
          maxNumberOfFiles: 10,
          allowedFileTypes: ['image/*'],
          metaWithUrl: false,
          withCredentials: false,
          timeout: 10 * 1000,
          meta: {
            is_public: '1',
          },
          customInsert(res: UploadResponse, insertFn: (url: string, alt: string, href: string) => void) {
            if (res.code === 200 && res.data) {
              const url = resolveUploadStorageUrl(res.data)
              if (!url) {
                console.error('Upload error: missing file url', res)
                return
              }
              insertFn(url, res.data.filename || 'image', url)
            } else {
              console.error('Upload error', res)
            }
          },
        },
      },
    }),
    [placeholder, uploadAction, uploadHeaders],
  )

  const handleChange = (nextEditor: IDomEditor) => {
    const nextHtml = nextEditor.getHtml()
    setHtml(nextHtml)
    lastEmittedRef.current = nextHtml
    onChange?.(nextHtml)
  }

  const handleMediaSelect = ({ url, alt }: AttachmentSelectPayload) => {
    if (!editor) return
    if (!url) {
      message.warning(t('attachment.editor_public_required'))
      return
    }
    editor.dangerouslyInsertHtml(`<img src="${escapeAttr(url)}" alt="${escapeAttr(alt || 'image')}" />`)
  }

  return (
    <div
      className={`wang-editor-wrapper${darkMode ? ' dark-mode' : ''}`}
      style={{ width: normalizedWidth }}
    >
      <Toolbar editor={editor} defaultConfig={toolbarConfig} mode={mode} />
      <div className="wang-editor-media-bar">
        <Button size="small" onClick={() => mediaPickerRef.current?.openPicker()}>
          {t('attachment.select_from_library')}
        </Button>
      </div>
      <Editor
        defaultConfig={editorConfig}
        value={html}
        onCreated={setEditor}
        onChange={handleChange}
        mode={mode}
        style={{ height, overflowY: 'hidden' }}
      />
      <AttachmentImageField ref={mediaPickerRef} pickerOnly onSelect={handleMediaSelect} />
    </div>
  )
}
