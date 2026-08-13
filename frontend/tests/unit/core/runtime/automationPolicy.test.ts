import { describe, it, expect } from 'vitest'
import {
  isAutomatedDevBrowserRuntime,
  shouldRunBackgroundNetworkTasks,
  shouldPollServerRefreshState,
  shouldWarmHighFrequencyRouteData,
  shouldFetchRemoteI18nPack,
  shouldReportShellActivity,
} from '../../../../src/core/runtime/automationPolicy'

describe('isAutomatedDevBrowserRuntime', () => {
  it('returns false when neither dev nor webdriver', () => {
    expect(isAutomatedDevBrowserRuntime({ dev: false, webdriver: false })).toBe(false)
  })
  it('returns false when dev only', () => {
    expect(isAutomatedDevBrowserRuntime({ dev: true, webdriver: false })).toBe(false)
  })
  it('returns false when webdriver only', () => {
    expect(isAutomatedDevBrowserRuntime({ dev: false, webdriver: true })).toBe(false)
  })
  it('returns true when both dev and webdriver', () => {
    expect(isAutomatedDevBrowserRuntime({ dev: true, webdriver: true })).toBe(true)
  })
})

describe('shouldRunBackgroundNetworkTasks', () => {
  it('returns true when not automated', () => {
    expect(shouldRunBackgroundNetworkTasks({ dev: false, webdriver: false })).toBe(true)
  })
  it('returns false when automated dev browser', () => {
    expect(shouldRunBackgroundNetworkTasks({ dev: true, webdriver: true })).toBe(false)
  })
})

describe('shouldPollServerRefreshState', () => {
  it('always returns true', () => {
    expect(shouldPollServerRefreshState()).toBe(true)
    expect(shouldPollServerRefreshState({ dev: true, webdriver: true })).toBe(true)
  })
})

describe('shouldWarmHighFrequencyRouteData', () => {
  it('returns true when not automated', () => {
    expect(shouldWarmHighFrequencyRouteData({ dev: false, webdriver: false })).toBe(true)
  })
  it('returns false when automated', () => {
    expect(shouldWarmHighFrequencyRouteData({ dev: true, webdriver: true })).toBe(false)
  })
})

describe('shouldFetchRemoteI18nPack', () => {
  it('returns true when not automated', () => {
    expect(shouldFetchRemoteI18nPack({ dev: false, webdriver: false })).toBe(true)
  })
  it('returns false when automated', () => {
    expect(shouldFetchRemoteI18nPack({ dev: true, webdriver: true })).toBe(false)
  })
})

describe('shouldReportShellActivity', () => {
  it('returns true when not automated', () => {
    expect(shouldReportShellActivity({ dev: false, webdriver: false })).toBe(true)
  })
  it('returns false when automated', () => {
    expect(shouldReportShellActivity({ dev: true, webdriver: true })).toBe(false)
  })
})

describe('default runtime (no explicit args)', () => {
  it('isAutomatedDevBrowserRuntime() with no args does not throw', () => {
    expect(typeof isAutomatedDevBrowserRuntime()).toBe('boolean')
  })

  it('shouldRunBackgroundNetworkTasks() with no args does not throw', () => {
    expect(typeof shouldRunBackgroundNetworkTasks()).toBe('boolean')
  })
})


