import request from "../utils/request";
import { createCRUDApi, extendApi } from "../utils/apiFactory";
import { normalizeListResponse } from "../utils/normalize";

const baseArticleApi = createCRUDApi("articles");

const articleApi = extendApi(baseArticleApi, {
  export: (params) => {
    return request({
      url: "/articles/export",
      method: "post",
      data: params,
    });
  },
  import: (file) => {
    const formData = new FormData();
    formData.append("file", file);
    return request({
      url: "/articles/import",
      method: "post",
      data: formData,
      headers: {
        "Content-Type": "multipart/form-data",
      },
    });
  },
});

export async function getArticleList(params) {
  const res = await articleApi.list(params);
  return normalizeListResponse(res);
}

export function getArticleDetail(id) {
  return articleApi.detail(id);
}

export function createArticle(data) {
  return articleApi.create(data);
}

export function updateArticle(id, data) {
  return articleApi.update(id, data);
}

export function deleteArticle(id) {
  return articleApi.delete(id);
}

export function exportArticle(params) {
  return articleApi.export(params);
}

export function importArticle(file) {
  return articleApi.import(file);
}
