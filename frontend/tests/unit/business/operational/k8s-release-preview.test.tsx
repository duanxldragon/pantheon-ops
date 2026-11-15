import type { ReactNode } from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import ReleasePreview from '../../../../src/modules/business/k8s/release/ReleasePreview';
import { createK8sRelease, previewK8sRelease, previewK8sRollback, rollbackK8sRelease } from '../../../../src/modules/business/k8s/api';
import { RequestError } from '../../../../src/api/request';
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('../../../../src/components', () => ({ AppModal: ({ children }: { children: ReactNode }) => <div role="dialog">{children}</div>, ChangeDiff: ({ state }: { state: string }) => <div data-testid="preview-state">{state}</div> }));
vi.mock('../../../../src/modules/business/k8s/api', () => ({ createK8sRelease: vi.fn(), previewK8sRelease: vi.fn(), previewK8sRollback: vi.fn(), rollbackK8sRelease: vi.fn() }));
afterEach(() => { cleanup(); vi.resetAllMocks(); });
const payload = { name: 'api', clusterId: 1, namespace: 'default', workloadType: 'deployment', workloadName: 'api', image: 'new:v2' };
const intent = { operation: 'create' as const, payload };
const result = () => ({ intentId: 'key', intentVersion: 'authenticated-ticket', currentVersion: '1', targetVersion: 'hash', targetCount: 1, environment: 'test', namespace: 'default', workloadType: 'deployment', workloadName: 'api', containerName: 'app', changes: [], riskKeys: [], rollbackAvailable: true, expiresAt: new Date(Date.now() + 600000).toISOString() });
const ready = () => waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('ready'));
const confirm = () => fireEvent.click(screen.getByRole('button', { name: 'business.k8s.release.preview.confirm' }));

it('requires confirmation and submits exactly the reviewed image intent', async () => {
 vi.mocked(previewK8sRelease).mockResolvedValue(result()); vi.mocked(createK8sRelease).mockResolvedValue({} as never);
 const accepted = vi.fn(); render(<ReleasePreview intent={intent} onClose={vi.fn()} onAccepted={accepted} />); await ready();
 expect(createK8sRelease).not.toHaveBeenCalled(); confirm(); await waitFor(() => expect(accepted).toHaveBeenCalledOnce());
 const reviewedKey = vi.mocked(previewK8sRelease).mock.calls[0][0].idempotencyKey;
 expect(createK8sRelease).toHaveBeenCalledWith({ ...payload, idempotencyKey: reviewedKey, intentVersion: 'authenticated-ticket' });
});

it('aborts a changed target review and ignores the old late response', async () => {
 let resolve!: (value: ReturnType<typeof result>) => void;
 vi.mocked(previewK8sRelease).mockReturnValueOnce(new Promise((done) => { resolve = done; })).mockReturnValueOnce(new Promise(() => {}));
 const props = { onClose: vi.fn(), onAccepted: vi.fn() };
 const { rerender } = render(<ReleasePreview intent={intent} {...props} />);
 const signal = vi.mocked(previewK8sRelease).mock.calls[0][1];
 rerender(<ReleasePreview intent={{ ...intent, payload: { ...payload, image: 'different:v3' } }} {...props} />);
 expect(signal?.aborted).toBe(true); await act(async () => resolve(result()));
 expect(screen.getByTestId('preview-state').textContent).toBe('loading'); confirm(); expect(createK8sRelease).not.toHaveBeenCalled();
});

it('refreshes only after a definite conflict and uses a new request intent', async () => {
 vi.mocked(previewK8sRelease).mockResolvedValue(result()); vi.mocked(createK8sRelease).mockRejectedValue(new RequestError({ kind: 'business', code: 409 }));
 render(<ReleasePreview intent={intent} onClose={vi.fn()} onAccepted={vi.fn()} />); await ready(); confirm();
 await waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('stale'));
 fireEvent.click(screen.getByRole('button', { name: 'business.k8s.release.preview.refresh' })); await ready();
 expect(vi.mocked(previewK8sRelease).mock.calls[1][0].idempotencyKey).not.toBe(vi.mocked(previewK8sRelease).mock.calls[0][0].idempotencyKey);
});

it('retains the exact request during an ambiguous failure and does not create a fresh preview', async () => {
 vi.mocked(previewK8sRelease).mockResolvedValue(result()); vi.mocked(createK8sRelease).mockRejectedValueOnce(new Error('transport interrupted')).mockResolvedValueOnce({} as never);
 const accepted = vi.fn(); render(<ReleasePreview intent={intent} onClose={vi.fn()} onAccepted={accepted} />); await ready(); confirm();
 await screen.findByText('business.k8s.release.preview.uncertain');
 const refresh = screen.getByRole('button', { name: 'business.k8s.release.preview.refresh' }) as HTMLButtonElement; expect(refresh.disabled).toBe(true);
 fireEvent.click(refresh); expect(previewK8sRelease).toHaveBeenCalledOnce();
 fireEvent.click(screen.getByRole('button', { name: 'business.k8s.release.preview.retrySame' })); await waitFor(() => expect(accepted).toHaveBeenCalledOnce());
 expect(vi.mocked(createK8sRelease).mock.calls[1]).toEqual(vi.mocked(createK8sRelease).mock.calls[0]);
});

it('cannot execute twice if the accepted list refresh fails', async () => {
 vi.mocked(previewK8sRelease).mockResolvedValue(result()); vi.mocked(createK8sRelease).mockResolvedValue({} as never);
 render(<ReleasePreview intent={intent} onClose={vi.fn()} onAccepted={vi.fn().mockRejectedValue(new Error('list error'))} />); await ready(); confirm();
 await screen.findByText('business.k8s.release.preview.refreshFailed'); confirm(); expect(createK8sRelease).toHaveBeenCalledOnce();
});

it('rejects an expired preview before execution', async () => {
 vi.mocked(previewK8sRelease).mockResolvedValue({ ...result(), expiresAt: new Date(Date.now() - 1).toISOString() });
 render(<ReleasePreview intent={intent} onClose={vi.fn()} onAccepted={vi.fn()} />);
 await waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('stale')); confirm(); expect(createK8sRelease).not.toHaveBeenCalled();
});

it('reviews rollback separately and carries its own ticket', async () => {
 vi.mocked(previewK8sRollback).mockResolvedValue(result()); vi.mocked(rollbackK8sRelease).mockResolvedValue({} as never);
 const accepted = vi.fn(); render(<ReleasePreview intent={{ operation: 'rollback', releaseId: 9 }} onClose={vi.fn()} onAccepted={accepted} />); await ready(); confirm();
 await waitFor(() => expect(accepted).toHaveBeenCalledOnce());
 expect(rollbackK8sRelease).toHaveBeenCalledWith(9, vi.mocked(previewK8sRollback).mock.calls[0][1], 'authenticated-ticket'); expect(createK8sRelease).not.toHaveBeenCalled();
});
