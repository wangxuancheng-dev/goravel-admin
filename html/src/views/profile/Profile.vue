<template>
  <div class="profile-container">
    <el-alert
      v-if="userStore.adminInfo?.must_change_password"
      type="warning"
      :closable="false"
      show-icon
      style="margin-bottom: 16px;"
      :title="$t('profile.must_change_password_title')"
      :description="$t('profile.must_change_password_desc')"
    />
    <el-row :gutter="20">
      <el-col :span="8">
        <el-card class="profile-card">
          <template #header>
            <div class="card-header">
              <span>{{ $t('profile.basic_info') }}</span>
            </div>
          </template>
          <div class="avatar-section">
            <el-avatar :size="100" :src="adminInfo.avatar" class="avatar">
              <el-icon><User /></el-icon>
            </el-avatar>
            <div class="avatar-actions">
              <el-button type="primary" link @click="handleOpenAvatarDialog">
                {{ $t('profile.change_avatar') }}
              </el-button>
            </div>
          </div>
          <el-descriptions :column="1" border class="info-descriptions">
            <el-descriptions-item :label="$t('profile.username')">
              {{ adminInfo.username }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('profile.nickname')">
              {{ adminInfo.nickname || '-' }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('profile.email')">
              {{ adminInfo.email || '-' }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('profile.phone')">
              {{ adminInfo.phone || '-' }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('profile.department')">
              {{ (adminInfo.department && adminInfo.department.name) ? adminInfo.department.name : '-' }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('profile.roles')">
              <template v-if="adminInfo.roles && adminInfo.roles.length > 0">
                <el-tag
                  v-for="role in adminInfo.roles"
                  :key="role.id"
                  style="margin-right: 5px;"
                >
                  {{ role.name }}
                </el-tag>
              </template>
              <span v-else>-</span>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <el-col :span="16">
        <el-card>
          <el-tabs v-model="activeTab">
            <el-tab-pane :label="$t('profile.edit_info')" name="info">
              <el-form
                ref="infoFormRef"
                :model="infoForm"
                :rules="infoRules"
                label-width="120px"
                style="max-width: 600px;"
              >
                <el-form-item :label="$t('profile.nickname')" prop="nickname">
                  <el-input v-model="infoForm.nickname" />
                </el-form-item>
                <el-form-item :label="$t('profile.email')" prop="email">
                  <el-input v-model="infoForm.email" type="email" />
                </el-form-item>
                <el-form-item :label="$t('profile.phone')" prop="phone">
                  <el-input v-model="infoForm.phone" />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" @click="handleUpdateInfo" :loading="infoSubmitting" :disabled="getButtonState('profile.update').disabled">
                    {{ $t('common.save') }}
                  </el-button>
                  <el-button @click="handleResetInfo">{{ $t('common.reset') }}</el-button>
                </el-form-item>
              </el-form>
            </el-tab-pane>

            <el-tab-pane
              v-if="canManageOwnPassword"
              :label="$t('profile.change_password')"
              name="password"
            >
              <el-form
                ref="passwordFormRef"
                :model="passwordForm"
                :rules="passwordRules"
                label-width="120px"
                style="max-width: 600px;"
              >
                <el-form-item :label="$t('profile.old_password')" prop="old_password">
                  <el-input
                    v-model="passwordForm.old_password"
                    type="password"
                    show-password
                  />
                </el-form-item>
                <el-form-item :label="$t('profile.new_password')" prop="new_password">
                  <el-input
                    v-model="passwordForm.new_password"
                    type="password"
                    show-password
                  />
                </el-form-item>
                <el-form-item :label="$t('profile.confirm_password')" prop="confirm_password">
                  <el-input
                    v-model="passwordForm.confirm_password"
                    type="password"
                    show-password
                  />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" @click="handleUpdatePassword" :loading="passwordSubmitting" :disabled="!canSubmitOwnPassword">
                    {{ $t('common.save') }}
                  </el-button>
                  <el-button @click="handleResetPassword">{{ $t('common.reset') }}</el-button>
                </el-form-item>
              </el-form>
            </el-tab-pane>

            <el-tab-pane
              v-if="getButtonState('google_authenticator.manage').show"
              :label="$t('profile.google_authenticator')"
              name="2fa"
              :disabled="getButtonState('google_authenticator.manage').disabled"
            >
              <div class="google-authenticator-section">
                <el-alert
                  v-if="!googleAuthStatus.is_bound"
                  :title="$t('profile.google_auth_not_bound')"
                  type="info"
                  :closable="false"
                  show-icon
                  style="margin-bottom: 20px;"
                />
                <el-alert
                  v-else
                  :title="$t('profile.google_auth_bound')"
                  type="success"
                  :closable="false"
                  show-icon
                  style="margin-bottom: 20px;"
                />

                <div v-if="!googleAuthStatus.is_bound" class="bind-section">
                  <el-steps :active="bindStep" finish-status="success" style="margin-bottom: 30px;">
                    <el-step :title="$t('profile.step1_scan_qr')" />
                    <el-step :title="$t('profile.step2_verify')" />
                  </el-steps>

                  <div v-if="bindStep === 0" class="qr-code-section">
                    <el-alert
                      :title="$t('profile.scan_qr_tip')"
                      type="warning"
                      :closable="false"
                      show-icon
                      style="margin-bottom: 20px;"
                    />
                    <div class="qr-code-container">
                      <div v-if="qrCodeLoading" class="loading-container">
                        <el-icon class="is-loading"><Loading /></el-icon>
                        <span>{{ $t('common.loading') }}</span>
                      </div>
                      <div v-else-if="qrCodeData.qr_code_image" class="qr-code-wrapper">
                        <img :src="qrCodeData.qr_code_image" alt="QR Code" class="qr-code-image" />
                        <div class="qr-code-info">
                          <p><strong>{{ $t('profile.secret_key') }}:</strong> {{ qrCodeData.secret }}</p>
                          <p class="tip-text">{{ $t('profile.save_secret_tip') }}</p>
                        </div>
                      </div>
                    </div>
                    <el-button
                      type="primary"
                      @click="bindStep = 1"
                      :disabled="!qrCodeData.secret || getButtonState('google_authenticator.manage').disabled"
                    >
                      {{ $t('profile.next_step') }}
                    </el-button>
                  </div>

                  <div v-if="bindStep === 1" class="verify-section">
                    <el-form
                      ref="bindFormRef"
                      :model="bindForm"
                      :rules="bindRules"
                      label-width="120px"
                      style="max-width: 400px;"
                    >
                      <el-form-item :label="$t('profile.verification_code')" prop="code">
                        <el-input
                          v-model="bindForm.code"
                          :placeholder="$t('profile.enter_6_digit_code')"
                          maxlength="6"
                          style="width: 200px;"
                        />
                      </el-form-item>
                      <el-form-item>
                        <el-button @click="bindStep = 0">{{ $t('common.back') }}</el-button>
                        <el-button
                          type="primary"
                          @click="handleBindGoogleAuth"
                          :loading="bindSubmitting"
                          :disabled="getButtonState('google_authenticator.manage').disabled"
                        >
                          {{ $t('profile.bind') }}
                        </el-button>
                      </el-form-item>
                    </el-form>
                  </div>
                </div>

                <div v-else class="unbind-section">
                  <el-form
                    ref="unbindFormRef"
                    :model="unbindForm"
                    :rules="unbindRules"
                    label-width="120px"
                    style="max-width: 400px;"
                  >
                    <el-form-item :label="$t('profile.verification_code')" prop="code">
                      <el-input
                        v-model="unbindForm.code"
                        :placeholder="$t('profile.enter_6_digit_code')"
                        maxlength="6"
                        style="width: 200px;"
                      />
                    </el-form-item>
                    <el-form-item>
                      <el-button
                        type="danger"
                        @click="handleUnbindGoogleAuth"
                        :loading="unbindSubmitting"
                        :disabled="getButtonState('google_authenticator.manage').disabled"
                      >
                        {{ $t('profile.unbind') }}
                      </el-button>
                    </el-form-item>
                  </el-form>
                </div>
              </div>
            </el-tab-pane>

            <el-tab-pane :label="$t('profile.sessions')" name="sessions">
              <ProfileSessions />
            </el-tab-pane>
          </el-tabs>
        </el-card>
      </el-col>
    </el-row>

    <!-- 头像选择对话框 -->
    <el-dialog
      v-model="showAvatarDialog"
      :title="$t('profile.change_avatar')"
      width="500px"
    >
      <div class="avatar-selector">
        <div class="avatar-grid">
          <div
            v-for="avatar in defaultAvatars"
            :key="avatar"
            class="avatar-item"
            :class="{ active: selectedAvatar === avatar }"
            @click="selectedAvatar = avatar"
          >
            <el-avatar :size="60" :src="avatar">
              <el-icon><User /></el-icon>
            </el-avatar>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showAvatarDialog = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleSaveAvatar" :loading="avatarSubmitting" :disabled="!selectedAvatar">
          {{ $t('common.confirm') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { User, Plus, Loading } from '@element-plus/icons-vue'
import { getProfile, updateProfile, updatePassword } from '../../api/profile'
import { 
  getGoogleAuthenticatorStatus, 
  getGoogleAuthenticatorQRCode, 
  bindGoogleAuthenticator, 
  unbindGoogleAuthenticator 
} from '../../api/auth'
import ProfileSessions from './ProfileSessions.vue'
import { useUserStore } from '../../store/user'
import { usePermission } from '../../composables/usePermission'

const { t } = useI18n()
const route = useRoute()
const userStore = useUserStore()
const { getButtonState } = usePermission()

const mustChangePassword = computed(() => !!userStore.adminInfo?.must_change_password)
const passwordPerm = computed(() => getButtonState('password.update'))
// Force-change must always reach password form; otherwise require password.update
const canManageOwnPassword = computed(
  () => mustChangePassword.value || passwordPerm.value.show
)
const canSubmitOwnPassword = computed(
  () => mustChangePassword.value || !passwordPerm.value.disabled
)

const activeTab = ref(route.query.tab === 'password' ? 'password' : 'info')
const infoFormRef = ref(null)
const passwordFormRef = ref(null)
const infoSubmitting = ref(false)
const passwordSubmitting = ref(false)
const showAvatarDialog = ref(false)
const selectedAvatar = ref('')
const avatarSubmitting = ref(false)

// 谷歌验证码相关
const googleAuthStatus = ref({ is_bound: false })
const qrCodeLoading = ref(false)
const qrCodeData = ref({ secret: '', qr_code_image: '' })
const bindStep = ref(0)
const bindSubmitting = ref(false)
const unbindSubmitting = ref(false)
const bindFormRef = ref(null)
const unbindFormRef = ref(null)

const bindForm = reactive({
  code: ''
})

const unbindForm = reactive({
  code: ''
})

// 系统默认头像列表（使用UI Avatars服务生成）
const defaultAvatars = [
  'https://ui-avatars.com/api/?name=A&background=409EFF&color=fff&size=128', // 蓝色
  'https://ui-avatars.com/api/?name=B&background=67C23A&color=fff&size=128', // 绿色
  'https://ui-avatars.com/api/?name=C&background=E6A23C&color=fff&size=128', // 橙色
  'https://ui-avatars.com/api/?name=D&background=F56C6C&color=fff&size=128', // 红色
  'https://ui-avatars.com/api/?name=E&background=9C27B0&color=fff&size=128', // 紫色
  'https://ui-avatars.com/api/?name=F&background=00BCD4&color=fff&size=128', // 青色
  'https://ui-avatars.com/api/?name=G&background=FF9800&color=fff&size=128', // 深橙色
  'https://ui-avatars.com/api/?name=H&background=4CAF50&color=fff&size=128', // 深绿色
  'https://ui-avatars.com/api/?name=I&background=2196F3&color=fff&size=128', // 亮蓝色
  'https://ui-avatars.com/api/?name=J&background=FF5722&color=fff&size=128', // 深红色
  'https://ui-avatars.com/api/?name=K&background=795548&color=fff&size=128', // 棕色
  'https://ui-avatars.com/api/?name=L&background=607D8B&color=fff&size=128', // 蓝灰色
  'https://ui-avatars.com/api/?name=M&background=3F51B5&color=fff&size=128', // 靛蓝色
  'https://ui-avatars.com/api/?name=N&background=009688&color=fff&size=128', // 青绿色
  'https://ui-avatars.com/api/?name=O&background=FFC107&color=fff&size=128', // 琥珀色
  'https://ui-avatars.com/api/?name=P&background=E91E63&color=fff&size=128', // 粉红色
  'https://ui-avatars.com/api/?name=Q&background=8BC34A&color=fff&size=128', // 浅绿色
  'https://ui-avatars.com/api/?name=R&background=CDDC39&color=fff&size=128', // 黄绿色
  'https://ui-avatars.com/api/?name=S&background=FFEB3B&color=333&size=128', // 黄色
  'https://ui-avatars.com/api/?name=T&background=FF9800&color=fff&size=128', // 橙色
  'https://ui-avatars.com/api/?name=U&background=9E9E9E&color=fff&size=128', // 灰色
  'https://ui-avatars.com/api/?name=V&background=673AB7&color=fff&size=128', // 深紫色
  'https://ui-avatars.com/api/?name=W&background=00ACC1&color=fff&size=128', // 深青色
  'https://ui-avatars.com/api/?name=X&background=5C6BC0&color=fff&size=128', // 蓝紫色
  'https://ui-avatars.com/api/?name=Y&background=F44336&color=fff&size=128', // 亮红色
  'https://ui-avatars.com/api/?name=Z&background=26A69A&color=fff&size=128', // 青蓝色
]

const adminInfo = computed(() => userStore.adminInfo || {})

const infoForm = reactive({
  nickname: '',
  email: '',
  phone: ''
})

const passwordForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const validateConfirmPassword = (rule, value, callback) => {
  if (value !== passwordForm.new_password) {
    callback(new Error(t('profile.password_not_match')))
  } else {
    callback()
  }
}

const validateEmail = (rule, value, callback) => {
  if (value && value.trim() !== '') {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailRegex.test(value)) {
      callback(new Error(t('profile.email_invalid')))
    } else {
      callback()
    }
  } else {
    callback()
  }
}

const validatePhone = (rule, value, callback) => {
  if (value && value.trim() !== '') {
    const phoneRegex = /^1[3-9]\d{9}$/
    if (!phoneRegex.test(value)) {
      callback(new Error(t('profile.phone_invalid')))
    } else {
      callback()
    }
  } else {
    callback()
  }
}

const infoRules = {
  email: [
    { validator: validateEmail, trigger: 'blur' }
  ],
  phone: [
    { validator: validatePhone, trigger: 'blur' }
  ]
}

const passwordRules = {
  old_password: [
    { required: true, message: t('profile.old_password_required'), trigger: 'blur' }
  ],
  new_password: [
    { required: true, message: t('profile.new_password_required'), trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: t('profile.confirm_password_required'), trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

const validateGoogleCode = (rule, value, callback) => {
  if (!value || value.trim() === '') {
    callback(new Error(t('profile.google_code_required')))
  } else if (!/^\d{6}$/.test(value)) {
    callback(new Error(t('profile.google_code_format')))
  } else {
    callback()
  }
}

const bindRules = {
  code: [
    { validator: validateGoogleCode, trigger: 'blur' }
  ]
}

const unbindRules = {
  code: [
    { validator: validateGoogleCode, trigger: 'blur' }
  ]
}

const loadProfile = async () => {
  try {
    const res = await getProfile()
    if (res.data && res.data.admin) {
      const admin = res.data.admin
      
      let department = null
      const dept = admin.Department || admin.department
      
      if (dept) {
        const deptId = dept.ID || dept.id || dept.Id || (dept.ID !== undefined ? dept.ID : null)
        const deptName = dept.Name || dept.name || dept.Name || ''
        
        if (deptId && deptId > 0) {
          department = {
            id: deptId,
            name: deptName || '-'
          }
        } else if (deptName && deptName !== '') {
          department = {
            id: 0,
            name: deptName
          }
        }
      }
      
      if (!department && (admin.DepartmentID || admin.department_id)) {
        const deptId = admin.DepartmentID || admin.department_id
        if (deptId && deptId > 0) {
          department = {
            id: deptId,
            name: '-'
          }
        }
      }
      
      const rolesArray = admin.Roles || admin.roles || []
      const roleMap = new Map()
      rolesArray.forEach(role => {
        const roleId = role.ID || role.id
        if (roleId && !roleMap.has(roleId)) {
          roleMap.set(roleId, {
            id: roleId,
            name: role.Name || role.name,
            slug: role.Slug || role.slug
          })
        }
      })
      const uniqueRoles = Array.from(roleMap.values())
      
      const transformedAdmin = {
        id: admin.ID || admin.id,
        username: admin.Username || admin.username,
        nickname: admin.Nickname || admin.nickname,
        email: admin.Email || admin.email,
        phone: admin.Phone || admin.phone,
        avatar: admin.Avatar || admin.avatar,
        department_id: admin.DepartmentID || admin.department_id,
        department: department,
        roles: uniqueRoles,
        // 保留权限信息，避免权限丢失
        permissions: admin.permissions || admin.Permissions || userStore.permissions || []
      }
      
      infoForm.nickname = transformedAdmin.nickname || ''
      infoForm.email = transformedAdmin.email || ''
      infoForm.phone = transformedAdmin.phone || ''
      
      // 使用 fetchUserInfo 来更新所有信息（包括权限），而不是只调用 setAdminInfo
      // 这样可以确保权限信息不会丢失
      await userStore.fetchUserInfo(true)
      
      // 更新表单数据（因为 fetchUserInfo 可能已经更新了 adminInfo）
      const currentAdminInfo = userStore.adminInfo
      if (currentAdminInfo) {
        infoForm.nickname = currentAdminInfo.nickname || ''
        infoForm.email = currentAdminInfo.email || ''
        infoForm.phone = currentAdminInfo.phone || ''
      }
    }
  } catch (error) {
    console.error('Load profile error:', error)
    if (!error.__handled) {
      const errorMessage = error.response?.data?.message || error.message || t('common.operation_failed')
      ElMessage.error(errorMessage)
    }
  }
}

const handleUpdateInfo = async () => {
  if (!infoFormRef.value) return

  await infoFormRef.value.validate(async (valid) => {
    if (valid) {
      infoSubmitting.value = true
      try {
        const res = await updateProfile(infoForm)
        if (res.data && res.data.admin) {
          const admin = res.data.admin
          
          // 处理部门数据
          let department = null
          const dept = admin.Department || admin.department
          if (dept) {
            const deptId = dept.ID || dept.id || dept.Id
            const deptName = dept.Name || dept.name || dept.Name
            if (deptId && deptId > 0) {
              department = {
                id: deptId,
                name: deptName || '-'
              }
            }
          }
          
          // 如果部门数据为空，但 department_id 存在，尝试从 department_id 获取
          if (!department && (admin.DepartmentID || admin.department_id)) {
            const deptId = admin.DepartmentID || admin.department_id
            if (deptId && deptId > 0) {
              department = {
                id: deptId,
                name: '-' // 暂时显示 '-'，后续可以从部门列表获取名称
              }
            }
          }
          
          // 处理角色数据（去重）
          const rolesArray = admin.Roles || admin.roles || []
          const roleMap = new Map()
          rolesArray.forEach(role => {
            const roleId = role.ID || role.id
            if (roleId && !roleMap.has(roleId)) {
              roleMap.set(roleId, {
                id: roleId,
                name: role.Name || role.name,
                slug: role.Slug || role.slug
              })
            }
          })
          const uniqueRoles = Array.from(roleMap.values())
          
          // 转换数据格式（PascalCase -> snake_case）
          const transformedAdmin = {
            id: admin.ID || admin.id,
            username: admin.Username || admin.username,
            nickname: admin.Nickname || admin.nickname,
            email: admin.Email || admin.email,
            phone: admin.Phone || admin.phone,
            avatar: admin.Avatar || admin.avatar,
            department_id: admin.DepartmentID || admin.department_id,
            department: department,
            roles: uniqueRoles
          }
          
          userStore.setAdminInfo(transformedAdmin)
          ElMessage.success(t('profile.update_success'))
          // 重新获取用户信息（包括权限），确保权限信息不会丢失
          await userStore.fetchUserInfo(true)
        }
      } catch (error) {
        console.error('Update info error:', error)
        // 如果错误已经在响应拦截器中处理过，就不再重复显示
        if (!error.__handled) {
          const errorMessage = error.response?.data?.message || error.message || t('common.operation_failed')
          ElMessage.error(errorMessage)
        }
      } finally {
        infoSubmitting.value = false
      }
    }
  })
}

const handleResetInfo = () => {
  loadProfile()
  infoFormRef.value?.resetFields()
}

const handleUpdatePassword = async () => {
  if (!passwordFormRef.value) return

  await passwordFormRef.value.validate(async (valid) => {
    if (valid) {
      passwordSubmitting.value = true
      try {
        await updatePassword(passwordForm)
        ElMessage.success(t('profile.password_update_success'))
        handleResetPassword()
        // 重新获取用户信息（包括权限），确保权限信息不会丢失
        await userStore.fetchUserInfo(true)
      } catch (error) {
        console.error('Update password error:', error)
        // 如果错误已经在响应拦截器中处理过，就不再重复显示
        if (!error.__handled) {
          const errorMessage = error.response?.data?.message || error.message || t('common.operation_failed')
          ElMessage.error(errorMessage)
        }
      } finally {
        passwordSubmitting.value = false
      }
    }
  })
}

const handleResetPassword = () => {
  passwordForm.old_password = ''
  passwordForm.new_password = ''
  passwordForm.confirm_password = ''
  passwordFormRef.value?.resetFields()
}

const handleOpenAvatarDialog = () => {
  // 打开对话框时，如果已有头像，设置为选中状态
  selectedAvatar.value = adminInfo.value.avatar || ''
  showAvatarDialog.value = true
}

const handleSaveAvatar = async () => {
  if (!selectedAvatar.value) {
    ElMessage.warning(t('profile.please_select_avatar'))
    return
  }

  avatarSubmitting.value = true
  try {
    await updateProfile({ avatar: selectedAvatar.value })
    await loadProfile()
    ElMessage.success(t('profile.avatar_update_success'))
    showAvatarDialog.value = false
    selectedAvatar.value = ''
    // 重新获取用户信息（包括权限），确保权限信息不会丢失
    await userStore.fetchUserInfo(true)
  } catch (error) {
    console.error('Update avatar error:', error)
    if (!error.__handled) {
      const errorMessage = error.response?.data?.message || error.message || t('common.operation_failed')
      ElMessage.error(errorMessage)
    }
  } finally {
    avatarSubmitting.value = false
  }
}

// 加载谷歌验证码状态
const loadGoogleAuthStatus = async () => {
  try {
    const res = await getGoogleAuthenticatorStatus()
    if (res.data) {
      googleAuthStatus.value = res.data
    }
  } catch (error) {
    console.error('Load google auth status error:', error)
    if (!error.__handled) {
      const errorMessage = error.response?.data?.message || error.message || t('common.operation_failed')
      ElMessage.error(errorMessage)
    }
  }
}

// 加载二维码
const loadQRCode = async () => {
  qrCodeLoading.value = true
  try {
    const res = await getGoogleAuthenticatorQRCode()
    if (res.data) {
      qrCodeData.value = res.data
    }
  } catch (error) {
    console.error('Load QR code error:', error)
    // 如果错误已经在拦截器中处理过，不再重复显示
    if (!error?.__handled) {
      const errorMessage = error.response?.data?.message || error.translatedMessage || error.message || t('common.operation_failed')
      ElMessage.error(errorMessage)
    }
  } finally {
    qrCodeLoading.value = false
  }
}

// 绑定谷歌验证码
const handleBindGoogleAuth = async () => {
  if (!bindFormRef.value) return

  await bindFormRef.value.validate(async (valid) => {
    if (valid) {
      bindSubmitting.value = true
      try {
        await bindGoogleAuthenticator({
          secret: qrCodeData.value.secret,
          code: bindForm.code
        })
        ElMessage.success(t('profile.bind_success'))
        bindForm.code = ''
        bindStep.value = 0
        await loadGoogleAuthStatus()
      } catch (error) {
        console.error('Bind google auth error:', error)
        // 如果错误已经在拦截器中处理过，不再重复显示
        if (!error?.__handled) {
          const errorMessage = error.response?.data?.message || error.translatedMessage || error.message || t('common.operation_failed')
          ElMessage.error(errorMessage)
        }
      } finally {
        bindSubmitting.value = false
      }
    }
  })
}

// 解绑谷歌验证码
const handleUnbindGoogleAuth = async () => {
  if (!unbindFormRef.value) return

  await unbindFormRef.value.validate(async (valid) => {
    if (valid) {
      try {
        await ElMessageBox.confirm(
          t('profile.unbind_confirm'),
          t('common.confirm'),
          {
            confirmButtonText: t('common.confirm'),
            cancelButtonText: t('common.cancel'),
            type: 'warning'
          }
        )
        
        unbindSubmitting.value = true
        try {
          await unbindGoogleAuthenticator({
            code: unbindForm.code
          })
          ElMessage.success(t('profile.unbind_success'))
          unbindForm.code = ''
          await loadGoogleAuthStatus()
        } catch (error) {
          console.error('Unbind google auth error:', error)
          // 如果错误已经在拦截器中处理过，不再重复显示
          if (!error?.__handled) {
            const errorMessage = error.response?.data?.message || error.translatedMessage || error.message || t('common.operation_failed')
            ElMessage.error(errorMessage)
          }
        } finally {
          unbindSubmitting.value = false
        }
      } catch (error) {
        // 用户取消
      }
    }
  })
}

// 监听标签页切换
watch(activeTab, (newTab) => {
  if (newTab === '2fa' && !getButtonState('google_authenticator.manage').disabled) {
    loadGoogleAuthStatus()
    if (!googleAuthStatus.value.is_bound && !qrCodeData.value.secret) {
      loadQRCode()
    }
  }
})

onMounted(() => {
  loadProfile()
  if (!getButtonState('google_authenticator.manage').disabled) {
    loadGoogleAuthStatus()
  }
})
</script>

<style scoped>
.profile-container {
  padding: 0;
}

.profile-card {
  height: 100%;
}

.card-header {
  font-weight: 500;
  font-size: 16px;
}

.avatar-section {
  text-align: center;
  margin-bottom: 20px;
}

.avatar {
  margin-bottom: 10px;
}

.avatar-actions {
  margin-top: 10px;
}

.info-descriptions {
  margin-top: 20px;
}

.avatar-selector {
  padding: 20px 0;
}

.avatar-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 15px;
  max-height: 400px;
  overflow-y: auto;
}

.avatar-item {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 8px;
  border: 2px solid var(--border-color-light);
  border-radius: var(--border-radius-lg);
  cursor: pointer;
  transition: all 0.3s;
}

.avatar-item:hover {
  border-color: var(--el-color-primary);
  transform: scale(1.05);
}

.avatar-item.active {
  border-color: var(--el-color-primary);
  background-color: var(--el-color-primary-light-9);
}

.google-authenticator-section {
  padding: 20px 0;
}

.bind-section,
.unbind-section {
  max-width: 600px;
}

.qr-code-section {
  text-align: center;
}

.qr-code-container {
  margin: 30px 0;
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  gap: 10px;
}

.qr-code-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  padding: 20px;
  border-radius: var(--border-radius-lg);
  margin-bottom: 20px;
}

.qr-code-image {
  width: 200px;
  height: 200px;
  border: 1px solid var(--border-color-light);
  border-radius: var(--border-radius-lg);
  background: var(--card-bg);
  padding: 10px;
}

.qr-code-info {
  text-align: left;
  width: 100%;
  max-width: 400px;
}

.qr-code-info p {
  margin: 8px 0;
  word-break: break-all;
}

.tip-text {
  color: var(--text-color-secondary);
  font-size: 12px;
  margin-top: 10px !important;
}

.verify-section {
  padding: 20px 0;
}


</style>

