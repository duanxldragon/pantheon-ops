import type { DistributionTaskRow } from './api';

export const TASK_STATUSES = [
  'pending',
  'running',
  'completed',
  'partial',
  'failed',
  'cancelled',
] as const;

export function isTaskActive(status: string): boolean {
  return status === 'pending' || status === 'running';
}

export function taskStatusColor(status: string): string {
  if (status === 'completed') return 'green';
  if (status === 'failed') return 'red';
  if (status === 'running' || status === 'transferring') return 'arcoblue';
  if (status === 'partial' || status === 'retrying') return 'orange';
  return 'gray';
}

export function canStartTask(task: DistributionTaskRow, now = Date.now()): boolean {
  if (task.status !== 'pending') return false;
  if (!task.scheduledAt) return true;
  const dueAt = Date.parse(task.scheduledAt);
  return Number.isFinite(dueAt) && dueAt <= now;
}

export function canRetryTask(task: DistributionTaskRow): boolean {
  return task.status === 'failed' || task.status === 'partial';
}

export function normalizeScheduledAt(value: unknown): string | undefined {
  if (value === null || value === undefined || value === '') return undefined;
  if (typeof value !== 'string' && typeof value !== 'number' && !(value instanceof Date)) {
    throw new Error('business.filedist.create.invalidSchedule');
  }
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) throw new Error('business.filedist.create.invalidSchedule');
  return date.toISOString();
}
