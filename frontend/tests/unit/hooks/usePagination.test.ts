import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { usePagination } from '../../../src/hooks/usePagination'

describe('usePagination', () => {
  it('returns defaults', () => {
    const { result } = renderHook(() => usePagination())
    expect(result.current.current).toBe(1)
    expect(result.current.pageSize).toBe(10)
    expect(result.current.total).toBe(0)
  })

  it('respects initial options', () => {
    const { result } = renderHook(() => usePagination({ defaultCurrent: 3, defaultPageSize: 20, total: 100 }))
    expect(result.current.current).toBe(3)
    expect(result.current.pageSize).toBe(20)
    expect(result.current.total).toBe(100)
  })

  it('setCurrent updates page', () => {
    const { result } = renderHook(() => usePagination())
    act(() => result.current.setCurrent(5))
    expect(result.current.current).toBe(5)
  })

  it('setPageSize resets to page 1', () => {
    const { result } = renderHook(() => usePagination({ defaultCurrent: 3 }))
    act(() => result.current.setPageSize(50))
    expect(result.current.pageSize).toBe(50)
    expect(result.current.current).toBe(1)
  })

  it('reset restores defaults', () => {
    const { result } = renderHook(() => usePagination({ defaultCurrent: 2, defaultPageSize: 20 }))
    act(() => result.current.setCurrent(5))
    act(() => result.current.reset())
    expect(result.current.current).toBe(2)
    expect(result.current.pageSize).toBe(20)
  })

  it('paginationProps.onChange updates current', () => {
    const { result } = renderHook(() => usePagination())
    act(() => result.current.paginationProps.onChange?.(4, 10))
    expect(result.current.current).toBe(4)
  })
})
