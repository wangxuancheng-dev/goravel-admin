import { describe, expect, it } from 'vitest'
import { normalizeEntity, normalizeTreeList } from './normalize'

describe('normalizeEntity', () => {
  it('adds snake_case aliases and id from ID', () => {
    expect(normalizeEntity({ ID: 3, UserName: 'a', status: 1 })).toMatchObject({
      ID: 3,
      id: 3,
      UserName: 'a',
      user_name: 'a',
      status: 1,
    })
  })

  it('normalizes nested children', () => {
    const tree = normalizeTreeList([{ ID: 1, children: [{ ID: 2 }] }]) as Array<{
      ID: number
      id: number
      children: Array<{ ID: number; id: number }>
    }>
    expect(tree[0].id).toBe(1)
    expect(tree[0].children[0].id).toBe(2)
  })
})
