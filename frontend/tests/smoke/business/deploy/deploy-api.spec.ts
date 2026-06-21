import { expect, request as playwrightRequest, test, type APIRequestContext, type APIResponse } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  loginByApi,
  type BrowserLoginResult,
} from '../../helpers/auth';

type ApiEnvelope<T> = {
  code: number;
  data: T;
  message?: string;
};

type HostRow = {
  id: number;
  hostname: string;
  ip: string;
  status: string;
};

type BizScopeRow = {
  id: number;
};

type PackageRow = {
  id: number;
  name: string;
  version: string;
  sourceFileName?: string;
};

type TaskRow = {
  id: number;
  name?: string;
  status: string;
  packageId?: number;
  packageName?: string;
  businessScopeId?: number;
  businessScopeName?: string;
  targetIds?: number[];
  executorType?: string;
  action?: string;
  remark?: string;
  hosts: Array<{
    id: number;
    status: string;
    hostId: number;
    stdout?: string;
    traceSteps?: Array<{ phase?: string; message?: string }>;
  }>;
};

type UploadResult = {
  objectKey: string;
  originalName: string;
  url: string;
};

async function expectSuccess<T>(response: APIResponse): Promise<T> {
  expect(response.ok()).toBeTruthy();
  const payload = await response.json() as ApiEnvelope<T>;
  expect(payload.code).toBe(200);
  return payload.data;
}

test.describe.serial('Deploy business api smoke', () => {
  let login: BrowserLoginResult;
  let apiContext: APIRequestContext;

  test.beforeAll(async () => {
    const loginContext = await playwrightRequest.newContext();
    login = await loginByApi(loginContext, adminCredentials);
    apiContext = await playwrightRequest.newContext({
      extraHTTPHeaders: {
        Authorization: `Bearer ${login.accessToken}`,
        'X-CSRF-Token': login.csrfToken,
        Cookie: `pantheon_csrf_token=${login.csrfToken}`,
      },
    });
    await loginContext.dispose();
  });

  test.afterAll(async () => {
    await apiContext?.dispose();
  });

  test('deploy api creates starts and marks a manual task', async () => {
    const token = `deploy-smoke-${Date.now()}`;
    const scope = await expectSuccess<BizScopeRow>(
      await apiContext.post(`${apiBaseUrl}/business/bizscope`, {
        data: {
          code: token,
          name: token,
          environment: 'prod',
          status: 'active',
        },
      }),
    );

    const host = await expectSuccess<HostRow>(
      await apiContext.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        data: {
          hostname: token,
          ip: `10.245.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
          os: 'linux',
          businessScopeId: scope.id,
          labels: [{ key: 'biz', val: token }],
        },
      }),
    );
    expect(host.status).toBe('assigned');

    const pkg = await expectSuccess<PackageRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/packages`, {
        data: {
          name: token,
          version: '1.0.0',
          installCommand: 'echo install',
          status: 'enabled',
        },
      }),
    );

    const task = await expectSuccess<TaskRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/tasks`, {
        data: {
          name: token,
          packageId: pkg.id,
          businessScopeId: scope.id,
          targetType: 'host',
          targetIds: [host.id],
          executorType: 'manual',
        },
      }),
    );
    expect(task.status).toBe('draft');

    const started = await expectSuccess<TaskRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/tasks/${task.id}/start`),
    );
    expect(started.status).toBe('running');
    expect(started.hosts).toHaveLength(1);

    await expectSuccess(
      await apiContext.post(`${apiBaseUrl}/business/deploy/task-hosts/${started.hosts[0].id}/result`, {
        data: { status: 'success', stdout: 'installed' },
      }),
    );

    const detail = await expectSuccess<TaskRow>(
      await apiContext.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`),
    );
    expect(detail.status).toBe('success');

    const hostDetail = await expectSuccess<HostRow>(
      await apiContext.get(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`),
    );
    expect(hostDetail.status).toBe('online');

    await apiContext.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`).catch(() => undefined);
    await apiContext.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`).catch(() => undefined);
    await apiContext.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`).catch(() => undefined);
  });

  test('deploy api updates a draft task before execution', async () => {
    const token = `deploy-update-${Date.now()}`;

    const scopeA = await expectSuccess<BizScopeRow>(
      await apiContext.post(`${apiBaseUrl}/business/bizscope`, {
        data: {
          code: `${token}-a`,
          name: `${token}-scope-a`,
          environment: 'dev',
          status: 'active',
        },
      }),
    );
    const scopeB = await expectSuccess<BizScopeRow>(
      await apiContext.post(`${apiBaseUrl}/business/bizscope`, {
        data: {
          code: `${token}-b`,
          name: `${token}-scope-b`,
          environment: 'prod',
          status: 'active',
        },
      }),
    );

    const hostA = await expectSuccess<HostRow>(
      await apiContext.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        data: {
          hostname: `${token}-host-a`,
          ip: `10.249.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
          os: 'linux',
          businessScopeId: scopeA.id,
          labels: [{ key: 'biz', val: `${token}-a` }],
        },
      }),
    );
    const hostB = await expectSuccess<HostRow>(
      await apiContext.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        data: {
          hostname: `${token}-host-b`,
          ip: `10.250.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
          os: 'linux',
          businessScopeId: scopeB.id,
          labels: [{ key: 'biz', val: `${token}-b` }],
        },
      }),
    );

    const pkgA = await expectSuccess<PackageRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/packages`, {
        data: {
          name: `${token}-pkg-a`,
          version: '1.0.0',
          installCommand: 'echo install a',
          status: 'enabled',
        },
      }),
    );
    const pkgB = await expectSuccess<PackageRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/packages`, {
        data: {
          name: `${token}-pkg-b`,
          version: '2.0.0',
          installCommand: 'echo install b',
          status: 'enabled',
        },
      }),
    );

    const task = await expectSuccess<TaskRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/tasks`, {
        data: {
          name: `${token}-task`,
          packageId: pkgA.id,
          businessScopeId: scopeA.id,
          targetType: 'host',
          targetIds: [hostA.id],
          executorType: 'manual',
        },
      }),
    );
    expect(task.status).toBe('draft');

    const updated = await expectSuccess<TaskRow>(
      await apiContext.put(`${apiBaseUrl}/business/deploy/tasks/${task.id}`, {
        data: {
          name: `${token}-task-updated`,
          packageId: pkgB.id,
          businessScopeId: scopeB.id,
          targetType: 'host',
          targetIds: [hostB.id],
          executorType: 'simulated',
          action: 'upgrade',
          templateParams: {
            action: 'upgrade',
          },
          remark: 'updated in smoke',
        },
      }),
    );

    expect(updated.name).toBe(`${token}-task-updated`);
    expect(updated.packageId).toBe(pkgB.id);
    expect(updated.packageName).toBe(`${token}-pkg-b`);
    expect(updated.businessScopeId).toBe(scopeB.id);
    expect(updated.businessScopeName).toBe(`${token}-scope-b`);
    expect(updated.targetIds).toEqual([hostB.id]);
    expect(updated.executorType).toBe('simulated');
    expect(updated.action).toBe('upgrade');
    expect(updated.remark).toBe('updated in smoke');
    expect(updated.status).toBe('draft');

    await expectSuccess<null>(
      await apiContext.delete(`${apiBaseUrl}/business/deploy/tasks/${task.id}`),
    );
    await apiContext.delete(`${apiBaseUrl}/business/cmdb/hosts/${hostA.id}`).catch(() => undefined);
    await apiContext.delete(`${apiBaseUrl}/business/cmdb/hosts/${hostB.id}`).catch(() => undefined);
    await apiContext.delete(`${apiBaseUrl}/business/bizscope/${scopeA.id}`).catch(() => undefined);
    await apiContext.delete(`${apiBaseUrl}/business/bizscope/${scopeB.id}`).catch(() => undefined);
    await apiContext.delete(`${apiBaseUrl}/business/deploy/packages/${pkgA.id}`).catch(() => undefined);
    await apiContext.delete(`${apiBaseUrl}/business/deploy/packages/${pkgB.id}`).catch(() => undefined);
  });

  test('deploy api deletes a draft task and rejects delete after start', async () => {
    const token = `deploy-delete-${Date.now()}`;

    const scope = await expectSuccess<BizScopeRow>(
      await apiContext.post(`${apiBaseUrl}/business/bizscope`, {
        data: {
          code: token,
          name: `${token}-scope`,
          environment: 'dev',
          status: 'active',
        },
      }),
    );

    const host = await expectSuccess<HostRow>(
      await apiContext.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        data: {
          hostname: `${token}-host`,
          ip: `10.252.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
          os: 'linux',
          businessScopeId: scope.id,
          labels: [{ key: 'biz', val: token }],
        },
      }),
    );

    const pkg = await expectSuccess<PackageRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/packages`, {
        data: {
          name: token,
          version: '1.0.0',
          installCommand: 'echo install',
          status: 'enabled',
        },
      }),
    );

    const draftTask = await expectSuccess<TaskRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/tasks`, {
        data: {
          name: `${token}-draft`,
          packageId: pkg.id,
          businessScopeId: scope.id,
          targetType: 'host',
          targetIds: [host.id],
          executorType: 'manual',
        },
      }),
    );
    expect(draftTask.status).toBe('draft');

    await expectSuccess<null>(
      await apiContext.delete(`${apiBaseUrl}/business/deploy/tasks/${draftTask.id}`),
    );

    const draftDetailResponse = await apiContext.get(`${apiBaseUrl}/business/deploy/tasks/${draftTask.id}`);
    const draftDetailPayload = await draftDetailResponse.json() as ApiEnvelope<unknown>;
    expect(draftDetailPayload.code).not.toBe(200);

    const runningTask = await expectSuccess<TaskRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/tasks`, {
        data: {
          name: `${token}-running`,
          packageId: pkg.id,
          businessScopeId: scope.id,
          targetType: 'host',
          targetIds: [host.id],
          executorType: 'manual',
        },
      }),
    );
    expect(runningTask.status).toBe('draft');

    const started = await expectSuccess<TaskRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/tasks/${runningTask.id}/start`),
    );
    expect(started.status).toBe('running');

    const deleteRunningResponse = await apiContext.delete(`${apiBaseUrl}/business/deploy/tasks/${runningTask.id}`);
    const deleteRunningPayload = await deleteRunningResponse.json() as ApiEnvelope<unknown>;
    expect(deleteRunningPayload.code).not.toBe(200);
    expect(deleteRunningPayload.message || '').toContain('business.deploy.task.invalidDeleteState');

    await expectSuccess(
      await apiContext.post(`${apiBaseUrl}/business/deploy/task-hosts/${started.hosts[0].id}/result`, {
        data: { status: 'success', stdout: 'completed after delete lock assertion' },
      }),
    );

    await apiContext.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`).catch(() => undefined);
    await apiContext.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`).catch(() => undefined);
    await apiContext.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`).catch(() => undefined);
  });

  test('deploy api records uploaded package metadata and failed ssh trace', async () => {
    const token = `deploy-api-trace-${Date.now()}`;
    const sourceBuffer = Buffer.from('fake nginx source archive for api smoke', 'utf8');

    const scope = await expectSuccess<BizScopeRow>(
      await apiContext.post(`${apiBaseUrl}/business/bizscope`, {
        data: {
          code: token,
          name: `${token}-scope`,
          environment: 'dev',
          status: 'active',
        },
      }),
    );

    const host = await expectSuccess<HostRow>(
      await apiContext.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        data: {
          hostname: `${token}-host`,
          ip: `10.246.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
          sshPort: 22,
          os: 'linux',
          businessScopeId: scope.id,
          labels: [
            { key: 'biz', val: token },
            { key: 'env', val: 'dev' },
          ],
          remark: token,
        },
      }),
    );
    expect(host.status).toBe('assigned');

    const uploadPayload = await expectSuccess<UploadResult>(
      await apiContext.post(`${apiBaseUrl}/system/upload?scope=deploy/package`, {
        multipart: {
          file: {
            name: 'nginx-1.30.2.tar.gz',
            mimeType: 'application/gzip',
            buffer: sourceBuffer,
          },
        },
      }),
    );
    expect(uploadPayload.originalName).toBe('nginx-1.30.2.tar.gz');

    const pkg = await expectSuccess<PackageRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/packages`, {
        data: {
          name: `${token}-nginx`,
          version: '1.30.2',
          executionMode: 'fixed',
          templateCode: 'nginx_systemd',
          templateConfig: { scenario: 'systemd' },
          sourceObjectKey: uploadPayload.objectKey,
          sourceFileName: uploadPayload.originalName,
          sourceUrl: uploadPayload.url,
          uninstallCommand: 'systemctl stop nginx || true',
          status: 'enabled',
        },
      }),
    );
    expect(pkg.sourceFileName).toBe('nginx-1.30.2.tar.gz');

    const task = await expectSuccess<TaskRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/tasks`, {
        data: {
          name: `${token}-task`,
          packageId: pkg.id,
          businessScopeId: scope.id,
          targetType: 'host',
          targetIds: [host.id],
          executorType: 'ssh',
          templateParams: {
            installRoot: '/data/nginx',
            serviceName: 'nginx',
          },
          remark: token,
        },
      }),
    );
    expect(task.status).toBe('draft');

    const started = await expectSuccess<TaskRow>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/tasks/${task.id}/start`, {
        data: {
          sshUser: 'root',
          sshPassword: 'smoke-secret',
          hostFingerprint: 'SHA256:test',
          authMode: 'password',
        },
      }),
    );
    expect(started.status).toBe('failed');
    expect(started.businessScopeName).toContain(token);
    expect(started.hosts).toHaveLength(1);
    expect(started.hosts[0].traceSteps?.length || 0).toBeGreaterThan(0);

    const detail = await expectSuccess<TaskRow>(
      await apiContext.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`),
    );
    expect(detail.hosts[0].traceSteps?.map((step) => step.phase)).toEqual(
      expect.arrayContaining(['start', 'result', 'error']),
    );

    const hostDetail = await expectSuccess<HostRow>(
      await apiContext.get(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`),
    );
    expect(hostDetail.status).toBe('assigned');

    await apiContext.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`).catch(() => undefined);
    await apiContext.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`).catch(() => undefined);
    await apiContext.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`).catch(() => undefined);
  });
});
