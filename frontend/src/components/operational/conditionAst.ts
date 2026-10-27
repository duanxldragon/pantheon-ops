import type { ConditionAstNode, ConditionFieldOption } from './types';

export function collectConditionNodeIds(node: ConditionAstNode): string[] {
  return node.type === 'rule'
    ? [node.id]
    : [node.id, ...node.children.flatMap(collectConditionNodeIds)];
}

export function validateConditionAst(
  node: ConditionAstNode,
  fields: readonly ConditionFieldOption[],
): string[] {
  if (node.type === 'group') {
    return node.children.flatMap((child) => validateConditionAst(child, fields));
  }
  const field = fields.find((candidate) => candidate.key === node.field);
  return !field || field.disabled || !field.operators.includes(node.operator) ? [node.id] : [];
}
