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
        <FormField
          v-for="f in formFields"
          :key="f.prop"
          :field="f"
          :model="formData"
        />

        <el-divider content-position="left">{{
          $t("common.detail_rows")
        }}</el-divider>
        <div style="margin-bottom: 12px">
          <el-button type="primary" link @click="addDetailRow">{{
            $t("common.add")
          }}</el-button>
        </div>
        <el-table :data="formData.details" border size="small">
          <el-table-column :label="$t('product_name')" min-width="120">
            <template #default="{ row }">
              <el-input v-model="row.product_name" />
            </template>
          </el-table-column>
          <el-table-column :label="$t('quantity')" min-width="120">
            <template #default="{ row }">
              <el-input-number
                v-model="row.quantity"
                :controls="false"
                style="width: 100%"
              />
            </template>
          </el-table-column>
          <el-table-column :label="$t('unit_price')" min-width="120">
            <template #default="{ row }">
              <el-input v-model="row.unit_price" />
            </template>
          </el-table-column>
          <el-table-column
            :label="$t('common.operation')"
            width="80"
            fixed="right"
          >
            <template #default="{ $index }">
              <el-button type="danger" link @click="removeDetailRow($index)">{{
                $t("common.delete")
              }}</el-button>
            </template>
          </el-table-column>
        </el-table>
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

import { createQuote, updateQuote, getQuoteDetail } from "../../api/quote";
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
  quote_no: "",
  customer_name: "",
  status: 0,
  remark: "",
  details: [{}],
});

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (val) => emit("update:modelValue", val),
});

const dialogTitle = computed(() => {
  return formData.id ? t("quote.edit_quote") : t("quote.add_quote");
});

const formData = reactive(getFormInitialValue());

const addDetailRow = () => {
  if (!Array.isArray(formData.details)) formData.details = [];
  formData.details.push({});
};
const removeDetailRow = (index) => {
  formData.details.splice(index, 1);
  if (formData.details.length === 0) formData.details.push({});
};

const formRules = computed(() => {
  const rules = {};

  rules["quote_no"] = [
    { required: true, message: t("common.required"), trigger: "blur" },
  ];
  rules["customer_name"] = [
    { required: true, message: t("common.required"), trigger: "blur" },
  ];
  rules["status"] = [
    { required: true, message: t("common.required"), trigger: "blur" },
  ];
  return rules;
});

// Schema-driven form fields.
const formFields = computed(() => {
  const fields = [];

  fields.push({
    prop: "quote_no",
    label: t("quote_no"),
    type: "input",
    disabled: loading.value,
  });
  fields.push({
    prop: "customer_name",
    label: t("customer_name"),
    type: "input",
    disabled: loading.value,
  });
  fields.push({
    prop: "status",
    label: t("status"),
    type: "number",
    disabled: loading.value,
    min: 0,
  });
  fields.push({
    prop: "remark",
    label: t("remark"),
    type: "input",
    disabled: loading.value,
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
    const res = await getQuoteDetail(props.editId);
    if (res.data && res.data.quote) {
      const data = res.data.quote;
      const mapped = mapFields(data, getFormInitialValue());
      const normalizeRules = {};

      const normalized = normalizeFormData(mapped, normalizeRules);
      Object.assign(formData, normalized);

      formData.details =
        Array.isArray(data.details) && data.details.length
          ? data.details.map((row) => ({ ...row }))
          : [{}];
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

const isBlankDetailRow = (row) => {
  if (!row || typeof row !== "object") return true;
  {
    const v = row["product_name"];
    if (v !== undefined && v !== null && String(v).trim() !== "") return false;
  }
  {
    const v = row["quantity"];
    if (v !== undefined && v !== null && String(v).trim() !== "") return false;
  }
  {
    const v = row["unit_price"];
    if (v !== undefined && v !== null && String(v).trim() !== "") return false;
  }
  return true;
};

const validateDetailRows = () => {
  const rows = Array.isArray(formData.details) ? formData.details : [];
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i];
    if (isBlankDetailRow(row)) continue;
    {
      const v = row["product_name"];
      if (v === undefined || v === null || String(v).trim() === "") {
        ElMessage.warning(t("common.required") + ": product name");
        return false;
      }
    }
    {
      const v = row["quantity"];
      if (v === undefined || v === null || String(v).trim() === "") {
        ElMessage.warning(t("common.required") + ": qty");
        return false;
      }
    }
    {
      const v = row["unit_price"];
      if (v === undefined || v === null || String(v).trim() === "") {
        ElMessage.warning(t("common.required") + ": unit price");
        return false;
      }
    }
  }
  return true;
};

const handleSubmit = async () => {
  if (!formRef.value) return;

  await formRef.value.validate(async (valid) => {
    if (!valid) return;

    if (!validateDetailRows()) return;

    submitting.value = true;
    try {
      const data = { ...formData };
      delete data.id;

      data.details = (
        Array.isArray(formData.details) ? formData.details : []
      ).filter((row) => !isBlankDetailRow(row));

      if (props.editId) {
        await updateQuote(props.editId, data);
        ElMessage.success(t("common.update_success"));
      } else {
        await createQuote(data);
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
