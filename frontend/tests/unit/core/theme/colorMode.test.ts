import { describe, it, expect, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import {
  normalizeColorMode,
  getStoredColorMode,
  applyColorMode,
  initializePantheonColorMode,
  usePantheonColorMode,
} from '../../../../src/core/theme/colorMode'

describe('normalizeColorMode', () => {
  it('returns light for undefined', () => expect(normalizeColorMode(undefined)).toBe('light'))
  it('returns light for null', () => expect(normalizeColorMode(null)).toBe('light'))
  it('returns light for unknown value', () => expect(normalizeColorMode('blue')).toBe('light'))
  it('returns dark for dark', () => expect(normalizeColorMode('dark')).toBe('dark'))
  it('returns light for light', () => expect(normalizeColorMode('light')).toBe('light'))
})

describe('getStoredColorMode', () => {
  beforeEach(() => localStorage.clear())

  it('returns a valid mode when nothing stored', () => {
    expect(['light', 'dark']).toContain(getStoredColorMode())
  })

  it('returns stored dark value', () => {
    localStorage.setItem('pantheon_color_mode', 'dark')
    expect(getStoredColorMode()).toBe('dark')
  })

  it('ignores invalid stored value', () => {
    localStorage.setItem('pantheon_color_mode', 'invalid')
    expect(['light', 'dark']).toContain(getStoredColorMode())
  })
})

describe('applyColorMode', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-color-mode')
    document.body.removeAttribute('arco-theme')
  })

  it('sets data-color-mode on documentElement', () => {
    applyColorMode('dark')
    expect(document.documentElement.dataset.colorMode).toBe('dark')
  })

  it('sets arco-theme on body', () => {
    applyColorMode('light')
    expect(document.body.getAttribute('arco-theme')).toBe('light')
  })

  it('persists to localStorage when persist=true', () => {
    applyColorMode('dark', true)
    expect(localStorage.getItem('pantheon_color_mode')).toBe('dark')
  })

  it('does not persist when persist=false', () => {
    applyColorMode('dark', false)
    expect(localStorage.getItem('pantheon_color_mode')).toBeNull()
  })
})

describe('initializePantheonColorMode', () => {
  beforeEach(() => localStorage.clear())

  it('returns a valid color mode', () => {
    const result = initializePantheonColorMode()
    expect(['light', 'dark']).toContain(result)
  })
})

describe('usePantheonColorMode', () => {
  beforeEach(() => localStorage.clear())

  it('returns current color mode', () => {
    const { result } = renderHook(() => usePantheonColorMode())
    expect(['light', 'dark']).toContain(result.current.colorMode)
  })

  it('setColorMode updates the mode', () => {
    const { result } = renderHook(() => usePantheonColorMode())
    act(() => result.current.setColorMode('dark'))
    expect(result.current.colorMode).toBe('dark')
  })

  it('toggleColorMode flips the mode', () => {
    const { result } = renderHook(() => usePantheonColorMode())
    const initial = result.current.colorMode
    act(() => result.current.toggleColorMode())
    expect(result.current.colorMode).toBe(initial === 'dark' ? 'light' : 'dark')
  })
})
