import type { ReactNode } from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { RequestError } from '../../../../src/api/request';
import LogExplorer from '../../../../src/modules/business/observability/LogExplorer';

const provider = vi.hoisted(() => ({ query: vi.fn(), identities: new Set<() => void>() }));
vi.mock('@duanxldragon/pantheon-base-ui', () => ({ useAuthStore: { subscribe: (listener: () => void) => { provider.identities.add(listener); return () => provider.identities.delete(listener); } } }));
vi.mock('../../../../src/modules/business/observability/QueryWorkspaceHistory', () => ({ default: ({onBeforeRestore,onRestore}:{onBeforeRestore:()=>void;onRestore:(value:unknown)=>void}) => <><button onClick={()=>{onBeforeRestore();onRestore({datasourceId:1,recipe:null,time:{mode:'relative',duration:'15m'},limit:500,expressionUnavailable:true});}}>restore-fixture</button><button onClick={()=>{onBeforeRestore();onRestore({datasourceId:1,recipe:null,time:{mode:'absolute',start:'2026-11-11T00:00:00.123Z',end:'2026-11-11T00:15:00.456Z'},limit:500,expressionUnavailable:true});}}>restore-precise-fixture</button></> }));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('../../../../src/modules/business/observability/api', () => ({
 getDatasourceList: async () => ({ items: [{ id: 1, name: 'Loki', type: 'loki' }, { id: 2, name: 'Other Loki', type: 'loki' }] }), queryLogs: provider.query,
}));
vi.mock('../../../../src/modules/business/observability/LogResultPanels', () => ({ default: ({ rows, logs }: { rows: {id: string; message: string}[]; logs: ReactNode }) => <div>{logs}{rows.map((row) => <p key={row.id}>{row.message}</p>)}</div> }));
vi.mock('@arco-design/web-react', () => {
 const Box = ({ children }: { children?: ReactNode }) => <div>{children}</div>;
 const Select = Object.assign(({ value, onChange, children, options }: { value?: number; onChange: (id: number) => void; children?: ReactNode; options?: {value: number; label: string}[] }) => <select value={value} onChange={(event) => onChange(Number(event.target.value))}>{children}{options?.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select>, { Option: ({ value, children }: {value: number; children?: ReactNode}) => <option value={value}>{children}</option> });
 return { Card: Box, Space: Box, Tag: Box, Alert: ({ content }: { content: ReactNode }) => <div role="status">{content}</div>, Select,
  Input: ({ onChange, ...props }: Omit<React.ComponentProps<'input'>, 'onChange'> & {onChange: (value: string) => void}) => <input {...props} onChange={(event) => onChange(event.target.value)} />,
  Button: ({ children, disabled, onClick }: { children?: ReactNode; disabled?: boolean; onClick?: () => void }) => <button disabled={disabled} onClick={onClick}>{children}</button>, Message: { warning: vi.fn() } };
});
vi.mock('../../../../src/components', () => {
 const Box = ({ children }: { children?: ReactNode }) => <div>{children}</div>;
 return { PageContainer: Box, GovernanceSummaryBar: () => null, AppTable: ({columns, data}: {columns: {dataIndex: string; render?: (value: unknown, row: {id: string; labels: Record<string,string>}) => ReactNode}[]; data: {id: string; labels: Record<string,string>}[]}) => <section data-testid="label-rows">{data.map((row) => <div key={row.id}>{columns.find((column) => column.dataIndex === 'labels')?.render?.(undefined, row)}</div>)}</section>, buildStandardPagination: () => ({}),
  PageLoading: () => <p>loading</p>, PageEmpty: ({ description }: {description: string}) => <p>{description}</p>, PageError: () => <p>query-error</p>,
  SearchToolbar: ({ keyword, onKeywordChange, inlineFilters, extra }: {keyword: string; onKeywordChange: (value: string) => void; inlineFilters: ReactNode; extra: ReactNode}) => <div><input aria-label="expression" value={keyword} onChange={(event) => onKeywordChange(event.target.value)} />{inlineFilters}{extra}</div> };
});
const response = (message = 'first-result') => ({ historyUnavailable: false, historyId: 41, result: { status: 'success', data: { resultType: 'streams', result: [{ stream: { app: 'one' }, values: [['1794009600000000000', message]] }] } } });
async function query() { render(<LogExplorer />); await screen.findByText('Loki'); fireEvent.click(screen.getByRole('button', {name: 'common.search'})); }
afterEach(() => { cleanup(); vi.useRealTimers(); provider.query.mockReset(); });

describe('Loki completed-result authority', () => {
 it('bounds per-row label DOM growth and explicitly marks omitted labels', async () => {
  const value=response(); value.result.data.result[0].stream=Object.fromEntries(Array.from({length:500},(_,i)=>[`label${i}`,`value-${i}`])) as {app:string};
  provider.query.mockResolvedValue(value); await query(); await screen.findByText('first-result');
  expect(screen.getAllByText(/^label\d+=value-/).length).toBeLessThanOrEqual(6);
  expect(screen.getByText('business.observability.logs.labelsOmitted')).toBeTruthy();
 });
 it('bounds label preview text and indicates that values have been shortened', async () => {
  const value=response(); value.result.data.result[0].stream={app:'wide-value-'.repeat(1000)};
  provider.query.mockResolvedValue(value); await query(); await screen.findByText('first-result');
  expect(screen.getByTestId('label-rows').textContent!.length).toBeLessThan(200);
  expect(screen.getByText('business.observability.logs.labelsShortened')).toBeTruthy();
 });
 it('marks the previous result stale when expression or absolute bounds change', async () => {
  provider.query.mockResolvedValue(response()); await query(); await screen.findByText('first-result');
  fireEvent.change(screen.getByLabelText('expression'), {target: {value: '{app="other"}'}});
  expect(screen.getByText('business.observability.logs.inputsChanged')).toBeTruthy();
  expect(screen.getByText('first-result')).toBeTruthy();
 });
 it('marks provider warnings without displaying their potentially sensitive text', async () => {
  provider.query.mockResolvedValue({...response(), result: {...response().result, warnings: ['token=private-warning']}}); await query(); await screen.findByText('first-result');
  expect(screen.getByText('business.observability.logs.providerWarnings')).toBeTruthy();
  expect(screen.queryByText(/private-warning/)).toBeNull();
 });
 it('does not present an upstream failure as empty or successful log data', async () => {
  provider.query.mockResolvedValue({...response('unsafe-failed-data'), result: {...response('unsafe-failed-data').result, status: 'error', error: 'token=secret'}}); await query();
  await screen.findByText('query-error'); expect(screen.queryByText('unsafe-failed-data')).toBeNull();
 });
 it('rejects an unexpected result type rather than calling it empty logs', async () => {
  provider.query.mockResolvedValue({historyUnavailable:false, result:{status: 'success', data: {resultType: 'matrix', result: []}}}); await query();
  await screen.findByText('query-error'); expect(screen.queryByText('business.observability.logs.empty')).toBeNull();
 });
 it('selects the newest rows across streams before applying the display limit', async () => {
  const old=Array.from({length:500},(_,i)=>['1794009600000000000', `older-${i}`]);
  provider.query.mockResolvedValue({historyUnavailable:false, result:{status: 'success', data: {resultType: 'streams', result: [{stream: {}, values: old}, {stream: {}, values: [['1794009720000000000','newest-across-streams']]}]}}});
  await query(); await waitFor(() => expect(screen.getByText('newest-across-streams')).toBeTruthy());
  expect(screen.getByText('business.observability.logs.rowsOmitted')).toBeTruthy();
  expect(screen.queryByText('older-499')).toBeNull();
 });
});


const followButton = () => screen.getByRole('button', { name: 'business.observability.logs.follow' });
const stopButton = () => screen.getByRole('button', { name: 'business.observability.logs.stop' });
async function startFollow() {
 render(<LogExplorer />); await screen.findByText('Loki');
 vi.useFakeTimers(); fireEvent.click(followButton());
 await act(async () => { await Promise.resolve(); });
}

describe('Loki bounded polling lifecycle', () => {
 it('uses the rolling UTC window, serializes polls and stops between requests', async () => {
  provider.query.mockResolvedValue(response()); await startFollow();
  const params = provider.query.mock.calls[0][1];
  expect(params.time).toEqual({mode:'relative',duration:'15m'});
  expect(params.historyMode).toBe('coalesced'); expect(params.historyId).toBeUndefined();
  expect(params.limit).toBe(500);
  expect(screen.getByText('business.observability.logs.following')).toBeTruthy();
  expect(screen.getByText('business.observability.logs.followSnapshot')).toBeTruthy();
  await act(async () => { await vi.advanceTimersByTimeAsync(4999); });
  expect(provider.query).toHaveBeenCalledTimes(1);
  await act(async () => { await vi.advanceTimersByTimeAsync(1); });
  expect(provider.query).toHaveBeenCalledTimes(2);
  expect(provider.query.mock.calls[1][1].historyId).toBe(41);
  expect(stopButton().hasAttribute('disabled')).toBe(false);
  fireEvent.click(stopButton());
  await act(async () => { await vi.advanceTimersByTimeAsync(20_000); });
  expect(provider.query).toHaveBeenCalledTimes(2);
  expect(screen.getByText('first-result')).toBeTruthy();
  expect(screen.queryByText('business.observability.logs.following')).toBeNull();
 });
 it('never overlaps a pending request and discards its result after stop', async () => {
  let resolve!: (value: ReturnType<typeof response>) => void;
  provider.query.mockReturnValue(new Promise((done) => { resolve = done; })); await startFollow();
  await act(async () => { await vi.advanceTimersByTimeAsync(20_000); });
  expect(provider.query).toHaveBeenCalledTimes(1);
  const signal = provider.query.mock.calls[0][2] as AbortSignal;
  fireEvent.click(stopButton()); expect(signal.aborted).toBe(true);
  await act(async () => { resolve(response('late-after-stop')); await Promise.resolve(); });
  expect(screen.queryByText('late-after-stop')).toBeNull();
  await act(async () => { await vi.advanceTimersByTimeAsync(20_000); });
  expect(provider.query).toHaveBeenCalledTimes(1);
 });
 it('ends polling on edits, marks the rolling snapshot stale and requires an explicit run', async () => {
  provider.query.mockResolvedValue(response()); await startFollow();
  fireEvent.change(screen.getByLabelText('expression'), { target: { value: '{app="edited"}' } });
  await act(async () => { await vi.advanceTimersByTimeAsync(10_000); });
  expect(provider.query).toHaveBeenCalledTimes(1);
  expect(screen.getByText('business.observability.logs.inputsChanged')).toBeTruthy();
  expect(screen.queryByText('business.observability.logs.following')).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'common.search' }));
  await act(async () => { await Promise.resolve(); });
  expect(provider.query).toHaveBeenCalledTimes(2);
  expect(provider.query.mock.calls[1][1].rawQuery).toBe('{app="edited"}');
  expect(screen.queryByText('business.observability.logs.followSnapshot')).toBeNull();
 });
 it.each(['error', 'denied', 'invalid'])('stops and clears retained output on %s', async (kind) => {
  provider.query.mockResolvedValueOnce(response());
  if (kind === 'invalid') provider.query.mockResolvedValueOnce({historyUnavailable:false,result:{status: 'success', data: {resultType: 'matrix', result: []}}});
  else provider.query.mockRejectedValueOnce(kind === 'denied' ? new RequestError({ kind: 'forbidden', status: 403 }) : new Error('provider down'));
  await startFollow();
  await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
  expect(screen.queryByText('first-result')).toBeNull();
  expect(screen.queryByText('business.observability.logs.following')).toBeNull();
  await act(async () => { await vi.advanceTimersByTimeAsync(20_000); });
  expect(provider.query).toHaveBeenCalledTimes(2);
  expect(screen.getByText(kind === 'denied' ? 'business.observability.logs.forbidden' : 'query-error')).toBeTruthy();
 });
 it('clears the pending timer on unmount', async () => {
  provider.query.mockResolvedValue(response()); await startFollow(); cleanup();
  await act(async () => { await vi.advanceTimersByTimeAsync(20_000); });
  expect(provider.query).toHaveBeenCalledTimes(1);
 });
 it('clears the owned pending request on unmount', async () => {
  provider.query.mockReturnValue(new Promise(() => {})); await startFollow();
  const signal = provider.query.mock.calls[0][2] as AbortSignal; cleanup(); expect(signal.aborted).toBe(true);
 });
});


describe('Loki follow source ownership', () => {
 it('preserves restored UTC precision when raw text is explicitly re-entered',async()=>{
  provider.query.mockResolvedValue(response());await query();await screen.findByText('first-result');fireEvent.click(screen.getByRole('button',{name:'restore-precise-fixture'}));
  expect(provider.query).toHaveBeenCalledOnce();fireEvent.change(screen.getByLabelText('expression'),{target:{value:'{app="restored"}'}});fireEvent.click(screen.getByRole('button',{name:'common.search'}));
  await waitFor(()=>expect(provider.query).toHaveBeenCalledTimes(2));expect(provider.query.mock.calls[1][1].time).toEqual({mode:'absolute',start:'2026-11-11T00:00:00.123Z',end:'2026-11-11T00:15:00.456Z'});
 });
 it('stops follow, clears output and raw text on restore without automatically querying', async()=>{
  provider.query.mockResolvedValue(response());await startFollow();fireEvent.click(screen.getByRole('button',{name:'restore-fixture'}));
  expect((screen.getByLabelText('expression') as HTMLInputElement).value).toBe('');expect(screen.queryByText('first-result')).toBeNull();expect(screen.getByText('business.observability.workspace.restoredRun')).toBeTruthy();
  await act(async()=>{await vi.advanceTimersByTimeAsync(20000);});expect(provider.query).toHaveBeenCalledOnce();
 });
 it('retains successful provider output when history recording is unavailable', async()=>{
  provider.query.mockResolvedValue({...response(),historyUnavailable:true,historyId:undefined});await startFollow();expect(screen.getByText('first-result')).toBeTruthy();expect(screen.getByText('business.observability.workspace.historyUnavailable')).toBeTruthy();
  await act(async()=>{await vi.advanceTimersByTimeAsync(5000);});expect(provider.query).toHaveBeenCalledTimes(2);expect(provider.query.mock.calls[1][1].historyId).toBeUndefined();
 });
 it('aborts a pending query on an auth update and rejects an ignored-abort response', async()=>{
  let resolve!:(value:ReturnType<typeof response>)=>void;provider.query.mockReturnValue(new Promise((done)=>{resolve=done;}));await query();const signal=provider.query.mock.calls[0][2] as AbortSignal;
  act(()=>provider.identities.forEach((listener)=>listener()));expect(signal.aborted).toBe(true);
  await act(async()=>{resolve(response('late-previous-session'));});expect(screen.queryByText('late-previous-session')).toBeNull();expect((screen.getByLabelText('expression') as HTMLInputElement).value).toBe('');
 });
 it('discards a delayed denial from the previous source and stops following', async () => {
  let reject!: (error: unknown) => void;
  provider.query.mockReturnValueOnce(new Promise((_done, fail) => { reject = fail; }));
  await startFollow();
  const signal = provider.query.mock.calls[0][2] as AbortSignal;
  fireEvent.change(screen.getAllByRole('combobox')[0], { target: { value: '2' } });
  expect(signal.aborted).toBe(true);
  await act(async () => { reject(new RequestError({ kind: 'forbidden', status: 403 })); await Promise.resolve(); });
  expect(screen.queryByText('business.observability.logs.forbidden')).toBeNull();
  expect(screen.queryByText('business.observability.logs.following')).toBeNull();
  await act(async () => { await vi.advanceTimersByTimeAsync(10_000); });
  expect(provider.query).toHaveBeenCalledTimes(1);
  provider.query.mockResolvedValueOnce(response('new-source-result'));
  fireEvent.click(screen.getByRole('button', { name: 'common.search' }));
  await act(async () => { await Promise.resolve(); });
  expect(provider.query.mock.calls[1][0]).toBe(2);
  expect(screen.getByText('new-source-result')).toBeTruthy();
 });
 it('does not restart a poll when unrelated state or translator identities change', async () => {
  provider.query.mockResolvedValue(response()); await startFollow();
  await act(async () => { await vi.advanceTimersByTimeAsync(20_000); });
  expect(provider.query).toHaveBeenCalledTimes(5);
 });
});
