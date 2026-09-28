import React from 'react';
import { Button, Select, Tag } from '@arco-design/web-react';
import { validateConditionAst } from './conditionAst';
import type { ConditionAstNode, ConditionFieldOption, OperationalDataState } from './types';
import './operational.css';

export interface ConditionBuilderProps {
  title: React.ReactNode;
  value: ConditionAstNode;
  fields: ConditionFieldOption[];
  labels: {
    and: React.ReactNode;
    or: React.ReactNode;
    operatorLabels: Record<string, React.ReactNode> & { unknown: React.ReactNode };
    addRule: React.ReactNode;
    invalidField: React.ReactNode;
    unserializableValue: React.ReactNode;
  } & Record<OperationalDataState, React.ReactNode>;
  state?: OperationalDataState;
}

function formatConditionValue(value: unknown, unserializableValue: React.ReactNode): React.ReactNode {
  if (value === null || value === undefined) {
    return '';
  }
  if (typeof value === 'string') {
    return value;
  }
  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return value.toString();
  }
  if (typeof value === 'symbol') {
    return value.description ?? unserializableValue;
  }
  if (typeof value === 'function') {
    return value.name || unserializableValue;
  }
  try {
    return JSON.stringify(value) ?? unserializableValue;
  } catch {
    return unserializableValue;
  }
}

function renderNode(
  node: ConditionAstNode,
  fields: ConditionFieldOption[],
  invalidIds: Set<string>,
  labels: ConditionBuilderProps['labels'],
): React.ReactNode {
  if (node.type === 'group')
    return (
      <div className="condition-builder__node" key={node.id}>
        <Tag>{node.combinator === 'and' ? labels.and : labels.or}</Tag>
        <div className="condition-builder__tree">
          {node.children.map((child) => renderNode(child, fields, invalidIds, labels))}
        </div>
      </div>
    );
  return (
    <div
      className={
        invalidIds.has(node.id)
          ? 'condition-builder__node condition-builder__node--invalid'
          : 'condition-builder__node'
      }
      key={node.id}
      aria-invalid={invalidIds.has(node.id)}
    >
      <Select
        value={node.field}
        disabled
        options={fields.map((field) => ({ label: field.label, value: field.key }))}
      />
      <Tag>{labels.operatorLabels[node.operator] ?? labels.operatorLabels.unknown}</Tag>
      <span>{formatConditionValue(node.value, labels.unserializableValue)}</span>
      {invalidIds.has(node.id) ? <span>{labels.invalidField}</span> : null}
    </div>
  );
}
export default function ConditionBuilder({
  title,
  value,
  fields,
  labels,
  state = 'ready',
}: Readonly<ConditionBuilderProps>) {
  const invalidIds = new Set(validateConditionAst(value, fields));
  return (
    <section className="operational-primitive condition-builder" aria-label={String(title)}>
      <div className="operational-primitive__header">
        <div className="operational-primitive__title">{title}</div>
        <Button disabled>{labels.addRule}</Button>
      </div>
      {state !== 'ready' ? (
        <div className="operational-primitive__state">{labels[state]}</div>
      ) : (
        <div className="operational-primitive__body condition-builder__tree">
          {renderNode(value, fields, invalidIds, labels)}
        </div>
      )}
    </section>
  );
}
