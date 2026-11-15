import React from 'react';
import { Button, Input, Select, Space, Tag } from '@arco-design/web-react';
import { validateConditionAst } from './conditionAst';
import type {
  ConditionAstNode,
  ConditionFieldOption,
  ConditionRule,
  OperationalDataState,
} from './types';
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
  onChange?: (value: ConditionAstNode) => void;
  onValidationChange?: (invalidIds: string[]) => void;
  editorLabels?: {
    field: string;
    operator: string;
    value: string;
    remove: React.ReactNode;
    addGroup: React.ReactNode;
    combinator: string;
  };
  renderValueEditor?: (
    rule: ConditionRule,
    onValueChange: (value: unknown) => void,
  ) => React.ReactNode;
}

function formatConditionValue(value: unknown, unserializableValue: string): string;
function formatConditionValue(
  value: unknown,
  unserializableValue: React.ReactNode,
): React.ReactNode;
function formatConditionValue(
  value: unknown,
  unserializableValue: React.ReactNode,
): React.ReactNode {
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
  if (React.isValidElement(value)) {
    return unserializableValue;
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
      <span>
        {React.isValidElement(node.value)
          ? node.value
          : formatConditionValue(node.value, labels.unserializableValue)}
      </span>
      {invalidIds.has(node.id) ? <span>{labels.invalidField}</span> : null}
    </div>
  );
}
export default function ConditionBuilder({
  title,
  value,
  fields,
  labels,
  onChange,
  onValidationChange,
  editorLabels,
  renderValueEditor,
  state = 'ready',
}: Readonly<ConditionBuilderProps>) {
  const titleId = React.useId();
  // Equivalent inline field arrays must not produce a fresh validation value
  // that repeatedly updates the controlled parent from this effect.
  const validationSignature = JSON.stringify(validateConditionAst(value, fields));
  const validation = React.useMemo(
    () => JSON.parse(validationSignature) as string[],
    [validationSignature],
  );
  const invalidIds = new Set(validation);
  const validationCallback = React.useRef(onValidationChange);
  React.useEffect(() => {
    validationCallback.current = onValidationChange;
  }, [onValidationChange]);
  React.useEffect(() => {
    validationCallback.current?.(validation);
  }, [validation]);
  const editable = Boolean(onChange && editorLabels && state === 'ready');
  const changeNode = (
    id: string,
    transform: (node: ConditionAstNode) => ConditionAstNode | null,
  ) => {
    const visit = (node: ConditionAstNode): ConditionAstNode | null => {
      if (node.id === id) return transform(node);
      return node.type === 'rule'
        ? node
        : {
            ...node,
            children: node.children
              .map(visit)
              .filter((child): child is ConditionAstNode => child !== null),
          };
    };
    const next = visit(value);
    if (next) onChange?.(next);
  };
  const newRule = (): ConditionRule => {
    const field = fields.find((item) => !item.disabled && item.operators.length > 0);
    return {
      type: 'rule',
      id: crypto.randomUUID(),
      field: field?.key || '',
      operator: field?.operators[0] || '',
      value: '',
    };
  };
  const addRule = (id: string) =>
    changeNode(id, (node) =>
      node.type === 'group'
        ? { ...node, children: [...node.children, newRule()] }
        : {
            type: 'group',
            id: crypto.randomUUID(),
            combinator: 'and',
            children: [node, newRule()],
          },
    );
  const renderEditable = (node: ConditionAstNode, root = false): React.ReactNode => {
    if (!editorLabels) return null;
    const remove = !root && (
      <Button onClick={() => changeNode(node.id, () => null)}>{editorLabels.remove}</Button>
    );
    if (node.type === 'group')
      return (
        <div className="condition-builder__node" key={node.id}>
          <Space wrap>
            <Select
              value={node.combinator}
              aria-label={editorLabels.combinator}
              onChange={(combinator: 'and' | 'or') =>
                changeNode(node.id, (current) =>
                  current.type === 'group' ? { ...current, combinator } : current,
                )
              }
              options={[
                { value: 'and', label: labels.and },
                { value: 'or', label: labels.or },
              ]}
            />
            <Button onClick={() => addRule(node.id)}>{labels.addRule}</Button>
            <Button
              onClick={() =>
                changeNode(node.id, (current) =>
                  current.type === 'group'
                    ? {
                        ...current,
                        children: [
                          ...current.children,
                          {
                            type: 'group',
                            id: crypto.randomUUID(),
                            combinator: 'and',
                            children: [newRule()],
                          },
                        ],
                      }
                    : current,
                )
              }
            >
              {editorLabels.addGroup}
            </Button>
            {remove}
          </Space>
          <div className="condition-builder__tree">
            {node.children.map((child) => renderEditable(child))}
          </div>
        </div>
      );
    const field = fields.find((item) => item.key === node.field);
    const changeValue = (next: unknown) =>
      changeNode(node.id, (current) =>
        current.type === 'rule' ? { ...current, value: next } : current,
      );
    const renderRuleValue = () => {
      if (renderValueEditor) return renderValueEditor(node, changeValue);
      if (React.isValidElement(node.value)) return <span>{labels.unserializableValue}</span>;
      return (
        <Input
          value={formatConditionValue(node.value, '')}
          aria-label={editorLabels.value}
          onChange={changeValue}
        />
      );
    };
    return (
      <div
        className={`condition-builder__node${invalidIds.has(node.id) ? ' condition-builder__node--invalid' : ''}`}
        key={node.id}
        aria-invalid={invalidIds.has(node.id)}
      >
        <Select
          value={node.field}
          aria-label={editorLabels.field}
          options={fields.map((item) => ({
            value: item.key,
            label: item.label,
            disabled: item.disabled,
          }))}
          onChange={(key: string) =>
            changeNode(node.id, (current) =>
              current.type === 'rule'
                ? {
                    ...current,
                    field: key,
                    operator: fields.find((item) => item.key === key)?.operators[0] || '',
                  }
                : current,
            )
          }
        />
        <Select
          value={node.operator}
          aria-label={editorLabels.operator}
          options={(field?.operators || []).map((operator) => ({
            value: operator,
            label: labels.operatorLabels[operator] ?? labels.operatorLabels.unknown,
          }))}
          onChange={(operator: string) =>
            changeNode(node.id, (current) =>
              current.type === 'rule' ? { ...current, operator } : current,
            )
          }
        />
        {renderRuleValue()}
        {invalidIds.has(node.id) && <span>{labels.invalidField}</span>}
        {remove}
      </div>
    );
  };
  return (
    <section className="operational-primitive condition-builder" aria-labelledby={titleId}>
      <div className="operational-primitive__header">
        <div className="operational-primitive__title" id={titleId}>
          {title}
        </div>
        <Button disabled={!editable} onClick={() => addRule(value.id)}>
          {labels.addRule}
        </Button>
      </div>
      {state !== 'ready' ? (
        <div className="operational-primitive__state">{labels[state]}</div>
      ) : (
        <div className="operational-primitive__body condition-builder__tree">
          {editable ? renderEditable(value, true) : renderNode(value, fields, invalidIds, labels)}
        </div>
      )}
    </section>
  );
}
