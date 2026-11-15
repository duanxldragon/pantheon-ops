Read-only CODE review; use no tools. Supplied source only. Prove P1/P2 defects and minimal fix, otherwise APPROVE this bounded scope. Max 220 words; state omitted scope.

FILE pantheon-base\frontend\src\components\operational\ConditionBuilder.tsx
import React from 'react';
import { Button, Input, Select, Space, Tag } from '@arco-design/web-react';
import { validateConditionAst } from './conditionAst';
import type { ConditionAstNode, ConditionFieldOption, ConditionRule, OperationalDataState } from './types';
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
editorLabels?: { field: string; operator: string; value: string; remove: React.ReactNode; addGroup: React.ReactNode; combinator: string };
renderValueEditor?: (rule: ConditionRule, onValueChange: (value: unknown) => void) => React.ReactNode;
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
aria-invalid={invalidIds.has(node.id)} >
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
onChange,
onValidationChange,
editorLabels,
renderValueEditor,
state = 'ready',
}: Readonly<ConditionBuilderProps>) {
const invalidIds = new Set(validateConditionAst(value, fields));
const editable = Boolean(onChange && editorLabels && state === 'ready');
const changeNode = (id: string, transform: (node: ConditionAstNode) => ConditionAstNode | null) => {
const visit = (node: ConditionAstNode): ConditionAstNode | null => {
if (node.id === id) return transform(node);
return node.type === 'rule' ? node : { ...node, children: node.children.map(visit).filter((child): child is ConditionAstNode => child !== null) };
};
const next = visit(value);
if (next) { onChange?.(next); onValidationChange?.(validateConditionAst(next, fields)); }
};
const newRule = (): ConditionRule => {
const field = fields.find((item) => !item.disabled && item.operators.length > 0);
return { type: 'rule', id: crypto.randomUUID(), field: field?.key || '', operator: field?.operators[0] || '', value: '' };
};
const addRule = (id: string) => changeNode(id, (node) => node.type === 'group' ? { ...node, children: [...node.children, newRule()] } : { type: 'group', id: crypto.randomUUID(), combinator: 'and', children: [node, newRule()] });
const renderEditable = (node: ConditionAstNode, root = false): React.ReactNode => {
if (!editorLabels) return null;
const remove = !root && <Button onClick={() => changeNode(node.id, () => null)}>{editorLabels.remove}</Button>;
if (node.type === 'group') return <div className="condition-builder__node" key={node.id}>
<Space wrap><Select value={node.combinator} aria-label={editorLabels.combinator} onChange={(combinator: 'and' | 'or') => changeNode(node.id, (current) => current.type === 'group' ? { ...current, combinator } : current)} options={[{ value: 'and', label: labels.and }, { value: 'or', label: labels.or }]} />
<Button onClick={() => addRule(node.id)}>{labels.addRule}</Button>
<Button onClick={() => changeNode(node.id, (current) => current.type === 'group' ? { ...current, children: [...current.children, { type: 'group', id: crypto.randomUUID(), combinator: 'and', children: [newRule()] }] } : current)}>{editorLabels.addGroup}</Button>{remove}</Space>
<div className="condition-builder__tree">{node.children.map((child) => renderEditable(child))}</div>
</div>;
const field = fields.find((item) => item.key === node.field);
const changeValue = (next: unknown) => changeNode(node.id, (current) => current.type === 'rule' ? { ...current, value: next } : current);
return <div className={`condition-builder__node${invalidIds.has(node.id) ? ' condition-builder__node--invalid' : ''}`} key={node.id} aria-invalid={invalidIds.has(node.id)}>
<Select value={node.field} aria-label={editorLabels.field} options={fields.map((item) => ({ value: item.key, label: item.label, disabled: item.disabled }))} onChange={(key: string) => changeNode(node.id, (current) => current.type === 'rule' ? { ...current, field: key, operator: fields.find((item) => item.key === key)?.operators[0] || '' } : current)} />
<Select value={node.operator} aria-label={editorLabels.operator} options={(field?.operators || []).map((operator) => ({ value: operator, label: labels.operatorLabels[operator] ?? labels.operatorLabels.unknown }))} onChange={(operator: string) => changeNode(node.id, (current) => current.type === 'rule' ? { ...current, operator } : current)} />
{renderValueEditor ? renderValueEditor(node, changeValue) : <Input value={String(formatConditionValue(node.value, labels.unserializableValue))} aria-label={editorLabels.value} onChange={changeValue} />}
{invalidIds.has(node.id) && <span>{labels.invalidField}</span>}{remove}
</div>;
};
return (
<section className="operational-primitive condition-builder" aria-label={String(title)}>
<div className="operational-primitive__header">
<div className="operational-primitive__title">{title}</div>
<Button disabled={!editable} onClick={() => addRule(value.id)}>{labels.addRule}</Button>
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

FILE pantheon-base\frontend\src\components\operational\conditionAst.ts
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

FILE pantheon-base\frontend\tests\unit\components\operational\conditionEditing.test.tsx
import { useState } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import ConditionBuilder from '../../../../src/components/operational/ConditionBuilder';
import ExecutionStepRail from '../../../../src/components/operational/ExecutionStepRail';
import type { ConditionAstNode } from '../../../../src/components/operational/types';
const states = { loading: 'loading', empty: 'empty', error: 'error', forbidden: 'forbidden', stale: 'stale', partial: 'partial', ready: 'ready' };
const labels = { ...states, and: 'and', or: 'or', operatorLabels: { eq: 'equals', unknown: 'unknown' }, addRule: 'add rule', invalidField: 'invalid', unserializableValue: 'unavailable' };
const editorLabels = { field: 'field', operator: 'operator', value: 'value', remove: 'remove', addGroup: 'add group', combinator: 'combinator' };
const fields = [{ key: 'status', label: 'status', operators: ['eq'] }];
afterEach(cleanup);
describe('controlled condition editing', () => {
it('changes a value immutably and adds/removes rules using controlled state', () => {
const ast: ConditionAstNode = { type: 'group', id: 'root', combinator: 'and', children: [{ type: 'rule', id: 'one', field: 'status', operator: 'eq', value: 'online' }] };
Object.freeze(ast); Object.freeze(ast.children); Object.freeze(ast.children[0]);
const changed = vi.fn(); const validation = vi.fn();
function Fixture() { const [value,setValue] = useState(ast); return <ConditionBuilder title="conditions" value={value} fields={fields} labels={labels} editorLabels={editorLabels} onChange={(next) => { changed(next); setValue(next); }} onValidationChange={validation} />; }
render(<Fixture />);
fireEvent.change(screen.getByLabelText('value'), { target: { value: 'offline' } });
expect(ast.children[0]).toMatchObject({ value: 'online' });
expect(changed.mock.calls[0][0].children[0].value).toBe('offline');
expect(validation).toHaveBeenCalledWith([]);
fireEvent.click(screen.getAllByText('add rule')[0]);
expect(screen.getAllByLabelText('value')).toHaveLength(2);
fireEvent.click(screen.getAllByText('remove')[1]);
expect(screen.getAllByLabelText('value')).toHaveLength(1);
});
it('preserves readonly rendering and refuses editing forbidden state', () => {
const ast: ConditionAstNode = { type: 'rule', id: 'one', field: 'status', operator: 'eq', value: 'online' };
const change = vi.fn(); const { rerender } = render(<ConditionBuilder title="conditions" value={ast} fields={fields} labels={labels} />);
expect(screen.getByText('online')).toBeTruthy();expect(screen.getByText('add rule').closest('button')?.disabled).toBe(true);
rerender(<ConditionBuilder title="conditions" value={ast} fields={fields} labels={labels} editorLabels={editorLabels} onChange={change} state="forbidden" />);
fireEvent.click(screen.getByText('add rule'));expect(change).not.toHaveBeenCalled();expect(screen.queryByLabelText('value')).toBeNull();
});
it('renders the caller supplied status label on the step rail', () => {
render(<ExecutionStepRail title="steps" steps={[{ id:'one',title:'Install',status:'running' }]} labels={{ ...states, attempts:'attempts',duration:'duration',jumpToStep:'jump' }} statusLabels={{ running:'正在执行' }} />);
expect(screen.getByText('正在执行')).toBeTruthy();expect(screen.queryByText('running')).toBeNull();
});
});
