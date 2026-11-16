import { act, fireEvent, render, renderHook, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import OperationalLogPanel from '../../../../src/modules/business/shared/OperationalLogPanel';
import { useDeployTaskStream } from '../../../../src/modules/business/deploy/task/useDeployTaskStream';

vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('../../../../src/modules/business/deploy/api', () => ({ getDeployTaskEventStreamUrl: (id: number) => `/tasks/${id}/events` }));

class FakeSource {
  static instances: FakeSource[] = [];
  onopen?: () => void;
  onerror?: () => void;
  onmessage?: (event: { data: string }) => void;
  onclose?: () => void;
  task?: () => void;
  closed = false;
  constructor(readonly url: string) { FakeSource.instances.push(this); }
  addEventListener(_type: string, handler: () => void) { this.task = handler; }
  close() { this.closed = true; }
}

afterEach(() => { vi.unstubAllGlobals(); FakeSource.instances = []; });

describe('business operational log lifecycle', () => {
  it('keeps the SSE instance and browser replay cursor across snapshot/callback updates', () => {
    vi.stubGlobal('EventSource', FakeSource);
    const refresh = vi.fn();
    const nextRefresh = vi.fn();
    const { rerender, unmount } = renderHook(({ callback }) => useDeployTaskStream(7, true, callback), { initialProps: { callback: refresh } });
    const source = FakeSource.instances[0];
    act(() => source.task?.());
    rerender({ callback: nextRefresh });
    act(() => source.task?.());
    expect(FakeSource.instances).toHaveLength(1);
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(nextRefresh).toHaveBeenCalledTimes(1);
    expect(source.closed).toBe(false);
    unmount();
    expect(source.closed).toBe(true);
  });

  it('freezes the visible snapshot while receiving, resumes latest data and hides paused data after denial', () => {
    const props = { title: 'task / host', chunks: [{ sequence: 1, content: 'first' }] };
    const { rerender } = render(<OperationalLogPanel {...props} />);
    fireEvent.click(screen.getByText('business.deploy.logs.pause'));
    rerender(<OperationalLogPanel {...props} chunks={[...props.chunks, { sequence: 2, content: 'next' }]} state="stale" />);
    expect(screen.queryByText('next')).toBeNull();
    expect(screen.getByText('first')).toBeTruthy();
    expect(screen.getByText('business.deploy.logs.stale')).toBeTruthy();
    fireEvent.click(screen.getByText('business.deploy.logs.resume'));
    expect(screen.getByText('next')).toBeTruthy();
    fireEvent.click(screen.getByText('business.deploy.logs.pause'));
    rerender(<OperationalLogPanel {...props} state="forbidden" />);
    expect(screen.queryByText('first')).toBeNull();
    expect(screen.queryByText('next')).toBeNull();
  });

  });
