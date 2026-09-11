import { beforeEach, describe, expect, it, vi } from 'vitest'

const requestMock = vi.fn()

vi.mock('./request', () => ({
  default: (...args) => requestMock(...args),
}))

import { createCRUDApi } from './apiFactory'

describe('createCRUDApi', () => {
  beforeEach(() => {
    requestMock.mockReset()
    requestMock.mockResolvedValue({ code: 200 })
  })

  it('builds standard CRUD urls', async () => {
    const api = createCRUDApi('roles')

    await api.list({ page: 1 })
    expect(requestMock).toHaveBeenLastCalledWith({
      url: '/roles',
      method: 'get',
      params: { page: 1 },
    })

    await api.detail(9)
    expect(requestMock).toHaveBeenLastCalledWith({
      url: '/roles/9',
      method: 'get',
    })

    await api.create({ name: 'a' })
    expect(requestMock).toHaveBeenLastCalledWith({
      url: '/roles',
      method: 'post',
      data: { name: 'a' },
    })

    await api.update(9, { name: 'b' })
    expect(requestMock).toHaveBeenLastCalledWith({
      url: '/roles/9',
      method: 'put',
      data: { name: 'b' },
    })

    await api.delete(9)
    expect(requestMock).toHaveBeenLastCalledWith({
      url: '/roles/9',
      method: 'delete',
    })

    await api.batchDelete([1, 2])
    expect(requestMock).toHaveBeenLastCalledWith({
      url: '/roles/batch',
      method: 'delete',
      data: { ids: [1, 2] },
    })
  })
})
