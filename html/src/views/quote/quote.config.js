import { buildRangeSearchParams } from "@/utils/listPageHelpers";

export const quoteInitialSearchForm = {
  quote_no: "",
  customer_name: "",
  status: "",
  remark: "",
  created_at: [],
  updated_at: [],
};

const rangeSearchFields = ["created_at", "updated_at"];

export function buildQuoteListParams(form, baseParams) {
  return buildRangeSearchParams(form, baseParams, rangeSearchFields);
}

export function createQuoteSearchFields(t) {
  return [
    {
      prop: "quote_no",
      label: t("quote_no"),
      type: "input",
      clearable: true,
      width: "200px",
      advanced: false,
    },
    {
      prop: "customer_name",
      label: t("customer_name"),
      type: "input",
      clearable: true,
      width: "200px",
      advanced: false,
    },
    {
      prop: "status",
      label: t("table.status"),
      type: "input",
      clearable: true,
      width: "200px",
      advanced: false,
    },
    {
      prop: "remark",
      label: t("remark"),
      type: "input",
      clearable: true,
      width: "200px",
      advanced: false,
    },
    {
      prop: "created_at",
      label: t("common.created_at"),
      type: "datetimerange",
      clearable: true,
      width: "380px",
      advanced: false,
    },
    {
      prop: "updated_at",
      label: t("common.updated_at"),
      type: "datetimerange",
      clearable: true,
      width: "380px",
      advanced: false,
    },
  ];
}

export function createQuoteTableColumns(t, options = {}) {
  const { enableBatchActions = false } = options;
  const baseColumns = [
    { field: "id", title: t("table.id"), width: 80, sortable: true, key: "id" },
    {
      field: "quote_no",
      title: t("quote_no"),
      sortable: false,
      key: "quote_no",
    },
    {
      field: "customer_name",
      title: t("customer_name"),
      sortable: false,
      key: "customer_name",
    },
    { field: "status", title: t("status"), sortable: false, key: "status" },
    { field: "remark", title: t("remark"), sortable: false, key: "remark" },
    {
      field: "updated_at",
      title: t("table.updated_at"),
      width: 180,
      sortable: true,
      key: "updated_at",
    },
    {
      field: "created_at",
      title: t("table.created_at"),
      width: 180,
      sortable: true,
      key: "created_at",
    },
    {
      field: "operation",
      title: t("table.operation"),
      width: 220,
      fixed: "right",
      slot: "operation",
      key: "operation",
    },
  ];

  if (!enableBatchActions) {
    return baseColumns;
  }

  return [
    { type: "checkbox", width: 52, fixed: "left", key: "checkbox" },
    ...baseColumns,
  ];
}
