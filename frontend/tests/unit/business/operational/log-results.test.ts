import { describe, expect, it } from 'vitest';
import { boundedRawLogRows, summarizeLogRows, type LogRow } from '../../../../src/modules/business/observability/log-results';
const row = (id: string, timestamp: string, labels: Record<string, string> = { app: 'one' }, message = 'sample'): LogRow => ({ id, timestamp, labels, message });
describe('bounded returned Loki result views', () => {
  it('counts only returned rows and preserves UTC time order with distinct field values', () => {
    const result = summarizeLogRows([row('a', '1794009600000000000'), row('b', '1794009600000000001', { app: 'two' }), row('c', '1794009720000000000')]);
    expect(result.trend).toEqual([{ time: '2026-11-07T00:00:00.000Z', count: 2 }, { time: '2026-11-07T00:02:00.000Z', count: 1 }]);
    expect(result.fields).toEqual([{ name: 'app', rows: 3, distinct: 2 }]);
    expect(result.invalidTimes).toBe(0);
  });
  it('reports invalid timestamps rather than inventing epoch buckets', () => {
    const result=summarizeLogRows([row('a','bad'),row('b','NaN')]);
    expect(result.trend).toEqual([]); expect(result.invalidTimes).toBe(2);
  });
  it('limits field expansion on wide labels', () => {
    const labels=Object.fromEntries(Array.from({length:150},(_,i)=>[`field${i}`,'value']));
    const result=summarizeLogRows([row('a','1794009600000000000',labels)]);
    expect(result.fields).toHaveLength(100); expect(result.fieldsLimited).toBe(true);
  });
  it('keeps bounded raw output valid JSON without splitting Unicode rows', () => {
    const input=[row('a','1794009600000000000',{app:'one'},'汉字'.repeat(12)),row('b','1794009600000000000',{app:'one'},'x'.repeat(200))];
    const result=boundedRawLogRows(input,200);
    expect(new TextEncoder().encode(result.text).byteLength).toBeLessThanOrEqual(200);
    expect(JSON.parse(result.text)).toEqual([input[0]]); expect(result.omittedRows).toBe(1);
  });
});
