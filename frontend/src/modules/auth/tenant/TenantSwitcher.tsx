import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Dropdown, Menu, Message, Tag, Tooltip } from '@arco-design/web-react';
import { IconCheck, IconSwap } from '@arco-design/web-react/icon';
import type { TFunction } from 'i18next';
import { usePermission } from '../../../hooks/usePermission';
import { listSwitchableTenants, switchTenant, type SwitchableTenant } from './api';
import { usePlatformScope } from './usePlatformScope';

interface TenantSwitcherProps {
  /** Only meaningful under multi mode; the caller passes `tenantMode === 'multi'`. */
  enabled: boolean;
  t: TFunction;
}

/**
 * Header control that lets a multi-membership subject move between the tenants
 * it is an ACTIVE member of (contract §2.2). The list comes from
 * `/auth/tenants/switchable`, which only reports what the membership table
 * already grants — this control never widens scope.
 *
 * The control hides itself unless the account can actually reach at least two
 * tenants, so single-membership users never see a dead switcher.
 *
 * ## Why it also hides for platform-capable subjects
 *
 * A platform-capable subject is served by `TenantViewpointSwitcher` instead.
 * That control covers a strict superset — it can pin any active tenant, and it
 * is the only one of the two that offers a **platform viewpoint**, i.e. the
 * view an operator uses to manage the platform itself while multi mode is on.
 * This one cannot express that at all: its options come from
 * `tenant_memberships`, which the platform population usually has none of, and
 * choosing an entry re-issues the session bound to that tenant's *claim*. Under
 * the isolation model a claim resolves the role set inside the tenant, where a
 * platform operator owns none, so the result would be a console the operator
 * cannot use. Offering one control per subject kind is both clearer and safer.
 */
export function TenantSwitcher({ enabled, t }: Readonly<TenantSwitcherProps>) {
  const { isAdmin } = usePermission();
  const platformScope = usePlatformScope(enabled);
  const [tenants, setTenants] = useState<SwitchableTenant[]>([]);
  const [switching, setSwitching] = useState(false);
  // Same real scope signal as `TenantViewpointSwitcher`: a tenant
  // administrator also holds `system:tenant:list`, and the old proxy hid this
  // membership switcher from exactly the subject that needs it.
  const platformCapable = platformScope === 'platform' || isAdmin;

  useEffect(() => {
    // The render guard already hides the control when `enabled` is false, so
    // there is nothing to reset here — just skip the fetch. Platform-capable
    // subjects are skipped for the same reason: they are served by the
    // viewpoint control, which includes the platform view.
    // Wait for the scope signal: firing the membership list before it resolves
    // would send a request this control then immediately hides for.
    if (!enabled || platformCapable || platformScope === 'unknown') {
      return;
    }
    let cancelled = false;
    listSwitchableTenants()
      .then((resp) => {
        if (!cancelled) {
          setTenants(resp?.items ?? []);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setTenants([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [enabled, platformCapable, platformScope]);

  const current = useMemo(() => tenants.find((item) => item.isActive), [tenants]);

  const handleSwitch = useCallback(
    async (tenant: SwitchableTenant) => {
      if (tenant.isActive || switching) {
        return;
      }
      setSwitching(true);
      try {
        await switchTenant(tenant.id);
        Message.success(t('system.tenant.switch.success', { name: tenant.name }));
        // The session claim, roles and menu tree all change with the tenant, so a
        // full reload is the only way to guarantee a coherent shell.
        window.location.assign('/');
      } catch {
        setSwitching(false);
        Message.error(t('system.tenant.switch.error'));
      }
    },
    [switching, t],
  );

  if (!enabled || platformCapable || tenants.length < 2) {
    return null;
  }

  const droplist = (
    <Menu>
      {tenants.map((tenant) => (
        <Menu.Item key={String(tenant.id)} onClick={() => handleSwitch(tenant)}>
          <span className="tenant-switcher__item">
            <span className="tenant-switcher__item-name">{tenant.name}</span>
            {tenant.isActive ? <IconCheck /> : null}
          </span>
        </Menu.Item>
      ))}
    </Menu>
  );

  return (
    <Dropdown trigger="click" position="br" droplist={droplist}>
      <Tooltip content={t('system.tenant.switch.title')}>
        <Button
          type="text"
          className="app-shell__icon-btn tenant-switcher__trigger"
          loading={switching}
          aria-label={t('system.tenant.switch.title')}
        >
          <IconSwap />
          <span className="tenant-switcher__label">
            {current?.name ?? t('system.tenant.switch.empty')}
          </span>
          <Tag size="small" className="tenant-switcher__tag">
            {current?.code ?? '-'}
          </Tag>
        </Button>
      </Tooltip>
    </Dropdown>
  );
}
