import { describe, it, expect } from 'vitest'
import { checkPermission } from '../../../../src/core/permissions/checkPermission'

describe('checkPermission', () => {
  it('returns true when permission is empty', () => {
    expect(checkPermission(null, '')).toBe(true)
  })

  it('returns false when userInfo is null', () => {
    expect(checkPermission(null, 'user:read')).toBe(false)
  })

  it('returns true when user has admin role', () => {
    expect(checkPermission({ roles: ['admin'], perms: [] }, 'user:read')).toBe(true)
  })

  it('returns true when user has the exact permission', () => {
    expect(checkPermission({ roles: [], perms: ['user:read'] }, 'user:read')).toBe(true)
  })

  it('returns false when user lacks the permission', () => {
    expect(checkPermission({ roles: [], perms: ['user:write'] }, 'user:read')).toBe(false)
  })

  it('returns false when perms is undefined', () => {
    expect(checkPermission({ roles: [] }, 'user:read')).toBe(false)
  })
})
