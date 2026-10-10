import request from "../utils/request";
import { createCRUDApi, extendApi } from "../utils/apiFactory";
import { normalizeListResponse } from "../utils/normalize";

const baseQuoteApi = createCRUDApi("quotes");

const quoteApi = extendApi(baseQuoteApi, {
  export: (params) => {
    return request({
      url: "/quotes/export",
      method: "post",
      data: params,
    });
  },
});

export async function getQuoteList(params) {
  const res = await quoteApi.list(params);
  return normalizeListResponse(res);
}

export function getQuoteDetail(id) {
  return quoteApi.detail(id);
}

export function createQuote(data) {
  return quoteApi.create(data);
}

export function updateQuote(id, data) {
  return quoteApi.update(id, data);
}

export function deleteQuote(id) {
  return quoteApi.delete(id);
}

export function exportQuote(params) {
  return quoteApi.export(params);
}
