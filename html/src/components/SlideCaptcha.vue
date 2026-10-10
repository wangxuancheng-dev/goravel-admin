<template>
  <div class="slide-captcha">
    <GoCaptchaSlide :config="config" :data="slideData" :events="events" />
  </div>
</template>

<script setup>
/**
 * Puzzle slider captcha (go-captcha). The answer is only verified server-side on login.
 * Re-mount with a new :key (captcha_id) to reset after a refresh.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Slide as GoCaptchaSlide } from 'go-captcha-vue'
import 'go-captcha-vue/dist/style.css'

const props = defineProps({
  // Payload of GET /login/captcha when type=slide
  data: { type: Object, required: true }
})
const emit = defineEmits(['confirm', 'refresh'])

const { t } = useI18n()

const config = computed(() => ({
  width: 300,
  height: 220,
  title: t('login.slide_title'),
  showTheme: true
}))

const slideData = computed(() => ({
  image: props.data.master_image,
  thumb: props.data.tile_image,
  thumbX: props.data.tile_x,
  thumbY: props.data.tile_y,
  thumbWidth: props.data.tile_width,
  thumbHeight: props.data.tile_height
}))

const events = {
  confirm: (point) => {
    emit('confirm', String(Math.round(point.x)))
  },
  refresh: () => {
    emit('refresh')
  }
}
</script>

<style scoped>
.slide-captcha {
  display: flex;
  justify-content: center;
  width: 100%;
  margin-bottom: 8px;
}
</style>
