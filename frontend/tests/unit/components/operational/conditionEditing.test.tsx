import { isValidElement, useState } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import ConditionBuilder, { type ConditionBuilderProps } from '../../../../src/components/operational/ConditionBuilder';
import ExecutionStepRail from '../../../../src/components/operational/ExecutionStepRail';
import type { ConditionAstNode } from '../../../../src/components/operational/types';
const states = { loading: 'loading', empty: 'empty', error: 'error', forbidden: 'forbidden', stale: 'stale', partial: 'partial', ready: 'ready' };
const labels = { ...states, and: 'and', or: 'or', operatorLabels: { eq: 'equals', unknown: 'unknown' }, addRule: 'add rule', invalidField: 'invalid', unserializableValue: 'unavailable' };
const editorLabels = { field: 'field', operator: 'operator', value: 'value', remove: 'remove', addGroup: 'add group', combinator: 'combinator' };
const fields = [{ key: 'status', label: 'status', operators: ['eq'] }];
afterEach(cleanup);
describe('controlled condition editing', () => {
 it.each(['object', 'array'] as const)('does not retrigger fresh %s parent state updates for equivalent inline field options', (shape) => {
  const ast: ConditionAstNode = { type: 'rule', id: 'one', field: 'missing', operator: 'eq', value: 'online' };
  const validation = vi.fn();
  function Fixture() {
   const [invalid, setInvalid] = useState<string[] | { ids: string[] }>([]);
   return <><ConditionBuilder title="conditions" value={ast} fields={[{ ...fields[0], operators: [...fields[0].operators] }]} labels={labels} onValidationChange={(ids) => {
    validation(ids);
    if (validation.mock.calls.length > 3) throw new Error('Repeated equivalent validation updates');
    setInvalid(shape === 'array' ? [...ids] : { ids });
   }} /><span data-testid="invalid-count">{Array.isArray(invalid) ? invalid.length : invalid.ids.length}</span></>;
  }
  render(<Fixture />);
  expect(screen.getByTestId('invalid-count').textContent).toBe('1');
  expect(validation).toHaveBeenCalledExactlyOnceWith(['one']);
 });
 it('uses the latest callback only when semantic validation changes', () => {
  const ast: ConditionAstNode = { type: 'rule', id: 'one', field: 'missing', operator: 'eq', value: 'online' };
  const first = vi.fn(); const latest = vi.fn();
  const { rerender } = render(<ConditionBuilder title="conditions" value={ast} fields={fields} labels={labels} onValidationChange={first} />);
  expect(first).toHaveBeenCalledExactlyOnceWith(['one']);
  rerender(<ConditionBuilder title="conditions" value={{ ...ast, value: 'offline' }} fields={[{ ...fields[0], operators: ['eq'] }]} labels={labels} onValidationChange={latest} />);
  expect(latest).not.toHaveBeenCalled();
  rerender(<ConditionBuilder title="conditions" value={{ ...ast, field: 'status' }} fields={fields} labels={labels} onValidationChange={latest} />);
  expect(latest).toHaveBeenCalledExactlyOnceWith([]);
  expect(first).toHaveBeenCalledOnce();
 });
 it('reports validation on mount and externally changed values and fields', () => {
  const ast: ConditionAstNode = { type: 'rule', id: 'one', field: 'missing', operator: 'eq', value: 'online' };
  const validation = vi.fn();
  const { rerender } = render(<ConditionBuilder title="conditions" value={ast} fields={fields} labels={labels} onValidationChange={validation} />);
  expect(validation).toHaveBeenLastCalledWith(['one']);
  const valid: ConditionAstNode = { ...ast, field: 'status' };
  rerender(<ConditionBuilder title="conditions" value={valid} fields={fields} labels={labels} onValidationChange={validation} />);
  expect(validation).toHaveBeenLastCalledWith([]);
  rerender(<ConditionBuilder title="conditions" value={valid} fields={[]} labels={labels} onValidationChange={validation} />);
  expect(validation).toHaveBeenLastCalledWith(['one']);
 });
 it('keeps validation tied to the rendered controlled value when an edit is rejected', () => {
  const ast: ConditionAstNode = { type: 'rule', id: 'one', field: 'status', operator: 'eq', value: 'online' };
  const validation = vi.fn();
  const change = vi.fn();
  render(<ConditionBuilder title="conditions" value={ast} fields={fields} labels={labels} editorLabels={editorLabels} onChange={change} onValidationChange={validation} />);
  expect(validation).toHaveBeenCalledWith([]);
  validation.mockClear();
  fireEvent.click(screen.getByText('add rule'));
  expect(change).toHaveBeenCalledOnce();
  expect(screen.getAllByLabelText('value')).toHaveLength(1);
  expect(validation).not.toHaveBeenCalled();
 });
 it('labels ReactNode titles and avoids coercing a ReactNode fallback into editable data', () => {
  const cyclic: { self?: unknown } = {}; cyclic.self = cyclic;
  const ast: ConditionAstNode = { type: 'rule', id: 'one', field: 'status', operator: 'eq', value: cyclic };
  const change = vi.fn();
  const nodeLabels = { ...labels, unserializableValue: <span>unavailable value</span> };
  const { rerender } = render(<ConditionBuilder title={<span>structured conditions</span>} value={ast} fields={fields} labels={nodeLabels} editorLabels={editorLabels} onChange={change} />);
  expect(screen.getByRole('region', { name: 'structured conditions' })).toBeTruthy();
  expect((screen.getByLabelText('value') as HTMLInputElement).value).toBe('');
  expect(change).not.toHaveBeenCalled();
  fireEvent.change(screen.getByLabelText('value'), { target: { value: 'replacement' } });
  expect(change).toHaveBeenCalledWith({ ...ast, value: 'replacement' });
  rerender(<ConditionBuilder title={<span>structured conditions</span>} value={ast} fields={fields} labels={nodeLabels} />);
  expect(screen.getByText('unavailable value')).toBeTruthy();
 });
 it('renders a real React element rule value in readonly mode', () => {
  const ast: ConditionAstNode = { type: 'rule', id: 'one', field: 'status', operator: 'eq', value: <strong data-testid="condition-node-value">online node</strong> };
  render(<ConditionBuilder title="conditions" value={ast} fields={fields} labels={labels} />);
  expect(screen.getByTestId('condition-node-value').tagName).toBe('STRONG');
  expect(screen.getByText('online node')).toBeTruthy();
 });
 it('keeps React element rule values out of the default editable input', () => {
  const ast: ConditionAstNode = { type: 'rule', id: 'one', field: 'status', operator: 'eq', value: <strong>online node</strong> };
  const change = vi.fn();
  const { container } = render(<ConditionBuilder title="conditions" value={ast} fields={fields} labels={labels} editorLabels={editorLabels} onChange={change} />);
  expect(screen.queryByLabelText('value')).toBeNull();
  expect(screen.getByText('unavailable')).toBeTruthy();
  expect(container.textContent).not.toContain('"props"');
  expect(change).not.toHaveBeenCalled();
 });
 it('lets the provider edit a React element rule value without serializing it', () => {
  const ast: ConditionAstNode = { type: 'rule', id: 'one', field: 'status', operator: 'eq', value: <strong>online node</strong> };
  const change = vi.fn();
  const editor = vi.fn<NonNullable<ConditionBuilderProps['renderValueEditor']>>((rule, onValueChange) => <><span data-testid="provider-value">{isValidElement(rule.value) ? rule.value : null}</span><button onClick={() => onValueChange('replacement')}>replace node</button></>);
  render(<ConditionBuilder title="conditions" value={ast} fields={fields} labels={labels} editorLabels={editorLabels} onChange={change} renderValueEditor={editor} />);
  expect(editor.mock.calls[0][0].value).toBe(ast.value);
  expect(screen.getByTestId('provider-value').textContent).toBe('online node');
  fireEvent.click(screen.getByRole('button', { name: 'replace node' }));
  expect(change).toHaveBeenCalledExactlyOnceWith({ ...ast, value: 'replacement' });
 });
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
 it('names each step rail using its rendered ReactNode title', () => {
  render(<><ExecutionStepRail title={<span>execution steps</span>} steps={[]} labels={{ ...states, attempts:'attempts',duration:'duration',jumpToStep:'jump' }} />
   <ExecutionStepRail title={<span>recovery steps</span>} steps={[]} labels={{ ...states, attempts:'attempts',duration:'duration',jumpToStep:'jump' }} /></>);
  const execution = screen.getByRole('region', { name: 'execution steps' });
  const recovery = screen.getByRole('region', { name: 'recovery steps' });
  expect(document.getElementById(execution.getAttribute('aria-labelledby')!)?.textContent).toBe('execution steps');
  expect(document.getElementById(recovery.getAttribute('aria-labelledby')!)?.textContent).toBe('recovery steps');
  expect(execution.getAttribute('aria-labelledby')).not.toBe(recovery.getAttribute('aria-labelledby'));
 });
 it('renders ReactNode attempts and duration labels beside their metric values', () => {
  render(<ExecutionStepRail title="steps" steps={[{ id:'one',title:<strong>Install</strong>,status:'running',attempts:2,durationMs:350 }]} labels={{ ...states, attempts:<em>tries</em>,duration:<em>elapsed</em>,jumpToStep:'jump' }} />);
  expect(screen.getByText('tries').tagName).toBe('EM');
  expect(screen.getByText('elapsed').tagName).toBe('EM');
  expect(screen.getByRole('button', { name: 'jump 1' }).textContent).toContain('tries: 2');
  expect(screen.getByRole('button', { name: 'jump 1' }).textContent).toContain('elapsed: 350ms');
  expect(screen.queryByText(/\[object Object\]/)).toBeNull();
 });
});
