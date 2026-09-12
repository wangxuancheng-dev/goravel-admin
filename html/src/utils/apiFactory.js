import request from './request'

/**
 * 创建 CRUD API 工厂函数
 * @param {string} resource - 资源名称（如 'admins', 'roles'）
 * @returns {Object} CRUD API 方法
 */
export function createCRUDApi(resource) {
  return {
    /**
     * 获取列表
     * @param {Object} params - 查询参数
     * @returns {Promise} 请求结果
     */
    list: (params) => {
      return request({
        url: `/${resource}`,
        method: 'get',
        params
      })
    },

    /**
     * 获取详情
     * @param {string|number} id - 资源 ID
     * @returns {Promise} 请求结果
     */
    detail: (id) => {
      return request({
        url: `/${resource}/${id}`,
        method: 'get'
      })
    },

    /**
     * 创建资源
     * @param {Object} data - 创建数据
     * @returns {Promise} 请求结果
     */
    create: (data) => {
      return request({
        url: `/${resource}`,
        method: 'post',
        data
      })
    },

    /**
     * 更新资源
     * @param {string|number} id - 资源 ID
     * @param {Object} data - 更新数据
     * @returns {Promise} 请求结果
     */
    update: (id, data) => {
      return request({
        url: `/${resource}/${id}`,
        method: 'put',
        data
      })
    },

    /**
     * 删除资源
     * @param {string|number} id - 资源 ID
     * @param {Object} [data] - 可选 body（如 confirm_code）
     * @returns {Promise} 请求结果
     */
    delete: (id, data) => {
      return request({
        url: `/${resource}/${id}`,
        method: 'delete',
        ...(data ? { data } : {})
      })
    },

    /**
     * 批量删除
     * @param {Array<string|number>} ids - 资源 ID 数组
     * @returns {Promise} 请求结果
     */
    batchDelete: (ids) => {
      return request({
        url: `/${resource}/batch`,
        method: 'delete',
        data: { ids }
      })
    }
  }
}

/**
 * 扩展 CRUD API，添加自定义方法
 * @param {Object} baseApi - 基础 CRUD API
 * @param {Object} customMethods - 自定义方法
 * @returns {Object} 扩展后的 API
 */
export function extendApi(baseApi, customMethods) {
  return {
    ...baseApi,
    ...customMethods
  }
}

export default {
  createCRUDApi,
  extendApi
}

