export function normalizeRetentionOptions(rawValue: string | undefined, defaultOptions: number[] = [1, 7, 30]): number[] {
  if (!rawValue) {
    return defaultOptions;
  }
  try {
    const parsed = JSON.parse(rawValue) as unknown;
    if (!Array.isArray(parsed)) {
      return defaultOptions;
    }
    const normalized = Array.from(
      new Set(
        parsed.map(Number).filter((item) => Number.isInteger(item) && item > 0),
      ),
    ).sort((left, right) => right - left);
    return normalized.length > 0 ? normalized : defaultOptions;
  } catch {
    return defaultOptions;
  }
}

export function toCleanupTimestamp(value: string): string | undefined {
  const normalized = String(value || '').trim();
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/.exec(normalized);
  if (!match) {
    return undefined;
  }
  const [, year, month, day, hour, minute, second = '00'] = match;
  const localDate = new Date(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour),
    Number(minute),
    Number(second),
  );
  if (Number.isNaN(localDate.getTime())) {
    return undefined;
  }
  const offsetMinutes = -localDate.getTimezoneOffset();
  const sign = offsetMinutes >= 0 ? '+' : '-';
  const offsetHours = `${Math.floor(Math.abs(offsetMinutes) / 60)}`.padStart(2, '0');
  const offsetRemainMinutes = `${Math.abs(offsetMinutes) % 60}`.padStart(2, '0');
  return `${year}-${month}-${day}T${hour}:${minute}:${second}${sign}${offsetHours}:${offsetRemainMinutes}`;
}

export function loadRetentionSetting(
  group: { items: Array<{ settingKey: string; settingValue?: string }> },
  settingKey: string,
  setRetentionOptions: (opts: number[]) => void,
  setRetentionDays: (value: number | ((current: number) => number)) => void,
) {
  const setting = group.items.find((item) => item.settingKey === settingKey);
  const nextOptions = normalizeRetentionOptions(setting?.settingValue);
  setRetentionOptions(nextOptions);
  setRetentionDays((current) => (nextOptions.includes(current) ? current : nextOptions[0]));
}
