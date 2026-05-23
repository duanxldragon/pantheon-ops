import { expect, test } from '@playwright/test';
import { signInAsAdmin } from '../helpers/auth';

const systemTablePages = [
  { path: '/system/user' },
  { path: '/system/role' },
  { path: '/system/permission' },
  { path: '/system/menu' },
  { path: '/system/dept' },
  { path: '/system/post' },
  { path: '/system/dict' },
  { path: '/system/i18n' },
  { path: '/system/modules' },
  { path: '/system/session' },
  { path: '/system/login-log' },
  { path: '/system/operation-log' },
] as const;

const filterPanelPages = [
  '/system/user',
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
  { path: '/system/dept', triggerText: '新增' },
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

type DialogControlContract = {
  kind?: string;
  label?: string;
  wrapper: ControlBox;
  inner: ControlBox;
  nestedInput: ControlBox;
};

async function installEmeraldThemePreference(page: import('@playwright/test').Page) {
  await page.addInitScript(() => {
    localStorage.setItem('pantheon_theme', 'emerald');
  });
}

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

async function openSystemDialogPage(page: import('@playwright/test').Page, path: string) {
  await page.goto(path, { waitUntil: 'domcontentloaded' });
  await expect(page.locator('.app-shell__content')).toBeVisible();
  await expect(page.getByRole('button', { name: '新增', exact: true }).first()).toBeVisible({
    timeout: 20000,
  });
}

async function readDialogContract(
  page: import('@playwright/test').Page,
  options: { preserveFocus?: boolean } = {},
) {
  if (!options.preserveFocus) {
    await page.evaluate(() => {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    });
  }
  const dialogLocator = page.locator('.app-dialog').first();
  await expect(dialogLocator).toBeVisible();
  await page.waitForTimeout(250);
  return dialogLocator.evaluate((dialog) => {
    const header = dialog.querySelector<HTMLElement>('.arco-modal-header');
    const content = dialog.querySelector<HTMLElement>('.arco-modal-content');
    const footer = dialog.querySelector<HTMLElement>('.arco-modal-footer');
    const submitBar = dialog.querySelector<HTMLElement>('.submit-bar');
    const formSectionTitle = dialog.querySelector<HTMLElement>('.form-section__title');
    const wrapper = Array.from(
      dialog.querySelectorAll<HTMLElement>(
        '.arco-input-password, .arco-input-number, input.arco-input, .arco-select-view, .arco-tree-select-view, .arco-picker, .arco-textarea-wrapper, .arco-input-inner-wrapper',
      ),
    ).find(
      (element) =>
        !(
          element.classList.contains('arco-input-inner-wrapper') &&
          element.closest('.arco-input-password, .arco-input-number')
        ),
    );
    const input = wrapper?.classList.contains('arco-input')
      ? null
      : wrapper?.querySelector<HTMLElement>('.arco-input');
    const read = (element?: HTMLElement | null) => {
      if (!element) {
        return null;
      }
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return {
        alignItems: style.alignItems,
        backgroundColor: style.backgroundColor,
        borderTopStyle: style.borderTopStyle,
        borderTopWidth: style.borderTopWidth,
        bottom: Math.round(rect.bottom),
        boxShadow: style.boxShadow,
        display: style.display,
        fontSize: style.fontSize,
        fontWeight: style.fontWeight,
        height: Math.round(rect.height),
        justifyContent: style.justifyContent,
        left: Math.round(rect.left),
        maxHeight: style.maxHeight,
        overflowX: style.overflowX,
        overflowY: style.overflowY,
        outlineStyle: style.outlineStyle,
        outlineWidth: style.outlineWidth,
        paddingBottom: style.paddingBottom,
        paddingLeft: style.paddingLeft,
        paddingRight: style.paddingRight,
        paddingTop: style.paddingTop,
        right: Math.round(rect.right),
        top: Math.round(rect.top),
        width: Math.round(rect.width),
      };
    };

    return {
      viewportWidth: window.innerWidth,
      viewportHeight: window.innerHeight,
      dialog: read(dialog),
      header: read(header),
      content: read(content),
      footer: read(footer),
      submitBar: read(submitBar),
      formSectionTitle: read(formSectionTitle),
      wrapper: read(wrapper),
      input: read(input),
    };
  });
}

async function readDialogControlContracts(page: import('@playwright/test').Page) {
  const dialogLocator = page.locator('.app-dialog').first();
  await expect(dialogLocator).toBeVisible();
  return dialogLocator.evaluate((dialog) => {
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
    const controls = {
      text: dialog.querySelector<HTMLElement>(
        '.arco-form-item:not(:has(.arco-input-password)):not(:has(.arco-input-number)):not(:has(.arco-textarea-wrapper)):not(:has(.arco-select-view)) .arco-input-inner-wrapper',
      ),
      password: dialog.querySelector<HTMLElement>('.arco-input-password'),
      number: dialog.querySelector<HTMLElement>('.arco-input-number'),
      textarea: dialog.querySelector<HTMLElement>('.arco-textarea-wrapper'),
      select: dialog.querySelector<HTMLElement>('.arco-select-view'),
    };
    return Object.fromEntries(
      Object.entries(controls).map(([key, wrapper]) => {
        const inner =
          key === 'textarea'
            ? wrapper?.querySelector<HTMLElement>('.arco-textarea')
            : wrapper?.querySelector<HTMLElement>('.arco-input-inner-wrapper, .arco-input');
        const nestedInput =
          inner?.classList.contains('arco-input-inner-wrapper') ||
          inner?.classList.contains('arco-input-password')
            ? inner.querySelector<HTMLElement>('.arco-input')
            : inner;
        return [
          key,
          {
            wrapper: read(wrapper),
            inner: read(inner),
            nestedInput: read(nestedInput),
          },
        ];
      }),
    );
  });
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

function preferMeasuredBoxValue(...values: Array<string | null | undefined>) {
  return values.find((value) => Boolean(value && value.trim().length > 0));
}

function expectSingleLayerDialogControl(
  control: DialogControlContract,
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

  await navigateInShell(page, '/system/setting/basic');
  await expect(page.locator('.setting-page__group-nav-item--active')).toBeVisible();
  await expect(page.locator('.arco-tabs-header-nav-rounded')).toHaveCount(0);

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
      groupNavItem: read('.setting-page__group-nav-item--active'),
    };
  });

  expect(settingShellStyles.groupNavItem?.borderStyle).toContain('solid');
});

test('setting workspace keeps summary and group navigation on the shared page rhythm', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);
  await navigateInShell(page, '/system/setting');
  await expect(page.locator('.setting-overview-page > .page-header')).toHaveCount(0);
  await expect(page.locator('.governance-summary-bar')).toBeVisible();
  await expect(page.locator('.setting-page__group-nav-item').first()).toBeVisible();
  await expect(page.locator('.setting-page__overview-head')).toHaveCount(0);
  await expect(page.locator('.system-page-hero')).toHaveCount(0);

  const overviewContract = await page.evaluate(() => {
    const summary = document.querySelector<HTMLElement>('.governance-summary-bar');
    const groupGrid = document.querySelector<HTMLElement>('.setting-page__group-nav-grid');
    const groupItem = document.querySelector<HTMLElement>('.setting-page__group-nav-item');
    const groupTitle = document.querySelector<HTMLElement>('.setting-page__group-nav-title');
    const runtimeStrip = document.querySelector<HTMLElement>('.setting-page__runtime-strip');
    const overviewCard = document.querySelector<HTMLElement>('.setting-overview-page__group-card');

    const read = (element?: HTMLElement | null) => {
      if (!element) {
        return null;
      }
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return {
        display: style.display,
        fontSize: style.fontSize,
        gap: style.gap,
        height: Math.round(rect.height),
        paddingTop: style.paddingTop,
        paddingRight: style.paddingRight,
        paddingBottom: style.paddingBottom,
        paddingLeft: style.paddingLeft,
        top: Math.round(rect.top),
      };
    };

    return {
      summary: read(summary),
      groupGrid: read(groupGrid),
      groupItem: read(groupItem),
      groupTitle: read(groupTitle),
      runtimeStrip: read(runtimeStrip),
      overviewCard: read(overviewCard),
    };
  });

  expect(overviewContract.summary?.display).toBe('grid');
  expect(overviewContract.summary?.paddingTop).toBe('10px');
  expect(overviewContract.summary?.paddingBottom).toBe('10px');
  expect(overviewContract.groupGrid?.display).toBe('grid');
  expect(overviewContract.groupGrid?.gap).toBe('8px');
  expect(overviewContract.groupItem?.height).toBeLessThanOrEqual(120);
  expect(overviewContract.groupTitle?.fontSize).toBe('13px');
  expect(overviewContract.runtimeStrip?.display).toBe('flex');
  expect(overviewContract.overviewCard?.top).toBeGreaterThan(overviewContract.summary!.top);
});

test('setting group routes use distinct opened-tab labels', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  await navigateInShell(page, '/system/setting/basic');
  await expect(page.locator('.setting-group-page > .page-header')).toHaveCount(0);
  await expect(page.getByRole('tab', { name: '基础信息' })).toBeVisible();
  await expect(page.locator('.app-shell__header').getByRole('listitem').last()).toContainText(
    '基础信息',
  );

  await navigateInShell(page, '/system/setting/platform');
  await expect(page.locator('.setting-group-page > .page-header')).toHaveCount(0);
  await expect(page.getByRole('tab', { name: '平台能力' })).toBeVisible();
  await expect(page.locator('.app-shell__header').getByRole('listitem').last()).toContainText(
    '平台能力',
  );

  const settingTabLabels = await page
    .locator('.app-shell__tabs [role="tab"] .app-shell__tab-label')
    .allTextContents();
  expect(settingTabLabels).toEqual(expect.arrayContaining(['基础信息', '平台能力']));
  expect(settingTabLabels.filter((label) => label === '系统设置')).toHaveLength(0);
});

test('high-sensitivity config pages keep a single summary shell without hero walls', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  await navigateInShell(page, '/system/modules');
  await expect(page.locator('.page-container > .page-header')).toHaveCount(0);
  await expect(page.locator('.module-manager-page .governance-summary-bar')).toBeVisible();
  await expect(
    page.locator('.system-list__work-actions .module-manager-page__header-actions .arco-btn-primary'),
  ).toBeVisible();
  await expect(page.locator('.system-page-hero')).toHaveCount(0);
  await expect(page.locator('.system-list__hero')).toHaveCount(0);
  await expect(page.locator('.module-manager-page__intro')).toHaveCount(0);
  await expect(page.locator('.module-manager-page__stats')).toHaveCount(0);
  const moduleCssVariables = await readRootCssVariables(page, ['--shell-table-card-padding']);
  const [modulePaddingTop, modulePaddingRight, modulePaddingBottom, modulePaddingLeft] =
    expandPaddingValues(moduleCssVariables['--shell-table-card-padding']);

  const moduleShellContract = await page.evaluate(() => {
    const alert = document.querySelector<HTMLElement>('.module-manager-page__card .arco-alert');
    const tableCard = document.querySelector<HTMLElement>('.module-manager-page__card');
    const tableBody = tableCard?.querySelector<HTMLElement>(':scope > .arco-card-body');
    const table = document.querySelector<HTMLElement>(
      '.module-manager-page .app-table, .module-manager-page .arco-table',
    );
    const empty = document.querySelector<HTMLElement>('.module-manager-page .page-empty');
    const governanceBar = document.querySelector<HTMLElement>('.module-manager-page .governance-summary-bar');
    const metrics = Array.from(
      document.querySelectorAll<HTMLElement>('.module-manager-page .governance-summary-bar__metric'),
    );
    const read = (element?: HTMLElement | null) => {
      if (!element) {
        return null;
      }
      const style = window.getComputedStyle(element);
      return {
        backgroundColor: style.backgroundColor,
        borderTopLeftRadius: style.borderTopLeftRadius,
        borderTopRightRadius: style.borderTopRightRadius,
        paddingBottom: style.paddingBottom,
        paddingLeft: style.paddingLeft,
        paddingRight: style.paddingRight,
        paddingTop: style.paddingTop,
      };
    };
    return {
      alertCount: document.querySelectorAll('.module-manager-page__card .arco-alert').length,
      alertDisplay: alert ? window.getComputedStyle(alert).display : null,
      hasDataTable: Boolean(table),
      hasEmptyState: Boolean(empty),
      governanceColumns: governanceBar
        ? window.getComputedStyle(governanceBar).gridTemplateColumns
            .split(' ')
            .filter(Boolean).length
        : 0,
      metricCount: metrics.length,
      metricTopOffsets: metrics.map((item) => Math.round(item.getBoundingClientRect().top)),
      tableBody: read(tableBody),
      tableContainer: read(
        table?.querySelector<HTMLElement>('.arco-table-container') ?? empty,
      ),
      firstHeaderBackground: table?.querySelector<HTMLElement>('.arco-table-th')
        ? window.getComputedStyle(table.querySelector<HTMLElement>('.arco-table-th')!).backgroundColor
        : null,
    };
  });

  expect(moduleShellContract.alertCount).toBe(1);
  expect(moduleShellContract.alertDisplay).toBeTruthy();
  expect(moduleShellContract.hasDataTable || moduleShellContract.hasEmptyState).toBeTruthy();
  expect(moduleShellContract.governanceColumns).toBe(2);
  expect(moduleShellContract.metricCount).toBe(5);
  expect(new Set(moduleShellContract.metricTopOffsets).size).toBe(1);
  expect(moduleShellContract.tableBody?.paddingTop).toBe(modulePaddingTop);
  expect(moduleShellContract.tableBody?.paddingRight).toBe(modulePaddingRight);
  expect(moduleShellContract.tableBody?.paddingBottom).toBe(modulePaddingBottom);
  expect(moduleShellContract.tableBody?.paddingLeft).toBe(modulePaddingLeft ?? modulePaddingRight);
  if (moduleShellContract.hasDataTable) {
    expect(moduleShellContract.tableContainer?.borderTopLeftRadius).toBe('0px');
    expect(moduleShellContract.tableContainer?.borderTopRightRadius).toBe('0px');
    expect(moduleShellContract.firstHeaderBackground).toBe('rgb(247, 248, 250)');
  }

  await navigateInShell(page, '/system/generator');
  await expect(page.locator('.page-container > .page-header')).toHaveCount(0);
  await expect(page.locator('.generator-wizard__steps')).toBeVisible();
  await expect(page.locator('.system-list__work-actions .arco-btn')).toBeVisible();
  await expect(page.locator('.system-page-hero')).toHaveCount(0);
  await expect(page.locator('.system-list__hero')).toHaveCount(0);

  const generatorShellContract = await page.evaluate(() => {
    const steps = document.querySelector<HTMLElement>('.generator-wizard__steps');
    return {
      topAlertCount: document.querySelectorAll(
        '.generator-wizard-card > .arco-card-body > .arco-alert',
      ).length,
      stepsDisplay: steps ? window.getComputedStyle(steps).display : null,
    };
  });

  expect(generatorShellContract.topAlertCount).toBe(0);
  expect(generatorShellContract.stepsDisplay).toBeTruthy();
});

test('system table pages keep unified table card spacing radius and neutral headers', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  for (const pageMeta of systemTablePages) {
    await navigateInShell(page, pageMeta.path);
    await expect(page.locator('.page-container > .page-header')).toHaveCount(0);
    await expect(
      page
        .locator('.system-user-list__function-bar, .governance-summary-bar, .system-list__work-actions')
        .first(),
    ).toBeVisible();
    const cssVariables = await readRootCssVariables(page, ['--shell-table-card-padding']);
    const [tablePaddingTop, tablePaddingRight, tablePaddingBottom, tablePaddingLeft] =
      expandPaddingValues(cssVariables['--shell-table-card-padding']);
    const tableCardBodyLocator = page.locator('.system-list__table-card > .arco-card-body').first();
    await expect(tableCardBodyLocator, pageMeta.path).toBeVisible();
    await expect(tableCardBodyLocator, pageMeta.path).toHaveCSS('padding-top', tablePaddingTop);
    await expect(tableCardBodyLocator, pageMeta.path).toHaveCSS('padding-right', tablePaddingRight);
    await expect(tableCardBodyLocator, pageMeta.path).toHaveCSS(
      'padding-bottom',
      tablePaddingBottom,
    );
    await expect(tableCardBodyLocator, pageMeta.path).toHaveCSS(
      'padding-left',
      tablePaddingLeft ?? tablePaddingRight,
    );

    const tableContract = await tableCardBodyLocator.evaluate((body) => {
      const tableCard = body.closest<HTMLElement>('.system-list__table-card');
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
    const filterBodyLocator = page.locator('.filter-panel .filter-panel__body').first();
    await expect(filterBodyLocator).toBeVisible();
    const firstItemLocator = page.locator('.filter-panel .arco-form-item').first();
    const firstControlLocator = page
      .locator(
        '.filter-panel .arco-input-inner-wrapper, .filter-panel .arco-input, .filter-panel .arco-select-view, .filter-panel .arco-tree-select-view, .filter-panel .arco-picker',
      )
      .first();
    const actionButtonLocator = page.locator('.filter-panel__action-item .arco-btn').first();
    const cssVariables = await readRootCssVariables(page, [
      '--shell-filter-body-padding',
      '--shell-filter-form-item-margin-bottom',
      '--shell-filter-control-min-height',
    ]);
    const [filterPaddingTop, filterPaddingRight, filterPaddingBottom, filterPaddingLeft] =
      expandPaddingValues(cssVariables['--shell-filter-body-padding']);
    await expect(filterBodyLocator, `${path} paddingTop`).toHaveCSS('padding-top', filterPaddingTop);
    await expect(filterBodyLocator, `${path} paddingRight`).toHaveCSS(
      'padding-right',
      filterPaddingRight,
    );
    await expect(filterBodyLocator, `${path} paddingBottom`).toHaveCSS(
      'padding-bottom',
      filterPaddingBottom,
    );
    await expect(filterBodyLocator, `${path} paddingLeft`).toHaveCSS(
      'padding-left',
      filterPaddingLeft ?? filterPaddingRight,
    );
    await expect(firstItemLocator, `${path} firstItem marginBottom`).toHaveCSS(
      'margin-bottom',
      cssVariables['--shell-filter-form-item-margin-bottom'],
    );
    const firstControlHeight = await firstControlLocator.evaluate((element) =>
      Math.round(element.getBoundingClientRect().height),
    );
    expect(firstControlHeight, path).toBeGreaterThanOrEqual(
      Number.parseInt(cssVariables['--shell-filter-control-min-height'], 10),
    );
    const actionButtonHeight = await actionButtonLocator.evaluate((element) =>
      Math.round(element.getBoundingClientRect().height),
    );
    expect(actionButtonHeight, path).toBeGreaterThanOrEqual(
      Number.parseInt(cssVariables['--shell-filter-control-min-height'], 10),
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
    expect(governanceContract.main?.justifyContent, path).toBe('space-between');
    expect(governanceContract.main?.height, path).toBeGreaterThanOrEqual(32);
    if (governanceContract.actions) {
      expect(governanceContract.actions.justifyContent, path).toBe('flex-end');
    }
    expect(governanceContract.select?.width, path).toBeGreaterThanOrEqual(120);
  }
});

test('business actions stay in the work area and dialogs use single-layer inputs', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);
  await navigateInShell(page, '/system/user');

  await expect(page.locator('.page-container > .page-header')).toHaveCount(0);
  await expect(page.locator('.governance-summary-bar')).toBeVisible();
  await expect(page.locator('.system-user-list__hero')).toHaveCount(0);
  await expect(page.locator('.table-batch-action-bar__prefix-actions')).toBeVisible();
  await expect(page.locator('.table-batch-action-bar__prefix-actions').getByText('新增')).toBeVisible();
  await expect(page.locator('.table-batch-action-bar__prefix-actions').getByText('导入')).toBeVisible();
  await expect(page.locator('.table-batch-action-bar__prefix-actions').getByText('导出')).toBeVisible();

  const actionContract = await page.evaluate(() => {
    const summaryBar = document.querySelector<HTMLElement>('.governance-summary-bar');
    const prefixActions = document.querySelector<HTMLElement>(
      '.table-batch-action-bar__prefix-actions',
    );
    const batchBar = document.querySelector<HTMLElement>('.table-batch-action-bar');
    return {
      summaryOutsideBatch: Boolean(summaryBar && !batchBar?.contains(summaryBar)),
      prefixInsideBatch: Boolean(prefixActions && batchBar?.contains(prefixActions)),
      prefixHasCreate: Boolean(
        prefixActions &&
          Array.from(prefixActions.querySelectorAll('button')).some((button) =>
            /新增/.test(button.textContent || ''),
          ),
      ),
      prefixGap: prefixActions ? window.getComputedStyle(prefixActions).gap : null,
    };
  });

  expect(actionContract.summaryOutsideBatch).toBe(true);
  expect(actionContract.prefixInsideBatch).toBe(true);
  expect(actionContract.prefixHasCreate).toBe(true);
  expect(actionContract.prefixGap).toBe('6px 8px');

  await page.locator('.table-batch-action-bar__prefix-actions').getByText('新增').click();
  await expect(page.locator('.app-dialog')).toBeVisible();

  const controlContracts = await readVisibleControlContracts(page, '.app-dialog');
  expect(controlContracts.length).toBeGreaterThanOrEqual(5);
  for (const controlContract of controlContracts) {
    expectSingleLayerDialogControl(
      controlContract,
      `/system/user ${controlContract.label ?? controlContract.kind}`,
    );
  }

  const focusedTextContract = await readFocusedTextInputContract(page, '.app-dialog');
  expectSingleLayerDialogControl(focusedTextContract, '/system/user focused text input');
  expect(focusedTextContract.wrapper?.boxShadow, '/system/user focused wrapper shadow').not.toBe(
    'none',
  );
  if (focusedTextContract.inner) {
    expect(focusedTextContract.inner.boxShadow, '/system/user focused inner shadow').toBe('none');
  }
});

test('narrow mobile layout keeps actions dialogs and states inside the viewport', async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await signInAsAdmin(page);
  await navigateInShell(page, '/system/user');

  await expect(page.locator('.page-container > .page-header')).toHaveCount(0);
  await expect(page.locator('.governance-summary-bar')).toBeVisible();
  await expect(page.locator('.table-batch-action-bar__prefix-actions')).toBeVisible();

  const mobileContract = await page.evaluate(() => {
    const viewportWidth = window.innerWidth;
    const selectors = [
      '.page-container',
      '.governance-summary-bar',
      '.filter-panel',
      '.system-list__table-card',
      '.table-batch-action-bar',
      '.table-batch-action-bar__prefix-actions',
    ];
    const boxes = selectors.map((selector) => {
      const element = document.querySelector<HTMLElement>(selector);
      if (!element) {
        return { selector, exists: false };
      }
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return {
        selector,
        exists: true,
        display: style.display,
        left: Math.round(rect.left),
        right: Math.round(rect.right),
        width: Math.round(rect.width),
      };
    });
    return {
      viewportWidth,
      bodyScrollWidth: document.documentElement.scrollWidth,
      boxes,
    };
  });

  expect(mobileContract.bodyScrollWidth).toBeLessThanOrEqual(mobileContract.viewportWidth + 1);
  for (const box of mobileContract.boxes) {
    expect(box.exists, box.selector).toBe(true);
    expect(box.left, box.selector).toBeGreaterThanOrEqual(0);
    expect(box.right, box.selector).toBeLessThanOrEqual(mobileContract.viewportWidth + 1);
  }

  await page.locator('.table-batch-action-bar__prefix-actions').getByText('新增').click();
  await expect(page.locator('.app-dialog')).toBeVisible();
  const dialogContract = await readDialogContract(page);
  expect(dialogContract.dialog?.left).toBeGreaterThanOrEqual(0);
  expect(dialogContract.dialog?.right).toBeLessThanOrEqual(dialogContract.viewportWidth);
  expect(dialogContract.dialog?.bottom).toBeLessThanOrEqual(dialogContract.viewportHeight);
  expect(dialogContract.content?.overflowX).toBe('hidden');
  expect(dialogContract.content?.overflowY).toBe('auto');
  expect(dialogContract.header?.paddingLeft).toBe('16px');
  expect(dialogContract.content?.paddingLeft).toBe('16px');
  expect(dialogContract.footer?.paddingLeft).toBe('16px');
});

test('core system dialogs share modal spacing section and input contracts', async ({ page }) => {
  test.setTimeout(60000);
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  for (const entry of dialogEntryPages) {
    await navigateInShell(page, entry.path);
    await expect(page.locator('.page-container > .page-header')).toHaveCount(0);
    const createButton = page
      .locator(
        '.system-user-list__function-bar, .table-batch-action-bar__prefix-actions, .dict-page__actions, .system-list__work-actions, .list-header-actions__primary',
      )
      .getByRole('button', { name: entry.triggerText })
      .first();
    await expect(createButton, entry.path).toBeVisible();
    await expect(createButton, entry.path).toBeEnabled();
    await createButton.click();
    await expect(page.locator('.app-dialog')).toBeVisible();
    await page.evaluate(() => {
      document
        .querySelector<HTMLElement>('.app-dialog .arco-input-inner-wrapper .arco-input')
        ?.focus();
    });

    const dialogContract = await readDialogContract(page, { preserveFocus: true });
    expect(dialogContract.dialog?.borderTopWidth, entry.path).toBe('1px');
    expect(dialogContract.header?.height, entry.path).toBeGreaterThanOrEqual(64);
    expect(dialogContract.header?.paddingTop, entry.path).toBe('16px');
    expect(dialogContract.header?.paddingLeft, entry.path).toBe('24px');
    expect(dialogContract.header?.paddingBottom, entry.path).toBe('16px');
    expect(dialogContract.content?.paddingTop, entry.path).toBe('20px');
    expect(dialogContract.content?.paddingRight, entry.path).toBe('24px');
    expect(dialogContract.content?.paddingBottom, entry.path).toBe('24px');
    expect(dialogContract.content?.paddingLeft, entry.path).toBe('24px');
    expect(dialogContract.content?.overflowX, entry.path).toBe('hidden');
    expect(dialogContract.content?.overflowY, entry.path).toBe('auto');
    if (dialogContract.footer) {
      expect(dialogContract.footer.paddingTop, entry.path).toBe('16px');
      expect(dialogContract.footer.paddingRight, entry.path).toBe('24px');
      expect(dialogContract.footer.paddingBottom, entry.path).toBe('20px');
      expect(dialogContract.footer.paddingLeft, entry.path).toBe('24px');
    }
    if (dialogContract.submitBar) {
      expect(dialogContract.submitBar.justifyContent, entry.path).toBe('flex-end');
    }
    if (dialogContract.formSectionTitle) {
      expect(dialogContract.formSectionTitle.fontSize, entry.path).toBe('14px');
      expect(Number(dialogContract.formSectionTitle.fontWeight), entry.path).toBeGreaterThanOrEqual(600);
    }
    expect(dialogContract.wrapper?.borderTopWidth, entry.path).toBe('1px');
    if (dialogContract.input) {
      expect(dialogContract.input.borderTopWidth, entry.path).toBe('0px');
      expect(dialogContract.input.backgroundColor, entry.path).toBe('rgba(0, 0, 0, 0)');
      expect(dialogContract.input.boxShadow, entry.path).toBe('none');
      expect(dialogContract.input.outlineStyle, entry.path).toBe('none');
      expect(dialogContract.input.outlineWidth, entry.path).toBe('0px');
    }
    const sampledControlContracts = await readDialogControlContracts(page);
    for (const [controlName, controlContract] of Object.entries(sampledControlContracts)) {
      if (controlContract.wrapper) {
        expectSingleLayerDialogControl(controlContract, `${entry.path} sampled ${controlName}`);
      }
    }
    const visibleControlContracts = await readVisibleControlContracts(page, '.app-dialog');
    expect(visibleControlContracts.length, `${entry.path} visible controls`).toBeGreaterThan(0);
    for (const controlContract of visibleControlContracts) {
      expectSingleLayerDialogControl(
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

  for (const path of filterPanelPages) {
    await navigateInShell(page, path);
    await expect(page.locator('.filter-panel').first(), `${path} filter panel`).toBeVisible();
    const filterContracts = await readVisibleControlContracts(page, '.filter-panel');
    expect(filterContracts.length, `${path} filter controls`).toBeGreaterThan(0);
    for (const controlContract of filterContracts) {
      expectSingleLayerDialogControl(
        controlContract,
        `${path} filter ${controlContract.label ?? controlContract.kind}`,
        { requireWhiteBackground: false },
      );
    }

    const focusedFilterContract = await readFocusedTextInputContract(page, '.filter-panel');
    expectSingleLayerDialogControl(focusedFilterContract, `${path} focused filter text input`, {
      requireWhiteBackground: false,
    });
    expect(
      focusedFilterContract.wrapper?.boxShadow,
      `${path} focused filter wrapper shadow`,
    ).not.toBe('none');
    if (focusedFilterContract.inner) {
      expect(focusedFilterContract.inner.boxShadow, `${path} focused filter inner shadow`).toBe(
        'none',
      );
    }
  }

  await navigateInShell(page, '/system/i18n');
  const createTrigger = page
    .locator('.table-batch-action-bar__prefix-actions')
    .getByRole('button', { name: '新增' })
    .first();
  await expect(createTrigger).toBeVisible();
  await expect(createTrigger).toBeEnabled();
  await createTrigger.click();
  await expect(page.locator('.app-dialog')).toBeVisible();
  await expect(page.locator('.app-dialog .arco-select-view').first()).toBeVisible();
  const focusedDialogContract = await readFocusedTextInputContract(page, '.app-dialog');
  expect(focusedDialogContract.wrapper, '/system/i18n dialog focused wrapper').not.toBeNull();
  expectSingleLayerDialogControl(focusedDialogContract, '/system/i18n dialog focused text input');
  expect(
    focusedDialogContract.wrapper?.boxShadow,
    '/system/i18n dialog focused wrapper shadow',
  ).not.toBe('none');
  if (focusedDialogContract.inner) {
    expect(
      focusedDialogContract.inner.boxShadow,
      '/system/i18n dialog focused inner shadow',
    ).toBe('none');
  }
});

test('emerald theme dialogs keep themed select focus and single-line authorization headers', async ({
  page,
}) => {
  test.setTimeout(60000);
  await page.setViewportSize({ width: 1440, height: 900 });
  await installEmeraldThemePreference(page);
  await signInAsAdmin(page);

  await navigateInShell(page, '/system/role');
  const createTrigger = page
    .locator('.table-batch-action-bar__prefix-actions')
    .getByRole('button', { name: '新增' })
    .first();
  await expect(createTrigger).toBeVisible();
  await createTrigger.click();
  const roleDialog = page.locator('.app-dialog').first();
  await expect(roleDialog).toBeVisible();

  const roleStatusSelect = roleDialog
    .locator('.arco-form-item')
    .filter({ hasText: /状态|Status/i })
    .locator('.arco-select')
    .first();
  const roleStatusView = roleStatusSelect.locator('.arco-select-view');
  await expect(roleStatusView).toBeVisible();
  await roleStatusView.click();
  await expect(roleStatusSelect).toHaveClass(/arco-select-open/);
  await page.waitForTimeout(100);
  const roleStatusContract = await roleStatusView.evaluate((view) => {
    const style = window.getComputedStyle(view);
    return {
      borderTopColor: style.borderTopColor,
      boxShadow: style.boxShadow,
    };
  });
  expect(roleStatusContract.borderTopColor).toBeTruthy();
  expect(roleStatusContract.boxShadow).not.toContain('190, 218, 255');
  expect(roleStatusContract.boxShadow).not.toBe('none');

  const authorizationHeaderMetrics = await roleDialog.evaluate((dialog) => {
    return Array.from(
      dialog.querySelectorAll<HTMLElement>('.dialog-grid-card .arco-card-header-title'),
    )
      .slice(0, 3)
      .map((header) => {
        const text = header.querySelector<HTMLElement>('.arco-typography');
        const titleStyle = window.getComputedStyle(header);
        const textStyle = text ? window.getComputedStyle(text) : null;
        const lineHeight = Number.parseFloat(textStyle?.lineHeight || '0');
        const height = text?.getBoundingClientRect().height || 0;
        return {
          whiteSpace: textStyle?.whiteSpace ?? null,
          lines: lineHeight > 0 ? Math.round(height / lineHeight) : 0,
          headerOverflow: titleStyle.overflow,
        };
      });
  });
  expect(authorizationHeaderMetrics).toHaveLength(3);
  for (const metric of authorizationHeaderMetrics) {
    expect(metric.whiteSpace).toBe('nowrap');
    expect(metric.lines).toBeLessThanOrEqual(1);
    expect(metric.headerOverflow).toBe('hidden');
  }

  await roleDialog.locator('.arco-modal-close-icon').click();
  await expect(page.locator('.app-dialog')).toHaveCount(0);

  await openSystemDialogPage(page, '/system/menu');
  const menuCreateTrigger = page.getByRole('button', { name: '新增', exact: true }).first();
  await expect(menuCreateTrigger).toBeVisible();
  await menuCreateTrigger.click();
  const menuDialog = page.locator('.app-dialog').first();
  await expect(menuDialog).toBeVisible();

  const treeSelectContract = await menuDialog.evaluate((dialog) => {
    const control = dialog.querySelector<HTMLElement>('.arco-tree-select-view');
    const style = control ? window.getComputedStyle(control) : null;
    return {
      borderTopWidth: style?.borderTopWidth ?? null,
      borderTopStyle: style?.borderTopStyle ?? null,
      backgroundColor: style?.backgroundColor ?? null,
    };
  });
  expect(treeSelectContract.borderTopWidth).toBe('1px');
  expect(treeSelectContract.borderTopStyle).toBe('solid');
  expect(treeSelectContract.backgroundColor).toBe('rgb(255, 255, 255)');
});

test('governance drawers share overlay spacing and surface contracts', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);
  await navigateInShell(page, '/system/user');

  const drawerTrigger = page.getByRole('button', { name: '治理摘要' }).first();
  await expect(drawerTrigger).toBeVisible();
  await drawerTrigger.click();
  await expect(page.locator('.app-drawer')).toBeVisible();

  const drawerContract = await page.evaluate(() => {
    const drawer = document.querySelector<HTMLElement>('.app-drawer');
    const readElement = (element?: HTMLElement | null) => {
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
        paddingBottom: style.paddingBottom,
        paddingLeft: style.paddingLeft,
        paddingRight: style.paddingRight,
        paddingTop: style.paddingTop,
      };
    };
    const read = (selector: string) => readElement(drawer?.querySelector<HTMLElement>(selector));

    return {
      appDrawerClassName: drawer?.className ?? null,
      shell: readElement(drawer),
      content: read('.arco-drawer-content'),
      header: read('.arco-drawer-header'),
      body: read('.arco-drawer-body, .arco-drawer-content'),
      summaryCard: read('.governance-insight-drawer__summary'),
    };
  });

  expect(drawerContract.shell?.borderTopWidth).toBe('1px');
  expect(drawerContract.shell?.borderTopStyle).toBe('solid');
  expect(drawerContract.shell?.boxShadow).not.toBe('none');
  expect(drawerContract.header?.height).toBeGreaterThanOrEqual(64);
  expect(drawerContract.header?.paddingLeft).toBe('24px');
  expect(drawerContract.header?.paddingRight).toBe('24px');
  expect(drawerContract.body?.paddingTop).toBe('20px');
  expect(drawerContract.body?.paddingRight).toBe('24px');
  expect(drawerContract.body?.paddingBottom).toBe('24px');
  expect(drawerContract.body?.paddingLeft).toBe('24px');
  expect(drawerContract.summaryCard?.backgroundColor).toBe('rgb(255, 255, 255)');
});

test('empty loading error and destructive controls keep shared visual semantics', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);
  await navigateInShell(page, '/system/user');

  await page.evaluate(() => {
    const host = document.createElement('div');
    host.setAttribute('data-testid', 'visual-state-fixtures');
    host.innerHTML = `
      <div class="page-loading"><div class="arco-spin"></div></div>
      <div class="page-empty"><div class="page-empty__inner">empty</div></div>
    `;
    document.body.appendChild(host);
  });

  const stateContract = await page.evaluate(() => {
    const read = (selector: string) => {
      const element = document.querySelector<HTMLElement>(selector);
      if (!element) {
        return null;
      }
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return {
        backgroundColor: style.backgroundColor,
        borderTopStyle: style.borderTopStyle,
        borderTopWidth: style.borderTopWidth,
        display: style.display,
        height: Math.round(rect.height),
        justifyContent: style.justifyContent,
        lineHeight: style.lineHeight,
        paddingTop: style.paddingTop,
        width: Math.round(rect.width),
      };
    };
    return {
      loading: read('[data-testid="visual-state-fixtures"] .page-loading'),
      empty: read('[data-testid="visual-state-fixtures"] .page-empty'),
      emptyInner: read('[data-testid="visual-state-fixtures"] .page-empty__inner'),
    };
  });

  expect(stateContract.loading?.height).toBeGreaterThanOrEqual(240);
  expect(stateContract.loading?.display).toBe('flex');
  expect(stateContract.loading?.justifyContent).toBe('center');
  expect(stateContract.empty?.height).toBeGreaterThanOrEqual(220);
  expect(stateContract.empty?.paddingTop).toBe('16px');
  expect(stateContract.emptyInner?.borderTopStyle).toBe('dashed');

  await navigateInShell(page, '/system/user/not-a-number');
  await expect(page.locator('.app-shell__content .page-empty').first()).toBeVisible();
  await page.locator('[data-testid="visual-state-fixtures"]').evaluate((element) => {
    element.remove();
  });
  await navigateInShell(page, '/system/user');
  await expect(page.locator('.system-list__table-card .app-table')).toBeVisible();
  await page.locator('.system-user-list__table .arco-checkbox').nth(1).click({ force: true });
  await expect(page.getByRole('button', { name: '删除所选' })).toBeEnabled();

  const destructiveContract = await page.evaluate(() => {
    const listRoot = document.querySelector<HTMLElement>('.page-container') || document;
    const buttons = Array.from(listRoot.querySelectorAll<HTMLElement>('.arco-btn')).filter(
      (button) => /删除|delete/i.test(button.textContent || ''),
    );
    return buttons.map((button) => {
      const style = window.getComputedStyle(button);
      return {
        text: button.textContent?.trim() || '',
        hasDangerClass: Array.from(button.classList).some((className) =>
          className.includes('danger'),
        ),
        color: style.color,
      };
    });
  });

  expect(destructiveContract.length).toBeGreaterThan(0);
  for (const button of destructiveContract) {
    expect(button.hasDangerClass, button.text).toBe(true);
  }
});

test('dict management keeps both tabs on the shared list rhythm', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);
  await navigateInShell(page, '/system/dict');
  await expect(page.locator('.page-container > .page-header')).toHaveCount(0);
  await expect(page.locator('.dict-page__governance-bar')).toBeVisible();
  await expect(page.locator('.dict-page__table-card .dict-page__governance-bar')).toHaveCount(0);
  await expect(page.locator('.arco-tabs-content .dict-workbench__context-card')).toHaveCount(0);

  const readDictTabContract = async () =>
    page.evaluate(() => {
      const governanceBar = document.querySelector<HTMLElement>('.dict-page__governance-bar');
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
        governanceTop: governanceBar ? Math.round(governanceBar.getBoundingClientRect().top) : null,
        governancePaddingTop: governanceBar ? window.getComputedStyle(governanceBar).paddingTop : null,
        governancePaddingBottom: governanceBar
          ? window.getComputedStyle(governanceBar).paddingBottom
          : null,
        tableTop: tableCard ? Math.round(tableCard.getBoundingClientRect().top) : null,
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
    '--shell-filter-form-item-margin-bottom',
    '--shell-filter-control-min-height',
  ]);
  const [filterPaddingTop, filterPaddingRight, filterPaddingBottom, filterPaddingLeft] =
    expandPaddingValues(cssVariables['--shell-filter-body-padding']);

  const typeTabContract = await readDictTabContract();
  expect(typeTabContract.tableTop).toBeGreaterThan(typeTabContract.governanceTop!);
  expect(typeTabContract.governancePaddingTop).toBe('10px');
  expect(typeTabContract.governancePaddingBottom).toBe('10px');
  expect(typeTabContract.hasSharedTableCard).toBe(true);
  expect(typeTabContract.hasSharedSystemTable).toBe(true);
  expect(typeTabContract.filterBody?.paddingTop).toBe(filterPaddingTop);
  expect(typeTabContract.filterBody?.paddingRight).toBe(filterPaddingRight);
  expect(typeTabContract.filterBody?.paddingBottom).toBe(filterPaddingBottom);
  expect(typeTabContract.filterBody?.paddingLeft).toBe(filterPaddingLeft ?? filterPaddingRight);
  expect(typeTabContract.firstItem?.marginBottom).toBe(
    cssVariables['--shell-filter-form-item-margin-bottom'],
  );
  expect(typeTabContract.firstControl?.height).toBeGreaterThanOrEqual(
    Number.parseInt(cssVariables['--shell-filter-control-min-height'], 10),
  );
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
  expect(itemTabContract.firstControl?.height).toBeGreaterThanOrEqual(
    Number.parseInt(cssVariables['--shell-filter-control-min-height'], 10),
  );
  expect(itemTabContract.listActions?.justifyContent).toBe('space-between');
  expect(itemTabContract.listActionsPrimary?.justifyContent).toBe('flex-end');
  if (itemTabContract.hasVisibleTable) {
    expect(itemTabContract.batchBar?.display).toBe('grid');
    expect(itemTabContract.tableContainer?.borderTopLeftRadius).toBe('0px');
  }
});
