import assert from 'node:assert/strict';
import test from 'node:test';

import {
  shouldFetchRemoteI18nPack,
  shouldLoadShellNoticeSummary,
  shouldPollServerRefreshState,
  shouldReportShellActivity,
} from '../../src/core/runtime/automationPolicy.ts';
import { shouldWarmHighFrequencyRouteData } from '../../src/core/router/warmupPolicy.ts';

test('shouldWarmHighFrequencyRouteData disables speculative route data warmup for automated dev browsers', () => {
  assert.equal(shouldWarmHighFrequencyRouteData({ dev: true, webdriver: true }), false);
});

test('shouldWarmHighFrequencyRouteData keeps speculative route data warmup for interactive and production runtimes', () => {
  assert.equal(shouldWarmHighFrequencyRouteData({ dev: true, webdriver: false }), true);
  assert.equal(shouldWarmHighFrequencyRouteData({ dev: false, webdriver: true }), true);
  assert.equal(shouldWarmHighFrequencyRouteData({ dev: false, webdriver: false }), true);
});

test('automation runtime policies suppress non-essential background network for automated dev browsers', () => {
  const runtime = { dev: true, webdriver: true };
  assert.equal(shouldFetchRemoteI18nPack(runtime), false);
  // 刷新状态轮询在自动化环境下有意保持开启（refresh-sync 冒烟依赖跨上下文
  // 传播验证，见 automationPolicy.ts 内注释）。
  assert.equal(shouldPollServerRefreshState(runtime), true);
  assert.equal(shouldReportShellActivity(runtime), false);
  assert.equal(shouldLoadShellNoticeSummary(runtime), false);
});

test('automation runtime policies keep background network enabled outside automated dev browsers', () => {
  const runtimes = [
    { dev: true, webdriver: false },
    { dev: false, webdriver: true },
    { dev: false, webdriver: false },
  ];
  runtimes.forEach((runtime) => {
    assert.equal(shouldFetchRemoteI18nPack(runtime), true);
    assert.equal(shouldPollServerRefreshState(runtime), true);
    assert.equal(shouldReportShellActivity(runtime), true);
    assert.equal(shouldLoadShellNoticeSummary(runtime), true);
  });
});
