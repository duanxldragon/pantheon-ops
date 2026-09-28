import type { TaskLogChunk } from './types';

export const DEFAULT_TASK_LOG_WINDOW_SIZE = 500;

export function mergeTaskLogChunks(
  current: readonly TaskLogChunk[],
  incoming: readonly TaskLogChunk[],
  limit = DEFAULT_TASK_LOG_WINDOW_SIZE,
) {
  const bySequence = new Map<number, TaskLogChunk>();
  [...current, ...incoming].forEach((chunk) => {
    if (Number.isSafeInteger(chunk.sequence) && chunk.sequence >= 0) {
      bySequence.set(chunk.sequence, chunk);
    }
  });
  return [...bySequence.values()]
    .sort((left, right) => left.sequence - right.sequence)
    .slice(Math.max(0, bySequence.size - Math.max(1, limit)));
}

export function filterTaskLogChunks(
  chunks: readonly TaskLogChunk[],
  filters: { keyword?: string; levels?: readonly string[]; sources?: readonly string[] },
) {
  const keyword = filters.keyword?.trim().toLowerCase();
  const levels = new Set(filters.levels ?? []);
  const sources = new Set(filters.sources ?? []);
  return chunks.filter(
    (chunk) =>
      (!keyword || chunk.content.toLowerCase().includes(keyword)) &&
      (levels.size === 0 || levels.has(chunk.level ?? '')) &&
      (sources.size === 0 || sources.has(chunk.source ?? '')),
  );
}
