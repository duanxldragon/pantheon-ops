import { act, cleanup, fireEvent, render, renderHook, screen } from '@testing-library/react';
import { createElement } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useTaskDetail } from '@/modules/business/filedist/useTaskDetail';
import { buildCreateTaskPayload } from '@/modules/business/filedist/createTaskPayload';
import { canRetryTask, canStartTask, normalizeScheduledAt } from '@/modules/business/filedist/taskState';
import FiledistDetail from '@/modules/business/filedist/FiledistDetail';
import FiledistList from '@/modules/business/filedist/FiledistList';
import { FILE_CHOICE_PAGE_SIZE, useFileChoices } from '@/modules/business/filedist/useFileChoices';
import type { DistributionTaskRow } from '@/modules/business/filedist/api';
import type { DeployTargetResolution } from '@/modules/business/deploy/api';

const api = vi.hoisted(() => ({ getTaskDetail: vi.fn(), getTaskTargets: vi.fn(), getTaskEvents: vi.fn(), getFileList: vi.fn(), getTaskList: vi.fn(), deleteFile: vi.fn(), deleteTask: vi.fn(), uploadFile: vi.fn(), createTask: vi.fn(), startTask: vi.fn(), cancelTask: vi.fn(), retryTask: vi.fn() }));
const notifications = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
const permissions = vi.hoisted(() => ({ allowed: true }));
vi.mock('@/modules/business/filedist/api', () => api);
vi.mock('@/hooks/usePermission', () => ({ usePermission: () => ({ hasPerm: () => permissions.allowed }) }));
vi.mock('@/api/request', () => ({ isForbiddenRequestError: (error: unknown) => error instanceof Error && 'kind' in error && error.kind === 'forbidden', isRequestError: (error: unknown) => error instanceof Error && 'kind' in error }));
vi.mock('react-router-dom', () => ({ useNavigate: () => vi.fn(), useParams: () => ({ id: '1' }), useLocation: () => ({ pathname: '/business/filedist/tasks' }) }));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('@/modules/business/bizscope/api', () => ({ getBizScopeOptions: vi.fn() }));
vi.mock('@/modules/business/cmdb/group/api', () => ({ getGroupList: vi.fn() }));
vi.mock('@/modules/business/deploy/task/DeployTargetSelector', () => ({ default: () => null }));
vi.mock('@/components/patterns/forms/PasswordInput', () => ({ default: () => null }));
vi.mock('@/components', () => ({
  PageContainer: ({ children }: { children: React.ReactNode }) => createElement('div', {}, children),
  PageLoading: () => createElement('div', {}, 'loading'),
  PageError: ({ description, onRetry }: { description?: React.ReactNode; onRetry?: () => void }) => createElement('div', {}, description || 'error', onRetry ? createElement('button', { onClick: onRetry }, 'retry-list') : null),
  PageEmpty: () => createElement('div', {}, 'empty'),
  AppTable: ({ data, columns }: { data?: { id: number; name?: string }[]; columns?: { render?: (value: unknown, row: unknown) => React.ReactNode }[] }) => createElement('div', {}, 'targets', ...((data || []).map((row) => createElement('div', { key: row.id }, createElement('span', {}, row.name), columns?.at(-1)?.render?.(undefined, row))))),
  AppModal: () => null,
  GovernanceSummaryBar: () => null,
  SearchToolbar: ({ keyword, onKeywordChange }: { keyword: string; onKeywordChange: (value: string) => void }) => createElement('input', { 'aria-label': 'keyword', value: keyword, onChange: (event: React.ChangeEvent<HTMLInputElement>) => onKeywordChange(event.target.value) }),
  SubmitBar: () => null,
  buildStandardPagination: (_: unknown, options: unknown) => options,
}));
vi.mock('@arco-design/web-react', () => {
  const Container = ({ children }: { children?: React.ReactNode }) => createElement('div', {}, children);
  const Tabs = Object.assign(Container, { TabPane: Container });
  return {
    Alert: ({ content }: { content?: React.ReactNode }) => createElement('div', {}, content), Card: Container, Descriptions: Container, Space: Container, Tag: Container, Progress: Container, Tabs,
    Form: Object.assign(Container, { Item: Container, useForm: () => [{ resetFields: vi.fn(), setFieldsValue: vi.fn(), validate: vi.fn() }] }),
    Input: Object.assign(Container, { TextArea: Container }), Select: Object.assign(Container, { Option: Container }), DatePicker: Container, Divider: Container, InputNumber: Container, Pagination: Container, Modal: () => null, Upload: Container,
    Button: ({ children, onClick, disabled }: { children: React.ReactNode; onClick: () => void; disabled?: boolean }) => createElement('button', { onClick, disabled }, children),
    Popconfirm: ({ children, onOk }: { children: React.ReactNode; onOk: () => void }) => createElement('div', {}, children, createElement('button', { onClick: onOk }, 'confirm-cancel')),
    Message: notifications,
  };
});
const task = (id = 1, status = 'running'): DistributionTaskRow => ({ id, name: `task-${id}`, fileId: 1, status, progress: 0, totalTargets: 1, successCount: 0, failedCount: 0 });
function deferred<T>() { let resolve!: (value: T) => void; let reject!: (error: unknown) => void; const promise = new Promise<T>((accept, decline) => { resolve = accept; reject = decline; }); return { promise, resolve, reject }; }
const advance = async (ms = 0) => { await act(async () => { await vi.advanceTimersByTimeAsync(ms); }); };

beforeEach(() => { vi.useFakeTimers(); vi.resetAllMocks(); permissions.allowed = true; api.getTaskDetail.mockResolvedValue(task()); api.getTaskTargets.mockResolvedValue({ items: [] }); api.getTaskEvents.mockResolvedValue([]); api.getFileList.mockResolvedValue({ items: [], total: 0 }); api.getTaskList.mockResolvedValue({ items: [], total: 0 }); });
afterEach(() => { cleanup(); vi.useRealTimers(); });

describe('FileDist detail request lifetime', () => {
  it('does not restart polling while a sibling GET remains unsettled after early failure', async () => {
    const slow = deferred<{ items: [] }>();
    api.getTaskDetail.mockRejectedValueOnce(new Error('offline'));
    api.getTaskTargets.mockReturnValueOnce(slow.promise);
    const view = renderHook(() => useTaskDetail(1));
    await advance(); await advance(10000);
    expect(api.getTaskDetail).toHaveBeenCalledTimes(1);
    await act(async () => { slow.resolve({ items: [] }); await slow.promise; });
    expect(view.result.current.error).toBeTruthy();
    await advance(5000);
    expect(api.getTaskDetail).toHaveBeenCalledTimes(2);
  });
  it('drops old task content immediately and ignores an aborted late response', async () => {
    const old = deferred<DistributionTaskRow>();
    api.getTaskDetail.mockReturnValueOnce(old.promise).mockResolvedValueOnce(task(2, 'completed'));
    const view = renderHook(({ id }) => useTaskDetail(id), { initialProps: { id: 1 } });
    await advance();
    const signal = api.getTaskDetail.mock.calls[0][1] as AbortSignal;
    view.rerender({ id: 2 });
    expect(signal.aborted).toBe(true);
    expect(view.result.current.task).toBeNull();
    await advance();
    expect(view.result.current.task?.id).toBe(2);
    await act(async () => { old.resolve(task(1)); await old.promise; });
    expect(view.result.current.task?.id).toBe(2);
  });
  it('never overlaps progress polls and stops after terminal status', async () => {
    const slow = deferred<DistributionTaskRow>();
    api.getTaskDetail.mockResolvedValueOnce(task()).mockReturnValueOnce(slow.promise);
    const view = renderHook(() => useTaskDetail(1));
    await advance(); await advance(5000);
    expect(api.getTaskDetail).toHaveBeenCalledTimes(2);
    await advance(15000);
    expect(api.getTaskDetail).toHaveBeenCalledTimes(2);
    await act(async () => { slow.resolve(task(1, 'completed')); await slow.promise; });
    expect(view.result.current.task?.status).toBe('completed');
    await advance(15000);
    expect(api.getTaskDetail).toHaveBeenCalledTimes(2);
  });
  it('retains progress but surfaces a background failure and its recovery', async () => {
    const denied = new Error('permission denied');
    api.getTaskDetail.mockResolvedValueOnce(task()).mockRejectedValueOnce(denied).mockResolvedValueOnce(task(1, 'completed'));
    const view = renderHook(() => useTaskDetail(1));
    await advance(); await advance(5000);
    expect(view.result.current.task?.id).toBe(1);
    expect(view.result.current.refreshError).toBe(denied);
    await advance(5000);
    expect(view.result.current.refreshError).toBeNull();
    expect(view.result.current.task?.status).toBe('completed');
  });
  it('cancels outstanding requests and future polls on unmount', async () => {
    const pending = deferred<DistributionTaskRow>();
    api.getTaskDetail.mockReturnValueOnce(pending.promise);
    const view = renderHook(() => useTaskDetail(1)); await advance();
    const signal = api.getTaskDetail.mock.calls[0][1] as AbortSignal;
    view.unmount(); expect(signal.aborted).toBe(true);
    pending.resolve(task()); await advance(20000);
    expect(api.getTaskDetail).toHaveBeenCalledTimes(1);
  });
  it.each([0, -1, NaN, 1.5])('rejects malformed task identity %s before HTTP', async (id) => {
    const view = renderHook(() => useTaskDetail(id)); await advance();
    expect(view.result.current.loading).toBe(false);
    expect(view.result.current.error).toBeTruthy();
    expect(api.getTaskDetail).not.toHaveBeenCalled();
  });
});

describe('FileDist detail action mutual exclusion', () => {
  it.each([
    { kind: 'not-found', status: 404, key: 'business.filedist.taskMissing' },
    { kind: 'forbidden', status: 403, key: 'business.filedist.createForbidden' },
    { kind: 'server', status: 503, key: 'common.loadFailedDesc' },
  ])('classifies $status without mixing forbidden, missing and generic errors', async ({ kind, status, key }) => {
    api.getTaskDetail.mockRejectedValueOnce(Object.assign(new Error('failure'), { kind, status }));
    render(createElement(FiledistDetail)); await advance();
    expect(screen.getByText(key)).toBeTruthy();
    for (const other of ['business.filedist.taskMissing', 'business.filedist.createForbidden', 'common.loadFailedDesc']) {
      if (other !== key) expect(screen.queryByText(other)).toBeNull();
    }
  });
  it('does not report task missing for an initial connectivity failure', async () => {
    api.getTaskDetail.mockRejectedValueOnce(new Error('offline'));
    render(createElement(FiledistDetail)); await advance();
    expect(screen.queryByText('business.filedist.taskMissing')).toBeNull();
    expect(screen.getByText('common.loadFailedDesc')).toBeTruthy();
  });
  it('submits cancel only once while its request is unsettled', async () => {
    const pending = deferred<unknown>();
    api.cancelTask.mockReturnValue(pending.promise);
    render(createElement(FiledistDetail)); await advance();
    fireEvent.click(screen.getByText('confirm-cancel'));
    fireEvent.click(screen.getByText('confirm-cancel'));
    expect(api.cancelTask).toHaveBeenCalledTimes(1);
    await act(async () => { pending.resolve({}); await pending.promise; });
    expect(notifications.success).toHaveBeenCalledTimes(1);
  });
  it('does not offer control actions without update permission', async () => {
    permissions.allowed = false;
    render(createElement(FiledistDetail)); await advance();
    expect(screen.queryByText('confirm-cancel')).toBeNull();
    expect(screen.queryByText('business.filedist.start')).toBeNull();
    expect(api.cancelTask).not.toHaveBeenCalled();
  });
});

describe('FileDist intent and schedule', () => {
  const resolution: DeployTargetResolution = {
    selection: { businessScopeId: 7, hostIds: [11, 12], groupIds: [], includeScope: false, manual: [], excludedIds: [] },
    scopeVersion: 'scope-v1',
    hosts: [{ id: 11, hostname: 'a', ip: '10.0.0.11', stateVersion: 1 }, { id: 12, hostname: 'b', ip: '10.0.0.12', stateVersion: 2 }],
    unavailableCount: 0, ineligibleCount: 0, duplicateCount: 0, excludedCount: 0,
  } as DeployTargetResolution;
  const fingerprintA = `SHA256:${'A'.repeat(43)}`;
  const fingerprintB = `SHA256:${'B'.repeat(43)}`;
  const values = { name: ' release ', fileId: 1, businessScopeId: 7, username: ' ops ', targetPath: ' /opt/app ', hostFingerprints: { 11: fingerprintA, 12: fingerprintB, 99: fingerprintA }, password: ' secret ', privateKey: ' old-key ', maxRetries: 3 };
  it('uses only resolved identities/endpoints, per-host fingerprints and the selected auth mode', () => {
    const payload = buildCreateTaskPayload(values, resolution, 'password', undefined);
    expect(payload.targets.map((target) => [target.hostId, target.host, target.hostFingerprint])).toEqual([[11, '10.0.0.11', fingerprintA], [12, '10.0.0.12', fingerprintB]]);
    expect(payload.targets.every((target) => target.password === ' secret ' && !('privateKey' in target))).toBe(true);
    expect(payload).not.toHaveProperty('hostFingerprints');
    expect(payload.name).toBe('release');
    const keyPayload = buildCreateTaskPayload(values, resolution, 'privateKey', undefined);
    expect(keyPayload.targets.every((target) => target.privateKey === 'old-key' && !('password' in target))).toBe(true);
  });
  it('rejects stale scope, rejected targets, duplicate IDs and missing per-host fingerprints', () => {
    expect(() => buildCreateTaskPayload({ ...values, businessScopeId: 8 }, resolution, 'password', undefined)).toThrow('selectionRequired');
    expect(() => buildCreateTaskPayload(values, { ...resolution, unavailableCount: 1 }, 'password', undefined)).toThrow('selectionRequired');
    expect(() => buildCreateTaskPayload(values, { ...resolution, hosts: [resolution.hosts[0], resolution.hosts[0]] }, 'password', undefined)).toThrow('selectionRequired');
    expect(() => buildCreateTaskPayload({ ...values, hostFingerprints: { 11: fingerprintA } }, resolution, 'password', undefined)).toThrow('invalidFingerprint');
  });
  it('keeps explicit schedules and refuses malformed dates rather than running immediately', () => {
    expect(normalizeScheduledAt(undefined)).toBeUndefined();
    expect(normalizeScheduledAt('2026-11-20T03:00:00Z')).toBe('2026-11-20T03:00:00.000Z');
    expect(() => normalizeScheduledAt('invalid')).toThrow('invalidSchedule');
    expect(canStartTask({ ...task(1, 'pending'), scheduledAt: '2026-11-20T03:00:00Z' }, Date.parse('2026-11-20T02:59:59Z'))).toBe(false);
    expect(canStartTask({ ...task(1, 'pending'), scheduledAt: 'invalid' })).toBe(false);
    expect(canStartTask(task(1, 'cancelled'))).toBe(false);
    expect(canRetryTask(task(1, 'partial'))).toBe(true);
    expect(canRetryTask(task(1, 'cancelled'))).toBe(false);
  });
});

describe('FileDist bounded file picker', () => {
  const file = (id: number) => ({ id, storageId: 1, name: `file-${id}`, size: 1 });
  it('reaches files beyond the first 100 through bounded paging and remote search', async () => {
    api.getFileList.mockImplementation(async ({ page, keyword }) => ({ items: [file(keyword ? 125 : (page - 1) * FILE_CHOICE_PAGE_SIZE + 1)], total: keyword ? 1 : 125, page, pageSize: FILE_CHOICE_PAGE_SIZE }));
    const view = renderHook(() => useFileChoices(true)); await advance(250);
    expect(view.result.current.items[0].id).toBe(1);
    act(() => view.result.current.setPage(6));
    expect(view.result.current.items).toEqual([]);
    await advance(250);
    expect(view.result.current.items[0].id).toBe(101);
    expect(api.getFileList).toHaveBeenLastCalledWith({ keyword: '', page: 6, pageSize: 20 }, expect.any(AbortSignal));
    act(() => view.result.current.search('file-125')); await advance(250);
    expect(view.result.current.items[0].id).toBe(125);
    expect(view.result.current.total).toBe(1);
    expect(api.getFileList).toHaveBeenLastCalledWith({ keyword: 'file-125', page: 1, pageSize: 20 }, expect.any(AbortSignal));
  });
  it('cancels superseded search and prevents late rows or totals from returning', async () => {
    const old = deferred<{ items: ReturnType<typeof file>[]; total: number; page: number; pageSize: number }>();
    api.getFileList.mockReturnValueOnce(old.promise).mockResolvedValueOnce({ items: [file(125)], total: 1, page: 1, pageSize: 20 });
    const view = renderHook(() => useFileChoices(true)); await advance(250);
    const signal = api.getFileList.mock.calls[0][1] as AbortSignal;
    act(() => view.result.current.search('file-125'));
    expect(signal.aborted).toBe(true);
    await advance(250);
    await act(async () => { old.resolve({ items: [file(1)], total: 125, page: 1, pageSize: 20 }); await old.promise; });
    expect(view.result.current.items[0].id).toBe(125);
    expect(view.result.current.total).toBe(1);
  });
  it('clears prior options immediately when the create dialog reopens', async () => {
    const fresh = deferred<{ items: ReturnType<typeof file>[]; total: number; page: number; pageSize: number }>();
    api.getFileList.mockResolvedValueOnce({ items: [file(1)], total: 125, page: 1, pageSize: 20 }).mockReturnValueOnce(fresh.promise);
    const view = renderHook(({ enabled }) => useFileChoices(enabled), { initialProps: { enabled: true } }); await advance(250);
    expect(view.result.current.items[0].id).toBe(1);
    view.rerender({ enabled: false }); view.rerender({ enabled: true });
    expect(view.result.current.items).toEqual([]);
    expect(view.result.current.total).toBe(0);
    expect(view.result.current.loading).toBe(true);
    await advance(250);
    expect(api.getFileList).toHaveBeenCalledTimes(2);
  });
});

describe('FileDist list current-query refresh lifetime', () => {
  it.each([
    { kind: 'file', late: 'success' }, { kind: 'file', late: 'error' },
    { kind: 'task', late: 'success' }, { kind: 'task', late: 'error' },
  ])('ignores superseded manual $kind refresh $late', async ({ kind, late }) => {
    const list = kind === 'file' ? api.getFileList : api.getTaskList;
    const pending = deferred<{ items: { id: number; name: string; status: string }[]; total: number }>();
    list.mockRejectedValueOnce(new Error('initial failure')).mockReturnValueOnce(pending.promise).mockResolvedValueOnce({ items: [{ id: 2, name: 'current-query', status: 'completed' }], total: 1 });
    render(createElement(FiledistList)); await advance();
    fireEvent.click(screen.getByText('retry-list')); await advance();
    const signal = list.mock.calls[1][1] as AbortSignal | undefined;
    fireEvent.change(screen.getByLabelText('keyword'), { target: { value: 'new-query' } }); await advance();
    expect(screen.getByText('current-query')).toBeTruthy();
    await act(async () => {
      if (late === 'success') pending.resolve({ items: [{ id: 1, name: 'stale-query', status: 'completed' }], total: 200 });
      else pending.reject(new Error('old query failed'));
      await pending.promise.catch(() => undefined);
    });
    expect(screen.queryByText('stale-query')).toBeNull();
    expect(screen.queryByText('business.filedist.refreshFailed')).toBeNull();
    expect(screen.getByText('current-query')).toBeTruthy();
    expect(signal?.aborted).toBe(true);
  });
  it('shares the same in-flight task GET between mutation refresh and active polling', async () => {
    const slow = deferred<{ items: DistributionTaskRow[]; total: number }>();
    api.getTaskList.mockResolvedValueOnce({ items: [task(1), task(2, 'completed')], total: 2 }).mockReturnValueOnce(slow.promise).mockResolvedValue({ items: [task(1, 'completed')], total: 1 });
    api.deleteTask.mockResolvedValue({});
    render(createElement(FiledistList)); await advance();
    fireEvent.click(screen.getByText('confirm-cancel')); await advance();
    expect(api.getTaskList).toHaveBeenCalledTimes(2);
    await advance(15000);
    expect(api.getTaskList).toHaveBeenCalledTimes(2);
    await act(async () => { slow.resolve({ items: [task(1, 'completed')], total: 1 }); await slow.promise; });
    await advance(15000);
    expect(api.getTaskList).toHaveBeenCalledTimes(2);
  });
  it('refreshes the latest query when a mutation from an older render finishes', async () => {
    const deletion = deferred<unknown>();
    api.getTaskList.mockResolvedValue({ items: [task(1, 'completed')], total: 1 });
    api.deleteTask.mockReturnValueOnce(deletion.promise);
    render(createElement(FiledistList)); await advance();
    fireEvent.click(screen.getByText('confirm-cancel'));
    fireEvent.change(screen.getByLabelText('keyword'), { target: { value: 'new-query' } }); await advance();
    await act(async () => { deletion.resolve({}); await deletion.promise; }); await advance();
    expect(api.getTaskList).toHaveBeenCalledTimes(3);
    expect(api.getTaskList).toHaveBeenLastCalledWith({ keyword: 'new-query', status: '', page: 1, pageSize: 20 }, expect.any(AbortSignal));
  });
  it('serializes a fresh mutation read behind a GET that started before deletion', async () => {
    const stalePoll = deferred<{ items: DistributionTaskRow[]; total: number }>();
    api.getTaskList.mockResolvedValueOnce({ items: [task(1), task(2, 'completed')], total: 2 }).mockReturnValueOnce(stalePoll.promise).mockResolvedValueOnce({ items: [task(1, 'completed')], total: 1 });
    api.deleteTask.mockResolvedValue({});
    render(createElement(FiledistList)); await advance(); await advance(5000);
    fireEvent.click(screen.getByText('confirm-cancel')); await advance();
    expect(api.getTaskList).toHaveBeenCalledTimes(2);
    await act(async () => { stalePoll.resolve({ items: [task(1), task(2, 'completed')], total: 2 }); await stalePoll.promise; }); await advance();
    expect(api.getTaskList).toHaveBeenCalledTimes(3);
    expect(screen.queryByText('task-2')).toBeNull();
    expect(screen.getByText('task-1')).toBeTruthy();
    await advance(15000);
    expect(api.getTaskList).toHaveBeenCalledTimes(3);
  });
});
