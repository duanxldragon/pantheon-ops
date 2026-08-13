import { describe, it, expect } from 'vitest'
import { shouldPublishRefreshForVersionChange } from '../../../../src/core/refresh/versioning'

describe('shouldPublishRefreshForVersionChange', () => {
  it('returns true when version increases', () => {
    expect(shouldPublishRefreshForVersionChange(1, 2)).toBe(true)
  })

  it('returns false when version is the same', () => {
    expect(shouldPublishRefreshForVersionChange(2, 2)).toBe(false)
  })

  it('returns false when version decreases', () => {
    expect(shouldPublishRefreshForVersionChange(3, 2)).toBe(false)
  })

  it('returns false when previousVersion is undefined', () => {
    expect(shouldPublishRefreshForVersionChange(undefined, 1)).toBe(false)
  })
})
