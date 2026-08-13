import { describe, it, expect, beforeEach } from 'vitest'
import {
  hasExplicitLanguagePreference,
  setExplicitLanguagePreference,
  syncDefaultLanguagePreference,
  clearExplicitLanguagePreference,
  getDefaultLanguagePreference,
  LANGUAGE_STORAGE_KEY,
  LANGUAGE_EXPLICIT_STORAGE_KEY,
} from '../../../../src/core/settings/languagePreference'

describe('languagePreference', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('hasExplicitLanguagePreference returns false when not set', () => {
    expect(hasExplicitLanguagePreference()).toBe(false)
  })

  it('setExplicitLanguagePreference stores language and explicit flag', () => {
    setExplicitLanguagePreference('en-US')
    expect(localStorage.getItem(LANGUAGE_STORAGE_KEY)).toBe('en-US')
    expect(localStorage.getItem(LANGUAGE_EXPLICIT_STORAGE_KEY)).toBe('1')
    expect(hasExplicitLanguagePreference()).toBe(true)
  })

  it('syncDefaultLanguagePreference sets language when no explicit preference', () => {
    syncDefaultLanguagePreference('ja-JP')
    expect(localStorage.getItem(LANGUAGE_STORAGE_KEY)).toBe('ja-JP')
  })

  it('syncDefaultLanguagePreference does not override explicit preference', () => {
    setExplicitLanguagePreference('en-US')
    syncDefaultLanguagePreference('ja-JP')
    expect(localStorage.getItem(LANGUAGE_STORAGE_KEY)).toBe('en-US')
  })

  it('clearExplicitLanguagePreference removes explicit flag and resets to default', () => {
    setExplicitLanguagePreference('en-US')
    clearExplicitLanguagePreference('zh-CN')
    expect(localStorage.getItem(LANGUAGE_EXPLICIT_STORAGE_KEY)).toBeNull()
    expect(localStorage.getItem(LANGUAGE_STORAGE_KEY)).toBe('zh-CN')
    expect(hasExplicitLanguagePreference()).toBe(false)
  })

  it('getDefaultLanguagePreference returns zh-CN when nothing stored', () => {
    expect(getDefaultLanguagePreference()).toBe('zh-CN')
  })

  it('syncDefaultLanguagePreference returns the language', () => {
    const result = syncDefaultLanguagePreference('ko-KR')
    expect(result).toBe('ko-KR')
  })

  it('clearExplicitLanguagePreference returns the default language', () => {
    const result = clearExplicitLanguagePreference('fr-FR')
    expect(result).toBe('fr-FR')
  })

  it('hasExplicitLanguagePreference returns false in SSR (no document)', () => {
    const doc = globalThis.document
    // @ts-expect-error simulate SSR
    delete globalThis.document
    expect(hasExplicitLanguagePreference()).toBe(false)
    globalThis.document = doc
  })

  it('setExplicitLanguagePreference is a no-op in SSR', () => {
    const doc = globalThis.document
    // @ts-expect-error simulate SSR
    delete globalThis.document
    setExplicitLanguagePreference('en-US')
    globalThis.document = doc
    expect(localStorage.getItem(LANGUAGE_STORAGE_KEY)).toBeNull()
  })

  it('syncDefaultLanguagePreference returns language in SSR', () => {
    const doc = globalThis.document
    // @ts-expect-error simulate SSR
    delete globalThis.document
    const result = syncDefaultLanguagePreference('ko-KR')
    globalThis.document = doc
    expect(result).toBe('ko-KR')
  })

  it('clearExplicitLanguagePreference returns default in SSR', () => {
    const doc = globalThis.document
    // @ts-expect-error simulate SSR
    delete globalThis.document
    const result = clearExplicitLanguagePreference('fr-FR')
    globalThis.document = doc
    expect(result).toBe('fr-FR')
  })
})
