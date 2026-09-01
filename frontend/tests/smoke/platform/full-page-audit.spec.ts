/**
 * Full-page audit — one-off manual QA sweep (not part of CI gates).
 *
 * For every page in the app it:
 *  - logs in as admin and navigates
 *  - captures a full-page screenshot (desktop viewport)
 *  - collects console errors / pageerrors / 5xx API responses
 *  - checks for broken-state text and horizontal overflow (layout smell)
 *  - tries to open the page's primary "create" dialog and screenshots it,
 *    then checks the dialog for overflow / empty body
 *
 * Produces:
 *  - screenshots under test-results/full-page-audit/<key>.png (+ <key>__dialog.png)
 *  - a JSON findings file at test-results/full-page-audit/findings.json
 *
 * Run against an already-running app via:
 *   PANTHEON_EXTERNAL_WEB_SERVER=1 PANTHEON_WEB_BASE_URL=http://127.0.0.1:15173 \
 *   PANTHEON_API_BASE_URL=http://127.0.0.1:15173/api/v1 \
 *   npx playwright test tests/smoke/platform/full-page-audit.spec.ts --config playwright.config.ts --workers=1
 */
import fs from 'node:fs';
import path from 'node:path';
import { expect, test, type Locator, type Page } from '@playwright/test';
import {
  adminCredentials,
  installClientSession,
  loginByApi,
  primeChineseLocale,
} from '../helpers/auth';

type AuditPage = {
  key: string;
  path: string;
  title: string;
  layer: string;
  /** button label that opens the primary create dialog, if any */
  createButton?: string;
  /** expected dialog title after clicking createButton */
  dialogTitle?: string;
};

const auditPages: AuditPage[] = [
  { key: 'dashboard', layer: 'platform', path: '/dashboard', title: '工作台' },
  { key: 'auth-security', layer: 'system/auth', path: '/auth/security', title: '安全中心' },
  { key: 'login-log', layer: 'system/auth', path: '/system/login-log', title: '登录日志' },
  { key: 'session', layer: 'system/auth', path: '/system/session', title: '会话管理' },
  { key: 'profile', layer: 'system/iam', path: '/system/profile', title: '个人中心' },
  {
    key: 'user',
    layer: 'system/iam',
    path: '/system/user',
    title: '用户管理',
    createButton: '新增',
    dialogTitle: '新增用户',
  },
  { key: 'user-detail', layer: 'system/iam', path: '/system/user/1', title: '用户详情' },
  {
    key: 'role',
    layer: 'system/iam',
    path: '/system/role',
    title: '角色管理',
    createButton: '新增',
    dialogTitle: '新增角色',
  },
  {
    key: 'menu',
    layer: 'system/iam',
    path: '/system/menu',
    title: '菜单管理',
    createButton: '新增',
    dialogTitle: '新增菜单',
  },
  {
    key: 'permission',
    layer: 'system/iam',
    path: '/system/permission',
    title: '权限管理',
    createButton: '新增',
    dialogTitle: '新增权限',
  },
  { key: 'operation-log', layer: 'system/iam', path: '/system/operation-log', title: '操作日志' },
  {
    key: 'dept',
    layer: 'system/org',
    path: '/system/dept',
    title: '部门管理',
    createButton: '新增',
    dialogTitle: '新增部门',
  },
  {
    key: 'post',
    layer: 'system/org',
    path: '/system/post',
    title: '岗位管理',
    createButton: '新增',
    dialogTitle: '新增岗位',
  },
  {
    key: 'dict',
    layer: 'system/config',
    path: '/system/dict',
    title: '字典管理',
    createButton: '新增',
    dialogTitle: '新增字典类型',
  },
  { key: 'setting', layer: 'system/config', path: '/system/setting', title: '系统设置' },
  { key: 'i18n', layer: 'system/config', path: '/system/i18n', title: '国际化管理' },
  { key: 'modules', layer: 'system/lowcode', path: '/system/modules', title: '模块注册表' },
  { key: 'generator', layer: 'system/lowcode', path: '/system/generator', title: '模块生成器' },
];

const OUT_DIR = path.resolve(process.cwd(), 'test-results/full-page-audit');
fs.mkdirSync(OUT_DIR, { recursive: true });

type Finding = {
  key: string;
  path: string;
  title: string;
  layer: string;
  titleVisible: boolean;
  brokenState: string[];
  consoleErrors: string[];
  horizontalOverflow: { scrollW: number; clientW: number } | null;
  dialogOpened: boolean | null;
  dialogOverflow: boolean | null;
  dialogEmpty: boolean | null;
  notes: string[];
};

const findings: Finding[] = [];

function collectRuntimeErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on('console', (message) => {
    const text = message.text();
    if (
      message.type() === 'error' &&
      !/status of 404/i.test(text) &&
      !/ERR_CONNECTION_CLOSED/i.test(text) &&
      !/element\.ref was removed in React 19/i.test(text) &&
      !/favicon/i.test(text) &&
      !/Download the React DevTools/i.test(text)
    ) {
      errors.push(text.slice(0, 300));
    }
  });
  page.on('pageerror', (error) => errors.push(`[pageerror] ${error.message.slice(0, 300)}`));
  page.on('response', (response) => {
    if (response.status() >= 500 && response.url().includes('/api/')) {
      errors.push(`[5xx] ${response.status()} ${response.url()}`);
    }
  });
  return errors;
}

async function detectBrokenState(page: Page): Promise<string[]> {
  const markers = ['无权访问', '页面不存在', '加载失败', '网络异常', 'Something went wrong'];
  const hits: string[] = [];
  for (const m of markers) {
    const count = await page.getByText(m, { exact: false }).filter({ visible: true }).count();
    if (count > 0) hits.push(m);
  }
  return hits;
}

async function measureOverflow(
  scope: Page | Locator,
): Promise<{ scrollW: number; clientW: number } | null> {
  const handle = 'evaluate' in scope ? scope : scope;
  const res = await (handle as Page).evaluate(() => {
    const el = document.scrollingElement || document.documentElement;
    return { scrollW: el.scrollWidth, clientW: el.clientWidth };
  });
  // >4px slack tolerance
  return res.scrollW > res.clientW + 4 ? res : null;
}

test.describe('full page audit', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
  });

  test.afterAll(async () => {
    fs.writeFileSync(path.join(OUT_DIR, 'findings.json'), JSON.stringify(findings, null, 2));
  });

  // Regression guard for the shell layout: in vertical (sidebar) mode the Sider
  // must sit to the LEFT of the content column, not stacked above it. This
  // caught a real bug where `.arco-layout` kept its default `flex-direction:
  // column` because `.arco-layout-has-sider` was never emitted.
  test('shell layout: sidebar sits beside content (not stacked)', async ({ page }) => {
    const tokens = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, tokens);
    await page.goto('/system/user', { waitUntil: 'domcontentloaded' });
    await page.locator('.app-shell .arco-layout-sider').waitFor({ state: 'visible' });
    await page.waitForLoadState('domcontentloaded').catch(() => {});

    const geo = await page.evaluate(() => {
      const shell = document.querySelector('.app-shell');
      const sider = document.querySelector('.arco-layout-sider');
      const header = document.querySelector('.arco-layout-header');
      const rect = (el: Element | null) => {
        if (!el) return null;
        const r = el.getBoundingClientRect();
        return {
          x: Math.round(r.x),
          y: Math.round(r.y),
          w: Math.round(r.width),
          h: Math.round(r.height),
          right: Math.round(r.right),
        };
      };
      return {
        flexDir: shell ? getComputedStyle(shell).flexDirection : null,
        sider: rect(sider),
        header: rect(header),
      };
    });

    expect(geo.sider, 'sider should render').not.toBeNull();
    expect(geo.header, 'header should render').not.toBeNull();
    // Side-by-side means the header/content starts to the RIGHT of the sider's
    // right edge, and the sider is full-height (not a short stacked block).
    expect(geo.flexDir, 'shell should be a row so sider & content are columns').toBe('row');
    expect(geo.header!.x, 'header must start at/after the sider right edge').toBeGreaterThanOrEqual(
      geo.sider!.right - 4,
    );
    expect(geo.sider!.h, 'sider should be full-height, not a short stacked block').toBeGreaterThan(
      750,
    );
  });

  test('login page renders', async ({ page }) => {
    const errors = collectRuntimeErrors(page);
    await primeChineseLocale(page);
    await page.goto('/login', { waitUntil: 'domcontentloaded' });
    await expect(page.getByRole('button', { name: '登录' })).toBeVisible();
    await page.screenshot({ path: path.join(OUT_DIR, '00-login.png'), fullPage: true });
    findings.push({
      key: 'login',
      path: '/login',
      title: '登录',
      layer: 'auth',
      titleVisible: true,
      brokenState: [],
      consoleErrors: errors,
      horizontalOverflow: await measureOverflow(page),
      dialogOpened: null,
      dialogOverflow: null,
      dialogEmpty: null,
      notes: [],
    });
  });

  for (const ap of auditPages) {
    test(`${ap.layer}: ${ap.path}`, async ({ page }) => {
      const notes: string[] = [];
      const tokens = await loginByApi(page.request, adminCredentials);
      await installClientSession(page, tokens);
      const errors = collectRuntimeErrors(page);

      await page.goto(ap.path, { waitUntil: 'domcontentloaded' });
      // let async data settle
      await page.waitForLoadState('domcontentloaded').catch(() => notes.push('domcontentloaded timeout'));

      const titleVisible = await page
        .getByText(ap.title, { exact: false })
        .filter({ visible: true })
        .count()
        .then((c) => c > 0)
        .catch(() => false);
      if (!titleVisible) notes.push(`page title "${ap.title}" not visible`);

      const brokenState = await detectBrokenState(page);
      const horizontalOverflow = await measureOverflow(page);

      await page
        .screenshot({ path: path.join(OUT_DIR, `${ap.key}.png`), fullPage: true })
        .catch((e) => {
          notes.push(`screenshot failed: ${String(e).slice(0, 120)}`);
        });

      let dialogOpened: boolean | null = null;
      let dialogOverflow: boolean | null = null;
      let dialogEmpty: boolean | null = null;

      if (ap.createButton && ap.dialogTitle) {
        dialogOpened = false;
        try {
          const btn = page
            .getByRole('button', { name: ap.createButton, exact: true })
            .filter({ visible: true })
            .first();
          if ((await btn.count()) > 0) {
            await btn.click({ timeout: 5000 });
            const dialog = page
              .getByRole('dialog')
              .filter({ has: page.getByText(ap.dialogTitle, { exact: true }) });
            await expect(dialog).toBeVisible({ timeout: 5000 });
            dialogOpened = true;
            // Wait for the modal open transition to finish instead of a fixed delay.
            await dialog
              .evaluate((el) =>
                Promise.all(
                  el.getAnimations({ subtree: true }).map((a) => a.finished.catch(() => undefined)),
                ),
              )
              .catch(() => {});
            await page.screenshot({
              path: path.join(OUT_DIR, `${ap.key}__dialog.png`),
              fullPage: true,
              animations: 'disabled',
            });
            // dialog overflow / empty checks
            const box = await dialog.boundingBox();
            const bodyText = (await dialog.innerText().catch(() => '')).trim();
            dialogEmpty = bodyText.length < 3;
            dialogOverflow = await dialog
              .evaluate((el) => el.scrollWidth > el.clientWidth + 4)
              .catch(() => null);
            if (box && box.height > 860)
              notes.push(`dialog taller than viewport (${Math.round(box.height)}px)`);
            // close it (Escape)
            await page.keyboard.press('Escape').catch(() => {});
          } else {
            notes.push(`create button "${ap.createButton}" not found`);
          }
        } catch (e) {
          notes.push(`create dialog error: ${String(e).slice(0, 160)}`);
        }
      }

      findings.push({
        key: ap.key,
        path: ap.path,
        title: ap.title,
        layer: ap.layer,
        titleVisible,
        brokenState,
        consoleErrors: errors,
        horizontalOverflow,
        dialogOpened,
        dialogOverflow,
        dialogEmpty,
        notes,
      });

      // soft assertions — record, don't fail the whole sweep
      expect.soft(brokenState, `${ap.key}: broken-state text`).toEqual([]);
      expect.soft(errors, `${ap.key}: console errors`).toEqual([]);
    });
  }
});
