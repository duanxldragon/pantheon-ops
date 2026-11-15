import { randomUUID } from 'node:crypto';
import { expect, request, test, type APIRequestContext, type APIResponse } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  apiRequestHeaders,
  loginByApi,
  verifiedApiHeaders,
} from '../../helpers/auth';

async function success<T>(response: APIResponse): Promise<T> {
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code, payload.message).toBe(200);
  return payload.data as T;
}

async function forbidden(response: APIResponse) {
  const payload = await response.json();
  expect(response.status() === 403 || payload.code === 403).toBeTruthy();
}

const cases = [
  { name: 'view', permissions: ['business:cmdb:host:view'], canExport: false, canImport: false },
  { name: 'export', permissions: ['business:cmdb:host:export'], canExport: true, canImport: false },
  { name: 'import', permissions: ['business:cmdb:host:import'], canExport: false, canImport: true },
  {
    name: 'scoped',
    permissions: ['business:cmdb:host:export', 'business:cmdb:host:import'],
    canExport: true,
    canImport: true,
  },
];

for (const scenario of cases) {
  test(`CMDB ${scenario.name} permission keys enforce separate CSV capabilities and revoke the original session`, async () => {
    test.setTimeout(90_000);
    const admin = await request.newContext();
    const login = await loginByApi(admin, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `cmdb_csv_${scenario.name}_${randomUUID().slice(0, 12)}`;
    let roleId = 0;
    let userId = 0;
    const hostIds: number[] = [];
    let viewer: APIRequestContext | undefined;
    const cleanupFailures: string[] = [];
    try {
      const role = await success<{ id: number }>(
        await admin.post(`${apiBaseUrl}/system/role`, {
          headers,
          data: {
            roleName: token,
            roleKey: token,
            status: 1,
            sort: 99,
            menuIds: [],
            permissionKeys: scenario.permissions,
            dataScope: scenario.name === 'scoped' ? 'self' : 'all',
          },
        }),
      );
      roleId = role.id;
      const user = await success<{ id: number }>(
        await admin.post(`${apiBaseUrl}/system/user`, {
          headers,
          data: {
            username: token,
            password: 'ChangeMe123',
            nickname: token,
            status: 1,
            roleIds: [roleId],
          },
        }),
      );
      userId = user.id;
      viewer = await request.newContext();
      const viewerLogin = await loginByApi(viewer, { username: token, password: 'ChangeMe123' });
      const viewerHeaders = apiRequestHeaders(viewerLogin);
      const originalUser = await success<{ id: number; perms: string[] }>(
        await viewer.get(`${apiBaseUrl}/auth/me`, { headers: viewerHeaders }),
      );
      expect(originalUser.id).toBe(userId);
      expect(originalUser.perms.filter(key => key.startsWith('business:cmdb:')).sort()).toEqual([...scenario.permissions].sort());
      const exported = await viewer.get(`${apiBaseUrl}/business/cmdb/hosts/export`, {
        headers: viewerHeaders,
      });
      if (scenario.canExport) {
        expect(exported.ok()).toBeTruthy();
        expect(exported.headers()['content-type'], await exported.text()).toContain('text/csv');
        expect(await exported.text()).toContain('hostname');
      } else await forbidden(exported);
      const template = await viewer.get(`${apiBaseUrl}/business/cmdb/hosts/import-template`, {
        headers: viewerHeaders,
      });
      if (scenario.canImport) {
        expect(template.ok()).toBeTruthy();
        expect(template.headers()['content-type'], await template.text()).toContain('text/csv');
      } else await forbidden(template);
      // Invalid CSV exercises the authorized handler without creating host data.
      const imported = await viewer.post(`${apiBaseUrl}/business/cmdb/hosts/import`, {
        headers: viewerHeaders,
        multipart: {
          file: {
            name: 'invalid.csv',
            mimeType: 'text/csv',
            buffer: Buffer.from('hostname,ip,os\r\n,198.51.100.90,linux\r\n'),
          },
        },
      });
      if (scenario.canImport) {
        const result = await success<{ applied: boolean; failed: number }>(imported);
        expect(result.applied).toBe(false);
        expect(result.failed).toBeGreaterThan(0);
      } else await forbidden(imported);
      if (scenario.name === 'scoped') {
        const adminHost = await success<{ id: number }>(
          await admin.post(`${apiBaseUrl}/business/cmdb/hosts`, {
            headers,
            data: { hostname: `${token}-admin`, ip: '198.51.100.191', os: 'linux' },
          }),
        );
        hostIds.push(adminHost.id);
        const upload = (hostname: string, ip: string) =>
          viewer!.post(`${apiBaseUrl}/business/cmdb/hosts/import`, {
            headers: viewerHeaders,
            multipart: {
              file: {
                name: 'scoped.csv',
                mimeType: 'text/csv',
                buffer: Buffer.from(`hostname,ip,os\r\n${hostname},${ip},linux\r\n`),
              },
            },
          });
        const ownImport = await success<{ applied: boolean; failed: number }>(
          await upload(`${token}-own`, '198.51.100.192'),
        );
        expect(ownImport.applied).toBe(true);
        expect(ownImport.failed).toBe(0);
        const ownHosts = await success<{ items: Array<{ id: number }> }>(
          await admin.get(`${apiBaseUrl}/business/cmdb/hosts?keyword=${token}-own`, { headers }),
        );
        expect(ownHosts.items).toHaveLength(1);
        hostIds.push(ownHosts.items[0].id);
        const scopedExport = await viewer!.get(
          `${apiBaseUrl}/business/cmdb/hosts/export?keyword=${token}`,
          { headers: viewerHeaders },
        );
        const csv = await scopedExport.text();
        expect(csv).toContain(`${token}-own`);
        expect(csv).not.toContain(`${token}-admin`);
        const overwrite = await upload(`${token}-overwrite`, '198.51.100.191');
        const denied = await overwrite.json();
        expect(denied.code !== 200 || denied.data?.applied === false).toBe(true);
        const retained = await success<{ hostname: string }>(
          await admin.get(`${apiBaseUrl}/business/cmdb/hosts/${adminHost.id}`, { headers }),
        );
        expect(retained.hostname).toBe(`${token}-admin`);
      }

      // Grant and revoke go through the public role contract only. Reusing this
      // exact login warms /auth/me first and detects stale released permissions
      // without creating a replacement session or adding manual API policies.
      const revokedRole = await success<{ id: number; permissionKeys: string[] }>(
        await admin.put(`${apiBaseUrl}/system/role/${roleId}`, {
          headers: await verifiedApiHeaders(admin, login),
          data: {
            roleName: token, roleKey: token, status: 1, sort: 99,
            menuIds: [], permissionKeys: [], dataScope: scenario.name === 'scoped' ? 'self' : 'all',
          },
        }),
      );
      expect(revokedRole.id).toBe(roleId);
      expect(revokedRole.permissionKeys).toEqual([]);
      const roles = await success<{ items: Array<{ id: number; permissionKeys: string[] }> }>(
        await admin.get(`${apiBaseUrl}/system/role/list`, {
          headers, params: { roleKey: token, page: 1, pageSize: 10 },
        }),
      );
      const persistedRole = roles.items.find(row => row.id === roleId);
      expect(persistedRole).toBeDefined();
      expect(persistedRole?.permissionKeys).toEqual([]);
      const revokedUser = await success<{ id: number; perms: string[] }>(
        await viewer.get(`${apiBaseUrl}/auth/me`, { headers: viewerHeaders }),
      );
      expect(revokedUser.id).toBe(userId);
      expect(revokedUser.perms.filter(key => key.startsWith('business:cmdb:'))).toEqual([]);
      await forbidden(await viewer.get(`${apiBaseUrl}/business/cmdb/hosts`, { headers: viewerHeaders }));
      await forbidden(await viewer.get(`${apiBaseUrl}/business/cmdb/hosts/export`, { headers: viewerHeaders }));
      await forbidden(await viewer.get(`${apiBaseUrl}/business/cmdb/hosts/import-template`, { headers: viewerHeaders }));
      await forbidden(await viewer.post(`${apiBaseUrl}/business/cmdb/hosts/import`, {
        headers: viewerHeaders,
        multipart: {
          file: { name: 'revoked.csv', mimeType: 'text/csv', buffer: Buffer.from('hostname,ip,os\r\n,198.51.100.90,linux\r\n') },
        },
      }));
    } finally {
      await viewer?.dispose();
      const cleanupHeaders = await verifiedApiHeaders(admin, login);
      for (const id of hostIds) {
        const response = await admin.delete(`${apiBaseUrl}/business/cmdb/hosts/${id}`, {
          headers: cleanupHeaders,
        });
        const payload = await response.json();
        if (!response.ok() || payload.code !== 200)
          cleanupFailures.push(`host: ${payload.message ?? response.status()}`);
      }
      for (const [kind, id] of [
        ['user', userId],
        ['role', roleId],
      ] as const) {
        if (!id) continue;
        const response = await admin.delete(`${apiBaseUrl}/system/${kind}/${id}`, {
          headers: cleanupHeaders,
        });
        const payload = await response.json();
        if (!response.ok() || payload.code !== 200)
          cleanupFailures.push(`${kind}: ${payload.message ?? response.status()}`);
      }
      await admin.dispose();
      expect(cleanupFailures).toEqual([]);
    }
  });
}
