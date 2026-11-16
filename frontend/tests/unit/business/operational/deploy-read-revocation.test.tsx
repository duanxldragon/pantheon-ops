import { act, cleanup, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { getDeployTaskDetail, type DeployTaskRow } from '../../../../src/modules/business/deploy/api';
import { RequestError } from '../../../../src/api/request';
import { useDeployTaskSnapshot } from '../../../../src/modules/business/deploy/task/useDeployTaskSnapshot';
import { useDeployTaskStream } from '../../../../src/modules/business/deploy/task/useDeployTaskStream';

vi.mock('../../../../src/modules/business/deploy/api', () => ({ getDeployTaskDetail: vi.fn(), getDeployTaskEventStreamUrl: (id: number) => `/stream/${id}` }));
const task = (id: number) => ({ id, name: `task-${id}`, status: 'running', hosts: [] }) as unknown as DeployTaskRow;
const flush = () => act(async () => { await Promise.resolve(); });
class Source {
  static instances: Source[] = [];
  listeners: Record<string, () => void> = {};
  closed = false;
  onopen?: () => void;
  onerror?: () => void;
  constructor() { Source.instances.push(this); }
  addEventListener(name: string, callback: () => void) { this.listeners[name] = callback; }
  close() { this.closed = true; }
}
afterEach(() => { cleanup(); vi.resetAllMocks(); vi.unstubAllGlobals(); Source.instances = []; });
describe('task read revocation and generations', () => {
  it('clears retained task on silent forbidden and fences in-flight success after stream denial', async () => {
    vi.mocked(getDeployTaskDetail).mockResolvedValue(task(7));
    const { result } = renderHook(() => useDeployTaskSnapshot(7)); await flush();
    expect(result.current.task?.id).toBe(7);
    vi.mocked(getDeployTaskDetail).mockRejectedValueOnce(new RequestError({ kind: 'forbidden', status: 403, code: 403 }));
    await act(async () => { await result.current.loadData({ silent: true }); });
    expect(result.current.task).toBeNull(); expect(result.current.error).toBeInstanceOf(RequestError);
  });
  it('fences in-flight success and pending refreshes after stream denial', async () => {
    vi.mocked(getDeployTaskDetail).mockResolvedValue(task(7));
    const { result } = renderHook(() => useDeployTaskSnapshot(7)); await flush();
    let resolve: ((value: DeployTaskRow) => void) | undefined;
    vi.mocked(getDeployTaskDetail).mockImplementationOnce(() => new Promise((r) => { resolve = r; }));
    let pending: Promise<void>;
    act(() => { pending = result.current.loadData({ silent: true }); });
    await act(async () => { await result.current.loadData({ silent: true }); result.current.deny(); });
    await act(async () => { resolve?.(task(7)); await pending; });
    await act(async () => { await result.current.loadData({ silent: true }); });
    expect(result.current.task).toBeNull(); expect(result.current.error).toBeInstanceOf(RequestError);
    expect(getDeployTaskDetail).toHaveBeenCalledTimes(2);
  });
  it('marks ordinary silent failures stale and allows recoverable refresh', async () => {
    vi.mocked(getDeployTaskDetail).mockResolvedValue(task(7));
    const { result } = renderHook(() => useDeployTaskSnapshot(7)); await flush();
    vi.mocked(getDeployTaskDetail).mockRejectedValueOnce(new Error('offline'));
    await act(async () => { await result.current.loadData({ silent: true }); });
    expect(result.current.task?.id).toBe(7); expect(result.current.stale).toBe(true);
    await act(async () => { await result.current.loadData({ silent: true }); });
    expect(result.current.stale).toBe(false);
  });
  it('ignores old generation forbidden response and callback after task navigation', async () => {
    let reject: ((reason: unknown) => void) | undefined;
    vi.mocked(getDeployTaskDetail).mockImplementationOnce(() => new Promise((_, r) => { reject = r; })).mockResolvedValueOnce(task(8));
    const { result, rerender } = renderHook(({ id }) => useDeployTaskSnapshot(id), { initialProps: { id: 7 } });
    await flush();
    const oldDeny = result.current.deny;
    rerender({ id: 8 }); await flush();
    await act(async () => { reject?.({ status: 403 }); oldDeny(); });
    expect(result.current.task?.id).toBe(8); expect(result.current.error).toBeNull();
  });
  it('closes denied stream and ignores departed generation events', () => {
    vi.stubGlobal('EventSource', Source);
    const deny = vi.fn(); const refresh = vi.fn();
    const { rerender } = renderHook(({ id }) => useDeployTaskStream(id, true, refresh, deny), { initialProps: { id: 7 } });
    const old = Source.instances[0];
    rerender({ id: 8 });
    act(() => { old.listeners.forbidden(); old.listeners.task(); });
    expect(deny).not.toHaveBeenCalled(); expect(refresh).not.toHaveBeenCalled();
    const current = Source.instances[1];
    act(() => { current.listeners.forbidden(); current.listeners.task(); });
    expect(current.closed).toBe(true); expect(deny).toHaveBeenCalledOnce(); expect(refresh).not.toHaveBeenCalled();
  });
});


it('coalesces multiple in-flight refreshes into one trailing request without a busy loop', async () => {
  let resolveFirst: ((value: DeployTaskRow) => void) | undefined;
  let resolveTrailing: ((value: DeployTaskRow) => void) | undefined;
  vi.mocked(getDeployTaskDetail)
    .mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve; }))
    .mockImplementationOnce(() => new Promise((resolve) => { resolveTrailing = resolve; }));
  const { result } = renderHook(() => useDeployTaskSnapshot(7)); await flush();
  await act(async () => { await result.current.loadData({ silent: true }); await result.current.loadData({ silent: true }); });
  expect(getDeployTaskDetail).toHaveBeenCalledOnce();
  await act(async () => { resolveFirst?.(task(7)); });
  expect(result.current.task?.status).toBe('running');
  expect(getDeployTaskDetail).toHaveBeenCalledTimes(2);
  await act(async () => { resolveTrailing?.({ ...task(7), status: 'success' }); });
  await flush();
  expect(result.current.task?.status).toBe('success'); expect(result.current.stale).toBe(false);
  expect(getDeployTaskDetail).toHaveBeenCalledTimes(2);
});

it('clears pending refreshes on detail forbidden and prevents later loads in the denied generation', async () => {
  let reject: ((reason: unknown) => void) | undefined;
  vi.mocked(getDeployTaskDetail).mockImplementationOnce(() => new Promise((_, r) => { reject = r; }));
  const { result } = renderHook(() => useDeployTaskSnapshot(7)); await flush();
  await act(async () => { await result.current.loadData({ silent: true }); });
  await act(async () => { reject?.(new RequestError({ kind: 'forbidden', status: 403 })); });
  await act(async () => { await result.current.loadData({ silent: true }); });
  expect(result.current.task).toBeNull(); expect(result.current.error).toBeInstanceOf(RequestError);
  expect(getDeployTaskDetail).toHaveBeenCalledOnce();
});

it('does not run pending old-generation refreshes after navigation or from a departed callback', async () => {
  let resolve: ((value: DeployTaskRow) => void) | undefined;
  vi.mocked(getDeployTaskDetail).mockImplementationOnce(() => new Promise((r) => { resolve = r; })).mockResolvedValueOnce(task(8));
  const { result, rerender } = renderHook(({ id }) => useDeployTaskSnapshot(id), { initialProps: { id: 7 } }); await flush();
  const oldRefresh = result.current.loadData;
  await act(async () => { await oldRefresh({ silent: true }); });
  rerender({ id: 8 }); await flush();
  await act(async () => { resolve?.(task(7)); await oldRefresh({ silent: true }); });
  expect(result.current.task?.id).toBe(8); expect(result.current.error).toBeNull();
  expect(getDeployTaskDetail).toHaveBeenCalledTimes(2);
});

it('does not run pending refreshes or accept callbacks after disposal', async () => {
  let resolve: ((value: DeployTaskRow) => void) | undefined;
  vi.mocked(getDeployTaskDetail).mockImplementationOnce(() => new Promise((r) => { resolve = r; }));
  const { result, unmount } = renderHook(() => useDeployTaskSnapshot(7)); await flush();
  const oldRefresh = result.current.loadData;
  await act(async () => { await oldRefresh({ silent: true }); });
  unmount();
  await act(async () => { resolve?.(task(7)); await oldRefresh({ silent: true }); });
  expect(getDeployTaskDetail).toHaveBeenCalledOnce();
});
