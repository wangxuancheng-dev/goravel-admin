import { memo, useMemo } from 'react'
import GoCaptcha from 'go-captcha-react'
import { useTranslation } from 'react-i18next'

/** Slide captcha payload returned by GET /login/captcha when type=slide. */
export interface SlideCaptchaData {
  captcha_id: string
  master_image: string
  tile_image: string
  tile_width: number
  tile_height: number
  tile_x: number
  tile_y: number
}

interface SlideCaptchaProps {
  data: SlideCaptchaData
  /** Called with the x offset ("123") after the user releases the slider. */
  onConfirm: (answer: string) => void
  /** Called when the user clicks the built-in refresh icon. */
  onRefresh: () => void
}

/**
 * Puzzle slider captcha (go-captcha). The answer is only verified server-side on login.
 * Remount with a new `key` (captcha_id) to reset after a refresh.
 */
function SlideCaptcha({ data, onConfirm, onRefresh }: SlideCaptchaProps) {
  const { t } = useTranslation()

  const config = useMemo(
    () => ({
      width: 300,
      height: 220,
      title: t('login.slide_title'),
      showTheme: true,
    }),
    [t],
  )

  const slideData = useMemo(
    () => ({
      image: data.master_image,
      thumb: data.tile_image,
      thumbX: data.tile_x,
      thumbY: data.tile_y,
      thumbWidth: data.tile_width,
      thumbHeight: data.tile_height,
    }),
    [data],
  )

  const events = useMemo(
    () => ({
      confirm: (point: { x: number; y: number }) => {
        onConfirm(String(Math.round(point.x)))
      },
      refresh: onRefresh,
    }),
    [onConfirm, onRefresh],
  )

  return (
    <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 16 }}>
      <GoCaptcha.Slide config={config} data={slideData} events={events} />
    </div>
  )
}

export default memo(SlideCaptcha)
