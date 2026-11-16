export interface LogRow {
  id: string;
  timestamp: string;
  labels: Record<string, string>;
  message: string;
}

export function logTimestampMilliseconds(value: string) {
  if (/^\d{16,20}$/.test(value)) return Number(value.slice(0, -6));
  return Number.NaN;
}

// These are summaries of the bounded returned rows, never an estimate of all matching logs.
export function summarizeLogRows(rows: LogRow[]) {
  const times = rows.map((row) => logTimestampMilliseconds(row.timestamp)).filter(Number.isFinite);
  const first = times.length ? Math.min(...times) : 0;
  const last = times.length ? Math.max(...times) : 0;
  const bucketWidth = Math.max(60_000, Math.ceil((last - first + 1) / 60 / 60_000) * 60_000);
  const buckets = new Map<number, number>();
  for (const time of times) {
    const bucket = Math.floor((time - first) / bucketWidth) * bucketWidth + first;
    buckets.set(bucket, (buckets.get(bucket) ?? 0) + 1);
  }
  const fields = new Map<string, { rows: number; values: Set<string> }>();
  let fieldsLimited = false;
  for (const row of rows) {
    let seen = 0;
    for (const key in row.labels) {
      if (!Object.hasOwn(row.labels, key)) continue;
      if (seen++ >= 100) { fieldsLimited = true; break; }
      if (!fields.has(key) && fields.size >= 100) { fieldsLimited = true; continue; }
      const field = fields.get(key) ?? { rows: 0, values: new Set<string>() };
      field.rows++; field.values.add(row.labels[key]); fields.set(key, field);
    }
  }
  return {
    trend: [...buckets].sort(([a], [b]) => a - b).map(([time, count]) => ({ time: new Date(time).toISOString(), count })),
    invalidTimes: rows.length - times.length,
    fields: [...fields].map(([name, field]) => ({ name, rows: field.rows, distinct: field.values.size }))
      .sort((a, b) => b.rows - a.rows || a.name.localeCompare(b.name)),
    fieldsLimited,
  };
}

export function boundedRawLogRows(rows: LogRow[], maxBytes = 262_144) {
  const values: string[] = [];
  let bytes = 4;
  for (const row of rows) {
    const encoded = JSON.stringify(row);
    const size = new TextEncoder().encode(encoded).byteLength + 2;
    if (bytes + size > maxBytes) break;
    values.push(encoded); bytes += size;
  }
  return { text: `[\n${values.join(',\n')}\n]`, omittedRows: rows.length - values.length };
}

// The proxy limits response bytes. Validate the upstream envelope before interpreting
// an empty result, and apply the display budget after ordering all returned streams.
export function readLokiRows(response: unknown, limit: number) {
  const object = (value: unknown): value is Record<string, unknown> => typeof value === 'object' && value !== null && !Array.isArray(value);
  const invalid = () => { throw new Error('observability.logs.invalid_response'); };
  if (!object(response) || response.status !== 'success' || !object(response.data) || response.data.resultType !== 'streams' || !Array.isArray(response.data.result)) return invalid();
  if (response.warnings !== undefined && (!Array.isArray(response.warnings) || response.warnings.some((warning) => typeof warning !== 'string'))) return invalid();
  const rows: LogRow[] = [];
  for (const stream of response.data.result) {
    if (!object(stream) || !object(stream.stream) || !Array.isArray(stream.values) || Object.values(stream.stream).some((value) => typeof value !== 'string')) return invalid();
    for (const pair of stream.values) {
      if (!Array.isArray(pair) || pair.length !== 2 || typeof pair[0] !== 'string' || typeof pair[1] !== 'string') return invalid();
      rows.push({ id: `${pair[0]}-${rows.length}`, timestamp: pair[0], labels: stream.stream as Record<string, string>, message: pair[1] });
    }
  }
  rows.sort((left, right) => right.timestamp.localeCompare(left.timestamp));
  return { rows: rows.slice(0, limit), omittedRows: Math.max(0, rows.length - limit), atLimit: rows.length >= limit, warningCount: Array.isArray(response.warnings) ? response.warnings.length : 0 };
}
