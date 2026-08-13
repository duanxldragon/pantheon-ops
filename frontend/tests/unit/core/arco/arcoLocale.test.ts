import { describe, it, expect } from 'vitest'
import { resolveArcoLocale } from '../../../../src/core/arco/arcoLocale'

describe('resolveArcoLocale', () => {
  it('returns zh-CN locale for zh-CN', () => {
    const locale = resolveArcoLocale('zh-CN')
    expect(locale).toBeDefined()
    expect(typeof locale).toBe('object')
  })

  it('returns en-US locale for en-US', () => {
    const locale = resolveArcoLocale('en-US')
    expect(locale).toBeDefined()
  })

  it('falls back to zh-CN for unknown language', () => {
    const fallback = resolveArcoLocale('zh-CN')
    const unknown = resolveArcoLocale('xx-XX')
    expect(unknown).toBe(fallback)
  })

  it('falls back to zh-CN for null', () => {
    const fallback = resolveArcoLocale('zh-CN')
    expect(resolveArcoLocale(null)).toBe(fallback)
  })

  it('falls back to zh-CN for empty string', () => {
    const fallback = resolveArcoLocale('zh-CN')
    expect(resolveArcoLocale('')).toBe(fallback)
  })

  it('returns ja-JP locale for ja-JP', () => {
    const locale = resolveArcoLocale('ja-JP')
    expect(locale).toBeDefined()
  })
})
