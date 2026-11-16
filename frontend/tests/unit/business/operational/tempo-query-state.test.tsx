import type { ReactNode } from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { RequestError } from '../../../../src/api/request';
import TraceExplorer from '../../../../src/modules/business/observability/TraceExplorer';

const provider = vi.hoisted(() => ({ search: vi.fn(), detail: vi.fn(), sources: vi.fn(), warning: vi.fn(), error: vi.fn(), identities: new Set<() => void>() }));
vi.mock('@duanxldragon/pantheon-base-ui', () => ({ useAuthStore: { subscribe: (listener: () => void) => { provider.identities.add(listener); return () => provider.identities.delete(listener); } } }));
vi.mock('../../../../src/modules/business/observability/QueryWorkspaceHistory', () => ({ default: () => null }));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('../../../../src/modules/business/observability/api', () => ({
  getDatasourceList: provider.sources, searchTraces: provider.search, getTrace: provider.detail,
}));
vi.mock('@arco-design/web-react', () => {
  const Box = ({ children }: { children?: ReactNode }) => <div>{children}</div>;
  const Select = Object.assign(({ value, onChange, children, options, ...props }: { value?: number; onChange: (id: number) => void; children?: ReactNode; options?: { value: number; label: string }[]; 'aria-label'?: string }) => <select {...props} value={value} onChange={(event) => onChange(Number(event.target.value))}>{children}{options?.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select>, { Option: ({ value, children }: { value: number; children?: ReactNode }) => <option value={value}>{children}</option> });
  return { Card: Box, Space: Box, Tag: Box, Alert: ({ content }: { content: ReactNode }) => <div role="status">{content}</div>, Select,
    Input: ({ onChange, ...props }: Omit<React.ComponentProps<'input'>, 'onChange'> & { onChange: (value: string) => void }) => <input {...props} onChange={(event) => onChange(event.target.value)} />,
    Button: ({ children, disabled, onClick }: { children?: ReactNode; disabled?: boolean; onClick?: () => void }) => <button disabled={disabled} onClick={onClick}>{children}</button>, Message: { warning: provider.warning, error: provider.error } };
});
vi.mock('../../../../src/components', () => {
  const Box = ({ children }: { children?: ReactNode }) => <div>{children}</div>;
  return { PageContainer: Box, GovernanceSummaryBar: () => null, AppTable: ({ columns, data }: { columns: { fixed?: string; render?: (value: unknown, row: { traceID: string }) => ReactNode }[]; data: { traceID: string }[] }) => <section>{data.map((row) => <div key={row.traceID}><p>{row.traceID}</p>{columns.find((column) => column.fixed)?.render?.(undefined, row)}</div>)}</section>, buildStandardPagination: () => ({}),
    PageLoading: () => <p>loading</p>, PageEmpty: ({ description }: { description: string }) => <p>{description}</p>, PageError: () => <p>query-error</p>,
    AppDrawer: ({ visible, onCancel, children }: { visible: boolean; onCancel: () => void; children: ReactNode }) => visible ? <aside><button onClick={onCancel}>close-detail</button>{children}</aside> : null,
    SearchToolbar: ({ keyword, onKeywordChange, inlineFilters, extra }: { keyword: string; onKeywordChange: (value: string) => void; inlineFilters: ReactNode; extra: ReactNode }) => <div><input aria-label="expression" value={keyword} onChange={(event) => onKeywordChange(event.target.value)} />{inlineFilters}{extra}</div> };
});
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((yes) => { resolve = yes; });
  return { promise, resolve };
}
async function mount() {
  provider.sources.mockResolvedValue({ items: [{ id: 1, name: 'Tempo A', type: 'tempo' }, { id: 2, name: 'Tempo B', type: 'tempo' }] });
  render(<TraceExplorer />); await screen.findByText('Tempo A');
}
const run = () => fireEvent.click(screen.getByRole('button', { name: 'common.search' }));
const response = (traceID: string) => ({ historyUnavailable: false, result: { traces: [{ traceID }] } });
afterEach(() => { cleanup(); vi.resetAllMocks(); });
describe('Tempo request and source authority', () => {
  it('cancels search and rejects a late response when the provider ignores abort', async () => {
    const pending = deferred<ReturnType<typeof response>>(); provider.search.mockReturnValue(pending.promise);
    await mount(); run(); const signal = provider.search.mock.calls[0][2] as AbortSignal;
    fireEvent.click(screen.getByRole('button', { name: 'business.observability.logs.stop' })); expect(signal.aborted).toBe(true);
    await act(async () => pending.resolve(response('late-canceled')));
    expect(screen.queryByText('late-canceled')).toBeNull(); expect(provider.error).not.toHaveBeenCalled();
  });
  it('clears rows and details when changing source and ignores both old completions', async () => {
    provider.search.mockResolvedValueOnce(response('source-a'));
    const pendingDetail = deferred<Record<string, unknown>>(); provider.detail.mockReturnValue(pendingDetail.promise);
    await mount(); run(); await screen.findByText('source-a'); fireEvent.click(screen.getByRole('button', { name: 'common.detail' }));
    const detailSignal = provider.detail.mock.calls[0][2] as AbortSignal;
    const pendingSearch = deferred<ReturnType<typeof response>>(); provider.search.mockReturnValue(pendingSearch.promise); run();
    const searchSignal = provider.search.mock.calls[1][2] as AbortSignal;
    fireEvent.change(screen.getByRole('combobox', { name: 'business.observability.common.datasource' }), { target: { value: '2' } });
    expect(detailSignal.aborted).toBe(true); expect(searchSignal.aborted).toBe(true);
    await act(async () => { pendingDetail.resolve({ resource: 'old-detail' }); pendingSearch.resolve(response('old-search')); });
    expect(screen.queryByText('source-a')).toBeNull(); expect(screen.queryByText('old-search')).toBeNull(); expect(screen.queryByText(/old-detail/)).toBeNull();
    provider.search.mockResolvedValue(response('source-b')); run(); await screen.findByText('source-b'); expect(provider.search.mock.calls[2][0]).toBe(2);
  });
  it('aborts closed details and prevents old completion from contaminating another trace', async () => {
    provider.search.mockResolvedValue(response('trace-a'));
    const pending = deferred<Record<string, unknown>>(); provider.detail.mockReturnValueOnce(pending.promise);
    await mount(); run(); await screen.findByText('trace-a'); fireEvent.click(screen.getByRole('button', { name: 'common.detail' }));
    const signal = provider.detail.mock.calls[0][2] as AbortSignal;
    fireEvent.click(screen.getByRole('button', { name: 'close-detail' })); expect(signal.aborted).toBe(true);
    provider.detail.mockResolvedValue({ resource: 'fresh-detail' }); fireEvent.click(screen.getByRole('button', { name: 'common.detail' }));
    await screen.findByText(/fresh-detail/); await act(async () => pending.resolve({ resource: 'obsolete-detail' }));
    expect(screen.queryByText(/obsolete-detail/)).toBeNull(); expect(screen.getByText(/fresh-detail/)).toBeTruthy();
  });
  it('marks completed results stale and removes them after query denial', async () => {
    provider.search.mockResolvedValueOnce(response('authorized-trace')); await mount(); run(); await screen.findByText('authorized-trace');
    const input = screen.getByLabelText('business.observability.logs.startUTC') as HTMLInputElement;
    fireEvent.change(input, { target: { value: new Date(Date.parse(input.value + ':00Z') - 5 * 60000).toISOString().slice(0, 16) } });
    expect(screen.getByText('business.observability.traces.inputsChanged')).toBeTruthy();
    provider.search.mockRejectedValue(new RequestError({ kind: 'forbidden', status: 403 })); run();
    await screen.findByText('business.observability.traces.forbidden'); expect(screen.queryByText('authorized-trace')).toBeNull();
  });
  it('clears rows and closes details when detail authorization is revoked', async () => {
    provider.search.mockResolvedValue(response('revoked-trace')); provider.detail.mockRejectedValue(new RequestError({ kind: 'forbidden', status: 403 }));
    await mount(); run(); await screen.findByText('revoked-trace'); fireEvent.click(screen.getByRole('button', { name: 'common.detail' }));
    await screen.findByText('business.observability.traces.forbidden'); expect(screen.queryByText('revoked-trace')).toBeNull(); expect(screen.queryByRole('complementary')).toBeNull();
  });
  it('passes explicit UTC bounds and rejects invalid ranges before contacting Tempo', async () => {
    provider.search.mockResolvedValue({ historyUnavailable:false, result:{ traces: [] } }); await mount();
    fireEvent.change(screen.getByLabelText('business.observability.logs.startUTC'), { target: { value: '2026-11-10T00:00' } });
    fireEvent.change(screen.getByLabelText('business.observability.logs.endUTC'), { target: { value: '2026-11-10T00:15' } });
    run(); await waitFor(() => expect(provider.search).toHaveBeenCalledOnce());
    const params = provider.search.mock.calls[0][1]; expect(params.recipe.time.start).toMatch(/Z$/); expect(params.recipe.time.end).toMatch(/Z$/); expect(params.rawQuery).toBeUndefined();
    fireEvent.change(screen.getByLabelText('business.observability.logs.endUTC'), { target: { value: '2000-01-01T00:00' } }); run();
    expect(provider.search).toHaveBeenCalledOnce(); expect(provider.warning).toHaveBeenCalledWith('business.observability.logs.invalidRange');
  });
  it('aborts discovery and running requests on unmount', async () => {
    const pending = deferred<ReturnType<typeof response>>(); provider.search.mockReturnValue(pending.promise); await mount(); run();
    const signal = provider.search.mock.calls[0][2] as AbortSignal; const discoverySignal = provider.sources.mock.calls[0][0] as AbortSignal;
    cleanup(); expect(signal.aborted).toBe(true); expect(discoverySignal.aborted).toBe(true);
    await act(async () => pending.resolve(response('unmounted'))); expect(provider.error).not.toHaveBeenCalled();
  });
});
