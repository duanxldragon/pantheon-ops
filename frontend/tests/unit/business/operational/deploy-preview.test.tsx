import type { ReactNode } from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import DeployTaskPreview from '../../../../src/modules/business/deploy/task/DeployTaskPreview';
import { previewDeployTask, startDeployTask } from '../../../../src/modules/business/deploy/api';
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('../../../../src/components', () => ({
 AppModal: ({ children }: { children: ReactNode }) => <div role="dialog">{children}</div>,
 ChangeDiff: ({ state }: { state: string }) => <div data-testid="preview-state">{state}</div>,
}));
vi.mock('../../../../src/modules/business/deploy/api', () => ({ previewDeployTask: vi.fn(), startDeployTask: vi.fn() }));
afterEach(() => { cleanup(); vi.resetAllMocks(); vi.useRealTimers(); });
const result = () => ({ intentId: 'deploy:7', intentVersion: 'signed-ticket', currentVersion: 'revision', targetVersion: 'v2', targetCount: 1, businessScopeName: 'scope', changes: [], riskKeys: [], rollbackAvailable: false, expiresAt: new Date(Date.now() + 600000).toISOString() });
describe('Deploy confirmation lifecycle', () => {
 it('requires explicit confirmation and passes the reviewed ticket once', async () => {
  vi.mocked(previewDeployTask).mockResolvedValue(result());
  vi.mocked(startDeployTask).mockResolvedValue({} as never);
  const done = vi.fn(); const payload = {};
  render(<DeployTaskPreview taskId={7} payload={payload} onClose={vi.fn()} onStarted={done} />);
  await waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('ready'));
  expect(startDeployTask).not.toHaveBeenCalled();
  fireEvent.click(screen.getByText('business.deploy.preview.confirm'));
  await waitFor(() => expect(done).toHaveBeenCalledOnce());
  expect(startDeployTask).toHaveBeenCalledWith(7, { intentVersion: 'signed-ticket' }, expect.any(String));
 });
 it('clears the previous intent while a different task review is pending', async () => {
  vi.mocked(previewDeployTask).mockResolvedValueOnce(result()).mockReturnValueOnce(new Promise(() => {}));
  const payload = {}; const props = { payload, onClose: vi.fn(), onStarted: vi.fn() };
  const { rerender } = render(<DeployTaskPreview taskId={7} {...props} />);
  await waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('ready'));
  rerender(<DeployTaskPreview taskId={8} {...props} />);
  expect(screen.getByTestId('preview-state').textContent).toBe('loading');
  expect((screen.getByRole('button', { name: 'business.deploy.preview.confirm' }) as HTMLButtonElement).disabled).toBe(true);
  expect(screen.queryByText('business.deploy.preview.scopeSummary')).toBeNull();
 });
 it('keeps an accepted start successful when the parent refresh fails', async () => {
  vi.mocked(previewDeployTask).mockResolvedValue(result());
  vi.mocked(startDeployTask).mockResolvedValue({} as never);
  const onStarted = vi.fn().mockRejectedValue(new Error('refresh failed'));
  render(<DeployTaskPreview taskId={7} payload={{}} onClose={vi.fn()} onStarted={onStarted} />);
  await waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('ready'));
  fireEvent.click(screen.getByRole('button', { name: 'business.deploy.preview.confirm' }));
  await screen.findByText('business.deploy.preview.postStartRefreshFailed');
  expect(screen.getByText('business.deploy.preview.startAccepted')).toBeTruthy();
  expect(screen.queryByTestId('preview-state')).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'business.deploy.preview.refresh' }));
  fireEvent.click(screen.getByRole('button', { name: 'business.deploy.preview.confirm' }));
  expect(startDeployTask).toHaveBeenCalledOnce();
  expect(previewDeployTask).toHaveBeenCalledOnce();
 });
 it('cannot start an expired or denied preview', async () => {
  vi.mocked(previewDeployTask).mockResolvedValue({ ...result(), expiresAt: new Date(Date.now() - 1000).toISOString() });
  render(<DeployTaskPreview taskId={7} payload={{}} onClose={vi.fn()} onStarted={vi.fn()} />);
  await waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('stale'));
  fireEvent.click(screen.getByText('business.deploy.preview.confirm'));
  expect(startDeployTask).not.toHaveBeenCalled();
 });
 it('aborts a closed preview and ignores its late response', async () => {
  let resolve!: (value: ReturnType<typeof result>) => void;
  vi.mocked(previewDeployTask).mockReturnValue(new Promise((done) => { resolve = done; }));
  const payload = {}; const { unmount } = render(<DeployTaskPreview taskId={7} payload={payload} onClose={vi.fn()} onStarted={vi.fn()} />);
  const signal = vi.mocked(previewDeployTask).mock.calls[0][2]; unmount();
  expect(signal?.aborted).toBe(true);
  await act(async () => resolve(result()));
  expect(startDeployTask).not.toHaveBeenCalled();
 });
 it('clears a rejected ticket and requires a fresh review', async () => {
  vi.mocked(previewDeployTask).mockResolvedValue(result());vi.mocked(startDeployTask).mockRejectedValue(new Error('changed'));
  render(<DeployTaskPreview taskId={7} payload={{}} onClose={vi.fn()} onStarted={vi.fn()} />);
  await waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('ready'));
  fireEvent.click(screen.getByText('business.deploy.preview.confirm'));
  await waitFor(() => expect(screen.getByTestId('preview-state').textContent).toBe('stale'));
  fireEvent.click(screen.getByText('business.deploy.preview.confirm'));
  expect(startDeployTask).toHaveBeenCalledOnce();
 });
});


describe('Frozen configuration review', () => {
 it('displays proposed safe fields and never renders or searches hidden values', async () => {
  vi.mocked(previewDeployTask).mockResolvedValue({ ...result(), plan: {
   templateId: 1, templateName: 'frozen-template', templateVersion: '1.0.0', stepCount: 1, parameterCount: 3, truncated: true,
   steps: [{ id: 'step:1', name: 'frozen-step', type: 'package', action: 'install', packageName: 'safe-package', packageVersion: '2.0.0', failurePolicy: 'fail_fast', continueOnHostFailure: false,
    parameters: [{ key: 'port', value: '8012', hidden: false }, { key: 'serviceName', value: 'safe-service', hidden: false }, { key: 'rootPassword', value: 'never-visible-needle', hidden: true }],
   }],
  } });
  render(<DeployTaskPreview taskId={7} payload={{}} onClose={vi.fn()} onStarted={vi.fn()} />);
  await screen.findByText('frozen-step');
  expect(screen.getByText('business.deploy.preview.parametersUnknown')).toBeTruthy();
  expect(screen.getByText('business.deploy.preview.planTruncated')).toBeTruthy();
  expect(screen.getByText('8012')).toBeTruthy();
  expect(screen.getByText('business.deploy.preview.sensitive')).toBeTruthy();
  expect(screen.queryByText('never-visible-needle')).toBeNull();
  fireEvent.change(screen.getByLabelText('business.deploy.preview.search'), { target: { value: 'serviceName' } });
  expect(screen.getByText('safe-service')).toBeTruthy();
  expect(screen.queryByText('8012')).toBeNull();
  fireEvent.change(screen.getByLabelText('business.deploy.preview.search'), { target: { value: 'never-visible-needle' } });
  expect(screen.queryByText('frozen-step')).toBeNull();
  expect(screen.queryByText('rootPassword')).toBeNull();
 });
});
