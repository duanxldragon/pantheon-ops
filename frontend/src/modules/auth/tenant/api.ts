import { apiRequest } from '../../../api/request';

/**
 * One tenant the authenticated subject may switch into. Mirrors the backend
 * `tenantSummary` wire shape (modules/auth/login/login_tenant_switch.go).
 */
export interface SwitchableTenant {
  id: number;
  code: string;
  name: string;
  status: string;
  plan: string;
  role: string;
  /** True when this tenant is the one the current request resolved to. */
  isActive: boolean;
}

export interface SwitchableTenantResp {
  items: SwitchableTenant[];
}

export interface CurrentTenantResp {
  mode: string;
  tenantId: number;
  resolvedBy: string;
  tenant?: Omit<SwitchableTenant, 'role' | 'isActive'>;
}

export interface SwitchTenantResp {
  tenantId: number;
  sessionId: string;
  csrfToken: string;
  tenant?: Omit<SwitchableTenant, 'role' | 'isActive'>;
}

/** GET /auth/tenants/switchable — the caller's ACTIVE memberships (contract §2.2). */
export function listSwitchableTenants() {
  return apiRequest<SwitchableTenantResp>({
    url: '/auth/tenants/switchable',
    method: 'get',
  });
}

/** GET /auth/tenants/current — the tenant context this request resolved to (contract §3.1). */
export function getCurrentTenant() {
  return apiRequest<CurrentTenantResp>({
    url: '/auth/tenants/current',
    method: 'get',
  });
}

/** POST /auth/tenants/switch — re-issue the session bound to another tenant. */
export function switchTenant(tenantId: number) {
  return apiRequest<SwitchTenantResp>({
    url: '/auth/tenants/switch',
    method: 'post',
    data: { tenantId },
  });
}
