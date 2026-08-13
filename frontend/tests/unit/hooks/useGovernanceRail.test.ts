import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useGovernanceRail } from '../../../src/hooks/useGovernanceRail'

describe('useGovernanceRail', () => {
  it('starts collapsed by default', () => {
    const { result } = renderHook(() => useGovernanceRail())
    expect(result.current.expanded).toBe(false)
  })

  it('respects defaultExpanded option', () => {
    const { result } = renderHook(() => useGovernanceRail({ defaultExpanded: true }))
    expect(result.current.expanded).toBe(true)
  })

  it('open() expands', () => {
    const { result } = renderHook(() => useGovernanceRail())
    act(() => result.current.open())
    expect(result.current.expanded).toBe(true)
  })

  it('close() collapses', () => {
    const { result } = renderHook(() => useGovernanceRail({ defaultExpanded: true }))
    act(() => result.current.close())
    expect(result.current.expanded).toBe(false)
  })

  it('toggle() flips state', () => {
    const { result } = renderHook(() => useGovernanceRail())
    act(() => result.current.toggle())
    expect(result.current.expanded).toBe(true)
    act(() => result.current.toggle())
    expect(result.current.expanded).toBe(false)
  })

  it('open() does nothing when disabled', () => {
    const { result } = renderHook(() => useGovernanceRail({ enabled: false }))
    act(() => result.current.open())
    expect(result.current.expanded).toBe(false)
  })

  it('toggle() does nothing when disabled', () => {
    const { result } = renderHook(() => useGovernanceRail({ enabled: false }))
    act(() => result.current.toggle())
    expect(result.current.expanded).toBe(false)
  })

  it('expanded is always false when disabled', () => {
    const { result } = renderHook(() => useGovernanceRail({ defaultExpanded: true, enabled: false }))
    expect(result.current.expanded).toBe(false)
  })
})
