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
    const localDateTimeMatch = normalized.match(
      /^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2}))?)?$/,
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

export function isValidDateValue(value?: DateValue) {
  return parseDate(value) !== null;
}

function getInvalidFallback(value: DateValue) {
  if (value === null || value === undefined || value === '') {
    return '-';
  }
  return String(value);
}

function padDatePart(value: number) {
  return String(value).padStart(2, '0');
}

function formatDateParts(date: Date, withTime: boolean, withSeconds: boolean) {
  const datePart = `${date.getFullYear()}-${padDatePart(date.getMonth() + 1)}-${padDatePart(date.getDate())}`;
  if (!withTime) {
    return datePart;
  }
  const timePart = `${padDatePart(date.getHours())}:${padDatePart(date.getMinutes())}`;
  if (!withSeconds) {
    return `${datePart} ${timePart}`;
  }
  return `${datePart} ${timePart}:${padDatePart(date.getSeconds())}`;
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
  return formatDateParts(date, true, options?.withSeconds !== false);
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
