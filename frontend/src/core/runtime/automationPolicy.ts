interface AutomationRuntime {
  dev?: boolean;
  webdriver?: boolean;
}

function isDevRuntime() {
  return Boolean((import.meta as ImportMeta & { env?: { DEV?: boolean } }).env?.DEV);
}

function isAutomatedBrowserRuntime() {
  return Boolean(globalThis.navigator?.webdriver);
}

export function isAutomatedDevBrowserRuntime(runtime: AutomationRuntime = {}) {
  const dev = runtime.dev ?? isDevRuntime();
  const webdriver = runtime.webdriver ?? isAutomatedBrowserRuntime();
  return dev && webdriver;
}

export function shouldRunBackgroundNetworkTasks(runtime: AutomationRuntime = {}) {
  return !isAutomatedDevBrowserRuntime(runtime);
}

export function shouldWarmHighFrequencyRouteData(runtime: AutomationRuntime = {}) {
  return shouldRunBackgroundNetworkTasks(runtime);
}

export function shouldFetchRemoteI18nPack(runtime: AutomationRuntime = {}) {
  return shouldRunBackgroundNetworkTasks(runtime);
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export function shouldPollServerRefreshState(_runtime: AutomationRuntime = {}) {
  // Refresh state polling must remain active even in automated browser
  // environments so refresh sync smoke tests can verify cross-context
  // data propagation. The polling is a lightweight GET and does not affect
  // page state or cause visual flakiness.
  return true;
}

export function shouldReportShellActivity(runtime: AutomationRuntime = {}) {
  return shouldRunBackgroundNetworkTasks(runtime);
}

export function shouldLoadShellNoticeSummary(runtime: AutomationRuntime = {}) {
  return shouldRunBackgroundNetworkTasks(runtime);
}
