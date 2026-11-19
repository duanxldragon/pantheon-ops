import { act, cleanup, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useCatalogOptions } from '../../../../src/modules/business/service/useCatalogOptions';

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((accept, decline) => { resolve = accept; reject = decline; });
  return { promise, resolve, reject };
}

async function startLookup() {
  await act(async () => { await vi.advanceTimersByTimeAsync(250); });
}

beforeEach(() => { vi.useFakeTimers(); });
afterEach(() => { cleanup(); vi.useRealTimers(); });

describe('catalog option activation lifetime', () => {
  it('hides previous rows immediately when an unchanged lookup is re-enabled', async () => {
    const fresh = deferred<string[]>();
    const lookup = vi.fn<(signal: AbortSignal) => Promise<string[]>>()
      .mockResolvedValueOnce(['old-target']).mockReturnValueOnce(fresh.promise);
    const { result, rerender } = renderHook(({ enabled }) => useCatalogOptions(enabled, lookup), { initialProps: { enabled: true } });
    await startLookup();
    expect(result.current.rows).toEqual(['old-target']);
    rerender({ enabled: false });
    rerender({ enabled: true });
    expect(result.current.rows).toEqual([]);
    expect(result.current.loading).toBe(true);
    expect(result.current.error).toBeNull();
    expect(lookup).toHaveBeenCalledTimes(1);
    await startLookup();
    expect(result.current.rows).toEqual([]);
    expect(result.current.loading).toBe(true);
    await act(async () => { fresh.resolve(['fresh-target']); await fresh.promise; });
    expect(result.current.rows).toEqual(['fresh-target']);
    expect(result.current.loading).toBe(false);
  });

  it('hides an earlier denial while the reopened activation awaits fresh authorization', async () => {
    const denial = new Error('denied');
    const lookup = vi.fn<(signal: AbortSignal) => Promise<string[]>>()
      .mockRejectedValueOnce(denial).mockResolvedValueOnce(['authorized-target']);
    const { result, rerender } = renderHook(({ enabled }) => useCatalogOptions(enabled, lookup), { initialProps: { enabled: true } });
    await startLookup();
    expect(result.current.error).toBe(denial);
    rerender({ enabled: false });
    rerender({ enabled: true });
    expect(result.current.error).toBeNull();
    expect(result.current.loading).toBe(true);
    await startLookup();
    expect(result.current.rows).toEqual(['authorized-target']);
    expect(result.current.error).toBeNull();
  });

  it('does not query or show pending state while initially disabled', async () => {
    const lookup = vi.fn<(signal: AbortSignal) => Promise<string[]>>().mockResolvedValue([]);
    const { result } = renderHook(() => useCatalogOptions(false, lookup));
    await startLookup();
    expect(lookup).not.toHaveBeenCalled();
    expect(result.current).toMatchObject({ rows: [], error: null, loading: false });
  });

  it('cancels an activation closed before its debounce fires', async () => {
    const lookup = vi.fn<(signal: AbortSignal) => Promise<string[]>>().mockResolvedValue([]);
    const { result, rerender } = renderHook(({ enabled }) => useCatalogOptions(enabled, lookup), { initialProps: { enabled: true } });
    rerender({ enabled: false });
    await startLookup();
    expect(lookup).not.toHaveBeenCalled();
    expect(result.current).toMatchObject({ rows: [], error: null, loading: false });
  });

  it.each(['resolve', 'reject'] as const)('ignores a late %s from the aborted activation after reopening', async (completion) => {
    const old = deferred<string[]>();
    const fresh = deferred<string[]>();
    const lookup = vi.fn<(signal: AbortSignal) => Promise<string[]>>()
      .mockReturnValueOnce(old.promise).mockReturnValueOnce(fresh.promise);
    const { result, rerender } = renderHook(({ enabled }) => useCatalogOptions(enabled, lookup), { initialProps: { enabled: true } });
    await startLookup();
    const oldSignal = lookup.mock.calls[0][0];
    rerender({ enabled: false });
    expect(oldSignal.aborted).toBe(true);
    rerender({ enabled: true });
    await startLookup();
    await act(async () => {
      if (completion === 'resolve') old.resolve(['obsolete-target']);
      else old.reject(new Error('obsolete-denial'));
      await old.promise.catch(() => undefined);
    });
    expect(result.current).toMatchObject({ rows: [], error: null, loading: true });
    await act(async () => { fresh.resolve(['current-target']); await fresh.promise; });
    expect(result.current).toMatchObject({ rows: ['current-target'], error: null, loading: false });
  });

  it('hides results immediately when the target lookup changes and preserves current errors', async () => {
    const fresh = deferred<string[]>();
    const oldLookup = vi.fn<(signal: AbortSignal) => Promise<string[]>>().mockResolvedValue(['old-namespace']);
    const freshLookup = vi.fn<(signal: AbortSignal) => Promise<string[]>>().mockReturnValue(fresh.promise);
    const { result, rerender } = renderHook(({ lookup }) => useCatalogOptions(true, lookup), { initialProps: { lookup: oldLookup } });
    await startLookup();
    expect(result.current.rows).toEqual(['old-namespace']);
    rerender({ lookup: freshLookup });
    expect(oldLookup.mock.calls[0][0].aborted).toBe(true);
    expect(result.current).toMatchObject({ rows: [], error: null, loading: true });
    await startLookup();
    const error = new Error('current-provider-failure');
    await act(async () => { fresh.reject(error); await fresh.promise.catch(() => undefined); });
    expect(result.current).toMatchObject({ rows: [], error, loading: false });
  });
});
