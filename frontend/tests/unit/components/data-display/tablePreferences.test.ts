import { describe, expect, it } from 'vitest';
import {
  applyAppTablePreferences,
  createDefaultAppTablePreferences,
  readAppTablePreferences,
  sanitizeAppTablePreferences,
  writeAppTablePreferences,
} from '../../../../src/components/data-display/tablePreferences';

const columns = [
  { key: 'name', defaultVisible: true, hideable: true, width: 160 },
  { key: 'email', defaultVisible: true, hideable: true },
  { key: 'actions', defaultVisible: true, hideable: false },
];

describe('AppTable preferences', () => {
  it('drops stale columns and keeps mandatory columns visible', () => {
    const result = sanitizeAppTablePreferences(
      {
        version: 1,
        viewKey: 'system.users',
        density: 'compact',
        columns: [
          { key: 'email', visible: false, order: 0 },
          { key: 'actions', visible: false, order: 1 },
          { key: 'removed-by-permission', visible: true, order: 2 },
        ],
      },
      'system.users',
      columns,
    );

    expect(result?.density).toBe('compact');
    expect(result?.columns.map((item) => item.key)).toEqual(['name', 'email', 'actions']);
    expect(result?.columns.find((item) => item.key === 'actions')?.visible).toBe(true);
  });

  it('falls back safely for corrupt persisted data', () => {
    const storage = {
      getItem: () => '{not-json',
      setItem: () => undefined,
    };

    expect(readAppTablePreferences(storage, 'system.users', columns)).toBeNull();
  });

  it('falls back safely when browser storage is unavailable', () => {
    const storage = {
      getItem: () => {
        throw new Error('denied');
      },
      setItem: () => {
        throw new Error('full');
      },
    };
    const preferences = createDefaultAppTablePreferences('system.users', columns);

    expect(readAppTablePreferences(storage, 'system.users', columns)).toBeNull();
    expect(() => writeAppTablePreferences(storage, preferences)).not.toThrow();
  });

  it('persists only an explicitly named view and applies its column order', () => {
    const values = new Map<string, string>();
    const storage = {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, value),
    };
    const preferences = createDefaultAppTablePreferences('system.users', columns);
    preferences.columns = [
      { key: 'email', visible: true, order: 0 },
      { key: 'name', visible: true, order: 1, width: 240 },
      { key: 'actions', visible: true, order: 2 },
    ];
    writeAppTablePreferences(storage, preferences);

    expect(readAppTablePreferences(storage, 'system.users', columns)?.columns[0]?.key).toBe('name');
    expect(
      applyAppTablePreferences(
        [{ dataIndex: 'name' }, { dataIndex: 'email' }, { dataIndex: 'actions' }],
        preferences,
      ).map((column) => column.dataIndex),
    ).toEqual(['email', 'name', 'actions']);
  });
});
