import { buildRangeSearchParams } from "@/utils/listPageHelpers";

export const articleInitialSearchForm = {
  admin_id: "",
  title: "",
  content: "",
  status: "",
  created_at: [],
  updated_at: [],
};

const rangeSearchFields = ["created_at", "updated_at"];

export function buildArticleListParams(form, baseParams) {
  return buildRangeSearchParams(form, baseParams, rangeSearchFields);
}

export function createArticleSearchFields(t) {
  return [
    {
      prop: "admin_id",
      label: t("admin_id"),
      type: "input",
      clearable: true,
      width: "200px",
      advanced: false,
    },
    {
      prop: "title",
      label: t("title"),
      type: "input",
      clearable: true,
      width: "200px",
      advanced: false,
    },
    {
      prop: "content",
      label: t("content"),
      type: "input",
      clearable: true,
      width: "200px",
      advanced: false,
    },
    {
      prop: "status",
      label: t("table.status"),
      type: "select",
      clearable: true,
      width: "200px",
      advanced: false,
      apiUrl: "/options?type=dictionary&dictionary_type=status",
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

export function createArticleTableColumns(t, options = {}) {
  const { enableBatchActions = false } = options;
  const baseColumns = [
    { field: "id", title: t("table.id"), width: 80, sortable: true, key: "id" },
    {
      field: "admin_id",
      title: t("admin_id"),
      slot: "admin_id",
      sortable: false,
      key: "admin_id",
    },
    { field: "title", title: t("title"), sortable: false, key: "title" },
    {
      field: "content",
      title: t("content"),
      slot: "content",
      sortable: false,
      width: 220,
      key: "content",
    },
    {
      field: "status",
      title: t("table.status"),
      width: 100,
      sortable: false,
      slot: "status",
      key: "status",
    },
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

export function getadminDisplayName(value) {
  if (!value) return "-";
  return value.nickname || value.admin || "-";
}
