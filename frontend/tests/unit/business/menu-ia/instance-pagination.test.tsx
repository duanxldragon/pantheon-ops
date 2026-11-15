import type { ReactNode } from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import ServiceInstanceList from '../../../../src/modules/business/serviceinstance/ServiceInstanceList';

const fixture = vi.hoisted(() => ({ list: vi.fn() }));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('@duanxldragon/pantheon-base-ui', () => ({ usePermission: () => ({ hasPerm: (key: string) => key === 'business:service:list' }) }));
vi.mock('../../../../src/api/request', () => ({ isForbiddenRequestError: () => false }));
vi.mock('../../../../src/modules/business/service/api', () => ({
  getServiceInstanceList: fixture.list, deleteServiceInstance: vi.fn(), reconcileServiceInstance: vi.fn(),
}));
vi.mock('../../../../src/modules/business/serviceinstance/InstanceCreateModal', () => ({ default: () => null }));
vi.mock('@arco-design/web-react', () => ({
  Card: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  Space: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  Tag: ({ children }: { children: ReactNode }) => <span>{children}</span>,
  Button: ({ children }: { children: ReactNode }) => <button>{children}</button>,
  Popconfirm: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  Message: { success: vi.fn(), error: vi.fn() },
}));
interface Pagination {
  current: number;
  pageSize: number;
  total: number;
  onChange: (page: number, pageSize: number) => void;
}
vi.mock('../../../../src/components', () => ({
  PageContainer: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  GovernanceSummaryBar: () => null,
  SearchToolbar: () => null,
  PageLoading: () => <span>loading</span>,
  PageError: () => <span>failed</span>,
  PageForbidden: () => <span>forbidden</span>,
  PageEmpty: () => <span>empty-dataset</span>,
  TABLE_ACTION_COLUMN_WIDTH: { medium: 240 },
  buildStandardPagination: (_t: unknown, options: Pagination) => options,
  AppTable: ({ data, pagination }: { data: { serviceName: string }[]; pagination: Pagination }) => <div data-testid="table">
    {data.map((row) => <span key={row.serviceName}>{row.serviceName}</span>)}
    <span data-testid="current-page">{pagination.current}</span>
    <button disabled={pagination.current === 1} onClick={() => pagination.onChange(pagination.current - 1, pagination.pageSize)}>previous-page</button>
    <button disabled={pagination.current * pagination.pageSize >= pagination.total} onClick={() => pagination.onChange(pagination.current + 1, pagination.pageSize)}>next-page</button>
  </div>,
}));

beforeEach(() => { fixture.list.mockReset(); });
afterEach(cleanup);

describe('instance list pagination after dataset shrink', () => {
  it('keeps pagination for an empty second page and can recover the remaining first page', async () => {
    let total = 21;
    fixture.list.mockImplementation(async ({ page }: { page: number }) => {
      if (page === 2) {
        total = 20;
        return { items: [], total, page: 2, pageSize: 20 };
      }
      return { items: [{ id: 1, serviceName: 'remaining-instance' }], total, page: 1, pageSize: 20 };
    });
    render(<ServiceInstanceList />);
    await screen.findByText('remaining-instance');
    fireEvent.click(screen.getByRole('button', { name: 'next-page' }));
    await waitFor(() => expect(fixture.list).toHaveBeenCalledWith({ page: 2, pageSize: 20 }, expect.any(AbortSignal)));
    await waitFor(() => expect(screen.getByTestId('current-page').textContent).toBe('2'));
    expect(screen.queryByText('remaining-instance')).toBeNull();
    expect(screen.queryByText('empty-dataset')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'previous-page' }));
    await screen.findByText('remaining-instance');
    expect(screen.getByTestId('current-page').textContent).toBe('1');
  });

  it('shows the dataset empty state when there are no instances on any page', async () => {
    fixture.list.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 });
    render(<ServiceInstanceList />);
    await screen.findByText('empty-dataset');
    expect(screen.queryByTestId('table')).toBeNull();
  });
});
