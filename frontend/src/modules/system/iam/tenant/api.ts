import { apiRequest } from '../../../../api/request';

export type TenantStatus = 'active' | 'suspended' | 'archived' | 'deleted';
export type TenantPlan = 'free' | 'basic' | 'professional' | 'enterprise';
export type TenantMemberRole = 'owner' | 'admin' | 'member';

/** Lifecycle values an operator may set; `deleted` is reserved for the delete action. */
export type TenantEditableStatus = Exclude<TenantStatus, 'deleted'>;

export interface TenantRow {
  id: number;
  code: string;
  name: string;
  status: TenantStatus;
  plan: TenantPlan;
  metadata?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Per-status tenant counts served by the list endpoint. They apply every list
 * filter except `status`, so they stay consistent with the total no matter
 * which page is loaded.
 */
export interface TenantSummary {
  total: number;
  active: number;
  suspended: number;
  archived: number;
  deleted: number;
}

export interface TenantListQuery {
  keyword?: string;
  status?: TenantStatus;
  plan?: TenantPlan;
  page?: number;
  size?: number;
}

export interface TenantListResp {
  items: TenantRow[];
  total: number;
  page: number;
  size: number;
  summary?: TenantSummary;
}

export interface TenantPayload {
  code: string;
  name: string;
  plan?: TenantPlan;
  metadata?: string;
  /**
   * Initial tenant administrator. When `admin_username` is set the backend
   * provisions the tenant in one step: tenant-scoped roles, the admin user,
   * its role binding, the owner membership, the Casbin policies and a copy of
   * the platform settings. Omit it to create the tenant shell and add members
   * later.
   *
   * Wire fields are snake_case to match CreateTenantDTO in tenant_model.go.
   */
  admin_username?: string;
  admin_password?: string;
  admin_nickname?: string;
}

export interface TenantUpdatePayload {
  name?: string;
  status?: TenantEditableStatus;
  plan?: TenantPlan;
  metadata?: string;
}

export interface TenantMemberRow {
  id: number;
  tenant_id: number;
  user_id: number;
  username?: string;
  nickname?: string;
  role: TenantMemberRole;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface TenantMemberListResp {
  items: TenantMemberRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface TenantMemberPayload {
  user_id: number;
  role: TenantMemberRole;
}

export function getTenantList(params?: TenantListQuery) {
  return apiRequest<TenantListResp>({
    url: '/system/tenant/list',
    method: 'get',
    params,
  });
}

export function getTenant(id: number) {
  return apiRequest<TenantRow>({
    url: `/system/tenant/${id}`,
    method: 'get',
  });
}

export function createTenant(data: TenantPayload) {
  return apiRequest<TenantRow>({
    url: '/system/tenant',
    method: 'post',
    data,
  });
}

export function updateTenant(id: number, data: TenantUpdatePayload) {
  return apiRequest<Record<string, never>>({
    url: `/system/tenant/${id}`,
    method: 'put',
    data,
  });
}

export function deleteTenant(id: number) {
  return apiRequest<Record<string, never>>({
    url: `/system/tenant/${id}`,
    method: 'delete',
  });
}

export function getTenantMembers(id: number, params?: { page?: number; pageSize?: number }) {
  return apiRequest<TenantMemberListResp>({
    url: `/system/tenant/${id}/members`,
    method: 'get',
    params,
  });
}

export function addTenantMember(id: number, data: TenantMemberPayload) {
  return apiRequest<Record<string, never>>({
    url: `/system/tenant/${id}/members`,
    method: 'post',
    data,
  });
}

export function removeTenantMember(id: number, userId: number) {
  return apiRequest<Record<string, never>>({
    url: `/system/tenant/${id}/members/${userId}`,
    method: 'delete',
  });
}

/**
 * Tenant-ops console (contract §2.4/§2.5). Separate from the surfaces above:
 * these are platform-scoped reads that span tenants, so they live under
 * `/system/tenant-ops/*` and only a platform-capable subject can reach them.
 */

/** Quota thresholds a plan resolves to. `-1` means unlimited. */
export interface TenantQuota {
  max_users: number;
  max_roles: number;
  max_depts: number;
  max_api_keys: number;
  max_storage_mb: number;
}

export interface OverviewPlanRow {
  /** Raw value carried by `tenants.plan`; may be empty or unmanaged. */
  plan: string;
  name: string;
  /** False when the code resolves to no definition, so the page must not file it under free. */
  known: boolean;
  tenantCount: number;
  quota: TenantQuota;
}

export interface OverviewTenantRow {
  tenantId: number;
  code: string;
  name: string;
  status: TenantStatus;
  plan: string;
  planName: string;
  members: number;
  roles: number;
  depts: number;
  maxUsers: number;
  maxRoles: number;
  maxDepts: number;
  createdAt: string;
  updatedAt: string;
}

export interface OverviewResp {
  summary?: TenantSummary;
  totals: { members: number; roles: number; depts: number };
  plans: OverviewPlanRow[];
  items: OverviewTenantRow[];
  total: number;
  page: number;
  size: number;
}

export interface TenantMemberUsageRow {
  tenantId: number;
  tenantCode: string;
  tenantName: string;
  tenantStatus: TenantStatus;
  plan: string;
  planName: string;
  maxUsers: number;
  tenantUsage: number;
  userId: number;
  username: string;
  nickname: string;
  memberRole: TenantMemberRole;
  memberStatus: string;
  joinedAt: string;
}

export interface MemberUsageListResp {
  items: TenantMemberUsageRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface MemberUsageQuery {
  tenantId?: number;
  status?: 'active' | 'removed';
  role?: TenantMemberRole;
  keyword?: string;
  page?: number;
  pageSize?: number;
}

export type PlanStatus = 'active' | 'disabled';

export interface PlanRow {
  id: number;
  code: string;
  name: string;
  maxUsers: number;
  maxRoles: number;
  maxDepts: number;
  maxApiKeys: number;
  maxStorageMb: number;
  status: PlanStatus;
  sort: number;
  createdAt: string;
  updatedAt: string;
  tenantCount: number;
}

export interface PlanQuotaPayload {
  maxUsers?: number;
  maxRoles?: number;
  maxDepts?: number;
  maxApiKeys?: number;
  maxStorageMb?: number;
}

export interface PlanCreatePayload extends PlanQuotaPayload {
  code: string;
  name?: string;
  status?: PlanStatus;
  sort?: number;
}

export interface PlanUpdatePayload extends PlanQuotaPayload {
  name?: string;
  status?: PlanStatus;
  sort?: number;
}

/**
 * Overview filters. `plan` is any code the installation defines: plan codes are
 * DB-backed now, so the console must not restrict them to the built-in tiers.
 */
export interface TenantOverviewQuery {
  keyword?: string;
  status?: TenantStatus;
  plan?: string;
  page?: number;
  size?: number;
}

export function getTenantOverview(params?: TenantOverviewQuery) {
  return apiRequest<OverviewResp>({
    url: '/system/tenant-ops/overview',
    method: 'get',
    params,
  });
}

export function getAllTenantMembers(params?: MemberUsageQuery) {
  return apiRequest<MemberUsageListResp>({
    url: '/system/tenant-ops/member',
    method: 'get',
    params,
  });
}

/**
 * Plan definitions. `skipErrorMessage` is for callers that treat the list as
 * advisory (the tenant editor falls back to the built-in tiers when it cannot
 * read the table) and must not turn that fallback into a user-facing failure.
 */
export function getPlanList(options?: { skipErrorMessage?: boolean }) {
  return apiRequest<{ items: PlanRow[] }>({
    url: '/system/tenant-ops/plan/list',
    method: 'get',
    ...options,
  });
}

export function createPlan(data: PlanCreatePayload) {
  return apiRequest<PlanRow>({
    url: '/system/tenant-ops/plan',
    method: 'post',
    data,
  });
}

export function updatePlan(code: string, data: PlanUpdatePayload) {
  return apiRequest<PlanRow>({
    url: `/system/tenant-ops/plan/${encodeURIComponent(code)}`,
    method: 'put',
    data,
  });
}
