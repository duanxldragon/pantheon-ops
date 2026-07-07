export const TABLE_ACTION_COLUMN_WIDTH = {
  single: 108,
  compact: 164,
  medium: 232,
  wide: 288,
} as const;

export type TableActionColumnWidthPreset = keyof typeof TABLE_ACTION_COLUMN_WIDTH;
