import { Children, cloneElement, isValidElement, type ReactElement, type ReactNode } from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import InstanceCreateModal from '../../../../src/modules/business/serviceinstance/InstanceCreateModal';

const fixture = vi.hoisted(() => {
  const values: Record<string, unknown> = {};
  return {
    values,
    form: {
      setFieldsValue: (patch: Record<string, unknown>) => Object.assign(values, patch),
      validate: async () => ({ ...values }),
    },
    services: vi.fn(),
    clusters: vi.fn(),
    hosts: vi.fn(),
    create: vi.fn(),
  };
});

vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('../../../../src/api/request', () => ({ isForbiddenRequestError: () => false }));
vi.mock('../../../../src/modules/business/service/api', () => ({
  getServiceList: fixture.services, createServiceInstance: fixture.create,
}));
vi.mock('../../../../src/modules/business/k8s/api', () => ({ getK8sClusterList: fixture.clusters }));
vi.mock('../../../../src/modules/business/cmdb/host/api', () => ({ getHostList: fixture.hosts }));
vi.mock('../../../../src/components', () => ({
  AppModal: ({ children, onOk, okButtonProps }: { children: ReactNode; onOk: () => void; okButtonProps: { disabled: boolean } }) => <div>{children}<button onClick={onOk} disabled={okButtonProps.disabled}>save</button></div>,
  PageLoading: () => <span>loading</span>,
  PageError: () => <span>failed</span>,
  PageForbidden: () => <span>forbidden</span>,
}));
vi.mock('@arco-design/web-react', () => {
  const Form = Object.assign(({ children }: { children: ReactNode }) => <div>{children}</div>, {
    useForm: () => [fixture.form],
    Item: ({ children, field }: { children: ReactNode; field: string }) => {
      const child = Children.only(children);
      if (!isValidElement(child)) return null;
      const element = child as ReactElement<{ onChange?: (value: string | number) => void }>;
      return cloneElement(element, {
        'data-testid': field,
        value: fixture.values[field] ?? '',
        onChange: (value: string | number) => {
          fixture.values[field] = value;
          element.props.onChange?.(value);
        },
      } as Record<string, unknown>);
    },
  });
  const Select = Object.assign(({ children, value, onChange, disabled, 'data-testid': testId }: { children: ReactNode; value: string | number; onChange: (value: string | number) => void; disabled?: boolean; 'data-testid': string }) => <select data-testid={testId} value={value} disabled={disabled} onChange={(event) => onChange(Number(event.target.value) || event.target.value)}><option value="" />{children}</select>, {
    Option: ({ children, value }: { children: ReactNode; value: string | number }) => <option value={value}>{children}</option>,
  });
  return {
    Form, Select, Message: { success: vi.fn(), error: vi.fn() },
    Input: ({ value, onChange, 'data-testid': testId }: { value: string; onChange: (value: string) => void; 'data-testid': string }) => <input data-testid={testId} value={value} onChange={(event) => onChange(event.target.value)} />,
  };
});

function deferred() {
  let resolve!: (value: { items: { id: number; name: string }[] }) => void;
  const promise = new Promise<{ items: { id: number; name: string }[] }>((done) => { resolve = done; });
  return { promise, resolve };
}

beforeEach(() => {
  vi.clearAllMocks();
  fixture.clusters.mockReset();
  for (const key of Object.keys(fixture.values)) delete fixture.values[key];
  fixture.services.mockResolvedValue({ items: [
    { id: 1, businessScopeId: 31, applicationName: 'app-a', name: 'service-a', runtimeType: 'k8s' },
    { id: 2, businessScopeId: 42, applicationName: 'app-b', name: 'service-b', runtimeType: 'k8s' },
  ] });
});
afterEach(cleanup);

async function mount() {
  render(<InstanceCreateModal onClose={vi.fn()} onCreated={vi.fn()} />);
  await screen.findByTestId('serviceId');
}

describe('instance Kubernetes scope selection', () => {
  it('requests only clusters in the selected service scope and clears the previous target when switching', async () => {
    const second = deferred();
    fixture.clusters.mockResolvedValueOnce({ items: [{ id: 101, name: 'scope-a-cluster' }] }).mockReturnValueOnce(second.promise);
    await mount();
    fireEvent.change(screen.getByTestId('serviceId'), { target: { value: '1' } });
    await screen.findByRole('option', { name: 'scope-a-cluster' });
    expect(fixture.clusters).toHaveBeenNthCalledWith(1, { businessScopeId: 31, page: 1, pageSize: 100 });
    fireEvent.change(screen.getByTestId('k8sClusterId'), { target: { value: '101' } });
    expect(fixture.values.k8sClusterId).toBe(101);
    fireEvent.change(screen.getByTestId('serviceId'), { target: { value: '2' } });
    expect(fixture.values.k8sClusterId).toBeUndefined();
    expect((screen.getByRole('button', { name: 'save' }) as HTMLButtonElement).disabled).toBe(true);
    await waitFor(() => expect(fixture.clusters).toHaveBeenNthCalledWith(2, { businessScopeId: 42, page: 1, pageSize: 100 }));
    expect(screen.queryByRole('option', { name: 'scope-a-cluster' })).toBeNull();
    await act(async () => second.resolve({ items: [{ id: 202, name: 'scope-b-cluster' }] }));
    await screen.findByRole('option', { name: 'scope-b-cluster' });
    expect((screen.getByTestId('k8sClusterId') as HTMLSelectElement).value).toBe('');
    expect(screen.queryByRole('option', { name: 'scope-a-cluster' })).toBeNull();
  });

  it('ignores a late cluster response from the previous service scope', async () => {
    const first = deferred();
    fixture.clusters.mockReturnValueOnce(first.promise).mockResolvedValueOnce({ items: [{ id: 202, name: 'scope-b-cluster' }] });
    await mount();
    fireEvent.change(screen.getByTestId('serviceId'), { target: { value: '1' } });
    await waitFor(() => expect(fixture.clusters).toHaveBeenCalledTimes(1));
    fireEvent.change(screen.getByTestId('serviceId'), { target: { value: '2' } });
    await screen.findByRole('option', { name: 'scope-b-cluster' });
    await act(async () => first.resolve({ items: [{ id: 101, name: 'late-scope-a-cluster' }] }));
    expect(screen.queryByRole('option', { name: 'late-scope-a-cluster' })).toBeNull();
    expect(screen.getByRole('option', { name: 'scope-b-cluster' })).toBeTruthy();
  });
});
