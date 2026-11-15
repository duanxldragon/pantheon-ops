import type { ReactNode } from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import ArgoCDPreview from '../../../../src/modules/business/cicd/argocd/ArgoCDPreview';
import { previewArgoCDApplication, rollbackArgoCDApplication, syncArgoCDApplication, type ArgoCDApplicationRow } from '../../../../src/modules/business/cicd/argocd/api';
import { RequestError } from '../../../../src/api/request';

vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('@arco-design/web-react', async (original) => ({ ...await original<typeof import('@arco-design/web-react')>(), Select: ({ value, options, onChange, ...props }: { value: string; options: Array<{ value: string; label: string; disabled?: boolean }>; onChange: (value: string) => void }) => <select {...props} value={value} onChange={(event) => onChange(event.target.value)}>{options.map((option) => <option key={option.value} value={option.value} disabled={option.disabled}>{option.label}</option>)}</select> }));
vi.mock('../../../../src/components', () => ({
  AppModal: ({ children }: { children: ReactNode }) => <div role="dialog">{children}</div>,
  ChangeDiff: ({ state, lines }: { state: string; lines: Array<{ key: string; kind: string; sensitive: boolean }> }) => <div><div data-testid="preview-state">{state}</div><output data-testid="diff-lines">{JSON.stringify(lines)}</output></div>,
}));
vi.mock('../../../../src/modules/business/cicd/argocd/api', () => ({ previewArgoCDApplication: vi.fn(), rollbackArgoCDApplication: vi.fn(), syncArgoCDApplication: vi.fn() }));
afterEach(() => { cleanup(); vi.resetAllMocks(); });

const prefix = 'business.cicd.argocd.preview';
const revision = 'a'.repeat(40);
const oldRevision = 'b'.repeat(40);
const application = { id: 8, name: 'api', sourceTargetRev: 'main' } as ArgoCDApplicationRow;
const result = () => ({ intentId: 'argocd:provider-id', intentVersion: 'signed-ticket', currentVersion: '15', currentRevision: oldRevision, targetVersion: revision, namespace: 'test', targetCount: 2, changes: [{ group: 'apps', kind: 'Deployment', namespace: 'test', name: 'api', change: 'modified' as const, valuesHidden: true }, { group: '', kind: 'ConfigMap', namespace: 'test', name: 'retained', change: 'retained' as const, valuesHidden: true }], history: [{ id: 0, revision: oldRevision, current: false, rollbackAvailable: true }], historyTruncated: false, riskKeys: [], rollbackAvailable: false, healthCheckRequired: true, expiresAt: new Date(Date.now() + 300000).toISOString() });
const ready = () => waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('ready'));
const confirm = () => fireEvent.click(screen.getByRole('button', { name: `${prefix}.confirm` }));

it('waits for confirmation then executes the resolved revision with the exact preview ticket', async () => {
  vi.mocked(previewArgoCDApplication).mockResolvedValue(result()); vi.mocked(syncArgoCDApplication).mockResolvedValue({ id: 1, status: 'Running' });
  const accepted = vi.fn(); render(<ArgoCDPreview application={application} onClose={vi.fn()} onAccepted={accepted} />); await ready();
  expect(syncArgoCDApplication).not.toHaveBeenCalled(); confirm(); await waitFor(() => expect(accepted).toHaveBeenCalledOnce());
  expect(syncArgoCDApplication).toHaveBeenCalledWith(8, revision, 'signed-ticket');
  expect(previewArgoCDApplication).toHaveBeenCalledWith(8, { revision: 'main', rollback: false }, expect.any(AbortSignal));
});

it('classifies retained resources as unchanged, hides values, and searches provider identity', async () => {
  vi.mocked(previewArgoCDApplication).mockResolvedValue(result());
  render(<ArgoCDPreview application={application} onClose={vi.fn()} onAccepted={vi.fn()} />); await ready();
  const lines = JSON.parse(screen.getByTestId('diff-lines').textContent ?? '[]');
  expect(lines[1]).toMatchObject({ kind: 'unchanged', sensitive: true });
  fireEvent.change(screen.getByRole('textbox', { name: `${prefix}.search` }), { target: { value: 'deployment' } });
  expect(JSON.parse(screen.getByTestId('diff-lines').textContent ?? '[]')).toHaveLength(1);
});

it('locks execution after conflict until an explicit fresh review', async () => {
  vi.mocked(previewArgoCDApplication).mockResolvedValueOnce(result()).mockResolvedValueOnce({ ...result(), intentVersion: 'fresh-ticket' });
  vi.mocked(syncArgoCDApplication).mockRejectedValueOnce(new RequestError({ kind: 'business', code: 409 })).mockResolvedValueOnce({ id: 1, status: 'Running' });
  const accepted = vi.fn(); render(<ArgoCDPreview application={application} onClose={vi.fn()} onAccepted={accepted} />); await ready(); confirm();
  await waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('stale'));
  expect((screen.getByRole('button', { name: `${prefix}.confirm` }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(screen.getByRole('button', { name: `${prefix}.refresh` })); await ready(); confirm();
  await waitFor(() => expect(accepted).toHaveBeenCalledOnce());
  expect(syncArgoCDApplication).toHaveBeenLastCalledWith(8, revision, 'fresh-ticket');
});

it('recovers an ambiguous request with the same ticket and blocks changing the operation', async () => {
  vi.mocked(previewArgoCDApplication).mockResolvedValue(result());
  vi.mocked(syncArgoCDApplication).mockRejectedValueOnce(new Error('transport')).mockResolvedValueOnce({ id: 1, status: 'Running' });
  const accepted = vi.fn(); render(<ArgoCDPreview application={application} onClose={vi.fn()} onAccepted={accepted} />); await ready(); confirm();
  await screen.findByText(`${prefix}.uncertain`);
  expect((screen.getByRole('combobox') as HTMLSelectElement).disabled).toBe(true);
  expect((screen.getByRole('button', { name: `${prefix}.refresh` }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(screen.getByRole('button', { name: `${prefix}.retrySame` })); await waitFor(() => expect(accepted).toHaveBeenCalledOnce());
  expect(vi.mocked(syncArgoCDApplication).mock.calls[1]).toEqual(vi.mocked(syncArgoCDApplication).mock.calls[0]);
  expect(previewArgoCDApplication).toHaveBeenCalledOnce();
});

it('reviews native history rollback separately and ignores the prior aborted late response', async () => {
  let resolve!: (value: ReturnType<typeof result>) => void;
  vi.mocked(previewArgoCDApplication).mockResolvedValueOnce(result()).mockReturnValueOnce(new Promise((done) => { resolve = done; })).mockResolvedValueOnce({ ...result(), targetVersion: oldRevision, intentVersion: 'rollback-ticket', rollbackAvailable: true });
  vi.mocked(rollbackArgoCDApplication).mockResolvedValue({ id: 1, status: 'Running' });
  const accepted = vi.fn(); render(<ArgoCDPreview application={application} onClose={vi.fn()} onAccepted={accepted} />); await ready();
  fireEvent.click(screen.getByRole('button', { name: `${prefix}.refresh` }));
  // Programmatic test steering exercises cancellation while a request is in flight.
  fireEvent.change(screen.getByRole('combobox'), { target: { value: oldRevision } });
  await ready(); await act(async () => resolve(result()));
  expect(vi.mocked(previewArgoCDApplication).mock.calls[1][2]?.aborted).toBe(true);
  fireEvent.click(screen.getByRole('button', { name: `${prefix}.confirmRollback` })); await waitFor(() => expect(accepted).toHaveBeenCalledOnce());
  expect(rollbackArgoCDApplication).toHaveBeenCalledWith(8, oldRevision, 'rollback-ticket'); expect(syncArgoCDApplication).not.toHaveBeenCalled();
});

it('rejects expired previews and forbids execution after denied preview', async () => {
  vi.mocked(previewArgoCDApplication).mockResolvedValueOnce({ ...result(), expiresAt: new Date(Date.now() - 1).toISOString() }).mockRejectedValueOnce(new RequestError({ kind: 'forbidden', code: 403 }));
  render(<ArgoCDPreview application={application} onClose={vi.fn()} onAccepted={vi.fn()} />);
  await waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('stale')); confirm();
  fireEvent.click(screen.getByRole('button', { name: `${prefix}.refresh` }));
  await waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('forbidden')); confirm();
  expect(syncArgoCDApplication).not.toHaveBeenCalled();
});

it('does not execute again if list refresh fails after provider acceptance', async () => {
  vi.mocked(previewArgoCDApplication).mockResolvedValue(result()); vi.mocked(syncArgoCDApplication).mockResolvedValue({ id: 1, status: 'Running' });
  render(<ArgoCDPreview application={application} onClose={vi.fn()} onAccepted={vi.fn().mockRejectedValue(new Error('refresh'))} />); await ready(); confirm();
  await screen.findByText(`${prefix}.refreshFailed`); confirm(); expect(syncArgoCDApplication).toHaveBeenCalledOnce();
});
