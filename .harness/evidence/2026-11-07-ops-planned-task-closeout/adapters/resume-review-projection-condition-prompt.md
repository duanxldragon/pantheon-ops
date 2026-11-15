Independent CODE review, read-only source-only. Tools disabled. Report only proved P0/P1/P2 defects, concrete source line and minimal fix. Max 450 words. End with verdict APPROVE or CHANGES_REQUESTED for supplied source only, plus explicit scope limits. No production, complete project, or release acceptance. Generic shared source belongs to Base, business semantics to Ops. Unknown runtime dependencies must be flagged conditional rather than assumed defective.

FILE pantheon-base/frontend/src/components/operational/ConditionBuilder.tsx
1: import React from 'react';
2: import { Button, Input, Select, Space, Tag } from '@arco-design/web-react';
3: import { validateConditionAst } from './conditionAst';
4: import type { ConditionAstNode, ConditionFieldOption, ConditionRule, OperationalDataState } from './types';
5: import './operational.css';
6:
7: export interface ConditionBuilderProps {
8: title: React.ReactNode;
9: value: ConditionAstNode;
10: fields: ConditionFieldOption[];
11: labels: {
12: and: React.ReactNode;
13: or: React.ReactNode;
14: operatorLabels: Record<string, React.ReactNode> & { unknown: React.ReactNode };
15: addRule: React.ReactNode;
16: invalidField: React.ReactNode;
17: unserializableValue: React.ReactNode;
18: } & Record<OperationalDataState, React.ReactNode>;
19: state?: OperationalDataState;
20: onChange?: (value: ConditionAstNode) => void;
21: onValidationChange?: (invalidIds: string[]) => void;
22: editorLabels?: { field: string; operator: string; value: string; remove: React.ReactNode; addGroup: React.ReactNode; combinator: string };
23: renderValueEditor?: (rule: ConditionRule, onValueChange: (value: unknown) => void) => React.ReactNode;
24: }
25:
26: function formatConditionValue(value: unknown, unserializableValue: React.ReactNode): React.ReactNode {
27: if (value === null || value === undefined) {
28: return '';
29: }
30: if (typeof value === 'string') {
31: return value;
32: }
33: if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
34: return value.toString();
35: }
36: if (typeof value === 'symbol') {
37: return value.description ?? unserializableValue;
38: }
39: if (typeof value === 'function') {
40: return value.name || unserializableValue;
41: }
42: try {
43: return JSON.stringify(value) ?? unserializableValue;
44: } catch {
45: return unserializableValue;
46: }
47: }
48:
49: function renderNode(
50: node: ConditionAstNode,
51: fields: ConditionFieldOption[],
52: invalidIds: Set<string>,
53: labels: ConditionBuilderProps['labels'],
54: ): React.ReactNode {
55: if (node.type === 'group')
56: return (
57: <div className="condition-builder__node" key={node.id}>
58: <Tag>{node.combinator === 'and' ? labels.and : labels.or}</Tag>
59: <div className="condition-builder__tree">
60: {node.children.map((child) => renderNode(child, fields, invalidIds, labels))}
61: </div>
62: </div>
63: );
64: return (
65: <div
66: className={
67: invalidIds.has(node.id)
68: ? 'condition-builder__node condition-builder__node--invalid'
69: : 'condition-builder__node'
70: }
71: key={node.id}
72: aria-invalid={invalidIds.has(node.id)}
73: >
74: <Select
75: value={node.field}
76: disabled
77: options={fields.map((field) => ({ label: field.label, value: field.key }))}
78: />
79: <Tag>{labels.operatorLabels[node.operator] ?? labels.operatorLabels.unknown}</Tag>
80: <span>{formatConditionValue(node.value, labels.unserializableValue)}</span>
81: {invalidIds.has(node.id) ? <span>{labels.invalidField}</span> : null}
82: </div>
83: );
84: }
85: export default function ConditionBuilder({
86: title,
87: value,
88: fields,
89: labels,
90: onChange,
91: onValidationChange,
92: editorLabels,
93: renderValueEditor,
94: state = 'ready',
95: }: Readonly<ConditionBuilderProps>) {
96: const invalidIds = new Set(validateConditionAst(value, fields));
97: const editable = Boolean(onChange && editorLabels && state === 'ready');
98: const changeNode = (id: string, transform: (node: ConditionAstNode) => ConditionAstNode | null) => {
99: const visit = (node: ConditionAstNode): ConditionAstNode | null => {
100: if (node.id === id) return transform(node);
101: return node.type === 'rule' ? node : { ...node, children: node.children.map(visit).filter((child): child is ConditionAstNode => child !== null) };
102: };
103: const next = visit(value);
104: if (next) { onChange?.(next); onValidationChange?.(validateConditionAst(next, fields)); }
105: };
106: const newRule = (): ConditionRule => {
107: const field = fields.find((item) => !item.disabled && item.operators.length > 0);
108: return { type: 'rule', id: crypto.randomUUID(), field: field?.key || '', operator: field?.operators[0] || '', value: '' };
109: };
110: const addRule = (id: string) => changeNode(id, (node) => node.type === 'group' ? { ...node, children: [...node.children, newRule()] } : { type: 'group', id: crypto.randomUUID(), combinator: 'and', children: [node, newRule()] });
111: const renderEditable = (node: ConditionAstNode, root = false): React.ReactNode => {
112: if (!editorLabels) return null;
113: const remove = !root && <Button onClick={() => changeNode(node.id, () => null)}>{editorLabels.remove}</Button>;
114: if (node.type === 'group') return <div className="condition-builder__node" key={node.id}>
115: <Space wrap><Select value={node.combinator} aria-label={editorLabels.combinator} onChange={(combinator: 'and' | 'or') => changeNode(node.id, (current) => current.type === 'group' ? { ...current, combinator } : current)} options={[{ value: 'and', label: labels.and }, { value: 'or', label: labels.or }]} />
116: <Button onClick={() => addRule(node.id)}>{labels.addRule}</Button>
117: <Button onClick={() => changeNode(node.id, (current) => current.type === 'group' ? { ...current, children: [...current.children, { type: 'group', id: crypto.randomUUID(), combinator: 'and', children: [newRule()] }] } : current)}>{editorLabels.addGroup}</Button>{remove}</Space>
118: <div className="condition-builder__tree">{node.children.map((child) => renderEditable(child))}</div>
119: </div>;
120: const field = fields.find((item) => item.key === node.field);
121: const changeValue = (next: unknown) => changeNode(node.id, (current) => current.type === 'rule' ? { ...current, value: next } : current);
122: return <div className={`condition-builder__node${invalidIds.has(node.id) ? ' condition-builder__node--invalid' : ''}`} key={node.id} aria-invalid={invalidIds.has(node.id)}>
123: <Select value={node.field} aria-label={editorLabels.field} options={fields.map((item) => ({ value: item.key, label: item.label, disabled: item.disabled }))} onChange={(key: string) => changeNode(node.id, (current) => current.type === 'rule' ? { ...current, field: key, operator: fields.find((item) => item.key === key)?.operators[0] || '' } : current)} />
124: <Select value={node.operator} aria-label={editorLabels.operator} options={(field?.operators || []).map((operator) => ({ value: operator, label: labels.operatorLabels[operator] ?? labels.operatorLabels.unknown }))} onChange={(operator: string) => changeNode(node.id, (current) => current.type === 'rule' ? { ...current, operator } : current)} />
125: {renderValueEditor ? renderValueEditor(node, changeValue) : <Input value={String(formatConditionValue(node.value, labels.unserializableValue))} aria-label={editorLabels.value} onChange={changeValue} />}
126: {invalidIds.has(node.id) && <span>{labels.invalidField}</span>}{remove}
127: </div>;
128: };
129: return (
130: <section className="operational-primitive condition-builder" aria-label={String(title)}>
131: <div className="operational-primitive__header">
132: <div className="operational-primitive__title">{title}</div>
133: <Button disabled={!editable} onClick={() => addRule(value.id)}>{labels.addRule}</Button>
134: </div>
135: {state !== 'ready' ? (
136: <div className="operational-primitive__state">{labels[state]}</div>
137: ) : (
138: <div className="operational-primitive__body condition-builder__tree">
139: {editable ? renderEditable(value, true) : renderNode(value, fields, invalidIds, labels)}
140: </div>
141: )}
142: </section>
143: );
144: }

FILE pantheon-base/config/foundation-consumer-artifacts.json
1: {
2: "schemaVersion": 1,
3: "sourceOwner": "pantheon-base",
4: "reviewLegacyRegistry": {
5: "path": "config/review-loop-legacy-artifacts.json",
6: "artifactRoot": ".harness/archive"
7: }
8: }

FILE pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs
568: export function projectConsumerReviewRegistry(baseRoot, targetRoot, manifest, baseCommit) {
569: const descriptorPath = path.join(baseRoot, 'config/foundation-consumer-artifacts.json');
570: if (!fs.existsSync(descriptorPath)) return [];
571: const descriptor = JSON.parse(fs.readFileSync(descriptorPath, 'utf8'));
572: const contract = descriptor.reviewLegacyRegistry;
573: if (descriptor.schemaVersion !== 1 || descriptor.sourceOwner !== 'pantheon-base' ||
574: contract?.path !== 'config/review-loop-legacy-artifacts.json' ||
575: contract?.artifactRoot !== '.harness/archive') {
576: throw new Error('Invalid Base consumer review packaging contract');
577: }
578: const sourcePath = path.join(baseRoot, contract.path);
579: const sourceBytes = fs.readFileSync(sourcePath);
580: const source = JSON.parse(sourceBytes.toString('utf8'));
581: if (!Array.isArray(source.entries)) throw new Error('Base legacy review entries must be an array');
582: const excluded = [...(manifest.replaceRoots ?? []), ...(manifest.baseExcludedPaths ?? [])].map(repoPath);
583: const retained = [];
584: const external = [];
585: for (const entry of source.entries) {
586: if (typeof entry !== 'string' || !entry.startsWith('.harness/archive/') ||
587: entry.includes('..') || entry.includes('\\') || !entry.endsWith('/review.md')) {
588: throw new Error('Base legacy entry must be a canonical archive review');
589: }
590: if (!fs.existsSync(path.join(baseRoot, entry))) throw new Error(`Stale Base legacy entry: ${entry}`);
591: if (excluded.some((root) => isWithin(entry, root))) external.push(entry);
592: else {
593: if (!fs.existsSync(path.join(targetRoot, entry))) throw new Error(`Missing retained Base review: ${entry}`);
594: retained.push(entry);
595: }
596: }
597: if (external.length === 0) return [];
598: const targetPath = path.join(targetRoot, contract.path);
599: const projection = {
600: ...source,
601: description: 'Consumer projection of the Base archive registry. Excluded Base artifacts stay recorded below; all current consumer reviews remain subject to strict review validation.',
602: entries: retained,
603: projection: { sourceOwner: descriptor.sourceOwner, baseCommit,
604: sourceConfigSha256: crypto.createHash('sha256').update(sourceBytes).digest('hex'),
605: externalBaseArtifacts: external },
606: };
607: fs.mkdirSync(path.dirname(targetPath), { recursive: true });
608: fs.writeFileSync(targetPath, `${JSON.stringify(projection, null, 2)}\n`, 'utf8');
609: return [contract.path];
610: }
611:
612: export function assembleConsumerTree({ opsRoot = defaultOpsRoot, snapshot, targetRoot, manifestPath } = {}) {
613: const resolvedManifest = manifestPath ?? path.join(opsRoot, 'business-overlay.json');
614: const manifest = JSON.parse(fs.readFileSync(resolvedManifest, 'utf8'));
615: assertSafeManifest(manifest);
616:
617: const resolvedBase = snapshot?.baseRoot;
618: if (!resolvedBase || !snapshot.baseCommit) throw new Error('Verified release snapshot is required');
619: const resolvedTarget = path.resolve(opsRoot, targetRoot ?? '.tmp/business-overlay-rebuild');
620: const targetContainsSource = [path.resolve(opsRoot), path.resolve(resolvedBase)].some((source) => source === resolvedTarget || source.startsWith(resolvedTarget + path.sep));
