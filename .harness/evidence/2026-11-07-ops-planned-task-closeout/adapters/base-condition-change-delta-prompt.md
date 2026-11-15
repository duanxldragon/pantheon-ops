Read-only CODE review, no tools. Max 160 words. Proved P1/P2 plus minimal fixes; otherwise APPROVE supplied scope only. State omitted source.
Review controlled edit logic only. editable is Boolean(onChange && editorLabels && state===ready); read-only remains default. validateConditionAst is existing validator; value input supplied by provider when needed. No compiler/database behavior.
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
