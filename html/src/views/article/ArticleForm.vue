<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="1000px"
    @close="handleDialogClose"
  >
    <div v-loading="loading">
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="120px"
      >
        <el-form-item :label="$t('content')" prop="content">
          <WangEditor
            v-model="formData.content"
            :placeholder="$t('content_placeholder')"
            :height="400"
          />
        </el-form-item>
        <FormField
          v-for="f in formFields"
          :key="f.prop"
          :field="f"
          :model="formData"
        />
      </el-form>
    </div>
    <template #footer>
      <el-button @click="handleCancel">{{ $t("common.cancel") }}</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="submitting">{{
        $t("common.confirm")
      }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { ElMessage } from "element-plus";
import FormField from "../../components/Form/FormField.vue";

import WangEditor from "../../components/WangEditor.vue";

// Relation field: admin_id -> admins
import {
  createArticle,
  updateArticle,
  getArticleDetail,
} from "../../api/article";
import { mapFields, normalizeFormData } from "../../utils/normalizeFormData";
import ErrorHandler from "../../utils/errorHandler";

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  editId: {
    type: [Number, String],
    default: null,
  },
});

const emit = defineEmits(["update:modelValue", "success"]);

const { t } = useI18n();
const formRef = ref(null);
const submitting = ref(false);
const loading = ref(false);

// Reusable function to build initial form values.
const getFormInitialValue = () => ({
  admin_id: null,
  title: "",
  content: "",
  status: null,
});

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (val) => emit("update:modelValue", val),
});

const dialogTitle = computed(() => {
  return formData.id ? t("article.edit_article") : t("article.add_article");
});

const formData = reactive(getFormInitialValue());

const formRules = computed(() => {
  const rules = {};

  rules["admin_id"] = [
    { required: true, message: t("common.select_required"), trigger: "change" },
  ];
  rules["title"] = [
    { required: true, message: t("common.required"), trigger: "blur" },
  ];
  rules["status"] = [
    { required: true, message: t("common.select_required"), trigger: "change" },
  ];
  return rules;
});

// Schema-driven form fields.
const formFields = computed(() => {
  const fields = [];

  fields.push({
    prop: "admin_id",
    label: t("admin_id"),
    type: "select",
    disabled: loading.value,
  });
  fields.push({
    prop: "title",
    label: t("title"),
    type: "input",
    disabled: loading.value,
  });
  fields.push({
    prop: "status",
    label: t("status"),
    type: "radio",
    disabled: loading.value,
    apiUrl: "/options?type=dictionary&dictionary_type=status",
    clearable: true,
  });
  return fields;
});

watch(
  () => props.editId,
  async (newId) => {
    if (newId && dialogVisible.value) {
      await loadData();
    } else if (!newId && dialogVisible.value) {
      resetForm();
    }
  },
  { immediate: true },
);

watch(dialogVisible, (visible) => {
  if (visible) {
    if (props.editId) {
      loadData();
    } else {
      resetForm();
    }
  }
});

const loadData = async () => {
  if (!props.editId) {
    resetForm();
    return;
  }

  loading.value = true;
  try {
    const res = await getArticleDetail(props.editId);
    if (res.data && res.data.article) {
      const data = res.data.article;
      const mapped = mapFields(data, getFormInitialValue());
      const normalizeRules = {};

      normalizeRules["status"] = "string";
      const normalized = normalizeFormData(mapped, normalizeRules);
      Object.assign(formData, normalized);
    }
  } catch (error) {
    ErrorHandler.handle(error);
  } finally {
    loading.value = false;
  }
};

const resetForm = () => {
  Object.assign(formData, getFormInitialValue());
  formRef.value?.resetFields();
};

const handleDialogClose = () => {
  formRef.value?.resetFields();
};

const handleCancel = () => {
  dialogVisible.value = false;
};

const handleSubmit = async () => {
  if (!formRef.value) return;

  await formRef.value.validate(async (valid) => {
    if (!valid) return;

    submitting.value = true;
    try {
      const data = { ...formData };
      delete data.id;

      if (props.editId) {
        await updateArticle(props.editId, data);
        ElMessage.success(t("common.update_success"));
      } else {
        await createArticle(data);
        ElMessage.success(t("common.create_success"));
      }

      emit("success");
      dialogVisible.value = false;
    } catch (error) {
      ErrorHandler.handle(error);
    } finally {
      submitting.value = false;
    }
  });
};
</script>

<style scoped></style>
