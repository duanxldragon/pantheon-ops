import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Dropdown, Menu, Tag, Tooltip } from '@arco-design/web-react';
import { IconClose, IconCheck, IconEye } from '@arco-design/web-react/icon';
import type { TFunction } from 'i18next';
import { usePermission } from '../../../hooks/usePermission';
import {
  readTenantViewpointId,
  writeTenantViewpointId,
} from '../../../core/tenant/viewpoint';
import { usePlatformScope } from './usePlatformScope';
import { getTenant, getTenantList, type TenantRow } from '../../system/iam/tenant/api';

interface TenantViewpointSwitcherProps {
  /** Only meaningful under multi mode; the caller passes `tenantMode === 'multi'`. */
  enabled: boolean;
  t: TFunction;
}

/**
 * Scope control for platform-capable subjects (contract §3.1 source 1).
 *
 * Unlike `TenantSwitcher` — which moves a subject between tenants it is a
 * *member* of via a session re-issue — this control pins a working tenant
 * scope onto the current session through the `X-Tenant-Id` header. It exists
 * because the platform subject (tenant 0) usually holds no membership, so the
 * membership-based switcher never lists anything for it, yet the operator
 * still needs to reach "user / role / dept / post in tenant A".
 *
 * Confinement is not the frontend's job: the backend accepts the header only
 * from a platform-capable subject and validates the tenant against the master
 * row, refusing every other combination with 403 (tenant.forbidden).
 *
 * The control hides itself unless the subject is platform-capable (the same
 * surface the tenant console uses) and at least one active tenant exists.
 * Switching is a full shell reload — the same pattern `TenantSwitcher` uses —
 * so menus/permissions/caches regenerate for the pinned scope.
 */
export function TenantViewpointSwitcher({ enabled, t }: Readonly<TenantViewpointSwitcherProps>) {
  const { isAdmin } = usePermission();
  const platformScope = usePlatformScope(enabled);
  const [tenants, setTenants] = useState<TenantRow[]>([]);
  // Read once per mount: switching always reloads the shell, so a live view of
  // the persisted value would never legitimately change mid-mount.
  const currentViewpointId = useMemo(() => readTenantViewpointId(), []);

  // Real scope signal, not a permission key: a tenant administrator holds
  // `system:tenant:list` too, and using it here fired GET /system/tenant/list
  // on every page only to receive 403 tenant.platformOnly (acceptance F5).
  // `isAdmin` covers a global admin whose session also carries a tenant claim.
  const platformCapable = enabled && (platformScope === 'platform' || isAdmin);

  useEffect(() => {
    if (!platformCapable) {
      return;
    }
    let cancelled = false;
    getTenantList({ status: 'active', page: 1, size: 100 })
      .then((resp) => {
        if (!cancelled) {
          setTenants(resp?.items ?? []);
        }
      })
      .catch(() => {
        // A non-platform subject can still satisfy the permission proxy but
        // is rejected by the console's capability gate with 403; hiding the
        // switcher is the correct reaction, and the header it would attach
        // would be refused server-side anyway (tenant.forbidden).
        if (!cancelled) {
          setTenants([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [platformCapable]);

  const visible = platformCapable && tenants.length > 0;

  const handleSelect = useCallback(
    (viewpointId: number) => {
      if (viewpointId === currentViewpointId) {
        return;
      }
      writeTenantViewpointId(viewpointId);
      // Same rationale as TenantSwitcher: the working scope changes for every
      // subsequent request of the shell, so a full reload is the only way to
      // guarantee a coherent, cache-free state.
      window.location.assign('/');
    },
    [currentViewpointId],
  );

  if (!visible) {
    return null;
  }

  const droplist = (
    <Menu>
      <Menu.Item key="platform" onClick={() => handleSelect(0)}>
        <span className="tenant-switcher__item">
          <span className="tenant-switcher__item-name">{t('system.tenant.viewpoint.platform')}</span>
          {currentViewpointId === null || currentViewpointId === 0 ?          <IconCheck /> : null}
        </span>
      </Menu.Item>
      {tenants.map((tenant) => (
        <Menu.Item key={String(tenant.id)} onClick={() => handleSelect(tenant.id)}>
          <span className="tenant-switcher__item">
            <span className="tenant-switcher__item-name">{tenant.name}</span>
            {currentViewpointId === tenant.id ?          <IconCheck /> : null}
          </span>
        </Menu.Item>
      ))}
    </Menu>
  );

  // `currentViewpointId` never names a tenant missing from the fetched list
  // unless the tenant became inactive mid-session; the backend then refuses
  // every scoped call with 403 and the operator picks an entry above to leave.
  const currentTenant = tenants.find((tenant) => tenant.id === currentViewpointId);
  const label = currentTenant ? currentTenant.name : t('system.tenant.viewpoint.platform');
  const tagCode = currentTenant ? currentTenant.code : '0';

  return (
    <Dropdown trigger="click" position="br" droplist={droplist}>
      <Tooltip content={t('system.tenant.viewpoint.title')}>
        <Button
          type="text"
          className="app-shell__icon-btn tenant-switcher__trigger"
          aria-label={t('system.tenant.viewpoint.title')}
        >
          <IconEye />
          <span className="tenant-switcher__label">{label}</span>
          <Tag size="small" color="arcoblue" className="tenant-switcher__tag">
            {tagCode}
          </Tag>
        </Button>
      </Tooltip>
    </Dropdown>
  );
}

/**
 * Persistent strip rendered by the shell while a tenant viewpoint (id > 0) is
 * pinned. Its job is fail-safe context: an operator landing on /system/user
 * must know that every list/create acts inside the pinned tenant instead of
 * the platform population. It resolves the tenant name once per mount and
 * leaves silently if the viewpoint disappears or the tenant cannot be read.
 */
interface TenantViewpointBannerProps {
  t: TFunction;
}

export function TenantViewpointBanner({ t }: Readonly<TenantViewpointBannerProps>) {
  const viewpointId = useMemo(() => readTenantViewpointId(), []);
  const [name, setName] = useState<string | null>(null);

  useEffect(() => {
    if (viewpointId === null || viewpointId <= 0) {
      return;
    }
    let cancelled = false;
    getTenant(viewpointId)
      .then((tenant) => {
        if (!cancelled) {
          setName(tenant?.name ?? String(viewpointId));
        }
      })
      .catch(() => {
        // Tenant missing/suspended/archived or no console permission: keep
        // the banner with the raw id so the operator can leave the scope;
        // the backend 403s the scoped calls in either case.
        if (!cancelled) {
          setName(String(viewpointId));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [viewpointId]);

  if (viewpointId === null || viewpointId <= 0) {
    return null;
  }

  return (
    <output className="tenant-viewpoint-banner">
      <Tag color="arcoblue">{t('system.tenant.viewpoint.title')}</Tag>
      <span className="tenant-viewpoint-banner__text">
        {t('system.tenant.viewpoint.banner', { name: name ?? '#' + viewpointId })}
      </span>
      <Button
        type="text"
        size="mini"
        icon={<IconClose />}
        onClick={() => {
          writeTenantViewpointId(null);
          window.location.assign('/');
        }}
      >
        {t('system.tenant.viewpoint.exit')}
      </Button>
    </output>
  );
}
