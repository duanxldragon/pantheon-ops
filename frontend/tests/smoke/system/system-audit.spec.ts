import { test } from '../../fixtures/coverage';
import { expect } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  expectNoPageError,
  expectPageIdentityReady,
  expectVisiblePageTitle,
  requestHeaders,
  signInAsAdmin,
  withListViewOnlyViewer,
} from './system-pages-helpers';

test.beforeEach(async ({ page }) => {
  await signInAsAdmin(page);
});

test('operation log smoke: failed reason and detail summary are visible', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const failedUploadResponse = await page.request.post(
    `${apiBaseUrl}/system/upload?scope=profile/avatar`,
    {
      headers: await requestHeaders(page, accessToken),
      multipart: {
        file: {
          name: 'audit-failure.txt',
          mimeType: 'text/plain',
          buffer: Buffer.from('audit-failure', 'utf8'),
        },
      },
    },
  );
  expect(failedUploadResponse.ok()).toBeTruthy();
  const failedUploadPayload = await failedUploadResponse.json();
  expect(failedUploadPayload.code).not.toBe(200);
  expect(failedUploadPayload.message).toBe('upload.file.type_not_allowed');

  await page.goto('/system/operation-log', { waitUntil: 'domcontentloaded' });
  // SearchToolbar：关键词 + 行内下拉即时触发；低频筛选在“筛选”弹层内。
  // keyword 按存储值匹配（title 列存 i18n key、oper_name、request_id），
  // 所以这里用操作人 admin 覆盖关键词交互，行定位仍用渲染后的中文标题。
  const auditToolbar = page.locator('.search-toolbar');
  await auditToolbar.getByPlaceholder(/搜索/).fill('admin');
  await auditToolbar.getByPlaceholder(/搜索/).press('Enter');
  await auditToolbar.locator('.arco-select-view').first().click();
  await page.locator('.arco-select-option').filter({ hasText: '失败' }).first().click();
  await auditToolbar.locator('.arco-select-view').nth(1).click();
  await page.locator('.arco-select-option').filter({ hasText: '系统配置' }).first().click();
  await auditToolbar.locator('.search-toolbar__advanced-trigger').click();
  const auditPopover = page.locator('.search-toolbar__popover');
  await expect(auditPopover).toBeVisible();
  await auditPopover.locator('.arco-select-view').first().click();
  await page.locator('.arco-select-option').filter({ hasText: '参数/校验失败' }).first().click();
  await auditPopover.locator('.arco-select-view').nth(1).click();
  await page.locator('.arco-select-option').filter({ hasText: '上传配置' }).first().click();
  await page.keyboard.press('Escape');

  const firstRow = page.getByRole('row', { name: /上传文件/ }).first();
  await expect(firstRow).toBeVisible();
  await expect(firstRow).toContainText('上传文件');
  await expect(firstRow).toContainText('系统配置');
  await expect(firstRow).toContainText('当前文件类型不在允许范围内');
  await firstRow.getByRole('button', { name: '详情' }).click();

  const detailDialog = page.getByRole('dialog');
  await expect(detailDialog.getByText('失败原因')).toBeVisible();
  await expect(detailDialog.getByText('参数/校验失败').first()).toBeVisible();
  await expect(detailDialog.getByText('系统配置').first()).toBeVisible();
  await expect(detailDialog.getByText('上传配置').first()).toBeVisible();
  await expect(detailDialog.getByText('当前文件类型不在允许范围内').first()).toBeVisible();
  await expect(detailDialog.getByText('upload.file.type_not_allowed').first()).toBeVisible();
});

test('login-log permission smoke: list-only role can view page but cannot clear, export, or batch delete', async ({
  page,
}) => {
  await withListViewOnlyViewer(
    page,
    {
      slug: 'login_log',
      cleanupTag: 'system-pages:login-log-viewer',
      roleName: '登录日志只读烟测角色',
      nickname: '登录日志只读烟测用户',
      listPermission: 'system:login-log:list',
      apiPermissions: [
        ['/api/v1/system/login-log/list', 'GET'],
        ['/api/v1/system/setting/group/audit', 'GET'],
      ],
      route: '/system/login-log',
      identityLabel: '登录日志',
    },
    async (viewerPage) => {
      await expect(viewerPage.getByRole('button', { name: '导出' })).toBeDisabled();
      await expect(viewerPage.getByRole('button', { name: '清理日志' })).toHaveCount(0);
      await expect(viewerPage.getByRole('button', { name: '删除所选' })).toHaveCount(0);
    },
  );
});

test('session permission smoke: list-only role can view page but cannot revoke or clear sessions', async ({
  page,
}) => {
  await withListViewOnlyViewer(
    page,
    {
      slug: 'session',
      cleanupTag: 'system-pages:session-viewer',
      roleName: '会话只读烟测角色',
      nickname: '会话只读烟测用户',
      listPermission: 'system:session:list',
      apiPermissions: [['/api/v1/system/session/list', 'GET']],
      route: '/system/session',
      identityLabel: '会话管理',
    },
    async (viewerPage) => {
      await expect(viewerPage.getByRole('button', { name: '清理历史会话' })).toHaveCount(0);
      const revokeButtons = viewerPage.getByRole('button', { name: '下线会话' });
      await expect(revokeButtons.first()).toBeDisabled();
    },
  );
});

test('operation-log permission smoke: list-only role can view page but cannot clear, export, or batch delete', async ({
  page,
}) => {
  await withListViewOnlyViewer(
    page,
    {
      slug: 'operation_log',
      cleanupTag: 'system-pages:operation-log-viewer',
      roleName: '操作日志只读烟测角色',
      nickname: '操作日志只读烟测用户',
      listPermission: 'system:operation-log:list',
      apiPermissions: [
        ['/api/v1/system/operation-log/list', 'GET'],
        ['/api/v1/system/setting/group/audit', 'GET'],
      ],
      route: '/system/operation-log',
      identityLabel: '操作日志',
    },
    async (viewerPage) => {
      await expect(viewerPage.getByRole('button', { name: '导出' })).toBeDisabled();
      await expect(viewerPage.getByRole('button', { name: '清理日志' })).toHaveCount(0);
      await expect(viewerPage.getByRole('button', { name: '删除所选' })).toHaveCount(0);
    },
  );
});

test('login-log governance smoke: selecting rows enables batch delete affordance', async ({
  page,
}) => {
  await signInAsAdmin(page);
  await page.goto('/system/login-log', { waitUntil: 'domcontentloaded' });
  await expectPageIdentityReady(page, '登录日志');
  await expectNoPageError(page);

  const batchDeleteButton = page.getByRole('button', { name: '删除所选' });
  await expect(batchDeleteButton).toBeDisabled();
  await page.locator('.system-list__table .arco-checkbox').nth(1).click({ force: true });
  await expect(batchDeleteButton).toBeEnabled();
});

test('login-log governance smoke: pager exposes first and last page controls', async ({ page }) => {
  const pageSize = 10;
  const total = pageSize + 1;
  const totalPages = Math.ceil(total / pageSize);
  const loginLogItems = Array.from({ length: total }, (_, index) => ({
    id: 910000 + index,
    username: `${adminCredentials.username}-${index + 1}`,
    ipaddr: '127.0.0.1',
    loginLocation: '本地',
    browser: 'Chrome',
    os: 'Windows',
    status: 1,
    msg: '',
    loginTime: `2026-05-${String((index % 28) + 1).padStart(2, '0')} 09:00:00`,
  }));

  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/system\/login-log\/list(?:\?.*)?$/, async (route) => {
    const requestUrl = new URL(route.request().url());
    const currentPage = Number(requestUrl.searchParams.get('page') || '1');
    const currentPageSize = Number(requestUrl.searchParams.get('pageSize') || String(pageSize));
    const startIndex = (currentPage - 1) * currentPageSize;
    const items = loginLogItems.slice(startIndex, startIndex + currentPageSize);

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items,
          total,
          page: currentPage,
          pageSize: currentPageSize,
        },
      }),
    });
  });

  await page.goto('/system/login-log', { waitUntil: 'domcontentloaded' });
  await expectPageIdentityReady(page, '登录日志');
  await expectNoPageError(page);

  const pager = page.locator('.system-list__table').first();
  const firstPageButton = pager.getByRole('button', { name: /^(首页|First page)$/ });
  const lastPageButton = pager.getByRole('button', { name: /^(末页|Last page)$/ });
  const prevPageButton = pager.locator('.arco-pagination-item-prev');
  const nextPageButton = pager.locator('.arco-pagination-item-next');
  await expect(prevPageButton.getByRole('button', { name: /^(首页|First page)$/ })).toBeVisible();
  await expect(nextPageButton.getByRole('button', { name: /^(末页|Last page)$/ })).toBeVisible();
  await expect(pager.locator('.arco-pagination-item-active')).toContainText('1');
  await expect(firstPageButton).toBeVisible();
  await expect(lastPageButton).toBeVisible();
  await expect(firstPageButton).toBeDisabled();
  await expect(prevPageButton).toHaveClass(/arco-pagination-item-disabled/);
  await expect(nextPageButton).not.toHaveClass(/arco-pagination-item-disabled/);
  await expect(lastPageButton).toBeEnabled();

  await lastPageButton.click();
  await expect
    .poll(async () => pager.locator('.arco-pagination-item-active').innerText())
    .toBe(String(totalPages));
  await expect(firstPageButton).toBeEnabled();
  await expect(prevPageButton).not.toHaveClass(/arco-pagination-item-disabled/);
  await expect(nextPageButton).toHaveClass(/arco-pagination-item-disabled/);
  await expect(lastPageButton).toBeDisabled();

  await firstPageButton.click();
  await expect
    .poll(async () => pager.locator('.arco-pagination-item-active').innerText())
    .toBe('1');
  await expect(firstPageButton).toBeDisabled();
  await expect(prevPageButton).toHaveClass(/arco-pagination-item-disabled/);
  await expect(nextPageButton).not.toHaveClass(/arco-pagination-item-disabled/);
  await expect(lastPageButton).toBeEnabled();
});

test('login-log governance smoke: single-page data keeps boundary pager controls disabled', async ({
  page,
}) => {
  const pageSize = 10;
  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/system\/login-log\/list(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [
            {
              id: 920001,
              username: adminCredentials.username,
              ipaddr: '127.0.0.1',
              loginLocation: '本地',
              browser: 'Chrome',
              os: 'Windows',
              status: 1,
              msg: '',
              loginTime: '2026-05-20 09:00:00',
            },
          ],
          total: 1,
          page: 1,
          pageSize,
        },
      }),
    });
  });

  await page.goto('/system/login-log', { waitUntil: 'domcontentloaded' });
  await expectPageIdentityReady(page, '登录日志');
  await expectNoPageError(page);

  const pager = page.locator('.system-list__table').first();
  await expect(pager.locator('.arco-pagination')).toBeVisible();
  await expect(pager.getByRole('button', { name: /^(首页|First page)$/ })).toHaveCount(0);
  await expect(pager.getByRole('button', { name: /^(末页|Last page)$/ })).toHaveCount(0);
});

test('security-center smoke: client-side AppTable pagination exposes shared boundary controls', async ({
  page,
}) => {
  const sessions = Array.from({ length: 6 }, (_, index) => ({
    sessionId: `client-session-${index + 1}`,
    isCurrent: index === 0,
    lastIp: `127.0.0.${index + 1}`,
    browser: 'Chrome',
    os: 'Windows',
    device: `Device ${index + 1}`,
    userAgent: 'Mozilla/5.0',
    refreshExpiresAt: '2026-05-21 09:00:00',
    lastRefreshAt: '2026-05-20 09:00:00',
    lastActivityAt: '2026-05-20 09:00:00',
    createdAt: `2026-05-${String(index + 1).padStart(2, '0')} 08:00:00`,
  }));

  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/auth\/security$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          user: {
            id: 1,
            username: adminCredentials.username,
            nickname: '管理员',
          },
          currentSession: sessions[0],
          activeSessionCount: sessions.length,
          lastLoginAt: '2026-05-20 09:00:00',
          passwordExpired: false,
          recentSecurityEvents: [],
          policy: {
            passwordMinLength: 8,
            passwordRequireDigit: true,
            passwordRequireUpper: false,
            passwordHistoryLimit: 3,
            passwordExpireDays: 90,
            maxFailedAttempts: 5,
            lockMinutes: 30,
            sourceMaxFailedAttempts: 5,
            sourceWindowMinutes: 15,
            sourceLockMinutes: 30,
            sessionIdleMinutes: 30,
            maxActiveSessions: 10,
            sessionRetentionDays: 30,
            captchaEnabled: true,
            mfaEnabled: false,
            ssoEnabled: false,
          },
        },
      }),
    });
  });
  await page.route(/\/api\/v1\/auth\/sessions$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: sessions,
      }),
    });
  });
  await page.route(/\/api\/v1\/auth\/login-logs(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [],
          total: 0,
          page: 1,
          pageSize: 10,
        },
      }),
    });
  });

  await page.goto('/auth/security', { waitUntil: 'domcontentloaded' });
  await expectVisiblePageTitle(page, '安全中心');
  await expectNoPageError(page);
  await expect(page.locator('.page-split-layout--with-rail')).toBeVisible();
  await expect(
    page
      .locator('.page-main-column .arco-card')
      .filter({ hasText: /在线会话|Active Sessions/ })
      .first(),
  ).toBeVisible();

  const sessionCard = page
    .locator('.page-panel')
    .filter({ hasText: /在线会话|Active Sessions/ })
    .first();
  const firstPageButton = sessionCard.getByRole('button', { name: /^(首页|First page)$/ });
  const lastPageButton = sessionCard.getByRole('button', { name: /^(末页|Last page)$/ });
  const prevPageButton = sessionCard.locator('.arco-pagination-item-prev');
  const nextPageButton = sessionCard.locator('.arco-pagination-item-next');
  await expect(prevPageButton.getByRole('button', { name: /^(首页|First page)$/ })).toBeVisible();
  await expect(nextPageButton.getByRole('button', { name: /^(末页|Last page)$/ })).toBeVisible();
  await expect(firstPageButton).toBeVisible();
  await expect(lastPageButton).toBeVisible();
  await expect(firstPageButton).toBeDisabled();
  await expect(prevPageButton).toHaveClass(/arco-pagination-item-disabled/);
  await expect(nextPageButton).not.toHaveClass(/arco-pagination-item-disabled/);
  await expect(lastPageButton).toBeEnabled();

  await lastPageButton.click();
  await expect
    .poll(async () => sessionCard.locator('.arco-pagination-item-active').innerText())
    .toBe('2');
  await expect(nextPageButton).toHaveClass(/arco-pagination-item-disabled/);
  await expect(lastPageButton).toBeDisabled();
  await expect(firstPageButton).toBeEnabled();
});

test('operation-log governance smoke: selecting rows enables batch delete affordance', async ({
  page,
}) => {
  await signInAsAdmin(page);
  await page.goto('/system/operation-log', { waitUntil: 'domcontentloaded' });
  await expectPageIdentityReady(page, '操作日志');
  await expectNoPageError(page);

  const batchDeleteButton = page.getByRole('button', { name: '删除所选' });
  await expect(
    page.locator('.table-batch-action-bar').getByRole('button', { name: '导出' }),
  ).toBeVisible();
  await expect(batchDeleteButton).toBeDisabled();
  await page.locator('.system-list__table .arco-checkbox').nth(1).click({ force: true });
  await expect(batchDeleteButton).toBeEnabled();
});

test('session governance smoke: revocation uses the shared table batch bar', async ({
  page,
}) => {
  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/system\/session\/list(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [
            {
              sessionId: 'smoke-self-session',
              userId: 1,
              username: adminCredentials.username,
              nickname: 'Admin',
              lastIp: '127.0.0.1',
              browser: 'Chrome',
              os: 'Windows',
              device: 'Desktop',
              userAgent: 'Chrome/136',
              refreshExpiresAt: '2026-05-21T10:00:00+08:00',
              lastRefreshAt: '2026-05-20T09:30:00+08:00',
              lastActivityAt: '2026-05-20T09:30:00+08:00',
              revokedAt: null,
              createdAt: '2026-05-20T08:00:00+08:00',
            },
            {
              sessionId: 'smoke-target-session',
              userId: 2,
              username: 'other-admin',
              nickname: 'Other Admin',
              lastIp: '10.0.0.8',
              browser: 'Edge',
              os: 'Windows',
              device: 'Desktop',
              userAgent: 'Edge/136',
              refreshExpiresAt: '2026-05-21T10:00:00+08:00',
              lastRefreshAt: '2026-05-20T08:30:00+08:00',
              lastActivityAt: '2026-05-20T08:35:00+08:00',
              revokedAt: null,
              createdAt: '2026-05-20T07:00:00+08:00',
            },
          ],
          total: 2,
          activeCount: 2,
          revokedCount: 0,
          page: 1,
          pageSize: 10,
        },
      }),
    });
  });
  await page.goto('/system/session', { waitUntil: 'domcontentloaded' });
  await expectPageIdentityReady(page, '会话管理');
  await expectNoPageError(page);

  // 治理摘要：迁移到 GovernanceSummaryBar（eyebrow + title 两行）。
  // 与基线 selector 保持一致（shell-visual-contract / governance-insight-drawer），
  // 避免 Arco <Space> 包裹打断 `.system-page-template >` 的直系选择器。
  const summaryBar = page.locator('.governance-summary-bar').first();
  await expect(summaryBar).toBeVisible();
  await expect(summaryBar.locator('.governance-summary-bar__title-row')).toBeVisible();

  // 批量下线 + 手动清理共用 GovernanceCleanupBar（--governance 修饰；
  // 2026-07-20 维护者决策恢复受控手动清理入口，自动保留策略仍为主）
  const batchBar = page.locator('.page-panel.system-list__table-card .table-batch-action-bar').first();
  await expect(batchBar).toBeVisible();
  await expect(batchBar).toHaveClass(/table-batch-action-bar--governance/);
  const revokeSelectedButton = batchBar.getByRole('button', { name: '下线所选' });
  await expect(revokeSelectedButton).toBeDisabled();

  // 手动清理入口存在且受权限保护（admin 会话可见）
  await expect(batchBar.getByRole('button', { name: '清理历史会话', exact: true })).toBeVisible();

  // 选中一行后，批量下线按钮启用
  await page.locator('.app-table tbody .arco-checkbox').nth(1).click({ force: true });
  await expect(revokeSelectedButton).toBeEnabled();
});

test('security-event governance smoke: pending event can be acknowledged with a note', async ({
  page,
}) => {
  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/system\/security-event\/list(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [
            {
              id: 9001,
              userId: 7,
              username: 'risk-user',
              eventType: 'source_blocked',
              severity: 'high',
              sourceKey: 'ip:10.0.0.9',
              ip: '10.0.0.9',
              userAgent: 'Chrome',
              messageKey: 'auth.security.event.source_blocked',
              metadata: '{}',
              createdAt: '2026-05-20T10:00:00+08:00',
              acknowledgedAt: null,
              acknowledgedBy: 0,
              acknowledgedByUser: '',
              acknowledgementNote: '',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 10,
        },
      }),
    });
  });

  let acknowledgePayload: Record<string, unknown> | null = null;
  await page.route(/\/api\/v1\/system\/security-event\/9001\/acknowledge$/, async (route) => {
    acknowledgePayload = route.request().postDataJSON() as Record<string, unknown>;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 200, data: { acknowledged: true } }),
    });
  });

  await page.goto('/system/security-event', { waitUntil: 'domcontentloaded' });
  await expectPageIdentityReady(page, '安全事件');
  await expectNoPageError(page);

  await page.getByRole('button', { name: '确认事件' }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('textbox').fill('来源已复核，风险已处置');
  await dialog.locator('.arco-btn-primary').last().click();

  await expect.poll(() => acknowledgePayload).not.toBeNull();
  expect(acknowledgePayload).toMatchObject({
    acknowledgementNote: '来源已复核，风险已处置',
  });
});
