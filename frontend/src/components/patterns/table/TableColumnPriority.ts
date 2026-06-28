import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';

export type TableColumnPriority = 'core' | 'medium' | 'low';

const TABLE_COLUMN_PRIORITY_CLASS: Record<Exclude<TableColumnPriority, 'core'>, string> = {
  medium: 'app-table__col--hide-md',
  low: 'app-table__col--hide-lg',
};

export function withTableColumnPriority<T>(
  column: ColumnProps<T>,
  priority: TableColumnPriority,
): ColumnProps<T> {
  if (priority === 'core') {
    return column;
  }

  const currentClassName = Array.isArray(column.className)
    ? column.className.join(' ')
    : column.className || '';
  const nextClassName = [currentClassName, TABLE_COLUMN_PRIORITY_CLASS[priority]]
    .filter(Boolean)
    .join(' ');

  return {
    ...column,
    className: nextClassName,
  };
}
