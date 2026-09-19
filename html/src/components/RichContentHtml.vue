<template>
  <div
    ref="rootRef"
    class="rich-content-html"
    :class="contentClass"
    v-html="html"
  />
</template>

<script setup>
import { computed, ref, watch, onBeforeUnmount, nextTick } from 'vue'
import { renderContent } from '@/utils/markdown'
import { hydrateContentImages } from '@/utils/publicImage'

const props = defineProps({
  content: {
    type: String,
    default: ''
  },
  contentType: {
    type: String,
    default: 'auto'
  },
  contentClass: {
    type: [String, Array, Object],
    default: ''
  }
})

const rootRef = ref(null)
const html = computed(() => renderContent(props.content || '', props.contentType))

let revokeBlobs = () => {}

const hydrate = async () => {
  revokeBlobs()
  revokeBlobs = () => {}
  await nextTick()
  if (!rootRef.value || !html.value) return
  revokeBlobs = await hydrateContentImages(rootRef.value)
}

watch(html, () => {
  void hydrate()
}, { immediate: true })

onBeforeUnmount(() => {
  revokeBlobs()
})
</script>
