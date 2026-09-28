export type DateValue = string | number | Date | null | undefined;

interface FormatDateTimeOptions {
  withSeconds?: boolean;
}

type RelativeTimeUnit = Intl.RelativeTimeFormatUnit;

const relativeTimeFormatterCache = new Map<string, Intl.RelativeTimeFormat>();

function resolveLocale() {
  if (globalThis.document !== undefined) {
    const preferred = globalThis.localStorage.getItem('pantheon_lang');
    if (preferred) {
      return preferred;
    }
    if (globalThis.navigator.language) {
      return globalThis.navigator.language;
    }
  }
  return 'zh-CN';
}

function parseDate(value: DateValue) {
  if (value === null || value === undefined || value === '') {
    return null;
  }
  if (typeof value === 'string') {
    const normalized = value.trim();
    if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(normalized)) {
      const date = new Date(normalized);
      if (!Number.isNaN(date.getTime())) {
        return date;
      }
    }
    const localDateTimeMatch = /^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2}))?)?$/.exec(
      normalized,
    );
    if (localDateTimeMatch) {
      const [, year, month, day, hour = '00', minute = '00', second = '00'] = localDateTimeMatch;
      const date = new Date(
        Number(year),
        Number(month) - 1,
        Number(day),
        Number(hour),
        Number(minute),
        Number(second),
      );
      if (!Number.isNaN(date.getTime())) {
        return date;
      }
    }
  }
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return date;
}

function formatDateParts(date: Date, withTime: boolean, withSeconds: boolean) {
  // Use local getters to match how backend stores / sends local datetime strings
  // (e.g. "2026-08-11 14:23:45" → year/month/day/hour/minute/second all in local TZ).
  // This keeps formatDateTime output consistent with parseDate input so it round-trips correctly.
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  const datePart = `${year}-${month}-${day}`;
  if (!withTime) {
    return datePart;
  }
  const timePart = `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
  if (!withSeconds) {
    return `${datePart} ${timePart}`;
  }
  return `${datePart} ${timePart}:${String(date.getSeconds()).padStart(2, '0')}`;
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
  return formatDateParts(date, false, false);
}

export function formatDateTime(value?: DateValue, options?: FormatDateTimeOptions) {
  const date = parseDate(value);
  if (!date) {
    return getInvalidFallback(value);
  }
  return formatDateParts(date, true, options?.withSeconds === true);
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
