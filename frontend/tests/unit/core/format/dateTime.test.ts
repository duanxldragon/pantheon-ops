import { describe, it, expect } from 'vitest'
import { isValidDateValue, formatDate, formatDateTime, formatRelativeTime } from '../../../../src/core/format/dateTime'

describe('isValidDateValue', () => {
  it('returns true for ISO string', () => {
    expect(isValidDateValue('2026-07-22T10:00:00+08:00')).toBe(true)
  })

  it('returns true for local date string', () => {
    expect(isValidDateValue('2026-07-22')).toBe(true)
  })

  it('returns true for Date object', () => {
    expect(isValidDateValue(new Date(2026, 0, 1))).toBe(true)
  })

  it('returns false for null', () => {
    expect(isValidDateValue(null)).toBe(false)
  })

  it('returns false for undefined', () => {
    expect(isValidDateValue(undefined)).toBe(false)
  })

  it('returns false for empty string', () => {
    expect(isValidDateValue('')).toBe(false)
  })

  it('returns false for invalid string', () => {
    expect(isValidDateValue('not-a-date')).toBe(false)
  })
})

describe('formatDate', () => {
  it('formats a local date string to YYYY-MM-DD', () => {
    expect(formatDate('2026-07-22')).toBe('2026-07-22')
  })

  it('returns "-" for null', () => {
    expect(formatDate(null)).toBe('-')
  })

  it('returns "-" for undefined', () => {
    expect(formatDate(undefined)).toBe('-')
  })

  it('returns "-" for empty string', () => {
    expect(formatDate('')).toBe('-')
  })

  it('returns raw string for unparseable value', () => {
    expect(formatDate('bad')).toBe('bad')
  })
})

describe('formatDateTime', () => {
  it('formats local datetime string without seconds by default', () => {
    expect(formatDateTime('2026-07-22 14:30:00')).toBe('2026-07-22 14:30')
  })

  it('includes seconds when withSeconds is true', () => {
    expect(formatDateTime('2026-07-22 14:30:45', { withSeconds: true })).toBe('2026-07-22 14:30:45')
  })

  it('returns "-" for null', () => {
    expect(formatDateTime(null)).toBe('-')
  })

  it('returns raw string for unparseable value', () => {
    expect(formatDateTime('nope')).toBe('nope')
  })
})

describe('formatRelativeTime', () => {
  it('returns a non-empty string for valid dates', () => {
    const base = new Date('2026-07-22T10:00:00Z')
    const target = new Date('2026-07-21T10:00:00Z')
    const result = formatRelativeTime(target, base)
    expect(typeof result).toBe('string')
    expect(result.length).toBeGreaterThan(0)
  })

  it('returns "-" when target is null', () => {
    expect(formatRelativeTime(null, new Date())).toBe('-')
  })

  it('returns a string when base is null (falls back to epoch)', () => {
    const result = formatRelativeTime(new Date(), null)
    expect(typeof result).toBe('string')
    expect(result.length).toBeGreaterThan(0)
  })
})
