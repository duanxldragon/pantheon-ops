import { expect, test, type APIRequestContext, type APIResponse, type Page, type TestInfo } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import {
  adminCredentials,
  apiBaseUrl,
  apiRequestHeaders,
  authHeaders,
  installClientSession,
  loginByApi,
  verifiedApiHeaders,
  type LoginCredentials,
} from '../../helpers/auth';

const credentialApi = '/business/deploy/credentials';
const credentialPage = '/business/deploy/credential';
const taskPage = '/business/deploy/task';
const fixtureKey = `t05-${randomUUID().slice(0, 8)}`;
const operatorName = `${fixtureKey}-operator`;
const viewerName = `${fixtureKey}-viewer`;
const ownName = `${fixtureKey}-own`;
const ownScopeName = `${fixtureKey}-owner`;
const otherName = `${fixtureKey}-other`;
const peerName = `${fixtureKey}-same-dept-other-scope`;
const inactiveScopeName = `${fixtureKey}-inactive-owner`;

type Row = { id: number };
type MenuNode = Row & { path: string; children?: MenuNode[] };

async function success<T>(response: APIResponse): Promise<T> {
  expect(response.ok(), response.url()).toBeTruthy();
  const payload = await response.json() as { code: number; data: T; message?: string };
  expect(payload.code, `${response.url()}: ${payload.message ?? ''}`).toBe(200);
  return payload.data;
}

function menuId(nodes: MenuNode[], path: string): number | undefined {
  for (const node of nodes) {
    if (node.path === path) return node.id;
    const child = node.children && menuId(node.children, path);
    if (child) return child;
  }
}

function deferred() {
  let resolve!: () => void;
  const promise = new Promise<void>((done) => { resolve = done; });
  return { promise, resolve };
}

async function openCredentials(page: Page, credentials: LoginCredentials) {
  const session = await loginByApi(page.request, credentials);
  await installClientSession(page, session);
  await page.goto(credentialPage, { waitUntil: 'domcontentloaded' });
}

async function capture(page: Page, testInfo: TestInfo, state: string) {
  await expect.poll(() => page.evaluate(() =>
    document.documentElement.scrollWidth - document.documentElement.clientWidth,
  )).toBeLessThanOrEqual(1);
  const path = testInfo.outputPath(`t05-${state}.png`);
  await page.screenshot({ path, animations: 'disabled' });
  await testInfo.attach(`t05-${state}`, { path, contentType: 'image/png' });
}

test.describe('Deployment credential page with real scoped roles', () => {
  const cleanup: Array<{ path: string; verified: boolean }> = [];
  let admin: Awaited<ReturnType<typeof loginByApi>>;
  let operator: LoginCredentials;
  let viewer: LoginCredentials;
  let unprivileged: LoginCredentials;
  let ownDeptId: number;
  let ownScopeId: number;
  let peerScopeId: number;

  async function create(
    request: APIRequestContext,
    path: string,
    data: Record<string, unknown>,
    verified = false,
  ) {
    const headers = verified ? await verifiedApiHeaders(request, admin) : apiRequestHeaders(admin);
    const row = await success<Row>(await request.post(`${apiBaseUrl}${path}`, { headers, data }));
    cleanup.unshift({ path: `${path}/${row.id}`, verified });
    return row.id;
  }

  test.beforeAll(async ({ request }) => {
    admin = await loginByApi(request, adminCredentials);
    const headers = authHeaders(admin.accessToken);
    const [root] = await success<Row[]>(await request.get(`${apiBaseUrl}/system/dept/tree`, { headers }));
    const menus = await success<MenuNode[]>(await request.get(`${apiBaseUrl}/system/menu/tree`, { headers }));
    const allowedMenuId = menuId(menus, credentialPage);
    expect(allowedMenuId).toBeTruthy();
    const taskMenuId = menuId(menus, taskPage);
    expect(taskMenuId).toBeTruthy();
    const createDepartment = (deptName: string) => create(request, '/system/dept', {
      parentId: root.id, deptName, status: 1, sort: 99,
    }, true);
    const ownDept = await createDepartment(ownName);
    ownDeptId = ownDept;
    const otherDept = await createDepartment(otherName);

    for (const [name, deptId] of [[ownName, ownDept], [otherName, otherDept]] as const) {
      const scopeId = await create(request, '/business/bizscope', {
        deptId, code: name, name: name === ownName ? ownScopeName : name, environment: 'dev', status: 'active',
      });
      if (name === ownName) ownScopeId = scopeId;
      await create(request, credentialApi, {
        businessScopeId: scopeId, name, username: 'smoke-user', authMode: 'password',
        secret: randomUUID(),
      });
    }
    peerScopeId = await create(request, '/business/bizscope', {
      deptId: ownDept, code: peerName, name: peerName, environment: 'dev', status: 'active',
    });
    await create(request, '/business/bizscope', {
      deptId: ownDept, code: inactiveScopeName, name: inactiveScopeName, environment: 'dev', status: 'inactive',
    });

    for (const [username, canMutate] of [[operatorName, true], [viewerName, false]] as const) {
      const roleId = await create(request, '/system/role', {
        roleName: username,
        roleKey: username,
        dataScope: 'dept',
        menuIds: canMutate ? [allowedMenuId, taskMenuId] : [allowedMenuId],
        permissionKeys: [
          'business:deploy:credential:list',
          ...(canMutate ? [
            'business:deploy:credential:create',
            'business:deploy:credential:update',
            'business:deploy:credential:delete',
            'business:deploy:task:view',
            'business:deploy:task:detail',
            'business:deploy:task:start',
          ] : []),
        ],
        sort: 99,
        status: 1,
      }, true);

      const login = { username, password: `T05!${randomUUID()}` };
      await create(request, '/system/user', {
        ...login, nickname: username, deptId: ownDept, roleIds: [roleId], status: 1,
      }, true);
      if (canMutate) operator = login;
      else viewer = login;
    }

    const deniedName = `${fixtureKey}-denied`;
    const deniedRoleId = await create(request, '/system/role', {
      roleName: deniedName, roleKey: deniedName, dataScope: 'dept', menuIds: [], permissionKeys: [], status: 1,
    }, true);
    unprivileged = { username: deniedName, password: `T05!${randomUUID()}` };
    await create(request, '/system/user', {
      ...unprivileged, nickname: deniedName, deptId: ownDept, roleIds: [deniedRoleId], status: 1,
    }, true);
  });

  test.afterAll(async ({ request }) => {
    for (const { path, verified } of cleanup) {
      const headers = verified ? await verifiedApiHeaders(request, admin) : apiRequestHeaders(admin);
      await success(await request.delete(`${apiBaseUrl}${path}`, { headers }));
    }
  });

  for (const [width, height, colorScheme] of [[1440, 900, 'light'], [390, 844, 'dark']] as const) {
    test(`read-only department role sees only its credentials at ${width}px`, async ({ page }, testInfo) => {
      await page.setViewportSize({ width, height });
      await page.emulateMedia({ colorScheme });
      await openCredentials(page, viewer);
      const table = page.locator('.system-list__table');
      const row = table.getByRole('row').filter({ hasText: ownName });
      await expect(row.getByRole('cell', { name: ownScopeName, exact: true })).toBeVisible();
      await expect(row.getByText('平台级（管理员）', { exact: true })).toHaveCount(0);
      await expect(table.getByText(otherName, { exact: true })).toHaveCount(0);
      await expect(page.getByRole('button', { name: /^(新增|编辑|删除)$/ })).toHaveCount(0);
      await expect(page.locator('html')).toHaveAttribute('data-color-mode', colorScheme);
      await capture(page, testInfo, `readonly-${width}-${colorScheme}`);
    });
  }

  test('user without the credential page grant sees access denied', async ({ page }, testInfo) => {
    await openCredentials(page, unprivileged);
    await expect(page.getByText('你没有访问当前页面的权限', { exact: true })).toBeVisible();
    await expect(page.locator('.system-list__table')).toHaveCount(0);
    await capture(page, testInfo, 'credential-access-denied');
  });

  test('department operator creates, rotates, and removes a credential in its own scope', async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await openCredentials(page, operator);
    const add = page.getByRole('button', { name: '新增', exact: true });
    await expect(page.getByRole('row').filter({ hasText: ownName })).toBeVisible();
    await add.focus();
    await expect(add).toBeFocused();
    await add.press('Enter');

    const dialog = page.getByRole('dialog');
    await expect(dialog).toHaveText(/新增部署凭据/);
    const name = `${fixtureKey}-created`;
    await dialog.getByPlaceholder('例如：生产主机 root').fill(name);
    await dialog.locator('.arco-select-view').filter({ hasText: '选择该凭据可用的业务域' }).click();
    const options = page.locator('.arco-select-option:visible');
    await expect(options.filter({ hasText: ownScopeName })).toBeVisible();
    await expect(options.filter({ hasText: otherName })).toHaveCount(0);
    await expect(options.filter({ hasText: inactiveScopeName })).toHaveCount(0);
    await options.filter({ hasText: ownScopeName }).click();
    await dialog.locator('.arco-form-item').filter({ hasText: '用户名' }).getByRole('textbox').fill('smoke-operator');
    await dialog.locator('input[autocomplete="new-password"]').fill(randomUUID());

    const postStarted = deferred();
    const releasePost = deferred();
    await page.route(`**/api/v1${credentialApi}`, async (route) => {
      if (route.request().method() === 'POST') {
        postStarted.resolve();
        await releasePost.promise;
      }
      await route.continue();
    });

    const createResponse = page.waitForResponse((response) =>
      response.request().method() === 'POST' && new URL(response.url()).pathname.endsWith(credentialApi),
    );
    const save = dialog.getByRole('button', { name: '保存', exact: true });
    try {
      await save.click();
      await postStarted.promise;
      await expect(save).toBeDisabled();
      await capture(page, testInfo, 'operator-submitting-1440');
    } finally {
      releasePost.resolve();
    }
    const created = await success<Row>(await createResponse);
    cleanup.unshift({ path: `${credentialApi}/${created.id}`, verified: false });
    await expect(dialog).toBeHidden();

    const row = page.getByRole('row').filter({ hasText: name });
    await row.getByRole('button', { name: '编辑', exact: true }).click();
    const secret = dialog.locator('input[autocomplete="new-password"]');
    await expect(secret).toHaveValue('');
    await secret.fill(randomUUID());
    const updateResponse = page.waitForResponse((response) =>
      response.request().method() === 'PUT' && new URL(response.url()).pathname.endsWith(`${credentialApi}/${created.id}`),
    );
    await save.click();
    expect((await success<{ version: number }>(await updateResponse)).version).toBe(2);
    await expect(dialog).toBeHidden();
    await expect(row.getByRole('cell', { name: '2', exact: true })).toBeVisible();

    await row.getByRole('button', { name: '删除', exact: true }).click();
    const deleteResponse = page.waitForResponse((response) =>
      response.request().method() === 'DELETE' && new URL(response.url()).pathname.endsWith(`${credentialApi}/${created.id}`),
    );
    await page.locator('.arco-popconfirm:visible').getByRole('button', { name: /确定|确认/ }).click();
    await success(await deleteResponse);
    cleanup.shift();
    await expect(row).toHaveCount(0);
  });

  test('credential list displays a failed fetch and retries as the read-only role', async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 390, height: 844 });
    const entered = deferred();
    const release = deferred();
    let fail = true;
    await page.route(`**/api/v1${credentialApi}`, async (route) => {
      if (!fail) {
        await route.continue();
        return;
      }
      entered.resolve();
      await release.promise;
      await route.fulfill({ status: 503, json: { code: 503, message: 'request.failed' } });
    });
    try {
      await openCredentials(page, viewer);
      await entered.promise;
      await expect(page.locator('.page-loading:visible')).toHaveCount(1);
      await capture(page, testInfo, 'readonly-loading-390');
    } finally {
      release.resolve();
    }
    const retry = page.getByRole('button', { name: '重试', exact: true });
    await expect(retry).toBeVisible();
    await capture(page, testInfo, 'readonly-load-error-390');
    fail = false;
    await retry.click();
    await expect(page.locator('.system-list__table').getByRole('row').filter({ hasText: ownName })).toBeVisible();
    await expect(retry).toHaveCount(0);
  });

  test('department operator selects an active credential when preparing a scoped task', async ({ page }, testInfo) => {
    const hostId = await create(page.request, '/business/cmdb/hosts', {
      hostname: `${fixtureKey}-host`,
      ip: '192.0.2.11',
      os: 'linux',
      businessScopeId: ownScopeId,
      deptId: ownDeptId,
    });
    const packageId = await create(page.request, '/business/deploy/packages', {
      name: `${fixtureKey}-package`, version: '1.0.0', status: 'enabled', installCommand: 'true',
    });
    const taskName = `${fixtureKey}-task`;
    const taskId = await create(page.request, '/business/deploy/tasks', {
      name: taskName, packageId, businessScopeId: ownScopeId, targetType: 'host',
      targetIds: [hostId], executorType: 'ssh',
    });
    const disabledId = await create(page.request, credentialApi, {
      businessScopeId: ownScopeId, name: `${fixtureKey}-disabled`, username: 'smoke-user',
      authMode: 'password', secret: randomUUID(),
    });
    await success(await page.request.put(`${apiBaseUrl}${credentialApi}/${disabledId}`, {
      headers: apiRequestHeaders(admin), data: { status: 'disabled' },
    }));
    await create(page.request, credentialApi, {
      businessScopeId: peerScopeId, name: peerName, username: 'smoke-user',
      authMode: 'password', secret: randomUUID(),
    });

    await page.setViewportSize({ width: 1440, height: 900 });
    const session = await loginByApi(page.request, operator);
    await installClientSession(page, session);
    await page.goto(taskPage, { waitUntil: 'domcontentloaded' });
    const row = page.getByRole('row').filter({ hasText: taskName });
    await row.getByRole('button', { name: '启动', exact: true }).click();
    const dialog = page.getByRole('dialog');
    await dialog.locator('.arco-select-view').filter({ hasText: '请选择 SSH 凭证' }).click();
    const options = page.locator('.arco-select-option:visible');
    await expect(options.filter({ hasText: ownName })).toBeVisible();
    await expect(options.filter({ hasText: otherName })).toHaveCount(0);
    await expect(options.filter({ hasText: peerName })).toHaveCount(0);
    await expect(options.filter({ hasText: `${fixtureKey}-disabled` })).toHaveCount(0);
    await options.filter({ hasText: ownName }).click();
    await expect(dialog.locator('.arco-select-view-value').filter({ hasText: ownName })).toBeVisible();
    await capture(page, testInfo, 'operator-select-list-credential-1440');
    await dialog.getByRole('button', { name: '取消', exact: true }).click();
    await expect(dialog).toBeHidden();

    await page.goto(`${taskPage}/${taskId}`, { waitUntil: 'domcontentloaded' });
    await page.getByRole('button', { name: '启动', exact: true }).click();
    await dialog.locator('.arco-select-view').filter({ hasText: '请选择 SSH 凭证' }).click();
    await expect(options.filter({ hasText: ownName })).toBeVisible();
    await expect(options.filter({ hasText: otherName })).toHaveCount(0);
    await expect(options.filter({ hasText: peerName })).toHaveCount(0);
    await expect(options.filter({ hasText: `${fixtureKey}-disabled` })).toHaveCount(0);
    await options.filter({ hasText: ownName }).click();
    await expect(dialog.locator('.arco-select-view-value').filter({ hasText: ownName })).toBeVisible();
    await capture(page, testInfo, 'operator-select-detail-credential-1440');
  });
});
