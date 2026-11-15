import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import DeployTargetSelector from '../../../../src/modules/business/deploy/task/DeployTargetSelector';
import { getHostList, type HostRow } from '../../../../src/modules/business/cmdb/host/api';
import { resolveDeployTargets, type DeployTargetResolution } from '../../../../src/modules/business/deploy/api';
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('../../../../src/components', () => ({
 ContextSelector: ({ options, selected, state, onSelect }: { options: Array<{id: string; label: string}>; selected: Array<{id: string}>; state: string; onSelect: (option: unknown) => void }) => <div>
  <span data-testid="lookup-state">{state}</span>
  <span data-testid="selected">{selected.map((item) => item.id).join(',')}</span>
  {options.map((option) => <button key={option.id} onClick={() => onSelect(option)}>{option.label}</button>)}
 </div>,
}));
vi.mock('../../../../src/modules/business/cmdb/host/api', () => ({ getHostList: vi.fn() }));
vi.mock('../../../../src/modules/business/deploy/api', () => ({ resolveDeployTargets: vi.fn() }));
const host = { id: 11, hostname: 'host-one', ip: '10.1.0.1', status: 'assigned' } as HostRow;
const result = (): DeployTargetResolution => ({ selection: { businessScopeId: 1, hostIds: [11], groupIds: [], manual: [], excludedIds: [], includeScope: false }, scopeVersion: 'current', resolvedAt: '', hosts: [], duplicateCount: 0, unavailableCount: 0, ineligibleCount: 0, excludedCount: 0, sources: {} });
beforeEach(() => { vi.mocked(getHostList).mockResolvedValue({ items: [host], total: 1, page: 1, pageSize: 25 }); });
afterEach(() => { cleanup(); vi.resetAllMocks(); });
describe('Deploy selection boundaries', () => {
 it('invalidates the review and ignores an obsolete resolver after intent editing', async () => {
  let complete!: (value: DeployTargetResolution) => void;
  vi.mocked(resolveDeployTargets).mockReturnValue(new Promise((done) => { complete = done; }));
  const changed = vi.fn();
  render(<DeployTargetSelector scopeId={1} groups={[]} onChange={changed} />);
  await screen.findByText('host-one · 10.1.0.1');
  fireEvent.click(screen.getByText('business.deploy.selection.resolve'));
  const signal = vi.mocked(resolveDeployTargets).mock.calls[0][1];
  fireEvent.change(screen.getByLabelText('business.deploy.selection.manual'), { target: { value: '10.1.0.1' } });
  expect(signal?.aborted).toBe(true);
  await act(async () => complete(result()));
  expect(changed.mock.calls.every(([value]) => value === null)).toBe(true);
  expect(screen.queryByRole('status')).toBeNull();
 });
 it('aborts the old scope resolver and removes ghost host selection on context remount', async () => {
  vi.mocked(resolveDeployTargets).mockReturnValue(new Promise(() => {}));
  const changed = vi.fn();
  const { rerender } = render(<DeployTargetSelector key="1" scopeId={1} groups={[]} initialHostIds={[11]} onChange={changed} />);
  await screen.findByText('host-one · 10.1.0.1');
  expect(screen.getByTestId('selected').textContent).toBe('11');
  fireEvent.click(screen.getByText('business.deploy.selection.resolve'));
  const signal = vi.mocked(resolveDeployTargets).mock.calls[0][1];
  rerender(<DeployTargetSelector key="2" scopeId={2} groups={[]} onChange={changed} />);
  expect(signal?.aborted).toBe(true);
  expect(screen.getByTestId('selected').textContent).toBe('');
  await waitFor(() => expect(vi.mocked(getHostList).mock.lastCall?.[0]?.businessScopeId).toBe(2));
 });
 it('cancels obsolete host searches instead of accepting their late results', async () => {
  vi.mocked(getHostList).mockReturnValueOnce(new Promise(() => {}));
  render(<DeployTargetSelector scopeId={1} groups={[]} onChange={vi.fn()} />);
  await waitFor(() => expect(getHostList).toHaveBeenCalledOnce());
  const signal = vi.mocked(getHostList).mock.calls[0][1];
  fireEvent.change(screen.getByLabelText('business.deploy.selection.search'), { target: { value: 'new' } });
  expect(signal?.aborted).toBe(true);
  await waitFor(() => expect(getHostList).toHaveBeenCalledTimes(2));
  expect(vi.mocked(getHostList).mock.lastCall?.[0]?.keyword).toBe('new');
 });
 it('clears a failed resolution before a successful retry', async () => {
  vi.mocked(resolveDeployTargets).mockRejectedValueOnce(new Error('unavailable')).mockResolvedValueOnce(result());
  const changed = vi.fn();
  render(<DeployTargetSelector scopeId={1} groups={[]} onChange={changed} />);
  fireEvent.click(screen.getByText('business.deploy.selection.resolve'));
  await screen.findByText('business.deploy.selection.loadFailed');
  fireEvent.click(screen.getByText('business.deploy.selection.resolve'));
  await waitFor(() => expect(changed).toHaveBeenLastCalledWith(result()));
  expect(screen.queryByText('business.deploy.selection.loadFailed')).toBeNull();
 });
});
