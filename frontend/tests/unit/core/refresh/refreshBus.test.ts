import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { publishRefresh, subscribeRefresh, useRefreshSubscription } from '../../../../src/core/refresh/refreshBus'
import type { PantheonRefreshTopic } from '../../../../src/core/refresh/refreshBus'

describe('publishRefresh / subscribeRefresh', () => {
  it('delivers a single topic to a subscriber', () => {
    const received: PantheonRefreshTopic[] = []
    const unsub = subscribeRefresh('system:user:changed', (p) => received.push(p.topic))
    publishRefresh('system:user:changed')
    unsub()
    expect(received).toEqual(['system:user:changed'])
  })

  it('does not deliver unrelated topics', () => {
    const received: PantheonRefreshTopic[] = []
    const unsub = subscribeRefresh('system:role:changed', (p) => received.push(p.topic))
    publishRefresh('system:user:changed')
    unsub()
    expect(received).toHaveLength(0)
  })

  it('delivers multiple topics to matching subscribers', () => {
    const received: PantheonRefreshTopic[] = []
    const unsub = subscribeRefresh(
      ['system:user:changed', 'system:role:changed'],
      (p) => received.push(p.topic),
    )
    publishRefresh(['system:user:changed', 'system:role:changed'])
    unsub()
    expect(received).toContain('system:user:changed')
    expect(received).toContain('system:role:changed')
  })

  it('unsubscribe stops delivery', () => {
    const received: PantheonRefreshTopic[] = []
    const unsub = subscribeRefresh('system:menu:changed', (p) => received.push(p.topic))
    unsub()
    publishRefresh('system:menu:changed')
    expect(received).toHaveLength(0)
  })

  it('payload includes topic and at timestamp', () => {
    let payload: { topic: string; at: number } | null = null
    const unsub = subscribeRefresh('system:setting:changed', (p) => { payload = p })
    publishRefresh('system:setting:changed', 'test')
    unsub()
    expect(payload).not.toBeNull()
    expect(payload!.topic).toBe('system:setting:changed')
    expect(typeof payload!.at).toBe('number')
  })
})

describe('useRefreshSubscription', () => {
  it('calls handler when matching topic is published', () => {
    const received: PantheonRefreshTopic[] = []
    renderHook(() =>
      useRefreshSubscription('system:dept:changed', (p) => received.push(p.topic)),
    )
    act(() => publishRefresh('system:dept:changed'))
    expect(received).toContain('system:dept:changed')
  })

  it('does not call handler for non-matching topic', () => {
    const received: PantheonRefreshTopic[] = []
    renderHook(() =>
      useRefreshSubscription('system:post:changed', (p) => received.push(p.topic)),
    )
    act(() => publishRefresh('system:user:changed'))
    expect(received).toHaveLength(0)
  })
})
