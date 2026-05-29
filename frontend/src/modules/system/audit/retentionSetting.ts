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
        parsed.map((item) => Number(item)).filter((item) => Number.isInteger(item) && item > 0),
      ),
    ).sort((left, right) => right - left);
    return normalized.length > 0 ? normalized : defaultOptions;
  } catch {
    return defaultOptions;
  }
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
  setRetentionDays((current) => (nextOptions.includes(current as number) ? current : nextOptions[0]));
}
