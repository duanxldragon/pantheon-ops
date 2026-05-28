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
  latestDeployedAt?: string;
};

type TemplateRow = {
  id: number;
};

type TaskRow = {
  id: number;
  status: string;
  businessScopeName?: string;
  hosts: Array<{
    id: number;
    status: string;
    hostId: number;
    stdout?: string;
    traceSteps?: Array<{ phase?: string; message?: string }>;
  }>;
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

  test('deploy pages load under operations platform', async ({ page }, testInfo) => {
    const consoleErrors: string[] = [];
    page.on('console', (message) => {
      if (message.type() === 'error') {
        consoleErrors.push(message.text());
      }
    });

    await page.goto('/operations/deploy/package', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect(page.locator('.governance-summary-bar__title-row')).toBeVisible();
    await expect(page.locator('.governance-summary-bar__icon')).toBeVisible();
    await expect(page.locator('.governance-summary-bar')).toContainText('软件组件');
    await page.locator('.governance-summary-bar .arco-btn').click();
    await expect(page.locator('.governance-insight-drawer')).toBeVisible();
    await expect(page.locator('.table-batch-action-bar')).toBeVisible();
    await expect(page.locator('.filter-panel')).toBeVisible();
    await expect(page.locator('.system-list__table-card')).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('deploy-package-list.png'), fullPage: true });

    await page.goto('/operations/deploy/template', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect(page.locator('.governance-summary-bar__title-row')).toBeVisible();
    await expect(page.locator('.governance-summary-bar__icon')).toBeVisible();
    await expect(page.locator('.governance-summary-bar')).toContainText('任务模板');
    await page.locator('.governance-summary-bar .arco-btn').click();
    await expect(page.locator('.governance-insight-drawer')).toBeVisible();
    await expect(page.locator('.table-batch-action-bar')).toBeVisible();
    await expect(page.locator('.filter-panel')).toBeVisible();
    await expect(page.locator('.system-list__table-card')).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('deploy-template-list.png'), fullPage: true });

    await page.goto('/operations/deploy/task', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect(page.locator('.governance-summary-bar__title-row')).toBeVisible();
    await expect(page.locator('.governance-summary-bar__icon')).toBeVisible();
    await expect(page.locator('.governance-summary-bar')).toContainText('部署任务');
    await page.locator('.governance-summary-bar .arco-btn').click();
    await expect(page.locator('.governance-insight-drawer')).toBeVisible();
    await expect(page.locator('.table-batch-action-bar')).toBeVisible();
    await expect(page.locator('.filter-panel')).toBeVisible();
    await expect(page.locator('.system-list__table-card')).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('deploy-task-list.png'), fullPage: true });
    expect(consoleErrors).toEqual([]);
  });

  test('deploy task detail action opens a modal without leaving the list page', async ({ page, request }, testInfo) => {
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `deploy-detail-modal-${Date.now()}`;

    const scope = await expectSuccess<BizScopeRow>(
      await request.post(`${apiBaseUrl}/business/bizscope`, {
        headers,
        data: {
          code: token,
          name: token,
          environment: 'dev',
          status: 'active',
        },
      }),
    );

    const host = await expectSuccess<HostRow>(
      await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        headers,
        data: {
          hostname: `${token}-host`,
          ip: `10.248.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
          os: 'linux',
          businessScopeId: scope.id,
          labels: [{ key: 'biz', val: token }],
        },
      }),
    );

    const pkg = await expectSuccess<PackageRow>(
      await request.post(`${apiBaseUrl}/business/deploy/packages`, {
        headers,
        data: {
          name: token,
          version: '1.30.2',
          installCommand: 'echo install',
          status: 'enabled',
        },
      }),
    );

    const task = await expectSuccess<TaskRow>(
      await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
        headers,
        data: {
          name: `${token}-task`,
          packageId: pkg.id,
          businessScopeId: scope.id,
          targetType: 'host',
          targetIds: [host.id],
          executorType: 'manual',
        },
      }),
    );

    try {
      const started = await expectSuccess<TaskRow>(
        await request.post(`${apiBaseUrl}/business/deploy/tasks/${task.id}/start`, { headers }),
      );
      expect(started.hosts).toHaveLength(1);

      await expectSuccess(
        await request.post(`${apiBaseUrl}/business/deploy/task-hosts/${started.hosts[0].id}/result`, {
          headers,
          data: { status: 'success', stdout: 'installed via modal smoke' },
        }),
      );

      await page.goto('/operations/deploy/task', { waitUntil: 'networkidle' });
      await expect(page).toHaveURL(/\/operations\/deploy\/task$/);
      const taskKeywordInput = page.locator('.filter-panel input').first();
      await expect(taskKeywordInput).toBeVisible();
      await taskKeywordInput.fill(token);
      await page.getByRole('button', { name: '搜索' }).click();

      const row = page.getByRole('row').filter({ hasText: `${token}-task` }).first();
      await expect(row).toBeVisible();
      await row.getByRole('button', { name: '详情' }).click();

      const modal = page.locator('.arco-modal:visible').last();
      await expect(modal).toBeVisible();
      await expect(modal).toContainText(`${token}-task`);
      await expect(modal).toContainText(token);
      await expect(modal).toContainText(`${token}-host`);
      await expect(modal).toContainText('installed via modal smoke');
      await expect(page).toHaveURL(/\/operations\/deploy\/task$/);
      await page.screenshot({ path: testInfo.outputPath('deploy-task-detail-modal.png'), fullPage: true });
    } finally {
      await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
      await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
      await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
    }
  });

  test('deploy package detail action opens a modal without leaving the list page', async ({ page, request }, testInfo) => {
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `deploy-package-modal-${Date.now()}`;

    const pkg = await expectSuccess<PackageRow>(
      await request.post(`${apiBaseUrl}/business/deploy/packages`, {
        headers,
        data: {
          name: token,
          version: '1.30.2',
          executionMode: 'fixed',
          templateCode: 'nginx_systemd',
          sourceFileName: 'nginx-1.30.2.tar.gz',
          sourceUrl: `https://example.invalid/${token}.tar.gz`,
          status: 'enabled',
        },
      }),
    );

    try {
      await page.goto('/operations/deploy/package', { waitUntil: 'networkidle' });
      await expect(page).toHaveURL(/\/operations\/deploy\/package$/);
      const row = page.getByRole('row').filter({ hasText: token }).first();
      await expect(row).toBeVisible();
      await row.getByRole('button', { name: '详情' }).click();

      const modal = page.locator('.arco-modal:visible').last();
      await expect(modal).toBeVisible();
      await expect(modal).toContainText(token);
      await expect(modal).toContainText('nginx-1.30.2.tar.gz');
      await expect(modal).toContainText('模板说明');
      await expect(page).toHaveURL(/\/operations\/deploy\/package$/);
      await page.screenshot({ path: testInfo.outputPath('deploy-package-detail-modal.png'), fullPage: true });
    } finally {
      await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
    }
  });

  test('deploy template detail action opens a modal without leaving the list page', async ({ page, request }, testInfo) => {
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `deploy-template-modal-${Date.now()}`;

    const pkg = await expectSuccess<PackageRow>(
      await request.post(`${apiBaseUrl}/business/deploy/packages`, {
        headers,
        data: {
          name: `${token}-pkg`,
          version: '1.30.2',
          executionMode: 'fixed',
          templateCode: 'nginx_systemd',
          sourceFileName: 'nginx-1.30.2.tar.gz',
          status: 'enabled',
        },
      }),
    );

    const template = await expectSuccess<TemplateRow>(
      await request.post(`${apiBaseUrl}/business/deploy/templates`, {
        headers,
        data: {
          name: token,
          version: 'v1',
          packageId: pkg.id,
          executionMode: 'orchestrated',
          defaultAction: 'install',
          status: 'enabled',
          parameterSchema: {
            installRoot: '/data/nginx',
          },
          steps: [
            {
              stepCode: 'prepare',
              stepName: '准备目录',
              stepType: 'script',
              action: 'install',
              stepConfig: {
                script: 'mkdir -p /data/nginx',
              },
              sort: 1,
            },
            {
              stepCode: 'install',
              stepName: '安装组件',
              stepType: 'package',
              action: 'install',
              packageId: pkg.id,
              sort: 2,
            },
          ],
        },
      }),
    );

    try {
      await page.goto('/operations/deploy/template', { waitUntil: 'networkidle' });
      await expect(page).toHaveURL(/\/operations\/deploy\/template$/);
      const templateKeywordInput = page.locator('.filter-panel input').first();
      await expect(templateKeywordInput).toBeVisible();
      await templateKeywordInput.fill(token);
      await page.getByRole('button', { name: '搜索' }).click();

      const row = page.getByRole('row').filter({ hasText: token }).first();
      await expect(row).toBeVisible();
      await row.getByRole('button', { name: '详情' }).click();

      const modal = page.locator('.arco-modal:visible').last();
      await expect(modal).toBeVisible();
      await expect(modal).toContainText(token);
      await expect(modal).toContainText('准备目录');
      await expect(modal).toContainText('安装组件');
      await expect(modal).toContainText('/data/nginx');
      await expect(page).toHaveURL(/\/operations\/deploy\/template$/);
      await page.screenshot({ path: testInfo.outputPath('deploy-template-detail-modal.png'), fullPage: true });
    } finally {
      await request.delete(`${apiBaseUrl}/business/deploy/templates/${template.id}`, { headers }).catch(() => undefined);
      await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
    }
  });

  test('deploy detail page loads with summary and host table', async ({ page, request }, testInfo) => {
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `deploy-detail-${Date.now()}`;
    const scope = await expectSuccess<BizScopeRow>(
      await request.post(`${apiBaseUrl}/business/bizscope`, {
        headers,
        data: {
          code: token,
          name: token,
          environment: 'prod',
          status: 'active',
        },
      }),
    );

    const host = await expectSuccess<HostRow>(
      await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        headers,
        data: {
          hostname: token,
          ip: `10.244.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
          os: 'linux',
          businessScopeId: scope.id,
          labels: [{ key: 'biz', val: token }],
        },
      }),
    );
    expect(host.status).toBe('assigned');

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
          businessScopeId: scope.id,
          targetType: 'host',
          targetIds: [host.id],
          executorType: 'manual',
        },
      }),
    );

    try {
      await page.goto(`/operations/deploy/task/${task.id}`, { waitUntil: 'networkidle' });
      await expect(page.locator('.page-container')).toBeVisible();
      await expect(page.locator('.system-page-hero')).toBeVisible();
      await expect(page.locator('.system-list__table-card')).toBeVisible();
      await expect(page.locator('.page-header')).toBeVisible();
      await page.screenshot({ path: testInfo.outputPath('deploy-task-detail.png'), fullPage: true });
    } finally {
      await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers });
      await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
      await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
    }
  });

  test('deploy template supports script steps and task params render in detail', async ({ page, request }, testInfo) => {
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `deploy-script-template-${Date.now()}`;

    const scope = await expectSuccess<BizScopeRow>(
      await request.post(`${apiBaseUrl}/business/bizscope`, {
        headers,
        data: {
          code: token,
          name: `${token}-scope`,
          environment: 'dev',
          status: 'active',
        },
      }),
    );

    const host = await expectSuccess<HostRow>(
      await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        headers,
        data: {
          hostname: `${token}-host`,
          ip: `10.247.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
          os: 'linux',
          businessScopeId: scope.id,
          labels: [{ key: 'biz', val: token }],
        },
      }),
    );

    const pkg = await expectSuccess<PackageRow>(
      await request.post(`${apiBaseUrl}/business/deploy/packages`, {
        headers,
        data: {
          name: token,
          version: '1.30.2',
          installCommand: 'echo install',
          status: 'enabled',
        },
      }),
    );

    const template = await expectSuccess<TemplateRow>(
      await request.post(`${apiBaseUrl}/business/deploy/templates`, {
        headers,
        data: {
          name: token,
          version: 'v1',
          packageId: pkg.id,
          executionMode: 'orchestrated',
          defaultAction: 'install',
          status: 'enabled',
          parameterSchema: {
            installRoot: '/data/nginx',
            serviceName: 'nginx',
            configRoot: '/data/nginx/conf',
          },
          steps: [
            {
              stepCode: 'prepare',
              stepName: '准备目录',
              stepType: 'script',
              action: 'install',
              stepConfig: {
                precheckCommand: 'echo precheck {{hostIp}}',
                script: 'echo render {{installRoot}} {{configRoot}}',
                postcheckCommand: 'echo postcheck {{serviceName}}',
              },
              sort: 1,
            },
            {
              stepCode: 'install',
              stepName: '安装组件',
              stepType: 'package',
              action: 'install',
              packageId: pkg.id,
              sort: 2,
            },
          ],
        },
      }),
    );

    const task = await expectSuccess<TaskRow>(
      await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
        headers,
        data: {
          name: `${token}-task`,
          templateId: template.id,
          businessScopeId: scope.id,
          targetType: 'host',
          targetIds: [host.id],
          executorType: 'manual',
          templateParams: {
            installRoot: '/data/nginx-custom',
            serviceName: 'nginx-custom',
            configRoot: '/data/nginx-custom/conf',
          },
        },
      }),
    );

    try {
      await page.goto('/operations/deploy/template', { waitUntil: 'networkidle' });
      await page.locator('.filter-panel input').first().fill(token);
      await page.locator('.filter-panel .arco-btn-primary').first().click();
      await expect(page.getByText(token, { exact: true }).first()).toBeVisible();
      await page.screenshot({ path: testInfo.outputPath('deploy-script-template-list.png'), fullPage: true });

      await page.goto(`/operations/deploy/task/${task.id}`, { waitUntil: 'networkidle' });
      await expect(page.getByText('configRoot', { exact: true })).toBeVisible();
      await expect(page.getByText('/data/nginx-custom/conf', { exact: true })).toBeVisible();
      await expect(page.getByText('/data/nginx-custom', { exact: true })).toBeVisible();
      await page.screenshot({ path: testInfo.outputPath('deploy-script-template-task-detail.png'), fullPage: true });
    } finally {
      await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
      await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
      await request.delete(`${apiBaseUrl}/business/deploy/templates/${template.id}`, { headers }).catch(() => undefined);
      await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
    }
  });

  test('deploy api creates starts and marks a manual task', async ({ request }) => {
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `deploy-smoke-${Date.now()}`;
    const scope = await expectSuccess<BizScopeRow>(
      await request.post(`${apiBaseUrl}/business/bizscope`, {
        headers,
        data: {
          code: token,
          name: token,
          environment: 'prod',
          status: 'active',
        },
      }),
    );

    const host = await expectSuccess<HostRow>(
      await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        headers,
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
          businessScopeId: scope.id,
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
    const hostDetail = await expectSuccess<HostRow>(
      await request.get(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }),
    );
    expect(hostDetail.status).toBe('online');

    await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers });
    await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  });

  test('targeted smoke closes cmdb, bizscope, deploy with uploaded source package and visible task trace', async ({ page, request }, testInfo) => {
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `deploy-closed-loop-${Date.now()}`;
    const sourceBuffer = Buffer.from('fake nginx source archive for smoke', 'utf8');

    const scope = await expectSuccess<BizScopeRow>(
      await request.post(`${apiBaseUrl}/business/bizscope`, {
        headers,
        data: {
          code: token,
          name: `${token}-his-dev`,
          environment: 'dev',
          status: 'active',
        },
      }),
    );

    const host = await expectSuccess<HostRow>(
      await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        headers,
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

    const uploadResponse = await request.post(`${apiBaseUrl}/system/upload?scope=deploy/package`, {
      headers,
      multipart: {
        file: {
          name: 'nginx-1.30.2.tar.gz',
          mimeType: 'application/gzip',
          buffer: sourceBuffer,
        },
      },
    });
    const uploadPayload = await expectSuccess<{ objectKey: string; originalName: string; url: string }>(uploadResponse);
    expect(uploadPayload.originalName).toBe('nginx-1.30.2.tar.gz');

    const pkg = await expectSuccess<PackageRow>(
      await request.post(`${apiBaseUrl}/business/deploy/packages`, {
        headers,
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
      await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
        headers,
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
    expect(task.status).toBe('pending');

    const started = await expectSuccess<TaskRow>(
      await request.post(`${apiBaseUrl}/business/deploy/tasks/${task.id}/start`, {
        headers,
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
      await request.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`, { headers }),
    );
    expect(detail.hosts[0].traceSteps?.map((step) => step.phase)).toEqual(
      expect.arrayContaining(['start', 'result', 'error']),
    );

    await page.goto(`/operations/deploy/task/${task.id}`, { waitUntil: 'networkidle' });
    await expect(page.locator('.page-container')).toBeVisible();
    await expect(page.locator('.system-page-hero')).toContainText('目标主机');
    await expect(page.getByRole('cell', { name: '业务域' })).toBeVisible();
    await expect(page.getByText('installRoot', { exact: true })).toBeVisible();
    await expect(page.getByText('/data/nginx', { exact: true })).toBeVisible();
    await expect(page.getByRole('cell', { name: 'SSH 执行' })).toBeVisible();
    await expect(page.locator('.system-list__table-card')).toContainText('失败');
    await expect(page.locator('.system-list__table-card')).toContainText('connect');
    await expect(page.locator('.system-list__table-card')).toContainText('failed');
    await page.screenshot({ path: testInfo.outputPath('deploy-closed-loop-task-detail.png'), fullPage: true });

    const hostDetail = await expectSuccess<HostRow>(
      await request.get(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }),
    );
    expect(hostDetail.status).toBe('assigned');

    await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
    await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
    await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  });

  test('deploy pages stay within a phone viewport', async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 390, height: 844 });

    await page.goto('/operations/deploy/package', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect
      .poll(async () =>
        page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1),
      )
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('deploy-package-list-mobile.png'), fullPage: true });

    await page.goto('/operations/deploy/template', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect
      .poll(async () =>
        page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1),
      )
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('deploy-template-list-mobile.png'), fullPage: true });

    await page.goto('/operations/deploy/task', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect
      .poll(async () =>
        page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1),
      )
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('deploy-task-list-mobile.png'), fullPage: true });
  });
});
