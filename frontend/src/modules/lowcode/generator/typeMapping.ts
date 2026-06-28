/**
 * 模块生成器 - 类型映射表
 *
 * 定义 Go / TypeScript / SQL 三端类型对应关系
 * 基于项目真实使用的类型约定
 */

import type { FieldType } from './schema';

/**
 * 类型映射定义
 */
export interface TypeMapping {
  go: string; // Go 类型
  ts: string; // TypeScript 类型
  sql: string; // SQL 数据类型
  gorm?: string; // GORM 标签(可选)
}

/**
 * 字段类型映射表
 *
 * 基于项目实际使用的类型:
 * - system/user: string, int64, time.Time, bool
 * - business/cmdb: string, int
 */
export const TYPE_MAPPING: Record<FieldType, TypeMapping> = {
  string: {
    go: 'string',
    ts: 'string',
    sql: 'VARCHAR(255)',
    gorm: 'size:255',
  },
  text: {
    go: 'string',
    ts: 'string',
    sql: 'TEXT',
    gorm: 'type:text',
  },
  int: {
    go: 'int64',
    ts: 'number',
    sql: 'BIGINT',
    gorm: '',
  },
  float: {
    go: 'float64',
    ts: 'number',
    sql: 'DECIMAL(10,2)',
    gorm: 'type:decimal(10,2)',
  },
  bool: {
    go: 'bool',
    ts: 'boolean',
    sql: 'TINYINT(1)',
    gorm: '',
  },
  date: {
    go: 'time.Time',
    ts: 'string',
    sql: 'DATETIME',
    gorm: '',
  },
  enum: {
    go: 'string',
    ts: 'string',
    sql: 'VARCHAR(50)',
    gorm: 'size:50',
  },
  relation: {
    go: 'uint64',
    ts: 'number',
    sql: 'BIGINT UNSIGNED',
    gorm: '',
  },
};

/**
 * 获取 Go 类型
 */
export function getGoType(fieldType: FieldType): string {
  return TYPE_MAPPING[fieldType].go;
}

/**
 * 获取 TypeScript 类型
 */
export function getTSType(fieldType: FieldType): string {
  return TYPE_MAPPING[fieldType].ts;
}

/**
 * 获取 SQL 数据类型
 */
export function getSQLType(fieldType: FieldType): string {
  return TYPE_MAPPING[fieldType].sql;
}

/**
 * 获取 GORM 标签
 */
export function getGORMTag(fieldType: FieldType, extra?: string): string {
  const base = TYPE_MAPPING[fieldType].gorm || '';
  if (extra) {
    return base ? `${base};${extra}` : extra;
  }
  return base;
}

/**
 * 生成完整的 GORM 标签字符串
 *
 * 示例: `gorm:"primaryKey;autoIncrement" json:"id"`
 */
export function generateStructTags(
  fieldName: string,
  fieldType: FieldType,
  options?: {
    primaryKey?: boolean;
    autoIncrement?: boolean;
    unique?: boolean;
    index?: boolean;
    notNull?: boolean;
    defaultValue?: string;
    jsonOmit?: boolean; // json:"-"
    jsonName?: string; // 自定义JSON字段名
  },
): string {
  const gormTags: string[] = [];
  const jsonTags: string[] = [];

  // GORM 标签
  if (options?.primaryKey) gormTags.push('primaryKey');
  if (options?.autoIncrement) gormTags.push('autoIncrement');
  if (options?.unique) gormTags.push('uniqueIndex');
  if (options?.index) gormTags.push('index');
  if (options?.notNull) gormTags.push('not null');
  if (options?.defaultValue) gormTags.push(`default:${options.defaultValue}`);

  const typeMapping = TYPE_MAPPING[fieldType];
  if (typeMapping.gorm) {
    gormTags.push(typeMapping.gorm);
  }

  // JSON 标签
  if (options?.jsonOmit) {
    jsonTags.push('-');
  } else {
    jsonTags.push(options?.jsonName || fieldName);
  }

  // 组装标签
  const parts: string[] = [];
  if (gormTags.length > 0) {
    parts.push(`gorm:"${gormTags.join(';')}"`);
  }
  if (jsonTags.length > 0) {
    parts.push(`json:"${jsonTags.join(',')}"`);
  }

  return parts.join(' ');
}

/**
 * Go 类型导入包映射
 *
 * 某些类型需要额外的 import
 */
export const GO_TYPE_IMPORTS: Record<string, string> = {
  'time.Time': '"time"',
  'gorm.DeletedAt': '"gorm.io/gorm"',
};

/**
 * 获取生成 Model 所需的 import 列表
 */
export function getRequiredImports(fields: Array<{ type: FieldType }>): Set<string> {
  const imports = new Set<string>();

  // 默认导入
  imports.add('"time"');
  imports.add('"gorm.io/gorm"');

  // 根据字段类型添加
  fields.forEach((field) => {
    const goType = TYPE_MAPPING[field.type].go;
    if (GO_TYPE_IMPORTS[goType]) {
      imports.add(GO_TYPE_IMPORTS[goType]);
    }
  });

  return imports;
}

/**
 * TypeScript 类型工具函数
 */
export const TS_TYPE_UTILS = {
  /** 可选类型标记 */
  optional: (type: string) => `${type} | undefined`,
  /** 可空类型标记 */
  nullable: (type: string) => `${type} | null`,
  /** 数组类型 */
  array: (type: string) => `${type}[]`,
  /** 分页响应类型 */
  paginated: (type: string) => `{
  items: ${type}[];
  total: number;
  page: number;
  pageSize: number;
}`,
};
