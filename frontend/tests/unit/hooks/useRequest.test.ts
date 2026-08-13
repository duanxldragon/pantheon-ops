import { describe, it, expect, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useRequest } from '../../../src/hooks/useRequest'

describe('useRequest', () => {
  it('manual=false initializes loading as true', () => {
    const service = vi.fn().mockResolvedValue('result')
    const { result } = renderHook(() => useRequest(service))
    expect(result.current.loading).toBe(true)
    expect(service).not.toHaveBeenCalled()
  })

  it('manual=true does not auto-run', () => {
    const service = vi.fn().mockResolvedValue('x')
    const { result } = renderHook(() => useRequest(service, { manual: true }))
    expect(result.current.loading).toBe(false)
    expect(service).not.toHaveBeenCalled()
  })

  it('run() triggers service and returns data', async () => {
    const service = vi.fn().mockResolvedValue(42)
    const { result } = renderHook(() => useRequest(service, { manual: true }))
    let value: number | undefined
    await act(async () => { value = await result.current.run() })
    expect(value).toBe(42)
    expect(result.current.data).toBe(42)
  })

  it('sets error on failure', async () => {
    const err = new Error('fail')
    const service = vi.fn().mockRejectedValue(err)
    const { result } = renderHook(() => useRequest(service, { manual: true }))
    await act(async () => { await result.current.run() })
    expect(result.current.error).toBe(err)
    expect(result.current.data).toBeUndefined()
  })

  it('runAsync throws on failure', async () => {
    const service = vi.fn().mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useRequest(service, { manual: true }))
    await expect(act(() => result.current.runAsync())).rejects.toThrow('boom')
  })

  it('mutate updates data directly', async () => {
    const service = vi.fn().mockResolvedValue(1)
    const { result } = renderHook(() => useRequest(service, { manual: true }))
    act(() => result.current.mutate(99))
    expect(result.current.data).toBe(99)
  })

  it('mutate accepts updater function', async () => {
    const service = vi.fn().mockResolvedValue(10)
    const { result } = renderHook(() => useRequest(service, { manual: true }))
    act(() => result.current.mutate(10))
    act(() => result.current.mutate((prev) => (prev ?? 0) + 5))
    expect(result.current.data).toBe(15)
  })

  it('reset clears data and error', async () => {
    const service = vi.fn().mockResolvedValue('x')
    const { result } = renderHook(() => useRequest(service, { manual: true }))
    await act(async () => { await result.current.run() })
    act(() => result.current.reset())
    expect(result.current.data).toBeUndefined()
    expect(result.current.error).toBeNull()
  })

  it('refresh() re-runs the service', async () => {
    const service = vi.fn().mockResolvedValue('v1')
    const { result } = renderHook(() => useRequest(service, { manual: true }))
    await act(async () => { await result.current.run() })
    service.mockResolvedValue('v2')
    await act(async () => { await result.current.refresh() })
    expect(result.current.data).toBe('v2')
  })

  it('calls onSuccess callback', async () => {
    const onSuccess = vi.fn()
    const service = vi.fn().mockResolvedValue('ok')
    const { result } = renderHook(() => useRequest(service, { manual: true, onSuccess }))
    await act(async () => { await result.current.run() })
    expect(onSuccess).toHaveBeenCalledWith('ok', [])
  })

  it('calls onError callback', async () => {
    const onError = vi.fn()
    const err = new Error('e')
    const service = vi.fn().mockRejectedValue(err)
    const { result } = renderHook(() => useRequest(service, { manual: true, onError }))
    await act(async () => { await result.current.run() })
    expect(onError).toHaveBeenCalledWith(err, [])
  })
})
