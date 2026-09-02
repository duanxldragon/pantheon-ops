import type { LabelSchemaRow } from './api';

export function findLabelSchemaByKey(labelSchemas: LabelSchemaRow[], key?: string) {
  if (!key) return undefined;
  return labelSchemas.find((schema) => schema.key === key);
}

export function labelValueOptions(labelSchemas: LabelSchemaRow[], key?: string) {
  return findLabelSchemaByKey(labelSchemas, key)?.options || [];
}

export function isFreeValueLabel(labelSchemas: LabelSchemaRow[], key?: string) {
  const schema = findLabelSchemaByKey(labelSchemas, key);
  return !schema || schema.valueMode === 'free';
}
