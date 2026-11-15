import type { ReactNode } from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { RequestError } from '../../../../src/api/request';
import ServiceList from '../../../../src/modules/business/k8s/service/ServiceList';
const api = vi.hoisted(() => ({ clusters: vi.fn(), list: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn(), message: vi.fn() }));
vi.mock('react-i18next', () => { const t = (key: string) => key; return { useTranslation: () => ({ t }) }; });
vi.mock('../../../../src/hooks/usePermission', () => ({ usePermission: () => ({ hasPerm: () => true }) }));
vi.mock('../../../../src/modules/business/k8s/api', () => ({ getK8sClusterList: api.clusters, getK8sServiceList: api.list, createK8sService: api.create, updateK8sService: api.update, deleteK8sService: api.remove }));
vi.mock('@arco-design/web-react', () => {
  const Box = ({ children }: { children?: ReactNode }) => <div>{children}</div>;
  const Select = Object.assign(({ value, onChange, children, placeholder }: { value?: number; onChange: (value: number) => void; children?: ReactNode; placeholder?: string }) => <select aria-label={placeholder} value={value} onChange={(event) => onChange(Number(event.target.value))}>{children}</select>, { Option: ({ value, children }: { value: number; children?: ReactNode }) => <option value={value}>{children}</option> });
  return { Card: Box, Space: Box, Tag: Box, Typography: { Text: Box }, Select,
    Alert: ({ content, action }: { content: ReactNode; action: ReactNode }) => <div role="alert">{content}{action}</div>,
    Input: ({ onChange, ...props }: Omit<React.ComponentProps<'input'>, 'onChange'> & { onChange: (value: string) => void }) => <input {...props} onChange={(event) => onChange(event.target.value)} />,
    InputNumber: () => null, Popconfirm: ({ children, onOk }: { children: ReactNode; onOk: () => void }) => <div onClick={onOk}>{children}</div>,
    Button: ({ children, disabled, onClick }: { children: ReactNode; disabled?: boolean; onClick?: () => void }) => <button disabled={disabled} onClick={onClick}>{children}</button>,
    Message: { success: api.message, warning: api.message, error: api.message } };
});
vi.mock('../../../../src/components', () => {
  const Box = ({ children }: { children?: ReactNode }) => <div>{children}</div>;
  return { PageContainer: Box, GovernanceSummaryBar: () => null, TABLE_ACTION_COLUMN_WIDTH: { compact: 140 }, ListHeaderActions: ({ primary }: { primary: ReactNode }) => <div>{primary}</div>,
    AppModal: ({ visible, children, onOk, onCancel, okButtonProps }: { visible: boolean; children: ReactNode; onOk: () => void; onCancel: () => void; okButtonProps?: { disabled?: boolean } }) => visible ? <div role="dialog">{children}<button disabled={okButtonProps?.disabled} onClick={onOk}>save</button><button onClick={onCancel}>close</button></div> : null,
    AppTable: ({ data, columns }: { data: { name: string }[]; columns: { fixed?: string; render?: (value: unknown, row: { name: string }) => ReactNode }[] }) => <section>{data.map((row) => <div key={row.name}><p>{row.name}</p>{columns.find((column) => column.fixed)?.render?.(undefined, row)}</div>)}</section>,
    PageLoading: () => <p>loading</p>, PageEmpty: ({ description }: { description: string }) => <p>{description}</p>, PageError: () => <p>request-error</p>,
    SearchToolbar: ({ inlineFilters }: { inlineFilters: ReactNode }) => <div>{inlineFilters}</div> };
});
const row = (name = 'service-a') => ({ name, namespace: 'test', uid: `uid-${name}`, resourceVersion: '43', type: 'ClusterIP', clusterIP: '10.0.0.1', selector: { app: 'old' }, ports: [{ port: 80, targetPortName: 'http', targetPort: 0, protocol: 'TCP' }] });
function deferred<T>() { let resolve!: (value: T) => void; let reject!: (error: unknown) => void; const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; }); return { promise, resolve, reject }; }
async function mount() { api.clusters.mockResolvedValue({ items: [{ id: 1, name: 'Cluster A' }, { id: 2, name: 'Cluster B' }] }); render(<ServiceList />); await screen.findByText('service-a'); }
function edit() { fireEvent.click(screen.getByRole('button', { name: 'common.edit' })); fireEvent.change(screen.getByRole('textbox', { name: 'business.k8s.service.field.selector' }), { target: { value: 'app=new' } }); fireEvent.click(screen.getByRole('button', { name: 'save' })); }
afterEach(() => { cleanup(); vi.resetAllMocks(); });
describe('Kubernetes Service version authority', () => {
  it('sends the viewed UID/version and changes only the selected routing fields', async () => {
    api.list.mockResolvedValue({ items: [row()] }); api.update.mockResolvedValue(row()); await mount(); edit();
    await waitFor(() => expect(api.update).toHaveBeenCalledWith(1, 'test', 'service-a', { uid: 'uid-service-a', resourceVersion: '43', selector: { app: 'new' } }));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  });
  it('does not replay a conflict and requires explicit refresh/review', async () => {
    api.list.mockResolvedValue({ items: [row()] }); api.update.mockRejectedValue({ status: 409 }); await mount(); edit();
    await screen.findByRole('alert'); expect(screen.getByRole('button', { name: 'save' }).hasAttribute('disabled')).toBe(true);
    expect(api.update).toHaveBeenCalledOnce(); fireEvent.click(screen.getByRole('button', { name: 'common.refresh' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull()); expect(api.update).toHaveBeenCalledOnce();
  });
  it('clears visible rows and pending edits after permission revocation', async () => {
    api.list.mockResolvedValue({ items: [row()] }); api.update.mockRejectedValue(new RequestError({ kind: 'forbidden', status: 403 })); await mount(); edit();
    await screen.findByText('request-error'); expect(screen.queryByText('service-a')).toBeNull(); expect(screen.queryByRole('dialog')).toBeNull();
  });
  it('keeps a completed mutation from restoring the old cluster list after a source switch', async () => {
    const pending = deferred<ReturnType<typeof row>>(); api.list.mockResolvedValueOnce({ items: [row()] }).mockResolvedValue({ items: [row('service-b')] }); api.update.mockReturnValue(pending.promise);
    await mount(); edit(); fireEvent.change(screen.getByRole('combobox', { name: 'business.k8s.service.field.cluster' }), { target: { value: '2' } });
    await screen.findByText('service-b'); await act(async () => pending.resolve(row()));
    expect(screen.queryByText('service-a')).toBeNull(); expect(screen.getByText('service-b')).toBeTruthy(); expect(api.list).toHaveBeenCalledTimes(2);
  });
  it('ignores a revoked update outcome from a previously selected cluster', async () => {
    const pending = deferred<ReturnType<typeof row>>();
    api.list.mockResolvedValueOnce({ items: [row()] }).mockResolvedValue({ items: [row('service-b')] });
    api.update.mockReturnValue(pending.promise);
    await mount(); edit();
    fireEvent.change(screen.getByRole('combobox', { name: 'business.k8s.service.field.cluster' }), { target: { value: '2' } });
    await screen.findByText('service-b');
    await act(async () => pending.reject(new RequestError({ kind: 'forbidden', status: 403 })));
    expect(screen.getByText('service-b')).toBeTruthy();
    expect(screen.queryByText('request-error')).toBeNull();
  });
  it('ignores a revoked deletion outcome from a previously selected cluster', async () => {
    const pending = deferred<void>();
    api.list.mockResolvedValueOnce({ items: [row()] }).mockResolvedValue({ items: [row('service-b')] });
    api.remove.mockReturnValue(pending.promise);
    await mount(); fireEvent.click(screen.getByRole('button', { name: 'common.delete' }));
    fireEvent.change(screen.getByRole('combobox', { name: 'business.k8s.service.field.cluster' }), { target: { value: '2' } });
    await screen.findByText('service-b');
    await act(async () => pending.reject(new RequestError({ kind: 'forbidden', status: 403 })));
    expect(screen.getByText('service-b')).toBeTruthy();
    expect(screen.queryByText('request-error')).toBeNull();
  });
  it('deletes using both identity preconditions from the selected row', async () => {
    api.list.mockResolvedValue({ items: [row()] }); api.remove.mockResolvedValue(undefined); await mount(); fireEvent.click(screen.getByRole('button', { name: 'common.delete' }));
    await waitFor(() => expect(api.remove).toHaveBeenCalledWith(1, 'test', 'service-a', row()));
  });
  it('disables mutations when an older provider response lacks identity/version', async () => {
    api.list.mockResolvedValue({ items: [{ ...row(), uid: '', resourceVersion: '' }] }); await mount();
    expect(screen.getByRole('button', { name: 'common.edit' }).hasAttribute('disabled')).toBe(true); expect(screen.getByRole('button', { name: 'common.delete' }).hasAttribute('disabled')).toBe(true);
  });
});

