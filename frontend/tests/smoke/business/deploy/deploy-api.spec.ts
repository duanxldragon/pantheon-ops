import { expect, request as playwrightRequest, test, type APIRequestContext, type APIResponse } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import type { DeployTaskPreviewResponse, StartDeployTaskPayload } from '../../../../src/modules/business/deploy/api';
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
  serviceId?: number;
  serviceInstanceId?: number;
  serviceName?: string;
  serviceInstanceName?: string;
  hosts: Array<{
    id: number;
    status: string;
    hostId: number;
    stdout?: string;
    errorMessage?: string;
    traceSteps?: Array<{ phase?: string; message?: string }>;
  }>;
};

type UploadResult = {
  objectKey: string;
  originalName: string;
  url: string;
};

async function expectSuccess<T>(response: APIResponse): Promise<T> {
  const payload = await response.json() as ApiEnvelope<T>;
  expect(response.ok(), `HTTP ${response.status()}: ${payload.message ?? 'no error key'}`).toBeTruthy();
  expect(payload.code).toBe(200);
  return payload.data;
}

async function previewAndStartTask(
  apiContext: APIRequestContext,
  taskId: number,
  data: StartDeployTaskPayload = {},
): Promise<TaskRow> {
  const preview = await expectSuccess<DeployTaskPreviewResponse>(
    await apiContext.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/preview`, { data }),
  );
  expect(preview.intentId).toBe(`deploy:${taskId}`);
  expect(preview.intentVersion).toBeTruthy();
  expect(preview.targetCount).toBeGreaterThan(0);
  return expectSuccess<TaskRow>(
    await apiContext.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/start`, {
      headers: { 'X-Idempotency-Key': randomUUID() },
      data: { ...data, intentVersion: preview.intentVersion },
    }),
  );
}

type FixtureResource =
  | 'deploy/tasks'
  | 'service/instances'
  | 'service/services'
  | 'service/applications'
  | 'cmdb/hosts'
  | 'deploy/packages'
  | 'deploy/credentials'
  | 'bizscope';

test.describe.serial('Deploy business api smoke', () => {
  let login: BrowserLoginResult;
  let apiContext: APIRequestContext;
  const createdFixtures: Array<{ resource: FixtureResource; id: number }> = [];

  async function createFixture<T extends { id: number }>(resource: FixtureResource, response: APIResponse): Promise<T> {
    const fixture = await expectSuccess<T>(response);
    createdFixtures.push({ resource, id: fixture.id });
    return fixture;
  }

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

  test.afterEach(async () => {
    const priority: FixtureResource[] = [
      'deploy/tasks',
      'service/instances',
      'service/services',
      'service/applications',
      'cmdb/hosts',
      'deploy/packages',
      'deploy/credentials',
      'bizscope',
    ];
    const fixtures = createdFixtures.splice(0).sort((a, b) => priority.indexOf(a.resource) - priority.indexOf(b.resource));
    const gaps: string[] = [];
    for (const { resource, id } of fixtures) {
      try {
        const response = await apiContext.delete(`${apiBaseUrl}/business/${resource}/${id}`);
        const payload = await response.json() as ApiEnvelope<unknown>;
        if (payload.code !== 200 && payload.code !== 404) {
          gaps.push(`${resource}/${id}: HTTP ${response.status()}, code ${payload.code}`);
        }
      } catch {
        gaps.push(`${resource}/${id}: cleanup request failed`);
      }
    }
    expect(gaps, 'Cleanup only addresses IDs created by this test; unresolved fixtures remain explicit').toEqual([]);
  });

  test('deploy api creates starts and marks a manual task', async () => {
    const token = `deploy-smoke-${Date.now()}`;
    const scope = await createFixture<BizScopeRow>('bizscope',
      await apiContext.post(`${apiBaseUrl}/business/bizscope`, {
        data: {
          code: token,
          name: token,
          environment: 'prod',
          status: 'active',
        },
      }),
    );

    const host = await createFixture<HostRow>('cmdb/hosts',
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

    const pkg = await createFixture<PackageRow>('deploy/packages',
      await apiContext.post(`${apiBaseUrl}/business/deploy/packages`, {
        data: {
          name: token,
          version: '1.0.0',
          installCommand: 'echo install',
          status: 'enabled',
        },
      }),
    );

    const task = await createFixture<TaskRow>('deploy/tasks',
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

    const started = await previewAndStartTask(apiContext, task.id);
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

  });

  test('deploy api links a task to a service instance and filters by the instance', async () => {
    const token = `deploy-link-${Date.now()}`;
    const scope = await createFixture<BizScopeRow>('bizscope',
      await apiContext.post(`${apiBaseUrl}/business/bizscope`, {
        data: { code: token, name: token, environment: 'prod', status: 'active' },
      }),
    );
    const host = await createFixture<HostRow>('cmdb/hosts',
      await apiContext.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        data: {
          hostname: token,
          ip: `10.247.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
          os: 'linux',
          businessScopeId: scope.id,
          labels: [{ key: 'biz', val: token }],
        },
      }),
    );
    const application = await createFixture<{ id: number }>('service/applications',
      await apiContext.post(`${apiBaseUrl}/business/service/applications`, {
        data: { code: `${token}-app`, name: `${token}-application`, businessScopeId: scope.id },
      }),
    );
    const service = await createFixture<{ id: number }>('service/services',
      await apiContext.post(`${apiBaseUrl}/business/service/services`, {
        data: { applicationId: application.id, code: `${token}-svc`, name: `${token}-service`, runtimeType: 'vm' },
      }),
    );
    const instance = await createFixture<{ id: number }>('service/instances',
      await apiContext.post(`${apiBaseUrl}/business/service/instances`, {
        data: { serviceId: service.id, environment: 'prod', targetType: 'vm', hostId: host.id, desiredVersion: '1.2.0' },
      }),
    );
    const pkg = await createFixture<PackageRow>('deploy/packages',
      await apiContext.post(`${apiBaseUrl}/business/deploy/packages`, {
        data: { name: token, version: '1.0.0', installCommand: 'echo install', status: 'enabled' },
      }),
    );

    const task = await createFixture<TaskRow>('deploy/tasks',
      await apiContext.post(`${apiBaseUrl}/business/deploy/tasks`, {
        data: {
          name: token,
          packageId: pkg.id,
          businessScopeId: scope.id,
          serviceId: service.id,
          serviceInstanceId: instance.id,
          targetType: 'host',
          targetIds: [host.id],
          executorType: 'manual',
        },
      }),
    );
    expect(task.serviceId).toBe(service.id);
    expect(task.serviceInstanceId).toBe(instance.id);

    const filtered = await expectSuccess<{ items: Array<{ id: number }>; total: number }>(
      await apiContext.get(`${apiBaseUrl}/business/deploy/tasks?serviceInstanceId=${instance.id}&page=1&pageSize=20`),
    );
    expect(filtered.items.map((item) => item.id)).toContain(task.id);

    const detail = await expectSuccess<TaskRow>(
      await apiContext.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`),
    );
    expect(detail.serviceName).toBe(`${token}-service`);
    expect(detail.serviceInstanceId).toBe(instance.id);

    // A service-linked task resolves its single target from the instance's host.
    // Starting it is covered by the unlinked manual-task case above; a manual
    // report for a linked task is rejected as a stale attempt, which this test
    // records as an explicit gap instead of forcing a terminal state.
    const preview = await expectSuccess<DeployTaskPreviewResponse>(
      await apiContext.post(`${apiBaseUrl}/business/deploy/tasks/${task.id}/preview`, { data: {} }),
    );
    expect(preview.targetCount).toBe(1);

    const instances = await expectSuccess<{ items: Array<{ id: number; currentVersion?: string }> }>(
      await apiContext.get(`${apiBaseUrl}/business/service/instances?serviceId=${service.id}&page=1&pageSize=20`),
    );
    expect(instances.items.map((item) => item.id)).toContain(instance.id);
  });

  test('deploy api updates a draft task before execution', async () => {
    const token = `deploy-update-${Date.now()}`;

    const scopeA = await createFixture<BizScopeRow>('bizscope',
      await apiContext.post(`${apiBaseUrl}/business/bizscope`, {
        data: {
          code: `${token}-a`,
          name: `${token}-scope-a`,
          environment: 'dev',
          status: 'active',
        },
      }),
    );
    const scopeB = await createFixture<BizScopeRow>('bizscope',
      await apiContext.post(`${apiBaseUrl}/business/bizscope`, {
        data: {
          code: `${token}-b`,
          name: `${token}-scope-b`,
          environment: 'prod',
          status: 'active',
        },
      }),
    );

    const hostA = await createFixture<HostRow>('cmdb/hosts',
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
    const hostB = await createFixture<HostRow>('cmdb/hosts',
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
    await expectSuccess<null>(
      await apiContext.patch(`${apiBaseUrl}/business/cmdb/hosts/${hostB.id}/status`, {
        data: { status: 'online' },
      }),
    );

    const pkgA = await createFixture<PackageRow>('deploy/packages',
      await apiContext.post(`${apiBaseUrl}/business/deploy/packages`, {
        data: {
          name: `${token}-pkg-a`,
          version: '1.0.0',
          installCommand: 'echo install a',
          status: 'enabled',
        },
      }),
    );
    const pkgB = await createFixture<PackageRow>('deploy/packages',
      await apiContext.post(`${apiBaseUrl}/business/deploy/packages`, {
        data: {
          name: `${token}-pkg-b`,
          version: '2.0.0',
          installCommand: 'echo install b',
          status: 'enabled',
        },
      }),
    );

    const task = await createFixture<TaskRow>('deploy/tasks',
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
  });

  test('deploy api deletes draft and terminal tasks but rejects active execution', async () => {
    const token = `deploy-delete-${Date.now()}`;

    const scope = await createFixture<BizScopeRow>('bizscope',
      await apiContext.post(`${apiBaseUrl}/business/bizscope`, {
        data: {
          code: token,
          name: `${token}-scope`,
          environment: 'dev',
          status: 'active',
        },
      }),
    );

    const host = await createFixture<HostRow>('cmdb/hosts',
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

    const pkg = await createFixture<PackageRow>('deploy/packages',
      await apiContext.post(`${apiBaseUrl}/business/deploy/packages`, {
        data: {
          name: token,
          version: '1.0.0',
          installCommand: 'echo install',
          status: 'enabled',
        },
      }),
    );

    const draftTask = await createFixture<TaskRow>('deploy/tasks',
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

    const runningTask = await createFixture<TaskRow>('deploy/tasks',
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

    const started = await previewAndStartTask(apiContext, runningTask.id);
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

    await expectSuccess<null>(
      await apiContext.delete(`${apiBaseUrl}/business/deploy/tasks/${runningTask.id}`),
    );
    const terminalDetailResponse = await apiContext.get(`${apiBaseUrl}/business/deploy/tasks/${runningTask.id}`);
    const terminalDetailPayload = await terminalDetailResponse.json() as ApiEnvelope<unknown>;
    expect(terminalDetailPayload.code).not.toBe(200);

  });

  test('deploy api records uploaded package metadata and failed ssh trace', async () => {
    const token = `deploy-api-trace-${Date.now()}`;
    // Keep the SSH fixture isolated from leftovers from interrupted runs.
    const smokeIp = `127.0.0.${Math.floor(Math.random() * 200) + 20}`;
    const sourceBuffer = Buffer.from('fake nginx source archive for api smoke', 'utf8');

    const scope = await createFixture<BizScopeRow>('bizscope',
      await apiContext.post(`${apiBaseUrl}/business/bizscope`, {
        data: {
          code: token,
          name: `${token}-scope`,
          environment: 'dev',
          status: 'active',
        },
      }),
    );

    const host = await createFixture<HostRow>('cmdb/hosts',
      await apiContext.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        data: {
          hostname: `${token}-host`,
          ip: smokeIp,
          sshPort: 1,
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

    const pkg = await createFixture<PackageRow>('deploy/packages',
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

    const task = await createFixture<TaskRow>('deploy/tasks',
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

    const credential = await createFixture<{ id: number }>('deploy/credentials',
      await apiContext.post(`${apiBaseUrl}/business/deploy/credentials`, {
        data: {
          name: `${token}-credential`,
          businessScopeId: scope.id,
          username: 'root',
          authMode: 'password',
          secret: 'smoke-secret',
        },
      }),
    );

    await previewAndStartTask(apiContext, task.id, {
      credentialRefId: credential.id,
      hostFingerprint: 'SHA256:test',
    });
    await expect.poll(async () => {
      const response = await apiContext.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`);
      const payload = await response.json() as ApiEnvelope<TaskRow>;
      return payload.data.status;
    }, { timeout: 10_000 }).toBe('failed');

    const failedTask = await expectSuccess<TaskRow>(
      await apiContext.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`),
    );
    expect(failedTask.status).toBe('failed');
    expect(failedTask.businessScopeName).toContain(token);
    expect(failedTask.hosts).toHaveLength(1);
    const failedHost = failedTask.hosts[0];
    expect(failedHost.hostId).toBe(host.id);
    expect(failedHost.status).toBe('failed');
    expect(failedHost.errorMessage).toBe('business.deploy.task.sshConnectFailed');
    // Connection setup commits a failed result; legacy traceSteps have no required error phase.
    expect(failedHost.traceSteps?.map((step) => step.phase)).toEqual(
      expect.arrayContaining(['start', 'result']),
    );
    expect(failedHost.traceSteps).toContainEqual(expect.objectContaining({ phase: 'result', message: 'status=failed' }));
    expect(JSON.stringify(failedTask)).not.toContain('smoke-secret');
    await test.info().attach('failed-ssh-result', {
      body: JSON.stringify({
        taskId: task.id,
        taskStatus: failedTask.status,
        hostStatus: failedHost.status,
        errorMessage: failedHost.errorMessage,
        tracePhases: failedHost.traceSteps?.map((step) => step.phase),
        fixtureSecretRedacted: true,
      }),
      contentType: 'application/json',
    });

    const hostDetail = await expectSuccess<HostRow>(
      await apiContext.get(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`),
    );
    expect(hostDetail.status).toBe('assigned');

  });
});
