import { expect, test, type Locator, type Page, type Route } from '@playwright/test';
import { installOperationToken, signInAsAdmin } from '../../helpers/auth';
import { expectPagePathname } from '../../helpers/url-pattern';

type Deferred<T = void> = {
  promise: Promise<T>;
  resolve: (value: T | PromiseLike<T>) => void;
};

type GovernanceActionCase = {
  key: string;
  domain: 'system/auth' | 'system/iam';
  path: string;
  confirmText: string;
  confirmButtonLabel: string;
  successToast: string;
  errorToast: string;
  actionRoutePattern: RegExp;
  prepare: (page: Page) => Promise<Locator>;
};

// Log/session cleanup now opens a dialog with a shared irreversible-action
// warning (common.cleanupIrreversibleWarning) instead of an inline Popconfirm.
const CLEANUP_WARNING_TEXT = '清理后的记录不可恢复，请确认清理条件无误后再执行。';

function createDeferred<T = void>(): Deferred<T> {
  let resolve!: (value: T | PromiseLike<T>) => void;
  const promise = new Promise<T>((nextResolve) => {
    resolve = nextResolve;
  });
  return { promise, resolve };
}

function collectRuntimeErrors(page: Page) {
  const runtimeErrors: string[] = [];

  page.on('console', (message) => {
    if (message.type() === 'error') {
      runtimeErrors.push(message.text());
    }
  });

  page.on('pageerror', (error) => {
    runtimeErrors.push(error.message);
  });

  return runtimeErrors;
}

function expectNoRuntimeErrors(runtimeErrors: string[], allowedPatterns: RegExp[] = []) {
  const filteredErrors = runtimeErrors.filter((message) => !(
    /Failed to load resource: the server responded with a status of 502/i.test(message)
    || /Failed to load i18n pack RequestError: Request failed with status code 502/i.test(message)
    || allowedPatterns.some((pattern) => pattern.test(message))
  ));
  expect(filteredErrors).toEqual([]);
}

async function fulfillJson(route: Route, status: number, body: Record<string, unknown>) {
  await route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
}

async function expectToast(page: Page, text: string) {
  await expect(page.locator('.arco-message').getByText(text, { exact: false }).last()).toBeVisible();
}

async function waitForConfirmPopup(page: Page, text: string) {
  const popup = page
    .locator('.arco-popconfirm:visible, .arco-trigger-popup:visible, .arco-popover:visible, [role="tooltip"]:visible, [role="dialog"]:visible')
    .filter({ has: page.getByText(text, { exact: true }) })
    .last();
  await expect(popup).toBeVisible();
  await expect(popup.getByText(text, { exact: true })).toBeVisible();
  return popup;
}

function confirmButton(popup: Locator, label = '确定') {
  return popup.getByRole('button', { name: label, exact: true }).last();
}

async function completeSecondaryVerifyIfVisible(page: Page, password = '123456') {
  const dialog = page.getByRole('dialog', { name: '敏感操作验证' });
  try {
    await dialog.waitFor({ state: 'visible', timeout: 1500 });
  } catch {
    return;
  }
  await dialog.getByRole('textbox', { name: '密码' }).fill(password);
  await dialog.getByRole('button', { name: '确定', exact: true }).click();
}

async function mockAuditSettings(page: Page) {
  await page.route(/\/api\/v1\/system\/setting\/group\/audit$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: {
        groupKey: 'audit',
        items: [
          {
            settingKey: 'audit.login_log_retention_options',
            settingValue: '[1,7,30]',
          },
          {
            settingKey: 'audit.operation_log_retention_options',
            settingValue: '[1,7,30]',
          },
        ],
      },
    });
  });
}

async function prepareSessionRevoke(page: Page) {
  await page.route(/\/api\/v1\/system\/session\/list(?:\?.*)?$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: {
        items: [
          {
            sessionId: 'session-matrix-1',
            userId: 9001,
            username: 'matrix_user',
            nickname: '矩阵用户',
            lastIp: '127.0.0.2',
            browser: 'Chrome',
            os: 'Windows',
            device: 'Desktop',
            userAgent: 'Mozilla/5.0',
            refreshExpiresAt: '2026-05-01 08:00:00',
            lastRefreshAt: '2026-04-29 08:00:00',
            lastActivityAt: '2026-04-29 08:30:00',
            revokedAt: '',
            createdAt: '2026-04-29 08:00:00',
          },
        ],
        total: 1,
        activeCount: 1,
        revokedCount: 0,
        page: 1,
        pageSize: 10,
      },
    });
  });

  await page.goto('/system/session', { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: '下线会话', exact: true }).first().click();
  return waitForConfirmPopup(page, '确认下线该会话？');
}

async function prepareLoginCleanup(page: Page) {
  await mockAuditSettings(page);
  await page.route(/\/api\/v1\/system\/login-log\/list(?:\?.*)?$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: {
        items: [
          {
            id: 1001,
            username: 'admin',
            ipaddr: '127.0.0.1',
            loginLocation: '本地',
            browser: 'Chrome',
            os: 'Windows',
            status: 1,
            msg: '',
            loginTime: '2026-04-29 09:00:00',
          },
        ],
        total: 1,
        page: 1,
        pageSize: 10,
      },
    });
  });

  await page.goto('/system/login-log', { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: '清理日志', exact: true }).click();
  return waitForConfirmPopup(page, CLEANUP_WARNING_TEXT);
}

async function prepareOperationCleanup(page: Page) {
  await mockAuditSettings(page);
  await page.route(/\/api\/v1\/system\/operation-log\/list(?:\?.*)?$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: {
        items: [
          {
            id: 2001,
            title: 'system.user.create',
            businessType: 1,
            method: 'POST',
            operName: 'admin',
            operUrl: '/api/v1/system/user',
            operIp: '127.0.0.1',
            sourceDomain: 'iam',
            sourcePage: 'user',
            operParam: '{}',
            jsonResult: '{"code":200,"message":"ok"}',
            status: 1,
            failureCategory: '',
            errorMsg: '',
            operTime: '2026-04-29 10:00:00',
            costTime: 18,
          },
        ],
        total: 1,
        page: 1,
        pageSize: 10,
      },
    });
  });

  await page.goto('/system/operation-log', { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: '清理日志', exact: true }).click();
  return waitForConfirmPopup(page, CLEANUP_WARNING_TEXT);
}

async function prepareModuleUnregister(page: Page) {
  await page.route(/\/api\/v1\/lowcode\/dynamic-modules(?:\?.*)?$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: [
        {
          id: 3001,
          name: 'biz_matrix',
          displayName: '矩阵模块',
          scope: 'business',
          tableName: 'biz_matrix',
          status: 1,
          installedAt: '2026-04-29 11:00:00',
          builtIn: false,
        },
      ],
    });
  });

  await page.goto('/system/modules', { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: '卸载', exact: true }).click();
  return waitForConfirmPopup(page, '确认卸载该模块吗？');
}

const actionCases: GovernanceActionCase[] = [
  {
    key: 'session-revoke',
    domain: 'system/auth',
    path: '/system/session',
    confirmText: '确认下线该会话？',
    confirmButtonLabel: '确定',
    successToast: '会话已下线',
    errorToast: '请求失败，请稍后重试',
    actionRoutePattern: /\/api\/v1\/system\/session\/session-matrix-1$/,
    prepare: prepareSessionRevoke,
  },
  {
    key: 'login-log-cleanup',
    domain: 'system/auth',
    path: '/system/login-log',
    confirmText: CLEANUP_WARNING_TEXT,
    confirmButtonLabel: '清理',
    successToast: '已清理 1 条登录日志',
    errorToast: '请求失败，请稍后重试',
    actionRoutePattern: /\/api\/v1\/system\/login-log\/cleanup$/,
    prepare: prepareLoginCleanup,
  },
  {
    key: 'operation-log-cleanup',
    domain: 'system/iam',
    path: '/system/operation-log',
    confirmText: CLEANUP_WARNING_TEXT,
    confirmButtonLabel: '清理',
    successToast: '已清理 1 条历史操作日志，并记录 1 条清理审计',
    errorToast: '请求失败，请稍后重试',
    actionRoutePattern: /\/api\/v1\/system\/operation-log\/cleanup$/,
    prepare: prepareOperationCleanup,
  },
  {
    key: 'module-unregister',
    domain: 'system/iam',
    path: '/system/modules',
    confirmText: '确认卸载该模块吗？',
    confirmButtonLabel: '确定',
    successToast: '模块已卸载',
    errorToast: '模块卸载失败',
    actionRoutePattern: /\/api\/v1\/lowcode\/dynamic-modules\/biz_matrix\?dropTable=false$/,
    prepare: prepareModuleUnregister,
  },
];

test.describe('system governance action matrix', () => {
  test.describe.configure({ timeout: 180000 });

  test('confirm copy matrix keeps governance language natural', async ({ page }) => {
    for (const actionCase of actionCases) {
      const casePage = await page.context().newPage();
      const runtimeErrors = collectRuntimeErrors(casePage);

      try {
        const accessToken = await signInAsAdmin(casePage);
        await casePage.goto('/dashboard', { waitUntil: 'networkidle' });
        await installOperationToken(casePage, accessToken);
        const popup = await actionCase.prepare(casePage);
        await expect(popup.getByText(actionCase.confirmText, { exact: true })).toBeVisible();
        expectPagePathname(casePage, actionCase.path);
        expectNoRuntimeErrors(runtimeErrors);
      } finally {
        await casePage.close();
      }
    }
  });

  test('submitting matrix dismisses confirms and keeps governance pages stable', async ({ page }) => {
    for (const actionCase of actionCases) {
      const casePage = await page.context().newPage();
      const runtimeErrors = collectRuntimeErrors(casePage);
      const gate = createDeferred<void>();
      let intercepted = false;

      try {
        const accessToken = await signInAsAdmin(casePage);
        await casePage.goto('/dashboard', { waitUntil: 'networkidle' });
        await installOperationToken(casePage, accessToken);
        await casePage.route(actionCase.actionRoutePattern, async (route) => {
          intercepted = true;
          await gate.promise;
          const payload = actionCase.key.includes('cleanup')
            ? { code: 200, data: { clearedCount: 1 } }
            : actionCase.key.includes('batch-delete')
              ? { code: 200, data: { deletedCount: 1 } }
              : actionCase.key === 'module-unregister'
                ? { code: 200, data: { unregistered: true, message: 'ok' } }
                : { code: 200, data: { revoked: true } };
          await fulfillJson(route, 200, payload);
        });

        const popup = await actionCase.prepare(casePage);
        const submit = confirmButton(popup, actionCase.confirmButtonLabel);
        await submit.click({ noWaitAfter: true });
        await completeSecondaryVerifyIfVisible(casePage);

        await expect.poll(() => intercepted).toBeTruthy();

        gate.resolve();
        await expect(
          casePage
            .locator(
              '.governance-summary-bar, .system-list__table-card, .module-manager-page, .auth-security-page',
            )
            .first(),
        ).toBeVisible();
        expectNoRuntimeErrors(runtimeErrors);
      } finally {
        await casePage.close();
      }
    }
  });

  test('server error matrix keeps governance actions friendly and recoverable', async ({ page }) => {
    for (const actionCase of actionCases) {
      const casePage = await page.context().newPage();
      const runtimeErrors = collectRuntimeErrors(casePage);

      try {
        const accessToken = await signInAsAdmin(casePage);
        await casePage.goto('/dashboard', { waitUntil: 'networkidle' });
        await installOperationToken(casePage, accessToken);
        await casePage.route(actionCase.actionRoutePattern, async (route) => {
          await fulfillJson(route, 500, {
            code: 500,
            message: 'request.failed',
          });
        });

        const popup = await actionCase.prepare(casePage);
        await confirmButton(popup, actionCase.confirmButtonLabel).click({ noWaitAfter: true });
        await completeSecondaryVerifyIfVisible(casePage);

        await expectToast(casePage, actionCase.errorToast);
        await expect(
          casePage
            .locator(
              '.governance-summary-bar, .system-list__table-card, .module-manager-page, .auth-security-page',
            )
            .first(),
        ).toBeVisible();
        expectNoRuntimeErrors(runtimeErrors, [
          /Failed to load resource: the server responded with a status of 500/i,
          /^request\.failed$/,
          /RequestError:\s*request\.failed/i,
        ]);
      } finally {
        await casePage.close();
      }
    }
  });
});
