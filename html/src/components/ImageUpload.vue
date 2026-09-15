<template>
  <div class="image-upload-wrapper">
    <div v-if="!imageUrl" class="upload-buttons">
      <el-upload
        v-if="uploadMode === 'both' || uploadMode === 'direct'"
        :action="uploadAction"
        :headers="uploadHeaders"
        :data="uploadData"
        :before-upload="beforeUpload"
        :on-success="handleUploadSuccess"
        :on-error="handleUploadError"
        :show-file-list="false"
        :auto-upload="true"
        accept="image/*"
      >
        <el-button type="primary">
          <el-icon><UploadIcon /></el-icon>
          {{ $t('common.image_upload') }}
        </el-button>
      </el-upload>
      
      <el-button 
        v-if="uploadMode === 'both' || uploadMode === 'crop'"
        type="success" 
        @click="handleCropUpload"
      >
        <el-icon><CropIcon /></el-icon>
        {{ $t('common.crop_upload') }}
      </el-button>
    </div>

    <div v-if="imageUrl" class="image-preview">
      <el-image
        :src="imageUrl"
        :preview-src-list="[imageUrl]"
        fit="cover"
        style="width: 150px; height: 150px; border-radius: 4px;"
      />
      <div class="image-actions">
        <el-button 
          v-if="uploadMode === 'both' || uploadMode === 'crop'"
          type="primary" 
          size="small"
          @click="handleCropUpload"
        >
          <el-icon><CropIcon /></el-icon>
          {{ $t('common.re_crop') }}
        </el-button>
        <el-button 
          type="danger" 
          size="small"
          @click="handleRemove"
        >
          <el-icon><DeleteIcon /></el-icon>
          {{ $t('common.delete') }}
        </el-button>
      </div>
    </div>

    <!-- 图片裁剪对话框 -->
    <el-dialog
      v-model="cropDialogVisible"
      :title="$t('common.crop_upload')"
      width="800px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <div class="crop-container" style="height: 500px;">
        <vue-cropper
          ref="cropperRef"
          :img="cropOption.img"
          :output-size="cropOption.size"
          :output-type="cropOption.outputType"
          :info="true"
          :full="cropOption.full"
          :can-move="cropOption.canMove"
          :can-move-box="cropOption.canMoveBox"
          :fixed-box="cropOption.fixedBox"
          :original="cropOption.original"
          :auto-crop="cropOption.autoCrop"
          :auto-crop-width="cropOption.autoCropWidth"
          :auto-crop-height="cropOption.autoCropHeight"
          :center-box="cropOption.centerBox"
          :high="cropOption.high"
          :mode="cropOption.mode"
        />
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-upload
            action=""
            :auto-upload="false"
            :show-file-list="false"
            accept="image/*"
            :on-change="onCropFileChange"
            style="display: inline-block; margin-right: 10px;"
          >
            <el-button>{{ $t('common.select_image') }}</el-button>
          </el-upload>
          <el-button @click="cropDialogVisible = false">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" :loading="cropUploading" @click="handleCropConfirm">
            {{ $t('common.confirm') }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Upload, Delete, Crop } from '@element-plus/icons-vue'
import { markRaw } from 'vue'
import axios from 'axios'
import { buildAdminAuthHeaders } from '@/utils/authHeaders'
import 'vue-cropper/dist/index.css'
import { VueCropper } from 'vue-cropper'

const UploadIcon = markRaw(Upload)
const DeleteIcon = markRaw(Delete)
const CropIcon = markRaw(Crop)

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  height: {
    type: Number,
    default: 400
  },
  width: {
    type: Number,
    default: 400
  },
  aspectRatio: {
    type: Number,
    default: null // null 表示不限制比例
  },
  uploadMode: {
    type: String,
    default: 'both', // 'both' | 'direct' | 'crop'
    validator: (value) => ['both', 'direct', 'crop'].includes(value)
  }
})

const emit = defineEmits(['update:modelValue', 'change'])

const imagePath = ref(props.modelValue)
const cropDialogVisible = ref(false)
const cropperRef = ref(null)
const cropUploading = ref(false)
const cropFileName = ref('')

const cropOption = reactive({
  img: '',
  size: 1,
  full: false,
  outputType: 'png',
  canMove: true,
  fixedBox: false,
  original: false,
  canMoveBox: true,
  autoCrop: true,
  autoCropWidth: props.width || 400,
  autoCropHeight: props.height || 400,
  centerBox: false,
  high: true,
  max: 99999,
  mode: 'contain'
})

// 监听 props 变化
watch(() => props.modelValue, (newVal) => {
  imagePath.value = newVal
})

const normalizeUrl = (url) => {
  if (!url) return ''
  if (url.startsWith('http') || url.startsWith('blob:') || url.startsWith('data:')) return url

  const apiBaseURL = import.meta.env.VITE_API_BASE_URL
  const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'

  if (apiBaseURL) {
    const base = apiBaseURL.replace(/\/+$/, '')
    const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
    if (url.startsWith(prefix)) return `${base}${url}`
    return `${base}${prefix}${url.startsWith('/') ? '' : '/'}${url}`
  }

  const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
  if (url.startsWith(prefix)) return url
  return `${prefix}${url.startsWith('/') ? '' : '/'}${url}`
}

const imageUrl = computed(() => normalizeUrl(imagePath.value || ''))

const toSubmitUrl = (url) => {
  const value = String(url || '').trim()
  if (!value) return ''
  if (value.startsWith('/')) return value

  if (value.startsWith('http')) {
    const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
    const normalizedPrefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
    try {
      const parsed = new URL(value)
      if (parsed.pathname.startsWith(normalizedPrefix)) {
        return `${parsed.pathname}${parsed.search || ''}`
      }
      return value
    } catch {
      return value
    }
  }

  return value
}

const emitValue = (value) => {
  const finalValue = toSubmitUrl(value)
  emit('update:modelValue', finalValue)
  emit('change', finalValue)
}

// 上传配置
const uploadAction = computed(() => {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL
  const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
  if (apiBaseURL) {
    const base = apiBaseURL.replace(/\/+$/, '')
    const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
    return `${base}${prefix}/attachments/upload`
  }
  return `${apiPrefix}/attachments/upload`
})

const uploadHeaders = computed(() => buildAdminAuthHeaders())

const uploadData = computed(() => {
  return {}
})

// 上传前验证
const beforeUpload = (file) => {
  const isImage = file.type.startsWith('image/')
  if (!isImage) {
    ElMessage.error('只能上传图片文件')
    return false
  }
  
  const maxSize = 10 * 1024 * 1024 // 10MB
  if (file.size > maxSize) {
    ElMessage.error('图片大小不能超过 10MB')
    return false
  }
  
  return true
}

// 上传成功
const handleUploadSuccess = (response) => {
  if (response.code === 200 && response.data) {
    const path = toSubmitUrl(response.data.file_url || response.data.preview_url || '')
    imagePath.value = path
    emitValue(path)
    ElMessage.success('上传成功')
  } else {
    ElMessage.error(response.message || '上传失败')
  }
}

// 上传失败
const handleUploadError = (error) => {
  console.error('Upload error', error)
  ElMessage.error('上传失败')
}

// 裁剪上传
const handleCropUpload = () => {
  cropDialogVisible.value = true
  cropOption.img = ''
  cropFileName.value = ''
}

// 选择图片文件
const onCropFileChange = (file) => {
  const isImage = file.raw.type.startsWith('image/')
  if (!isImage) {
    ElMessage.error('只能选择图片文件')
    return
  }
  
  const maxSize = 10 * 1024 * 1024
  if (file.size > maxSize) {
    ElMessage.error('图片大小不能超过 10MB')
    return
  }

  cropFileName.value = file.name
  const reader = new FileReader()
  reader.onload = (e) => {
    cropOption.img = e.target.result
  }
  reader.readAsDataURL(file.raw)
}

// 确认裁剪并上传
const handleCropConfirm = () => {
  if (!cropOption.img) {
    ElMessage.warning('请先选择图片')
    return
  }

  cropUploading.value = true
  cropperRef.value.getCropBlob((blob) => {
    if (!blob) {
      cropUploading.value = false
      return
    }

    const file = new File([blob], cropFileName.value || 'cropped-image.png', {
      type: blob.type
    })

    const formData = new FormData()
    formData.append('file', file)

    axios.post(uploadAction.value, formData, {
      headers: uploadHeaders.value,
      timeout: 10000
    }).then(response => {
      cropUploading.value = false
      if (response.data.code === 200 && response.data.data) {
        const path = toSubmitUrl(response.data.data.file_url || response.data.data.preview_url || '')
        imagePath.value = path
        emitValue(path)
        cropDialogVisible.value = false
        ElMessage.success('上传成功')
      } else {
        ElMessage.error(response.data.message || '上传失败')
      }
    }).catch(error => {
      cropUploading.value = false
      console.error('Crop upload error', error)
      ElMessage.error('上传失败')
    })
  })
}

// 删除图片
const handleRemove = () => {
  imagePath.value = ''
  emitValue('')
}
</script>

<style scoped>
.image-upload-wrapper {
  width: 100%;
}

.upload-buttons {
  display: flex;
  align-items: center;
  gap: 10px;
}

.image-preview {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
}

.image-actions {
  display: flex;
  gap: 10px;
}

.crop-container {
  display: flex;
  justify-content: center;
  align-items: center;
}
</style>
