import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useState } from 'react'
import {
  Button,
  Empty,
  Image,
  Input,
  Modal,
  Pagination,
  Select,
  Space,
  Spin,
} from 'antd'
import { PictureOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { getAttachmentList } from '@/api/attachment'
import { useAttachmentImagePreview } from '@/hooks/useAttachmentImagePreview'
import { useOptions } from '@/hooks/useOptions'
import {
  transformAttachmentRow,
  type AttachmentRow,
} from '@/pages/attachment/attachment.config'
import { PUBLIC_IMAGE_PATH_RE, resolveImageDisplayUrl } from '@/utils/publicImage'
import { toStablePublicAttachmentUrl } from '@/utils/attachmentUrl'
import './AttachmentImageField.scss'

export interface AttachmentSelectPayload {
  url: string
  alt: string
  item: AttachmentRow
}

export interface AttachmentImageFieldProps {
  value?: string
  onChange?: (value: string) => void
  onSelect?: (payload: AttachmentSelectPayload) => void
  placeholder?: string
  disabled?: boolean
  /** When true, only the picker dialog is rendered (for editors). */
  pickerOnly?: boolean
}

export interface AttachmentImageFieldRef {
  openPicker: () => void
}

function toSubmitUrl(url: string) {
  const value = String(url || '').trim()
  if (!value) return ''
  if (value.startsWith('/')) return value
  if (value.startsWith('http')) {
    try {
      const parsed = new URL(value)
      if (PUBLIC_IMAGE_PATH_RE.test(parsed.pathname)) {
        return `${parsed.pathname}${parsed.search || ''}`
      }
      return value
    } catch {
      return value
    }
  }
  return value
}

function ensureFileUrl(item: AttachmentRow) {
  if (item.file_url) return item.file_url
  if (item.id) return `/api/admin/public/images/${item.id}`
  return ''
}

const AttachmentImageField = forwardRef<AttachmentImageFieldRef, AttachmentImageFieldProps>(
  function AttachmentImageField(
    { value = '', onChange, onSelect, placeholder, disabled, pickerOnly = false },
    ref,
  ) {
    const { t } = useTranslation()
    const [pickerOpen, setPickerOpen] = useState(false)
    const [loading, setLoading] = useState(false)
    const [list, setList] = useState<AttachmentRow[]>([])
    const [keyword, setKeyword] = useState('')
    const [categoryId, setCategoryId] = useState<string | number | undefined>()
    const [page, setPage] = useState(1)
    const [pageSize] = useState(12)
    const [total, setTotal] = useState(0)
    const [selectedId, setSelectedId] = useState<string | number | null>(null)
    const [displayPreviewUrl, setDisplayPreviewUrl] = useState('')

    const { selectOptions: categoryOptions } = useOptions('attachment_category', pickerOpen)
    const { loadImageAsBlob, getImageUrl, getImageLoadingState } = useAttachmentImagePreview()

    const selectedItem = useMemo(
      () => list.find((item) => item.id === selectedId) || null,
      [list, selectedId],
    )

    const emitValue = useCallback(
      (next: string) => {
        onChange?.(toSubmitUrl(next))
      },
      [onChange],
    )

    const openPicker = useCallback(() => {
      setSelectedId(null)
      setPickerOpen(true)
    }, [])

    useImperativeHandle(ref, () => ({ openPicker }), [openPicker])

    useEffect(() => {
      if (pickerOnly) return
      let cancelled = false
      let revoke: (() => void) | undefined

      const run = async () => {
        setDisplayPreviewUrl('')
        const raw = String(value || '').trim()
        if (!raw) return
        const result = await resolveImageDisplayUrl(raw)
        if (cancelled) {
          result.revoke?.()
          return
        }
        revoke = result.revoke
        setDisplayPreviewUrl(result.url || '')
      }

      void run()
      return () => {
        cancelled = true
        revoke?.()
      }
    }, [value, pickerOnly])

    const loadList = useCallback(
      async (overrides?: {
        page?: number
        keyword?: string
        categoryId?: string | number
      }) => {
        const nextPage = overrides?.page ?? page
        const nextKeyword = overrides?.keyword ?? keyword
        const nextCategoryId = overrides?.categoryId !== undefined ? overrides.categoryId : categoryId
        setLoading(true)
        try {
          const res = await getAttachmentList({
            page: nextPage,
            page_size: pageSize,
            file_type: 'image',
            is_public: '1',
            keyword: String(nextKeyword || '').trim() || undefined,
            category_id: nextCategoryId || undefined,
          })
          const rows = (res.data?.list || res.data?.data || []) as Array<Record<string, unknown>>
          const nextList = rows.map((row) => {
            const item = transformAttachmentRow(row)
            item.file_url = ensureFileUrl(item)
            return item
          })
          setList(nextList)
          setTotal(Number(res.data?.total || 0))
          nextList.forEach((item) => {
            if (item.file_type === 'image') void loadImageAsBlob(item)
          })
        } catch {
          setList([])
          setTotal(0)
        } finally {
          setLoading(false)
        }
      },
      [page, pageSize, keyword, categoryId, loadImageAsBlob],
    )

    useEffect(() => {
      if (!pickerOpen) return
      void loadList()
    }, [pickerOpen, page, categoryId, loadList])

    const handleSearch = () => {
      setSelectedId(null)
      if (page !== 1) {
        setPage(1)
        return
      }
      void loadList({ page: 1 })
    }

    const handleReset = () => {
      setKeyword('')
      setCategoryId(undefined)
      setSelectedId(null)
      setPage(1)
      void loadList({ page: 1, keyword: '', categoryId: undefined })
    }

    const confirmSelect = (item: AttachmentRow | null) => {
      if (!item) return
      const url = toSubmitUrl(
        toStablePublicAttachmentUrl({
          id: Number(item.id),
          is_public: item.is_public,
          file_url: item.file_url,
        }),
      )
      const alt = item.display_name || item.filename || 'image'
      onSelect?.({ url, alt, item })
      if (!pickerOnly) {
        emitValue(url)
      }
      setPickerOpen(false)
    }

    return (
      <div className={`attachment-image-field${pickerOnly ? ' is-picker-only' : ''}`}>
        {!pickerOnly ? (
          <>
            <Space.Compact style={{ width: '100%' }}>
              <Input
                value={value}
                disabled={disabled}
                placeholder={placeholder || t('config.site_logo_placeholder')}
                allowClear
                onChange={(e) => emitValue(e.target.value)}
              />
              <Button disabled={disabled} icon={<PictureOutlined />} onClick={openPicker}>
                {t('common.select_from_attachment')}
              </Button>
            </Space.Compact>

            {displayPreviewUrl ? (
              <div className="attachment-image-field__preview">
                <Image
                  src={displayPreviewUrl}
                  width={120}
                  height={120}
                  style={{ objectFit: 'contain' }}
                  fallback=""
                />
              </div>
            ) : null}
          </>
        ) : null}

        <Modal
          open={pickerOpen}
          title={t('common.select_from_attachment')}
          width={780}
          destroyOnHidden
          onCancel={() => setPickerOpen(false)}
          afterOpenChange={(open) => {
            if (open) {
              setSelectedId(null)
              setPage(1)
            } else {
              setList([])
              setSelectedId(null)
            }
          }}
          footer={[
            <Button key="cancel" onClick={() => setPickerOpen(false)}>
              {t('common.cancel')}
            </Button>,
            <Button
              key="ok"
              type="primary"
              disabled={!selectedItem}
              onClick={() => confirmSelect(selectedItem)}
            >
              {t('common.confirm')}
            </Button>,
          ]}
        >
          <div className="attachment-image-field__toolbar">
            <Select
              allowClear
              placeholder={t('attachment.category')}
              style={{ width: 160 }}
              value={categoryId}
              options={categoryOptions}
              onChange={(val) => {
                setCategoryId(val)
                setPage(1)
                setSelectedId(null)
              }}
            />
            <Input
              allowClear
              placeholder={t('attachment.search_name_placeholder')}
              style={{ width: 280 }}
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              onPressEnter={handleSearch}
            />
            <Button type="primary" onClick={handleSearch}>
              {t('common.search')}
            </Button>
            <Button onClick={handleReset}>{t('common.reset')}</Button>
          </div>

          <Spin spinning={loading}>
            <div className="attachment-image-field__grid">
              {list.map((item) => {
                const thumb = getImageUrl(item)
                const loadingState = getImageLoadingState(item)
                return (
                  <div
                    key={String(item.id)}
                    className={`attachment-image-field__item${selectedId === item.id ? ' is-active' : ''}`}
                    onClick={() => setSelectedId(item.id)}
                    onDoubleClick={() => confirmSelect(item)}
                  >
                    {thumb ? (
                      <img src={thumb} alt="" className="attachment-image-field__thumb" />
                    ) : (
                      <div className="attachment-image-field__thumb-empty">
                        {loadingState === 'loading' ? <Spin size="small" /> : <PictureOutlined />}
                      </div>
                    )}
                    <div
                      className="attachment-image-field__name"
                      title={
                        item.display_name
                          ? `${item.display_name}\n${item.filename || ''}`
                          : item.filename || ''
                      }
                    >
                      <div className="attachment-image-field__name-main">
                        {item.display_name || item.filename}
                      </div>
                      {item.display_name ? (
                        <div className="attachment-image-field__name-sub">{item.filename}</div>
                      ) : null}
                    </div>
                  </div>
                )
              })}
              {!loading && list.length === 0 ? (
                <div className="attachment-image-field__empty">
                  <Empty description={t('common.no_data')} />
                </div>
              ) : null}
            </div>
          </Spin>

          <div className="attachment-image-field__pager">
            <Pagination
              current={page}
              pageSize={pageSize}
              total={total}
              size="small"
              showSizeChanger={false}
              onChange={(next) => {
                setPage(next)
                setSelectedId(null)
              }}
            />
          </div>
        </Modal>
      </div>
    )
  },
)

export default AttachmentImageField
