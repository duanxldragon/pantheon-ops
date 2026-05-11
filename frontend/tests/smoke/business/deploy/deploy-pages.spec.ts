import { expect, test, type APIResponse } from '@playwright/test';
import {
  apiBaseUrl,
  apiRequestHeaders,
  adminCredentials,
  installClientSession,
  loginByApi,
  primeChineseLocale,
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
};

type PackageRow = {
  id: number;
  name: string;
  version: string;
};

type TaskRow = {
  id: number;
  status: string;
  hosts: Array<{ id: number; status: string; hostId: number }>;
};

async function expectSuccess<T>(response: APIResponse): Promise<T> {
  expect(response.ok()).toBeTruthy();
  const payload = await response.json() as ApiEnvelope<T>;
  expect(payload.code).toBe(200);
  return payload.data;
}

test.describe('Deploy business module smoke', () => {
  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
  });

  test('deploy pages load under operations platform', async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', (message) => {
      if (message.type() === 'error') {
        consoleErrors.push(message.text());
      }
    });

    await page.goto('/operations/deploy/package', { waitUntil: 'networkidle' });
    await expect(page.getByRole('heading', { name: '软件组件' })).toBeVisible();
    await expect(page.locator('.filter-panel')).toBeVisible();

    await page.goto('/operations/deploy/task', { waitUntil: 'networkidle' });
    await expect(page.getByRole('heading', { name: '部署任务' })).toBeVisible();
    await expect(page.locator('.filter-panel')).toBeVisible();
    expect(consoleErrors).toEqual([]);
  });

  test('deploy api creates starts and marks a manual task', async ({ request }) => {
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `deploy-smoke-${Date.now()}`;

    const host = await expectSuccess<HostRow>(
      await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        headers,
        data: {
          hostname: token,
          ip: `10.245.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
          os: 'linux',
          status: 'online',
          labels: [{ key: 'biz', val: token }],
        },
      }),
    );

    const pkg = await expectSuccess<PackageRow>(
      await request.post(`${apiBaseUrl}/business/deploy/packages`, {
        headers,
        data: {
          name: token,
          version: '1.0.0',
          installCommand: 'echo install',
          status: 'enabled',
        },
      }),
    );

    const task = await expectSuccess<TaskRow>(
      await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
        headers,
        data: {
          name: token,
          packageId: pkg.id,
          targetType: 'host',
          targetIds: [host.id],
          executorType: 'manual',
        },
      }),
    );
    expect(task.status).toBe('pending');

    const started = await expectSuccess<TaskRow>(
      await request.post(`${apiBaseUrl}/business/deploy/tasks/${task.id}/start`, { headers }),
    );
    expect(started.status).toBe('running');
    expect(started.hosts).toHaveLength(1);

    await expectSuccess(
      await request.post(`${apiBaseUrl}/business/deploy/task-hosts/${started.hosts[0].id}/result`, {
        headers,
        data: { status: 'success', stdout: 'installed' },
      }),
    );

    const detail = await expectSuccess<TaskRow>(
      await request.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`, { headers }),
    );
    expect(detail.status).toBe('success');

    await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers });
  });
});
