export type DateValue = string | number | Date | null | undefined;

interface FormatDateTimeOptions {
  withSeconds?: boolean;
}

type RelativeTimeUnit = Intl.RelativeTimeFormatUnit;

const dateTimeFormatterCache = new Map<string, Intl.DateTimeFormat>();
const relativeTimeFormatterCache = new Map<string, Intl.RelativeTimeFormat>();

function resolveLocale() {
  if (typeof window !== 'undefined') {
    const preferred = window.localStorage.getItem('pantheon_lang');
    if (preferred) {
      return preferred;
    }
    if (window.navigator.language) {
      return window.navigator.language;
    }
  }
  return 'zh-CN';
}

function parseDate(value: DateValue) {
  if (value === null || value === undefined || value === '') {
    return null;
  }
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return date;
}

export function isValidDateValue(value?: DateValue) {
  return parseDate(value) !== null;
}

function getInvalidFallback(value: DateValue) {
  if (value === null || value === undefined || value === '') {
    return '-';
  }
  return String(value);
}

function getDateTimeFormatter(options?: FormatDateTimeOptions) {
  const locale = resolveLocale();
  const withSeconds = options?.withSeconds !== false;
  const cacheKey = `${locale}:${withSeconds ? 'seconds' : 'minutes'}`;
  const cached = dateTimeFormatterCache.get(cacheKey);
  if (cached) {
    return cached;
  }
  const formatter = new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: withSeconds ? '2-digit' : undefined,
    hour12: false,
  });
  dateTimeFormatterCache.set(cacheKey, formatter);
  return formatter;
}

function getDateFormatter() {
  const locale = resolveLocale();
  const cacheKey = `${locale}:date`;
  const cached = dateTimeFormatterCache.get(cacheKey);
  if (cached) {
    return cached;
  }
  const formatter = new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  });
  dateTimeFormatterCache.set(cacheKey, formatter);
  return formatter;
}

function getRelativeTimeFormatter() {
  const locale = resolveLocale();
  const cached = relativeTimeFormatterCache.get(locale);
  if (cached) {
    return cached;
  }
  const formatter = new Intl.RelativeTimeFormat(locale, {
    numeric: 'auto',
    style: 'long',
  });
  relativeTimeFormatterCache.set(locale, formatter);
  return formatter;
}

function resolveRelativeUnit(diffMs: number): { value: number; unit: RelativeTimeUnit } {
  const absMs = Math.abs(diffMs);
  const second = 1000;
  const minute = second * 60;
  const hour = minute * 60;
  const day = hour * 24;
  const month = day * 30;
  const year = day * 365;

  if (absMs >= year) {
    return { value: Math.round(diffMs / year), unit: 'year' };
  }
  if (absMs >= month) {
    return { value: Math.round(diffMs / month), unit: 'month' };
  }
  if (absMs >= day) {
    return { value: Math.round(diffMs / day), unit: 'day' };
  }
  if (absMs >= hour) {
    return { value: Math.round(diffMs / hour), unit: 'hour' };
  }
  if (absMs >= minute) {
    return { value: Math.round(diffMs / minute), unit: 'minute' };
  }
  return { value: Math.round(diffMs / second), unit: 'second' };
}

export function formatDate(value?: DateValue) {
  const date = parseDate(value);
  if (!date) {
    return getInvalidFallback(value);
  }
  return getDateFormatter().format(date);
}

export function formatDateTime(value?: DateValue, options?: FormatDateTimeOptions) {
  const date = parseDate(value);
  if (!date) {
    return getInvalidFallback(value);
  }
  return getDateTimeFormatter(options).format(date);
}

export function formatRelativeTime(value?: DateValue, baseTime: DateValue = Date.now()) {
  const targetDate = parseDate(value);
  const baseDate = parseDate(baseTime);
  if (!targetDate || !baseDate) {
    return getInvalidFallback(value);
  }
  const diffMs = targetDate.getTime() - baseDate.getTime();
  const { value: relativeValue, unit } = resolveRelativeUnit(diffMs);
  return getRelativeTimeFormatter().format(relativeValue, unit);
}
