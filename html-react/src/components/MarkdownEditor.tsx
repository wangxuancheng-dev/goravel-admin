import { useMemo, useRef } from 'react'
import { App, Button } from 'antd'
import MDEditor, { commands, type ICommand, type TextAreaTextApi } from '@uiw/react-md-editor'
import '@uiw/react-md-editor/markdown-editor.css'
import { useTranslation } from 'react-i18next'
import request from '@/utils/request'
import { useAppStore } from '@/stores/app'
import { resolveUploadStorageUrl } from '@/utils/attachmentUrl'
import { markdownToHtml } from '@/utils/markdown'
import AttachmentImageField, {
  type AttachmentImageFieldRef,
  type AttachmentSelectPayload,
} from './AttachmentImageField'
import './MarkdownEditor.scss'

interface MarkdownEditorProps {
  value?: string
  onChange?: (value: string) => void
  height?: number
  placeholder?: string
}

const MAX_IMAGE_SIZE = 5 * 1024 * 1024

function escapeMdAlt(value: string) {
  return String(value || 'image').replace(/[[\]]/g, '')
}

export default function MarkdownEditor({
  value = '',
  onChange,
  height = 400,
  placeholder,
}: MarkdownEditorProps) {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const darkMode = useAppStore((s) => s.darkMode)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const pendingApiRef = useRef<TextAreaTextApi | null>(null)
  const textApiRef = useRef<TextAreaTextApi | null>(null)
  const mediaPickerRef = useRef<AttachmentImageFieldRef>(null)

  const uploadImage = async (file: File) => {
    if (!file.type.startsWith('image/')) return ''
    if (file.size > MAX_IMAGE_SIZE) return ''
    const formData = new FormData()
    formData.append('file', file)
    formData.append('is_public', '1')
    const res = await request({
      url: '/attachments/upload',
      method: 'post',
      data: formData,
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    const data = (res.data || {}) as Record<string, unknown>
    const nested = (data.data || data) as { id?: number; is_public?: number; file_url?: string }
    return resolveUploadStorageUrl(nested) || ''
  }

  const imageUploadCommand: ICommand = useMemo(
    () => ({
      ...commands.image,
      execute: (_state, api) => {
        pendingApiRef.current = api
        textApiRef.current = api
        fileInputRef.current?.click()
      },
    }),
    [],
  )

  const mediaLibraryCommand: ICommand = useMemo(
    () => ({
      name: 'media-library',
      keyCommand: 'media-library',
      buttonProps: { 'aria-label': t('attachment.select_from_library'), title: t('attachment.select_from_library') },
      icon: <span style={{ fontSize: 12 }}>{t('attachment.select_from_library')}</span>,
      execute: (_state, api) => {
        textApiRef.current = api
        mediaPickerRef.current?.openPicker()
      },
    }),
    [t],
  )

  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file || !pendingApiRef.current) return
    try {
      const url = await uploadImage(file)
      if (url) {
        pendingApiRef.current.replaceSelection(`![image](${url})`)
      }
    } catch {
      // upload errors handled by request interceptor
    } finally {
      pendingApiRef.current = null
    }
  }

  const handleMediaSelect = ({ url, alt }: AttachmentSelectPayload) => {
    if (!url) {
      message.warning(t('attachment.editor_public_required'))
      return
    }
    const snippet = `![${escapeMdAlt(alt)}](${url})`
    if (textApiRef.current) {
      textApiRef.current.replaceSelection(snippet)
      return
    }
    onChange?.(value ? `${value}\n${snippet}` : snippet)
  }

  return (
    <div className="markdown-editor-wrapper" data-color-mode={darkMode ? 'dark' : 'light'}>
      <div className="markdown-editor-media-bar">
        <Button
          size="small"
          onClick={() => {
            // Prefer cursor insert when a prior toolbar action captured TextApi; else append.
            mediaPickerRef.current?.openPicker()
          }}
        >
          {t('attachment.select_from_library')}
        </Button>
      </div>
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        style={{ display: 'none' }}
        onChange={(e) => void handleFileChange(e)}
      />
      <MDEditor
        value={value}
        onChange={(next) => onChange?.(next || '')}
        height={height}
        textareaProps={{ placeholder }}
        data-color-mode={darkMode ? 'dark' : 'light'}
        preview="live"
        visibleDragbar
        extraCommands={[mediaLibraryCommand]}
        commandsFilter={(command) => (command.name === 'image' ? imageUploadCommand : command)}
        components={{
          preview: (source) => (
            <div
              className="wmde-markdown wmde-markdown-color"
              dangerouslySetInnerHTML={{ __html: markdownToHtml(source) }}
            />
          ),
        }}
      />
      <AttachmentImageField ref={mediaPickerRef} pickerOnly onSelect={handleMediaSelect} />
    </div>
  )
}
