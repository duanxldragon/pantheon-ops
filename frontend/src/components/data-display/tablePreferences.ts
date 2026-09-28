export type AppTableDensity = 'compact' | 'standard' | 'comfortable';

export interface AppTableColumnPreference {
  key: string;
  visible: boolean;
  order: number;
  width?: number | string;
}

export interface AppTablePreferences {
  version: 1;
  viewKey: string;
  density: AppTableDensity;
  columns: AppTableColumnPreference[];
  updatedAt: string;
}

export interface AppTableColumnMeta {
  key: string;
  defaultVisible: boolean;
  hideable: boolean;
  width?: number | string;
}

export const APP_TABLE_PREFERENCES_VERSION = 1;
export const APP_TABLE_PREFERENCES_PREFIX = 'pantheon:app-table:view:';

export function buildAppTablePreferenceStorageKey(viewKey: string) {
  return `${APP_TABLE_PREFERENCES_PREFIX}${viewKey}`;
}

export function createDefaultAppTablePreferences(
  viewKey: string,
  columns: AppTableColumnMeta[],
  density: AppTableDensity = 'standard',
): AppTablePreferences {
  return {
    version: APP_TABLE_PREFERENCES_VERSION,
    viewKey,
    density,
    columns: columns.map((column, index) => ({
      key: column.key,
      visible: column.defaultVisible,
      order: index,
      width: column.width,
    })),
    updatedAt: new Date(0).toISOString(),
  };
}

export function sanitizeAppTablePreferences(
  candidate: unknown,
  viewKey: string,
  columns: AppTableColumnMeta[],
): AppTablePreferences | null {
  if (!candidate || typeof candidate !== 'object') {
    return null;
  }

  const record = candidate as Partial<AppTablePreferences>;
  if (record.version !== APP_TABLE_PREFERENCES_VERSION || record.viewKey !== viewKey) {
    return null;
  }

  const defaultPreferences = createDefaultAppTablePreferences(viewKey, columns);
  const columnMetaByKey = new Map(columns.map((column) => [column.key, column]));
  const incomingColumns = Array.isArray(record.columns) ? record.columns : [];
  const sanitizedByKey = new Map<string, AppTableColumnPreference>();

  incomingColumns.forEach((column, index) => {
    if (!column || typeof column !== 'object') {
      return;
    }
    const key = typeof column.key === 'string' ? column.key : '';
    const meta = columnMetaByKey.get(key);
    if (!meta) {
      return;
    }
    sanitizedByKey.set(key, {
      key,
      visible: meta.hideable ? column.visible !== false : true,
      order: Number.isFinite(column.order) ? Number(column.order) : index,
      width:
        typeof column.width === 'number' || typeof column.width === 'string'
          ? column.width
          : meta.width,
    });
  });

  const mergedColumns = defaultPreferences.columns.map(
    (column) => sanitizedByKey.get(column.key) || column,
  );
  const density: AppTableDensity =
    record.density === 'compact' || record.density === 'comfortable' ? record.density : 'standard';

  return {
    ...defaultPreferences,
    density,
    columns: mergedColumns,
    updatedAt:
      typeof record.updatedAt === 'string' && record.updatedAt.trim()
        ? record.updatedAt
        : defaultPreferences.updatedAt,
  };
}

export function readAppTablePreferences(
  storage: Pick<Storage, 'getItem'> | undefined,
  viewKey: string,
  columns: AppTableColumnMeta[],
) {
  if (!storage) {
    return null;
  }

  let raw: string | null;
  try {
    raw = storage.getItem(buildAppTablePreferenceStorageKey(viewKey));
  } catch {
    return null;
  }
  if (!raw) {
    return null;
  }

  try {
    return sanitizeAppTablePreferences(JSON.parse(raw), viewKey, columns);
  } catch {
    return null;
  }
}

export function writeAppTablePreferences(
  storage: Pick<Storage, 'setItem'> | undefined,
  preferences: AppTablePreferences,
) {
  if (!storage) {
    return;
  }
  try {
    storage.setItem(
      buildAppTablePreferenceStorageKey(preferences.viewKey),
      JSON.stringify(preferences),
    );
  } catch {
    return;
  }
}

export function applyAppTablePreferences<T extends { columnKey?: string; dataIndex?: unknown }>(
  columns: T[],
  preferences: AppTablePreferences,
) {
  const preferenceByKey = new Map(preferences.columns.map((column) => [column.key, column]));
  return columns
    .filter((column) => {
      const key = resolveAppTableColumnKey(column);
      const preference = key ? preferenceByKey.get(key) : undefined;
      return preference?.visible !== false;
    })
    .map((column) => {
      const key = resolveAppTableColumnKey(column);
      const preference = key ? preferenceByKey.get(key) : undefined;
      return preference?.width === undefined ? column : { ...column, width: preference.width };
    })
    .sort((left, right) => {
      const leftKey = resolveAppTableColumnKey(left);
      const rightKey = resolveAppTableColumnKey(right);
      const leftOrder = leftKey ? preferenceByKey.get(leftKey)?.order : undefined;
      const rightOrder = rightKey ? preferenceByKey.get(rightKey)?.order : undefined;
      return (leftOrder ?? Number.MAX_SAFE_INTEGER) - (rightOrder ?? Number.MAX_SAFE_INTEGER);
    });
}

export function resolveAppTableColumnKey(column: { columnKey?: string; dataIndex?: unknown }) {
  if (typeof column.columnKey === 'string' && column.columnKey.trim()) {
    return column.columnKey;
  }
  return typeof column.dataIndex === 'string' && column.dataIndex.trim()
    ? column.dataIndex
    : undefined;
}
