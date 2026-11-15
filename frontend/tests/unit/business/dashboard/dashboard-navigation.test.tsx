import { useState, type ComponentType, type ReactNode } from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, useNavigate } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import DeployTaskList from '../../../../src/modules/business/deploy/task/DeployTaskList';
import ClusterList from '../../../../src/modules/business/k8s/cluster/ClusterList';
import WorkloadList from '../../../../src/modules/business/k8s/workload/WorkloadList';

const fixture = vi.hoisted(() => ({
  deployList: vi.fn(async () => ({ items: [], total: 0, page: 1, pageSize: 10 })),
  clusterList: vi.fn(async () => ({ items: [], total: 0, page: 1, pageSize: 20 })),
  workloadList: vi.fn(async () => ({ items: [{ name: 'api', namespace: 'payments', kind: 'Deployment', status: 'ready', replicas: 1, readyReplicas: 1, images: [] }, { name: 'api-other', namespace: 'payments', kind: 'Deployment', status: 'ready', replicas: 1, readyReplicas: 1, images: [] }], total: 2 })),
  t: (key: string) => key,
}));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: fixture.t }) }));
vi.mock('@duanxldragon/pantheon-base-ui', () => ({ apiRequest: vi.fn(), downloadFile: vi.fn(), uploadImportFile: vi.fn() }));
vi.mock('../../../../src/hooks/usePermission', () => ({ usePermission: () => ({ hasPerm: () => false }) }));
vi.mock('../../../../src/modules/business/deploy/api', async (original) => ({
  ...await original<object>(), getDeployTaskList: fixture.deployList,
  getDeployPackageList: vi.fn(async () => ({ items: [] })), getDeployTemplateList: vi.fn(async () => ({ items: [] })),
}));
vi.mock('../../../../src/modules/business/k8s/api', async (original) => ({
  ...await original<object>(), getK8sClusterList: fixture.clusterList,
  getK8sWorkloadList: fixture.workloadList,
}));
vi.mock('../../../../src/modules/business/cmdb/group/api', () => ({ getGroupList: vi.fn(async () => []) }));
vi.mock('../../../../src/modules/business/cmdb/host/api', () => ({ getHostList: vi.fn(async () => ({ items: [] })) }));
vi.mock('../../../../src/modules/business/bizscope/api', () => ({ getBizScopeOptions: vi.fn(async () => []) }));
vi.mock('../../../../src/modules/business/deploy/task/DeployTargetSelector', () => ({ default: () => null }));
vi.mock('../../../../src/modules/business/deploy/task/DeployTaskPreview', () => ({ default: () => null }));
vi.mock('../../../../src/modules/business/k8s/cluster/ClusterForm', () => ({ default: () => null }));
vi.mock('../../../../src/modules/business/k8s/workload/PodLogViewer', () => ({ default: () => null }));
vi.mock('@arco-design/web-react', () => {
  const Wrap = ({ children }: { children?: ReactNode }) => <div>{children}</div>;
  const Select = Object.assign(({ value, onChange, children }: { value?: string; onChange?: (value: string) => void; children?: ReactNode }) =>
    <select value={value ?? ''} onChange={(event) => onChange?.(event.target.value)}><option value="" />{children}</select>,
  { Option: ({ value, children }: { value: string; children: ReactNode }) => <option value={value}>{children}</option> });
  return {
    Alert: ({ content, onClose }: { content: ReactNode; onClose?: () => void }) => <div>{content}<button onClick={onClose}>clear filter</button></div>, Button: Wrap, Card: Wrap, Descriptions: Wrap, Divider: () => null,
    Form: Object.assign(Wrap, { useForm: () => [{}], Item: Wrap }),
    Input: () => null, InputNumber: () => null, Message: { error: vi.fn() }, Popconfirm: Wrap, Select, Space: Wrap, Tag: Wrap,
    Typography: { Text: Wrap },
  };
});
vi.mock('../../../../src/components', () => {
  const Wrap = ({ children }: { children?: ReactNode }) => <div>{children}</div>;
  return {
    AppDrawer: () => null, AppModal: () => null, AppTable: ({ data }: { data: { name?: string }[] }) => <div>{data?.map((row) => <div key={row.name}>{row.name}</div>)}</div>, AsyncContentState: Wrap,
    GovernanceInsightDrawer: () => null, GovernanceRailSummary: () => null,
    GovernanceRailToggleButton: () => null, GovernanceSummaryBar: () => null,
    ListHeaderActions: () => null, PageContainer: Wrap, PageEmpty: () => null, PageError: () => null,
    PageLoading: () => null, SubmitBar: () => null, TableBatchActionBar: () => null,
    SearchToolbar: ({ keyword, onKeywordChange, inlineFilters }: { keyword: string; onKeywordChange: (value: string) => void; inlineFilters: ReactNode }) =>
      <div><input aria-label="keyword" value={keyword} onChange={(event) => onKeywordChange(event.target.value)} />{inlineFilters}</div>,
    buildStandardPagination: () => ({}), TABLE_ACTION_COLUMN_WIDTH: 180,
    useGovernanceRail: () => ({ expanded: false, toggle: () => {} }),
  };
});

function Navigation({ Page }: { Page: ComponentType }) {
  const navigate = useNavigate();
  const [tick, setTick] = useState(0);
  return <div data-tick={tick}>
    <button onClick={() => setTick((value) => value + 1)}>rerender</button>
    <button onClick={() => navigate('?keyword=first&status=failed&businessScopeId=11&editId=99')}>edit intent</button>
    <button onClick={() => navigate('?keyword=second&status=running&businessScopeId=22')}>next dashboard target</button>
    <Page />
  </div>;
}
afterEach(() => { cleanup(); fixture.deployList.mockClear(); fixture.clusterList.mockClear(); });

describe('dashboard navigation to mounted business lists', () => {
  it('preserves the exact dashboard creation window in deployment list requests', async () => {
    render(<MemoryRouter initialEntries={['/list?createdFrom=2026-11-13T10%3A00%3A00Z&createdTo=2026-11-14T10%3A00%3A00Z']}><DeployTaskList /></MemoryRouter>);
    await waitFor(() => expect(fixture.deployList).toHaveBeenLastCalledWith(expect.objectContaining({ createdFrom: '2026-11-13T10:00:00Z', createdTo: '2026-11-14T10:00:00Z' })));
    expect(screen.getByText('business.dashboard.timeFilter')).toBeDefined();
  });

  it('requests the dashboard workload cluster/namespace/kind and filters by exact name with a clear action', async () => {
    render(<MemoryRouter initialEntries={['/list?clusterId=71&namespace=payments&kind=deployment&keyword=api']}><WorkloadList /></MemoryRouter>);
    await waitFor(() => expect(fixture.workloadList).toHaveBeenLastCalledWith({ clusterId: 71, namespace: 'payments', kind: 'deployment' }));
    await waitFor(() => expect(screen.getByText('api')).toBeDefined());
    expect(screen.queryByText('api-other')).toBeNull();
    fireEvent.click(screen.getByText('clear filter'));
    await waitFor(() => expect(screen.getByText('api-other')).toBeDefined());
  });

  for (const [name, Page, list] of [
    ['deploy', DeployTaskList, fixture.deployList], ['clusters', ClusterList, fixture.clusterList],
  ] as const) {
    it(`${name}: applies new URL filters while preserving manual edits for unchanged filters and editId`, async () => {
      render(<MemoryRouter initialEntries={['/list?keyword=first&status=failed&businessScopeId=11']}>
        <Navigation Page={Page} />
      </MemoryRouter>);
      await waitFor(() => expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ keyword: 'first', status: 'failed', businessScopeId: 11 })));
      fireEvent.change(screen.getByLabelText('keyword'), { target: { value: 'operator edit' } });
      await waitFor(() => expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ keyword: 'operator edit' })));
      fireEvent.click(screen.getByText('rerender'));
      expect((screen.getByLabelText('keyword') as HTMLInputElement).value).toBe('operator edit');
      fireEvent.click(screen.getByText('edit intent'));
      await waitFor(() => expect((screen.getByLabelText('keyword') as HTMLInputElement).value).toBe('operator edit'));
      fireEvent.click(screen.getByText('next dashboard target'));
      await waitFor(() => expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ keyword: 'second', status: 'running', businessScopeId: 22, page: 1 })));
      expect((screen.getByLabelText('keyword') as HTMLInputElement).value).toBe('second');
    });
  }
});
