export const TABLE_COLUMN_WIDTH = {
  // Base size scale
  xs: 96,
  sm: 120,
  md: 160,
  lg: 220,
  xl: 280,
  tree: 300,
  time: 168,
  path: 240,
  content: 340,
  // Semantic aliases
  status: 96,
  count: 96,
  method: 96,
  scope: 96,
  code: 120,
  identity: 120,
  owner: 120,
  name: 160,
  location: 160,
  datetime: 168,
  tagGroup: 220,
  diagnostics: 220,
  routePath: 240,
  keyPath: 240,
  treeLabel: 300,
  body: 340,
} as const;

export type TableColumnWidthPreset = keyof typeof TABLE_COLUMN_WIDTH;
