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

const dialogEntryPages = [
  { path: '/system/user', triggerText: '新增' },
  { path: '/system/role', triggerText: '新增' },
  { path: '/system/menu', triggerText: '新增' },
  { path: '/system/post', triggerText: '新增' },
  { path: '/system/dict', triggerText: '新增' },
  { path: '/system/i18n', triggerText: '新增' },
] as const;

type ControlBox = {
  backgroundColor: string;
  borderTopStyle: string;
  borderTopWidth: string;
  boxShadow: string;
  height: number;
  outlineStyle: string;
  outlineWidth: string;
} | null;

type ControlContract = {
  kind?: string;
  label?: string;
  wrapper: ControlBox;
  inner: ControlBox;
  nestedInput: ControlBox;
};

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

async function readVisibleControlContracts(
  page: import('@playwright/test').Page,
  rootSelector: string,
) {
  const rootLocator = page.locator(rootSelector).first();
  await expect(rootLocator).toBeVisible();
  return rootLocator.evaluate((root) => {
    const outerSelector = [
      '.arco-input-password',
      '.arco-input-number',
      '.arco-select-view',
      '.arco-tree-select-view',
      '.arco-picker',
      '.arco-textarea-wrapper',
      'textarea.arco-textarea',
      '.arco-input-inner-wrapper',
      'input.arco-input',
    ].join(',');
    const read = (element?: HTMLElement | null) => {
      if (!element) {
        return null;
      }
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return {
        backgroundColor: style.backgroundColor,
        borderTopStyle: style.borderTopStyle,
        borderTopWidth: style.borderTopWidth,
        boxShadow: style.boxShadow,
        height: Math.round(rect.height),
        outlineStyle: style.outlineStyle,
        outlineWidth: style.outlineWidth,
      };
    };
    const isOuterControl = (element: HTMLElement) => {
      if (
        element.matches('input.arco-input, textarea.arco-textarea') &&
        element.closest(
          '.arco-input-inner-wrapper, .arco-input-password, .arco-input-number, .arco-textarea-wrapper',
        )
      ) {
        return false;
      }
      const nestedOuter = element.closest('.arco-input-password, .arco-input-number');
      if (element.classList.contains('arco-input-inner-wrapper') && nestedOuter) {
        return nestedOuter === element;
      }
      return true;
    };
    const kindOf = (element: HTMLElement) => {
      if (element.classList.contains('arco-input-password')) {
        return 'password';
      }
      if (element.classList.contains('arco-input-number')) {
        return 'number';
      }
      if (element.classList.contains('arco-select-view')) {
        return 'select';
      }
      if (element.classList.contains('arco-tree-select-view')) {
        return 'treeSelect';
      }
      if (element.classList.contains('arco-picker')) {
        return 'picker';
      }
      if (element.classList.contains('arco-textarea-wrapper')) {
        return 'textarea';
      }
      if (element.matches('textarea.arco-textarea')) {
        return 'textarea';
      }
      return 'text';
    };
    return Array.from(root.querySelectorAll<HTMLElement>(outerSelector))
      .filter((element) => {
        const rect = element.getBoundingClientRect();
        return rect.width > 0 && rect.height > 0 && isOuterControl(element);
      })
      .map((wrapper, index) => {
        const kind = kindOf(wrapper);
        let inner: HTMLElement | null | undefined = null;
        if (!wrapper.matches('input.arco-input, textarea.arco-textarea')) {
          inner =
            kind === 'textarea'
              ? wrapper.querySelector<HTMLElement>('.arco-textarea')
              : wrapper.querySelector<HTMLElement>('.arco-input-inner-wrapper, .arco-input');
        }
        const nestedInput =
          inner?.classList.contains('arco-input-inner-wrapper') ||
          inner?.classList.contains('arco-input-password')
            ? inner.querySelector<HTMLElement>('.arco-input')
            : inner;
        return {
          kind,
          label: `${kind}#${index + 1}`,
          wrapper: read(wrapper),
          inner: read(inner),
          nestedInput: read(nestedInput),
        };
      });
  });
}

async function readFocusedTextInputContract(
  page: import('@playwright/test').Page,
  rootSelector: string,
) {
  const rootLocator = page.locator(rootSelector).first();
  await expect(rootLocator).toBeVisible();
  return rootLocator.evaluate((root) => {
    const wrappedInput = Array.from(
      root.querySelectorAll<HTMLElement>('.arco-input-inner-wrapper'),
    ).find((element) => {
      const input = element.querySelector<HTMLElement>('.arco-input');
      const rect = element.getBoundingClientRect();
      return Boolean(
        input &&
          rect.width > 0 &&
          rect.height > 0 &&
          !element.closest('.arco-input-password, .arco-input-number'),
      );
    });
    const standaloneInput = Array.from(
      root.querySelectorAll<HTMLElement>('input.arco-input'),
    ).find((element) => {
      const rect = element.getBoundingClientRect();
      return Boolean(
        rect.width > 0 &&
          rect.height > 0 &&
          !element.closest('.arco-input-inner-wrapper, .arco-input-password, .arco-input-number'),
      );
    });
    const wrapper = wrappedInput ?? standaloneInput;
    const input = wrapper?.matches('input.arco-input')
      ? wrapper
      : wrapper?.querySelector<HTMLElement>('.arco-input');
    input?.focus();
    const read = (element?: HTMLElement | null) => {
      if (!element) {
        return null;
      }
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return {
        backgroundColor: style.backgroundColor,
        borderTopStyle: style.borderTopStyle,
        borderTopWidth: style.borderTopWidth,
        boxShadow: style.boxShadow,
        height: Math.round(rect.height),
        outlineStyle: style.outlineStyle,
        outlineWidth: style.outlineWidth,
      };
    };
    if (wrapper?.matches('input.arco-input')) {
      return {
        wrapper: read(wrapper),
        inner: null,
        nestedInput: null,
      };
    }
    return {
      wrapper: read(wrapper),
      inner: read(input),
      nestedInput: read(input),
    };
  });
}

function expectSingleLayerControl(
  control: ControlContract,
  label: string,
  options: { requireWhiteBackground?: boolean } = {},
) {
  const { requireWhiteBackground = true } = options;
  expect(control.wrapper, label).not.toBeNull();
  if (!control.wrapper) {
    return;
  }
  expect(control.wrapper.borderTopWidth, label).toBe('1px');
  expect(control.wrapper.borderTopStyle, label).toBe('solid');
  if (requireWhiteBackground) {
    expect(control.wrapper.backgroundColor, label).toBe('rgb(255, 255, 255)');
  }
  if (control.inner && control.inner !== control.wrapper) {
    expect(control.inner.borderTopWidth, `${label} inner border`).toBe('0px');
    expect(control.inner.backgroundColor, `${label} inner background`).toBe('rgba(0, 0, 0, 0)');
    expect(control.inner.boxShadow, `${label} inner shadow`).toBe('none');
    expect(control.inner.outlineStyle, `${label} inner outline`).toBe('none');
    expect(control.inner.outlineWidth, `${label} inner outline width`).toBe('0px');
  }
  if (control.nestedInput && control.nestedInput !== control.wrapper) {
    expect(control.nestedInput.borderTopWidth, `${label} nested border`).toBe('0px');
    expect(control.nestedInput.backgroundColor, `${label} nested background`).toBe(
      'rgba(0, 0, 0, 0)',
    );
    expect(control.nestedInput.boxShadow, `${label} nested shadow`).toBe('none');
    expect(control.nestedInput.outlineStyle, `${label} nested outline`).toBe('none');
    expect(control.nestedInput.outlineWidth, `${label} nested outline width`).toBe('0px');
  }
}

async function readRootCssVariables(
  page: import('@playwright/test').Page,
  variableNames: string[],
) {
  return page.evaluate((names) => {
    const style = window.getComputedStyle(document.documentElement);
    return Object.fromEntries(names.map((name) => [name, style.getPropertyValue(name).trim()]));
  }, variableNames);
}

function expandPaddingValues(value: string) {
  const parts = value.split(/\s+/).filter(Boolean);
  const [top = '0px', right = top, bottom = top, left = right] = parts;
  const normalize = (part: string) => (part === '0' ? '0px' : part);
  return [top, right, bottom, left].map(normalize);
}

test('platform shell breadcrumb and function bars do not clip text or use inset mixed borders', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);
  await navigateInShell(page, '/system/user');
  await expect(page.locator('.table-batch-action-bar')).toBeVisible();
  await expect(page.locator('.app-table .arco-table-th').first()).toBeVisible();

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
  expect(userShellStyles.batchBar?.borderStyle).not.toContain('inset');
  expect(userShellStyles.batchBar?.boxShadow).toBe('none');
  expect(userShellStyles.batchButton?.borderStyle).toBe('none');
  expect(userShellStyles.tableHeader?.backgroundColor).toBe('rgb(247, 248, 250)');

  await navigateInShell(page, '/system/setting');
  const roundedTabLocator = page.locator(
    '.arco-tabs-header-nav-rounded .arco-tabs-header-title-active',
  );
  const roundedTabCount = await roundedTabLocator.count();

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

  if (roundedTabCount > 0) {
    await expect(roundedTabLocator.first()).toBeVisible();
    expect(settingShellStyles.roundedTab?.boxShadow).toBe('none');
    expect(settingShellStyles.roundedTab?.borderStyle).toContain('solid');
  }
});

test('system table pages keep unified table card spacing radius and neutral headers', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  for (const pageMeta of systemTablePages) {
    await navigateInShell(page, pageMeta.path);
    await expect(page.locator('.system-list__table-card').first()).toBeVisible();
    const cssVariables = await readRootCssVariables(page, ['--shell-table-card-padding']);
    const [tablePaddingTop, tablePaddingRight, tablePaddingBottom, tablePaddingLeft] =
      expandPaddingValues(cssVariables['--shell-table-card-padding']);

    const tableContract = await page.evaluate(() => {
      const tableCard = document.querySelector<HTMLElement>('.system-list__table-card');
      const body = tableCard?.querySelector<HTMLElement>('.arco-card-body');
      const table = tableCard?.querySelector<HTMLElement>('.app-table');
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
        hasDataTable: Boolean(table),
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
    expect(tableContract.body?.paddingTop, pageMeta.path).toBe(tablePaddingTop);
    expect(tableContract.body?.paddingRight, pageMeta.path).toBe(tablePaddingRight);
    expect(tableContract.body?.paddingBottom, pageMeta.path).toBe(tablePaddingBottom);
    expect(tableContract.body?.paddingLeft, pageMeta.path).toBe(
      tablePaddingLeft ?? tablePaddingRight,
    );
    if (tableContract.hasDataTable) {
      expect(tableContract.container?.borderTopLeftRadius, pageMeta.path).toBe('0px');
      expect(tableContract.container?.borderTopRightRadius, pageMeta.path).toBe('0px');
      expect(tableContract.firstHeader?.backgroundColor, pageMeta.path).toBe('rgb(247, 248, 250)');
      expect(tableContract.fixedColumnShadow, pageMeta.path).toBe('none');
      expect(tableContract.scrollBeforeShadow, pageMeta.path).toBe('none');
      expect(tableContract.scrollAfterShadow, pageMeta.path).toBe('none');
    }
  }
});

test('system filter panels and governance bars keep one formal rhythm', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  for (const path of filterPanelPages) {
    await navigateInShell(page, path);
    await expect(page.locator('.filter-panel').first()).toBeVisible();
    const cssVariables = await readRootCssVariables(page, [
      '--shell-filter-body-padding',
      '--shell-filter-control-min-height',
      '--shell-filter-form-item-margin-bottom',
    ]);
    const [filterPaddingTop, filterPaddingRight, filterPaddingBottom, filterPaddingLeft] =
      expandPaddingValues(cssVariables['--shell-filter-body-padding']);
    const filterControlMinHeight = Number.parseInt(
      cssVariables['--shell-filter-control-min-height'],
      10,
    );

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

    expect(filterContract.body?.paddingTop, path).toBe(filterPaddingTop);
    expect(filterContract.body?.paddingRight, path).toBe(filterPaddingRight);
    expect(filterContract.body?.paddingBottom, path).toBe(filterPaddingBottom);
    expect(filterContract.body?.paddingLeft, path).toBe(filterPaddingLeft ?? filterPaddingRight);
    expect(filterContract.firstItem?.marginBottom, path).toBe(
      cssVariables['--shell-filter-form-item-margin-bottom'],
    );
    expect(filterContract.firstControl?.height, path).toBeGreaterThanOrEqual(
      filterControlMinHeight,
    );
    expect(filterContract.actionButton?.height, path).toBeGreaterThanOrEqual(
      filterControlMinHeight,
    );
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
  const cssVariables = await readRootCssVariables(page, [
    '--shell-filter-body-padding',
    '--shell-filter-control-min-height',
    '--shell-filter-form-item-margin-bottom',
  ]);
  const [filterPaddingTop, filterPaddingRight, filterPaddingBottom, filterPaddingLeft] =
    expandPaddingValues(cssVariables['--shell-filter-body-padding']);
  const filterControlMinHeight = Number.parseInt(
    cssVariables['--shell-filter-control-min-height'],
    10,
  );

  const typeTabContract = await readDictTabContract();
  expect(typeTabContract.hasSharedTableCard).toBe(true);
  expect(typeTabContract.hasSharedSystemTable).toBe(true);
  expect(typeTabContract.filterBody?.paddingTop).toBe(filterPaddingTop);
  expect(typeTabContract.filterBody?.paddingRight).toBe(filterPaddingRight);
  expect(typeTabContract.filterBody?.paddingBottom).toBe(filterPaddingBottom);
  expect(typeTabContract.filterBody?.paddingLeft).toBe(filterPaddingLeft ?? filterPaddingRight);
  expect(typeTabContract.firstItem?.marginBottom).toBe(
    cssVariables['--shell-filter-form-item-margin-bottom'],
  );
  expect(typeTabContract.firstControl?.height).toBeGreaterThanOrEqual(filterControlMinHeight);
  expect(typeTabContract.listActions?.justifyContent).toBe('space-between');
  expect(typeTabContract.listActionsPrimary?.justifyContent).toBe('flex-end');
  expect(typeTabContract.batchBar?.display).toBe('grid');
  expect(typeTabContract.tableContainer?.borderTopLeftRadius).toBe('0px');

  await page.getByRole('tab', { name: '字典项' }).click();
  await expect(page.locator('.dict-page__table-card .filter-panel').first()).toBeVisible();

  const itemTabContract = await readDictTabContract();
  expect(itemTabContract.hasSharedTableCard).toBe(true);
  expect(itemTabContract.hasSharedSystemTable || !itemTabContract.hasVisibleTable).toBe(true);
  expect(itemTabContract.filterBody?.paddingTop).toBe(filterPaddingTop);
  expect(itemTabContract.filterBody?.paddingRight).toBe(filterPaddingRight);
  expect(itemTabContract.filterBody?.paddingBottom).toBe(filterPaddingBottom);
  expect(itemTabContract.filterBody?.paddingLeft).toBe(filterPaddingLeft ?? filterPaddingRight);
  expect(itemTabContract.firstItem?.marginBottom).toBe(
    cssVariables['--shell-filter-form-item-margin-bottom'],
  );
  expect(itemTabContract.firstControl?.height).toBeGreaterThanOrEqual(filterControlMinHeight);
  expect(itemTabContract.listActions?.justifyContent).toBe('space-between');
  expect(itemTabContract.listActionsPrimary?.justifyContent).toBe('flex-end');
  if (itemTabContract.hasVisibleTable) {
    expect(itemTabContract.batchBar?.display).toBe('grid');
    expect(itemTabContract.tableContainer?.borderTopLeftRadius).toBe('0px');
  }
});

test('user management keeps function actions before lightweight status and batch actions', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);
  await navigateInShell(page, '/system/user');

  await expect(page.locator('.page-container > .page-header')).toHaveCount(0);
  await expect(page.locator('.governance-summary-bar')).toHaveCount(0);
  await expect(page.locator('.system-user-list__hero')).toHaveCount(0);
  await expect(page.locator('.system-user-list__function-bar')).toBeVisible();
  await expect(page.locator('.system-user-list__function-bar').getByText('新增')).toBeVisible();
  await expect(page.locator('.system-user-list__function-bar').getByText('导入')).toBeVisible();
  await expect(page.locator('.system-user-list__function-bar').getByText('导出')).toBeVisible();
  await expect(page.locator('.system-user-list__status-strip')).toBeVisible();
  await expect(page.locator('.system-user-list__status-item')).toHaveCount(5);
  await expect(page.locator('.table-batch-action-bar__prefix-actions')).toBeVisible();
  await expect(page.locator('.table-batch-action-bar__prefix-actions').getByText('新增')).toHaveCount(0);

  const actionContract = await page.evaluate(() => {
    const functionBar = document.querySelector<HTMLElement>('.system-user-list__function-bar');
    const statusStrip = document.querySelector<HTMLElement>('.system-user-list__status-strip');
    const prefixActions = document.querySelector<HTMLElement>(
      '.table-batch-action-bar__prefix-actions',
    );
    const batchBar = document.querySelector<HTMLElement>('.table-batch-action-bar');
    const readTop = (element: HTMLElement | null) =>
      element ? Math.round(element.getBoundingClientRect().top) : null;
    return {
      functionBarTop: readTop(functionBar),
      statusStripTop: readTop(statusStrip),
      functionBarHasCreate: Boolean(
        functionBar &&
          Array.from(functionBar.querySelectorAll('button')).some((button) =>
            /新增/.test(button.textContent || ''),
          ),
      ),
      statusInsideBatch: Boolean(statusStrip && batchBar?.contains(statusStrip)),
      prefixInsideBatch: Boolean(prefixActions && batchBar?.contains(prefixActions)),
      prefixGap: prefixActions ? window.getComputedStyle(prefixActions).gap : null,
    };
  });

  expect(actionContract.functionBarHasCreate).toBe(true);
  expect(actionContract.statusInsideBatch).toBe(true);
  expect(actionContract.prefixInsideBatch).toBe(true);
  expect(actionContract.prefixGap).toBe('6px 8px');
  expect(actionContract.functionBarTop).not.toBeNull();
  expect(actionContract.statusStripTop).not.toBeNull();
  expect(actionContract.functionBarTop!).toBeLessThan(actionContract.statusStripTop!);

  await page.locator('.system-user-list__function-bar').getByText('新增').click();
  await expect(page.locator('.app-dialog')).toBeVisible();

  const controlContracts = await readVisibleControlContracts(page, '.app-dialog');
  expect(controlContracts.length).toBeGreaterThanOrEqual(5);
  for (const controlContract of controlContracts) {
    expectSingleLayerControl(
      controlContract,
      `/system/user ${controlContract.label ?? controlContract.kind}`,
    );
  }

  const focusedTextContract = await readFocusedTextInputContract(page, '.app-dialog');
  expectSingleLayerControl(focusedTextContract, '/system/user focused text input');
  expect(focusedTextContract.wrapper?.boxShadow, '/system/user focused wrapper shadow').not.toBe(
    'none',
  );
  if (focusedTextContract.inner) {
    expect(focusedTextContract.inner.boxShadow, '/system/user focused inner shadow').toBe('none');
  }
});

test('core system dialogs share single-layer input border contracts', async ({ page }) => {
  test.setTimeout(60000);
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  for (const entry of dialogEntryPages) {
    await navigateInShell(page, entry.path);
    const createButton = page
      .locator(
        '.system-user-list__function-bar, .table-batch-action-bar__prefix-actions, .dict-page__actions, .list-header-actions__primary, .system-list__work-actions',
      )
      .getByRole('button', { name: entry.triggerText })
      .first();
    await expect(createButton, entry.path).toBeVisible();
    await expect(createButton, entry.path).toBeEnabled();
    await createButton.click();
    await expect(page.locator('.app-dialog')).toBeVisible();

    const visibleControlContracts = await readVisibleControlContracts(page, '.app-dialog');
    expect(visibleControlContracts.length, `${entry.path} visible controls`).toBeGreaterThan(0);
    for (const controlContract of visibleControlContracts) {
      expectSingleLayerControl(
        controlContract,
        `${entry.path} ${controlContract.label ?? controlContract.kind}`,
      );
    }

    await page.locator('.app-dialog .arco-modal-close-icon').click();
    await expect(page.locator('.app-dialog')).toHaveCount(0);
  }
});

test('form controls use a single visible focus ring across modal and page surfaces', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  await navigateInShell(page, '/system/user');
  const filterContracts = await readVisibleControlContracts(page, '.filter-panel');
  expect(filterContracts.length, '/system/user filter controls').toBeGreaterThan(0);
  for (const controlContract of filterContracts) {
    expectSingleLayerControl(
      controlContract,
      `/system/user filter ${controlContract.label ?? controlContract.kind}`,
      { requireWhiteBackground: false },
    );
  }

  const focusedFilterContract = await readFocusedTextInputContract(page, '.filter-panel');
  expectSingleLayerControl(focusedFilterContract, '/system/user focused filter text input', {
    requireWhiteBackground: false,
  });
  expect(focusedFilterContract.wrapper?.boxShadow, '/system/user focused filter wrapper shadow').not.toBe(
    'none',
  );
  if (focusedFilterContract.inner) {
    expect(focusedFilterContract.inner.boxShadow, '/system/user focused filter inner shadow').toBe(
      'none',
    );
  }

  await page
    .locator('.system-user-list__function-bar, .list-header-actions__primary, .table-batch-action-bar__prefix-actions')
    .getByRole('button', { name: '新增' })
    .first()
    .click();
  await expect(page.locator('.app-dialog')).toBeVisible();
  const focusedDialogContract = await readFocusedTextInputContract(page, '.app-dialog');
  expectSingleLayerControl(focusedDialogContract, '/system/user focused dialog text input');
  expect(focusedDialogContract.wrapper?.boxShadow, '/system/user focused dialog wrapper shadow').not.toBe(
    'none',
  );
  if (focusedDialogContract.inner) {
    expect(focusedDialogContract.inner.boxShadow, '/system/user focused dialog inner shadow').toBe(
      'none',
    );
  }
});
