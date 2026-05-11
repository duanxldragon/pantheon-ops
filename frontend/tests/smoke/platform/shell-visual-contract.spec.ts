import { expect, test } from '@playwright/test';
import { signInAsAdmin } from '../helpers/auth';

const systemTablePages = [
  { path: '/system/user', title: '用户管理' },
  { path: '/system/role', title: '角色管理' },
  { path: '/system/permission', title: '权限管理' },
  { path: '/system/menu', title: '菜单管理' },
  { path: '/system/dept', title: '部门管理' },
  { path: '/system/post', title: '岗位管理' },
  { path: '/system/dict', title: '字典管理' },
  { path: '/system/setting', title: '系统设置' },
  { path: '/system/i18n', title: '国际化管理' },
  { path: '/system/modules', title: '模块注册表' },
  { path: '/system/session', title: '会话管理' },
  { path: '/system/login-log', title: '登录日志' },
  { path: '/system/operation-log', title: '操作日志' },
] as const;

const filterPanelPages = [
  '/system/user',
  '/system/role',
  '/system/permission',
  '/system/menu',
  '/system/dept',
  '/system/post',
  '/system/dict',
  '/system/i18n',
  '/system/session',
  '/system/login-log',
  '/system/operation-log',
] as const;

const governanceBarPages = [
  '/system/session',
  '/system/login-log',
  '/system/operation-log',
] as const;

async function navigateInShell(page: import('@playwright/test').Page, path: string) {
  if (page.url() === 'about:blank') {
    await page.goto('/dashboard', { waitUntil: 'networkidle' });
  }
  await page.evaluate((nextPath) => {
    window.history.pushState({}, '', nextPath);
    window.dispatchEvent(new PopStateEvent('popstate'));
  }, path);
  await expect(page).toHaveURL(new RegExp(`${path.replace(/\//g, '\\/')}$`));
}

test('platform shell breadcrumb and function bars do not clip text or use inset mixed borders', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);
  await navigateInShell(page, '/system/user');
  await expect(page.locator('.table-batch-action-bar')).toBeVisible();

  const userShellStyles = await page.evaluate(() => {
    const read = (selector: string) => {
      const element = document.querySelector<HTMLElement>(selector);
      if (!element) {
        return null;
      }
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return {
        selector,
        backgroundColor: style.backgroundColor,
        borderColor: style.borderColor,
        borderStyle: style.borderStyle,
        borderWidth: style.borderWidth,
        boxShadow: style.boxShadow,
        height: style.height,
        lineHeight: style.lineHeight,
        overflow: style.overflow,
        top: rect.top,
        bottom: rect.bottom,
      };
    };

    return {
      header: read('.app-shell__header'),
      breadcrumb: read('.app-shell__header-breadcrumb'),
      breadcrumbItem: read('.app-shell__header .arco-breadcrumb-item'),
      tab: read('.app-shell__tab--active'),
      tabLabel: read('.app-shell__tab--active .app-shell__tab-label'),
      batchBar: read('.table-batch-action-bar'),
      batchButton: read('.table-batch-action-bar .arco-btn-text'),
      tableHeader: read('.app-table .arco-table-th'),
    };
  });

  expect(userShellStyles.header?.lineHeight).toBe('normal');
  expect(userShellStyles.breadcrumb?.overflow).toBe('visible');
  expect(userShellStyles.breadcrumb?.lineHeight).toBe('20px');
  expect(userShellStyles.breadcrumbItem?.overflow).toBe('visible');
  expect(userShellStyles.breadcrumbItem?.lineHeight).toBe('24px');
  expect(userShellStyles.breadcrumbItem?.top).toBeGreaterThanOrEqual(
    userShellStyles.breadcrumb!.top - 1,
  );
  expect(userShellStyles.breadcrumbItem?.bottom).toBeLessThanOrEqual(
    userShellStyles.breadcrumb!.bottom + 1,
  );
  expect(userShellStyles.tab?.boxShadow).toBe('none');
  expect(userShellStyles.tab?.lineHeight).toBe('20px');
  expect(userShellStyles.tab?.borderColor).toBe('rgba(0, 0, 0, 0)');
  expect(userShellStyles.tabLabel?.lineHeight).toBe('20px');
  expect(userShellStyles.batchBar?.backgroundColor).toBe('rgba(0, 0, 0, 0)');
  expect(userShellStyles.batchBar?.borderStyle).toBe('none');
  expect(userShellStyles.batchBar?.boxShadow).toBe('none');
  expect(userShellStyles.batchButton?.borderStyle).toBe('none');
  expect(userShellStyles.tableHeader?.backgroundColor).toBe('rgb(247, 248, 250)');

  await navigateInShell(page, '/system/setting');
  await expect(
    page.locator('.arco-tabs-header-nav-rounded .arco-tabs-header-title-active'),
  ).toBeVisible();

  const settingShellStyles = await page.evaluate(() => {
    const read = (selector: string) => {
      const element = document.querySelector<HTMLElement>(selector);
      if (!element) {
        return null;
      }
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return {
        selector,
        borderColor: style.borderColor,
        borderStyle: style.borderStyle,
        boxShadow: style.boxShadow,
        height: style.height,
        lineHeight: style.lineHeight,
        overflow: style.overflow,
        top: rect.top,
        bottom: rect.bottom,
      };
    };

    return {
      roundedTab: read('.arco-tabs-header-nav-rounded .arco-tabs-header-title-active'),
    };
  });

  expect(settingShellStyles.roundedTab?.boxShadow).toBe('none');
  expect(settingShellStyles.roundedTab?.borderStyle).toContain('solid');
});

test('system table pages keep unified table card spacing radius and neutral headers', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  for (const pageMeta of systemTablePages) {
    await navigateInShell(page, pageMeta.path);
    await expect(page.getByRole('heading', { name: pageMeta.title })).toBeVisible();
    await expect(page.locator('.app-table').first()).toBeVisible();

    const tableContract = await page.evaluate(() => {
      const table = document.querySelector<HTMLElement>('.app-table');
      const tableCard = table?.closest<HTMLElement>('.system-list__table-card');
      const body = table?.closest<HTMLElement>('.arco-card-body');
      const container = table?.querySelector<HTMLElement>('.arco-table-container');
      const firstHeader = table?.querySelector<HTMLElement>('.arco-table-th');
      const fixedColumn = table?.querySelector<HTMLElement>(
        '.arco-table-col-fixed-left-last, .arco-table-col-fixed-right-first',
      );
      const scrollContent = table?.querySelector<HTMLElement>('.arco-table-content-scroll');

      const read = (element?: HTMLElement | null) => {
        if (!element) {
          return null;
        }
        const style = window.getComputedStyle(element);
        return {
          backgroundColor: style.backgroundColor,
          borderTopLeftRadius: style.borderTopLeftRadius,
          borderTopRightRadius: style.borderTopRightRadius,
          boxShadow: style.boxShadow,
          paddingBottom: style.paddingBottom,
          paddingLeft: style.paddingLeft,
          paddingRight: style.paddingRight,
          paddingTop: style.paddingTop,
        };
      };

      return {
        hasSharedTableCard: Boolean(tableCard),
        body: read(body),
        container: read(container),
        firstHeader: read(firstHeader),
        fixedColumnShadow: fixedColumn
          ? window.getComputedStyle(fixedColumn, '::after').boxShadow
          : 'none',
        scrollBeforeShadow: scrollContent
          ? window.getComputedStyle(scrollContent, '::before').boxShadow
          : 'none',
        scrollAfterShadow: scrollContent
          ? window.getComputedStyle(scrollContent, '::after').boxShadow
          : 'none',
      };
    });

    expect(tableContract.hasSharedTableCard, pageMeta.path).toBe(true);
    expect(tableContract.body?.paddingTop, pageMeta.path).toBe('12px');
    expect(tableContract.body?.paddingRight, pageMeta.path).toBe('14px');
    expect(tableContract.body?.paddingBottom, pageMeta.path).toBe('6px');
    expect(tableContract.body?.paddingLeft, pageMeta.path).toBe('14px');
    expect(tableContract.container?.borderTopLeftRadius, pageMeta.path).toBe('0px');
    expect(tableContract.container?.borderTopRightRadius, pageMeta.path).toBe('0px');
    expect(tableContract.firstHeader?.backgroundColor, pageMeta.path).toBe('rgb(247, 248, 250)');
    expect(tableContract.fixedColumnShadow, pageMeta.path).toBe('none');
    expect(tableContract.scrollBeforeShadow, pageMeta.path).toBe('none');
    expect(tableContract.scrollAfterShadow, pageMeta.path).toBe('none');
  }
});

test('system filter panels and governance bars keep one formal rhythm', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  for (const path of filterPanelPages) {
    await navigateInShell(page, path);
    await expect(page.locator('.filter-panel').first()).toBeVisible();

    const filterContract = await page.evaluate(() => {
      const panel = document.querySelector<HTMLElement>('.filter-panel');
      const body = panel?.querySelector<HTMLElement>('.filter-panel__body');
      const firstItem = panel?.querySelector<HTMLElement>('.arco-form-item');
      const firstControl = panel?.querySelector<HTMLElement>(
        '.arco-input-inner-wrapper, .arco-input, .arco-select-view, .arco-tree-select-view, .arco-picker',
      );
      const actionButton = panel?.querySelector<HTMLElement>(
        '.filter-panel__action-item .arco-btn',
      );

      const readBox = (element?: HTMLElement | null) => {
        if (!element) {
          return null;
        }
        const style = window.getComputedStyle(element);
        const rect = element.getBoundingClientRect();
        return {
          height: Math.round(rect.height),
          marginBottom: style.marginBottom,
          paddingBottom: style.paddingBottom,
          paddingLeft: style.paddingLeft,
          paddingRight: style.paddingRight,
          paddingTop: style.paddingTop,
        };
      };

      return {
        body: readBox(body),
        firstItem: readBox(firstItem),
        firstControl: readBox(firstControl),
        actionButton: readBox(actionButton),
      };
    });

    expect(filterContract.body?.paddingTop, path).toBe('16px');
    expect(filterContract.body?.paddingRight, path).toBe('16px');
    expect(filterContract.body?.paddingBottom, path).toBe('4px');
    expect(filterContract.body?.paddingLeft, path).toBe('16px');
    expect(filterContract.firstItem?.marginBottom, path).toBe('12px');
    expect(filterContract.firstControl?.height, path).toBeGreaterThanOrEqual(34);
    expect(filterContract.actionButton?.height, path).toBeGreaterThanOrEqual(34);
  }

  for (const path of governanceBarPages) {
    await navigateInShell(page, path);
    await expect(page.locator('.table-batch-action-bar--governance')).toBeVisible();

    const governanceContract = await page.evaluate(() => {
      const bar = document.querySelector<HTMLElement>('.table-batch-action-bar--governance');
      const main = bar?.querySelector<HTMLElement>('.table-batch-action-bar__main');
      const meta = bar?.querySelector<HTMLElement>('.table-batch-action-bar__meta');
      const actions = bar?.querySelector<HTMLElement>('.table-batch-action-bar__actions');
      const select = bar?.querySelector<HTMLElement>('.table-batch-action-bar__select');
      const read = (element?: HTMLElement | null) => {
        if (!element) {
          return null;
        }
        const style = window.getComputedStyle(element);
        const rect = element.getBoundingClientRect();
        return {
          alignItems: style.alignItems,
          backgroundColor: style.backgroundColor,
          borderStyle: style.borderStyle,
          boxShadow: style.boxShadow,
          gap: style.gap,
          height: Math.round(rect.height),
          justifyContent: style.justifyContent,
          width: Math.round(rect.width),
        };
      };

      return {
        bar: read(bar),
        main: read(main),
        meta: read(meta),
        actions: read(actions),
        select: read(select),
      };
    });

    expect(governanceContract.bar?.backgroundColor, path).toBe('rgba(0, 0, 0, 0)');
    expect(governanceContract.bar?.borderStyle, path).toBe('none');
    expect(governanceContract.bar?.boxShadow, path).toBe('none');
    expect(governanceContract.main?.alignItems, path).toBe('center');
    expect(governanceContract.main?.justifyContent, path).toBe('space-between');
    expect(governanceContract.main?.height, path).toBeGreaterThanOrEqual(32);
    expect(governanceContract.meta?.alignItems, path).toBe('center');
    if (governanceContract.actions) {
      expect(governanceContract.actions.justifyContent, path).toBe('flex-end');
    }
    expect(governanceContract.select?.width, path).toBe(200);
  }
});

test('dict management keeps both tabs on the shared list rhythm', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);
  await navigateInShell(page, '/system/dict');
  await expect(page.getByRole('heading', { name: '字典管理' })).toBeVisible();

  const readDictTabContract = async () =>
    page.evaluate(() => {
      const tableCard = document.querySelector<HTMLElement>('.dict-page__table-card');
      const activePanel =
        document.querySelector<HTMLElement>('.arco-tabs-content-item-active') ||
        document.querySelector<HTMLElement>('.arco-tabs-pane-active') ||
        tableCard;
      const filterBody = activePanel?.querySelector<HTMLElement>('.filter-panel__body');
      const firstItem = activePanel?.querySelector<HTMLElement>('.filter-panel .arco-form-item');
      const firstControl = activePanel?.querySelector<HTMLElement>(
        '.filter-panel .arco-input-inner-wrapper, .filter-panel .arco-input, .filter-panel .arco-select-view, .filter-panel .arco-tree-select-view, .filter-panel .arco-picker',
      );
      const listActions = activePanel?.querySelector<HTMLElement>('.list-header-actions');
      const listActionsPrimary = activePanel?.querySelector<HTMLElement>(
        '.list-header-actions__primary',
      );
      const batchBar = activePanel?.querySelector<HTMLElement>('.table-batch-action-bar');
      const appTable = activePanel?.querySelector<HTMLElement>('.app-table');
      const tableContainer = activePanel?.querySelector<HTMLElement>(
        '.app-table .arco-table-container',
      );

      const read = (element?: HTMLElement | null) => {
        if (!element) {
          return null;
        }
        const style = window.getComputedStyle(element);
        const rect = element.getBoundingClientRect();
        return {
          alignItems: style.alignItems,
          borderTopLeftRadius: style.borderTopLeftRadius,
          display: style.display,
          gap: style.gap,
          height: Math.round(rect.height),
          justifyContent: style.justifyContent,
          marginBottom: style.marginBottom,
          paddingBottom: style.paddingBottom,
          paddingLeft: style.paddingLeft,
          paddingRight: style.paddingRight,
          paddingTop: style.paddingTop,
        };
      };

      return {
        hasSharedTableCard: Boolean(tableCard),
        hasSharedSystemTable: Boolean(appTable?.classList.contains('system-list__table')),
        hasVisibleTable: Boolean(appTable),
        filterBody: read(filterBody),
        firstItem: read(firstItem),
        firstControl: read(firstControl),
        listActions: read(listActions),
        listActionsPrimary: read(listActionsPrimary),
        batchBar: read(batchBar),
        tableContainer: read(tableContainer),
      };
    });

  await expect(page.locator('.dict-page__table-card .filter-panel').first()).toBeVisible();

  const typeTabContract = await readDictTabContract();
  expect(typeTabContract.hasSharedTableCard).toBe(true);
  expect(typeTabContract.hasSharedSystemTable).toBe(true);
  expect(typeTabContract.filterBody?.paddingTop).toBe('16px');
  expect(typeTabContract.filterBody?.paddingRight).toBe('16px');
  expect(typeTabContract.filterBody?.paddingBottom).toBe('4px');
  expect(typeTabContract.filterBody?.paddingLeft).toBe('16px');
  expect(typeTabContract.firstItem?.marginBottom).toBe('12px');
  expect(typeTabContract.firstControl?.height).toBeGreaterThanOrEqual(34);
  expect(typeTabContract.listActions?.justifyContent).toBe('space-between');
  expect(typeTabContract.listActionsPrimary?.justifyContent).toBe('flex-end');
  expect(typeTabContract.batchBar?.display).toBe('grid');
  expect(typeTabContract.tableContainer?.borderTopLeftRadius).toBe('0px');

  await page.getByRole('tab', { name: '字典项' }).click();
  await expect(page.locator('.dict-page__table-card .filter-panel').first()).toBeVisible();

  const itemTabContract = await readDictTabContract();
  expect(itemTabContract.hasSharedTableCard).toBe(true);
  expect(itemTabContract.hasSharedSystemTable || !itemTabContract.hasVisibleTable).toBe(true);
  expect(itemTabContract.filterBody?.paddingTop).toBe('16px');
  expect(itemTabContract.filterBody?.paddingRight).toBe('16px');
  expect(itemTabContract.filterBody?.paddingBottom).toBe('4px');
  expect(itemTabContract.filterBody?.paddingLeft).toBe('16px');
  expect(itemTabContract.firstItem?.marginBottom).toBe('12px');
  expect(itemTabContract.firstControl?.height).toBeGreaterThanOrEqual(34);
  expect(itemTabContract.listActions?.justifyContent).toBe('space-between');
  expect(itemTabContract.listActionsPrimary?.justifyContent).toBe('flex-end');
  if (itemTabContract.hasVisibleTable) {
    expect(itemTabContract.batchBar?.display).toBe('grid');
    expect(itemTabContract.tableContainer?.borderTopLeftRadius).toBe('0px');
  }
});
