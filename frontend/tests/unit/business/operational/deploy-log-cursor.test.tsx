import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useDeployTaskLogs } from '../../../../src/modules/business/deploy/task/useDeployTaskLogs';
import { getDeployTaskLogs, type DeployTaskLogRow } from '../../../../src/modules/business/deploy/api';

vi.mock('../../../../src/modules/business/deploy/api', () => ({ getDeployTaskLogs: vi.fn() }));
vi.mock('../../../../src/api/request', () => ({ isForbiddenRequestError: (error: { code?: number }) => error?.code === 403 }));
const row = (sequence: number, content = `line-${sequence}`): DeployTaskLogRow => ({ sequence, content, taskHostId: 1, taskStepId: 1, stream: 'stdout', createdAt: '2026-11-08T00:00:00Z' });
const flush = () => act(async () => { await Promise.resolve(); });
afterEach(() => { vi.resetAllMocks(); vi.useRealTimers(); });

describe('task-bound log cursor and retained window', () => {
  it('hides old task rows immediately and requests a new task from zero', async () => {
    let release: ((value: { items: DeployTaskLogRow[]; total: number }) => void) | undefined;
    vi.mocked(getDeployTaskLogs).mockResolvedValueOnce({ items: [row(41, 'old task')], total: 1 })
      .mockImplementationOnce(() => new Promise((resolve) => { release = resolve; }));
    const { result, rerender } = renderHook(({ id }) => useDeployTaskLogs(id, false), { initialProps: { id: 7 } });
    await flush();
    expect(result.current.rows[0].content).toBe('old task');
    rerender({ id: 8 });
    expect(result.current.rows).toEqual([]);
    expect(result.current.state).toBe('loading');
    expect(getDeployTaskLogs).toHaveBeenLastCalledWith(8, 0, expect.any(AbortSignal), expect.any(Object));
    await act(async () => { release?.({ items: [row(1, 'new task')], total: 1 }); });
    expect(result.current.rows.map((item) => item.content)).toEqual(['new task']);
  });

  it('ignores late responses from a departed task and aborts its request', async () => {
    let release: ((value: { items: DeployTaskLogRow[]; total: number }) => void) | undefined;
    vi.mocked(getDeployTaskLogs).mockImplementationOnce(() => new Promise((resolve) => { release = resolve; }))
      .mockResolvedValueOnce({ items: [row(1, 'current task')], total: 1 });
    const { result, rerender } = renderHook(({ id }) => useDeployTaskLogs(id, false), { initialProps: { id: 7 } });
    const oldSignal = vi.mocked(getDeployTaskLogs).mock.calls[0][2];
    rerender({ id: 8 });
    await flush();
    await act(async () => { release?.({ items: [row(99, 'late old task')], total: 1 }); });
    expect(oldSignal?.aborted).toBe(true);
    expect(result.current.rows.map((item) => item.content)).toEqual(['current task']);
  });

  it('does not call a sparse high sequence a truncated buffer', async () => {
    vi.mocked(getDeployTaskLogs).mockResolvedValue({ items: [row(10001)], total: 1 });
    const { result } = renderHook(() => useDeployTaskLogs(7, false));
    await flush();
    expect(result.current.rows).toHaveLength(1);
    expect(result.current.truncated).toBe(false);
  });

  it('retains only 1000 rows across actual 500-row pages and resets that budget for a new task', async () => {
    vi.useFakeTimers();
    const page = (start: number) => ({ items: Array.from({ length: 500 }, (_, i) => row(start+i)), total: 1500 });
    vi.mocked(getDeployTaskLogs).mockResolvedValueOnce(page(1)).mockResolvedValueOnce(page(501))
      .mockResolvedValueOnce(page(1001)).mockResolvedValueOnce({ items: [], total: 1500 })
      .mockResolvedValueOnce({ items: [row(1, 'next task')], total: 1 });
    const { result, rerender } = renderHook(({ id }) => useDeployTaskLogs(id, false), { initialProps: { id: 7 } });
    await flush();
    await act(async () => { await vi.advanceTimersByTimeAsync(300); });
    expect(result.current.rows).toHaveLength(1000);
    expect(result.current.rows[0].sequence).toBe(501);
    expect(result.current.truncated).toBe(true);
    rerender({ id: 8 });
    await flush();
    expect(result.current.rows.map((item) => item.content)).toEqual(['next task']);
    expect(result.current.truncated).toBe(false);
  });

  it('retains the cursor on active-state change but clears it after access denial', async () => {
    vi.mocked(getDeployTaskLogs).mockResolvedValueOnce({ items: [row(9)], total: 1 })
      .mockRejectedValueOnce({ code: 403 }).mockResolvedValueOnce({ items: [row(1, 'recovered')], total: 1 });
    const { result, rerender } = renderHook(({ active }) => useDeployTaskLogs(7, active), { initialProps: { active: true } });
    await flush();
    rerender({ active: false });
    await flush();
    expect(getDeployTaskLogs).toHaveBeenNthCalledWith(2, 7, 9, expect.any(AbortSignal), expect.any(Object));
    expect(result.current.rows).toEqual([]);
    expect(result.current.state).toBe('forbidden');
    act(() => result.current.reload());
    await flush();
    expect(getDeployTaskLogs).toHaveBeenLastCalledWith(7, 0, expect.any(AbortSignal), expect.any(Object));
    expect(result.current.rows.map((item) => item.content)).toEqual(['recovered']);
  });
});


describe('provider source-bound cursor', () => {
  it('clears the window and starts from zero on host, attempt or step changes', async () => {
    vi.mocked(getDeployTaskLogs).mockResolvedValue({ items: [row(99)], total: 1 });
    const { result, rerender } = renderHook(({ hostId, attemptNo, stepId }: { hostId: number; attemptNo: number; stepId?: number }) =>
      useDeployTaskLogs(7, false, { hostId, attemptNo, stepId }), { initialProps: { hostId: 1, attemptNo: 1, stepId: undefined } as { hostId: number; attemptNo: number; stepId?: number } });
    await flush();
    for (const props of [{ hostId: 2, attemptNo: 1 }, { hostId: 2, attemptNo: 2 }, { hostId: 2, attemptNo: 2, stepId: 20 }]) {
      rerender(props); expect(result.current.rows).toEqual([]); await flush();
      expect(getDeployTaskLogs).toHaveBeenLastCalledWith(7, 0, expect.any(AbortSignal), expect.objectContaining(props));
    }
  });
  it('ignores a departed source response within the same task', async () => {
    let release: ((value: { items: DeployTaskLogRow[]; total: number }) => void) | undefined;
    vi.mocked(getDeployTaskLogs).mockImplementationOnce(() => new Promise((resolve) => { release = resolve; }))
      .mockResolvedValueOnce({ items: [row(3, 'selected source')], total: 1 });
    const { result, rerender } = renderHook(({ hostId }) => useDeployTaskLogs(7, false, { hostId }), { initialProps: { hostId: 1 } });
    rerender({ hostId: 2 }); await flush();
    await act(async () => { release?.({ items: [row(99, 'old source')], total: 1 }); });
    expect(result.current.rows.map((item) => item.content)).toEqual(['selected source']);
  });
  it('re-reads the first error from zero without duplicating or advancing to a later error', async () => {
    vi.mocked(getDeployTaskLogs).mockResolvedValue({ items: [row(4, 'first error')], total: 1 });
    const { result } = renderHook(() => useDeployTaskLogs(7, false, { hostId: 1, stepId: 9, firstError: true }));
    await flush(); act(() => result.current.reload()); await flush();
    expect(getDeployTaskLogs).toHaveBeenLastCalledWith(7, 0, expect.any(AbortSignal), expect.objectContaining({ firstError: true, stepId: 9 }));
    expect(result.current.rows.map((item) => item.content)).toEqual(['first error']);
    vi.mocked(getDeployTaskLogs).mockRejectedValueOnce(new Error('offline'));
    act(() => result.current.reload()); await flush();
    expect(result.current.state).toBe('stale');
  });
});
