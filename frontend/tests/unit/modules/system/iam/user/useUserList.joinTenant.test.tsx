import type { FormInstance } from '@arco-design/web-react/es/Form';
import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { UserCreatePayload } from '../../../../../../src/modules/system/iam/user/api';
import type { TenantMemberRole } from '../../../../../../src/modules/system/iam/tenant/api';

/** Create payload plus the form-only fields the join-tenant section adds. */
type FormValues = UserCreatePayload & {
  joinTenant?: boolean;
  membershipRole?: TenantMemberRole;
};

const {
  addTenantMemberMock,
  createUserMock,
  getUserListMock,
  successMock,
  errorMock,
  // A stable `t` identity matters: the hook memoises its loaders on `t`, so a
  // fresh function per render would re-fire every load effect on each render.
  stableT,
} = vi.hoisted(() => ({
  addTenantMemberMock: vi.fn(),
  createUserMock: vi.fn(),
  getUserListMock: vi.fn(),
  successMock: vi.fn(),
  errorMock: vi.fn(),
  stableT: (key: string) => key,
}));

vi.mock('../../../../../../src/components/feedback/message', () => ({
  message: {
    success: successMock,
    error: errorMock,
    warning: vi.fn(),
    info: vi.fn(),
  },
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: stableT, i18n: { language: 'zh-CN' } }),
}));

vi.mock('../../../../../../src/api/importExport', () => ({
  showImportResult: vi.fn(),
  uploadImportFile: vi.fn(),
}));

// The route warm-data cache is transparent to this test: always call through
// to the loader so the mocked API stays the single source of truth.
vi.mock('../../../../../../src/core/router/prefetch', () => ({
  resolveRouteWarmData: (_path: string, _key: string, loader: () => unknown) => loader(),
  invalidateRouteWarmDataMany: vi.fn(),
}));

vi.mock('../../../../../../src/core/refresh/refreshBus', () => ({
  publishRefresh: vi.fn(),
  useRefreshSubscription: vi.fn(),
}));

vi.mock('../../../../../../src/core/setting/publicSettings', () => ({
  usePublicSettings: () => ({ orgEnabled: false, orgRequiredForUser: false }),
}));

vi.mock('../../../../../../src/modules/system/iam/user/api', () => ({
  batchDeleteUsers: vi.fn(),
  batchUpdateUserStatus: vi.fn(),
  createUser: createUserMock,
  downloadUserImportTemplate: vi.fn(),
  exportUsers: vi.fn(),
  getUserDetail: vi.fn(),
  getUserList: getUserListMock,
  importUsers: vi.fn(),
  resetUserPassword: vi.fn(),
  updateUser: vi.fn(),
}));

vi.mock('../../../../../../src/modules/system/iam/tenant/api', () => ({
  addTenantMember: addTenantMemberMock,
}));

vi.mock('../../../../../../src/modules/system/iam/role/api', () => ({
  getRoleList: vi.fn(async () => ({ items: [], total: 0 })),
}));

vi.mock('../../../../../../src/modules/system/org/dept/api', () => ({
  getDeptTree: vi.fn(async () => []),
}));

vi.mock('../../../../../../src/modules/system/org/post/api', () => ({
  getPostList: vi.fn(async () => ({ items: [], total: 0 })),
}));

import { useUserList } from '../../../../../../src/modules/system/iam/user/useUserList';
import {
  clearTenantViewpoint,
  writeTenantViewpointId,
} from '../../../../../../src/core/tenant/viewpoint';

const createFormValues: FormValues = {
  username: 'ada',
  password: 'Secret#123',
  nickname: 'Ada',
  email: 'ada@example.com',
  phone: '',
  avatar: '',
  status: 1,
  roleIds: [],
  deptId: 0,
  postId: 0,
};

function createFormStub(values: FormValues) {
  return {
    validate: vi.fn(async () => values),
    setFieldsValue: vi.fn(),
    setFieldValue: vi.fn(),
    getFieldValue: vi.fn(() => 0),
    submit: vi.fn(),
  } as unknown as FormInstance<UserCreatePayload>;
}

function renderUserList(values: FormValues) {
  // The viewpoint is read once per mount, so it must be stored beforehand.
  return renderHook(() => useUserList(createFormStub(values)));
}

/** Waits for the mount-time loaders (list, roles, dept/post) to settle. */
async function settleMountLoads() {
  await act(async () => {});
  await waitFor(() => {
    expect(getUserListMock).toHaveBeenCalled();
  });
  await act(async () => {});
}

beforeEach(() => {
  sessionStorage.clear();
  clearTenantViewpoint();
  vi.clearAllMocks();
  getUserListMock.mockResolvedValue({ items: [], total: 0 });
  createUserMock.mockResolvedValue({ id: 77 });
  addTenantMemberMock.mockResolvedValue({});
});

describe('useUserList — create user inside a pinned tenant viewpoint', () => {
  it('adds the created user to the pinned tenant with the default member role', async () => {
    writeTenantViewpointId(9);
    const { result } = renderUserList({ ...createFormValues, joinTenant: true, membershipRole: 'member' });

    await act(async () => {
      await result.current.submitForm();
    });

    // The membership fields are form-only: they must never reach the user API.
    expect(createUserMock).toHaveBeenCalledWith(
      expect.objectContaining({ username: 'ada', password: 'Secret#123' }),
    );
    const createPayload = createUserMock.mock.calls[0][0] as Record<string, unknown>;
    expect(createPayload).not.toHaveProperty('joinTenant');
    expect(createPayload).not.toHaveProperty('membershipRole');

    expect(addTenantMemberMock).toHaveBeenCalledTimes(1);
    expect(addTenantMemberMock).toHaveBeenCalledWith(9, { user_id: 77, role: 'member' });
    expect(successMock).toHaveBeenCalledWith('system.user.joinTenantDone');
  });

  it('passes the chosen membership role through', async () => {
    writeTenantViewpointId(9);
    const { result } = renderUserList({ ...createFormValues, joinTenant: true, membershipRole: 'admin' });

    await act(async () => {
      await result.current.submitForm();
    });

    expect(addTenantMemberMock).toHaveBeenCalledWith(9, { user_id: 77, role: 'admin' });
  });

  it('skips the membership call when the operator opts out', async () => {
    writeTenantViewpointId(9);
    const { result } = renderUserList({ ...createFormValues, joinTenant: false, membershipRole: 'member' });

    await act(async () => {
      await result.current.submitForm();
    });

    expect(createUserMock).toHaveBeenCalledTimes(1);
    expect(addTenantMemberMock).not.toHaveBeenCalled();
  });

  it('never calls the membership endpoint without a pinned tenant', async () => {
    const { result } = renderUserList({ ...createFormValues, joinTenant: true, membershipRole: 'member' });

    await act(async () => {
      await result.current.submitForm();
    });

    expect(createUserMock).toHaveBeenCalledTimes(1);
    expect(addTenantMemberMock).not.toHaveBeenCalled();
  });

  it('does not pin the platform-global viewpoint onto the membership call', async () => {
    // Viewpoint 0 means "platform scope": there is no tenant to join.
    writeTenantViewpointId(0);
    const { result } = renderUserList({ ...createFormValues, joinTenant: true, membershipRole: 'member' });

    await act(async () => {
      await result.current.submitForm();
    });

    expect(addTenantMemberMock).not.toHaveBeenCalled();
  });

  it('keeps the created user when the membership call fails, and reports it', async () => {
    writeTenantViewpointId(9);
    addTenantMemberMock.mockRejectedValue(new Error('membership unavailable'));
    const { result } = renderUserList({ ...createFormValues, joinTenant: true, membershipRole: 'member' });

    await act(async () => {
      await result.current.submitForm();
    });

    // Creation succeeded and must not be rolled back by the integration step.
    expect(successMock).toHaveBeenCalledWith('common.createSuccess');
    expect(errorMock).toHaveBeenCalledWith('system.tenant.viewpoint.joinMembershipFailed');
    expect(errorMock).not.toHaveBeenCalledWith('common.actionFailed');
    expect(result.current.state.formVisible).toBe(false);
  });

  it('skips the membership call when the create response carries no usable id', async () => {
    writeTenantViewpointId(9);
    createUserMock.mockResolvedValue(undefined);
    const { result } = renderUserList({ ...createFormValues, joinTenant: true, membershipRole: 'member' });

    await act(async () => {
      await result.current.submitForm();
    });

    expect(addTenantMemberMock).not.toHaveBeenCalled();
  });

  it('exposes the pinned tenant id so the form can offer the join section', async () => {
    writeTenantViewpointId(9);
    const { result } = renderUserList(createFormValues);
    await settleMountLoads();

    expect(result.current.viewpointTenantId).toBe(9);
  });

  it('reports no viewpoint for a platform subject with no pinned tenant', async () => {
    const { result } = renderUserList(createFormValues);
    await settleMountLoads();

    expect(result.current.viewpointTenantId).toBeNull();
  });
});
