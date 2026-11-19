import type { DeployTargetResolution } from '../deploy/api';
import type { CreateTaskPayload } from './api';
import { normalizeScheduledAt } from './taskState';

export interface CreateTaskFormValues {
  name: string;
  fileId: number;
  businessScopeId: number;
  username: string;
  targetPath: string;
  hostFingerprints: Record<string, string>;
  password?: string;
  privateKey?: string;
  maxRetries: number;
}

export function buildCreateTaskPayload(
  values: CreateTaskFormValues,
  resolution: DeployTargetResolution,
  authMode: 'password' | 'privateKey',
  schedule: unknown,
): CreateTaskPayload {
  if (
    !resolution.scopeVersion ||
    resolution.selection.businessScopeId !== values.businessScopeId ||
    !resolution.hosts.length ||
    resolution.unavailableCount ||
    resolution.ineligibleCount
  ) {
    throw new Error('business.filedist.create.selectionRequired');
  }
  const scheduledAt = normalizeScheduledAt(schedule);
  const credential = authMode === 'password' ? values.password : values.privateKey?.trim();
  if (!credential || !values.name.trim() || !values.username.trim() || !values.targetPath.trim())
    throw new Error('business.filedist.create.required');
  const seen = new Set<number>();
  const targets = resolution.hosts.map((host) => {
    const hostFingerprint = values.hostFingerprints?.[String(host.id)]?.trim();
    if (!hostFingerprint || !/^SHA256:[A-Za-z0-9+/]{43}$/.test(hostFingerprint))
      throw new Error('business.filedist.create.invalidFingerprint');
    if (
      !Number.isSafeInteger(host.id) ||
      host.id <= 0 ||
      seen.has(host.id) ||
      !Number.isSafeInteger(host.stateVersion) ||
      host.stateVersion <= 0 ||
      !host.ip
    )
      throw new Error('business.filedist.create.selectionRequired');
    seen.add(host.id);
    return {
      hostId: host.id,
      stateVersion: host.stateVersion,
      host: host.ip,
      targetPath: values.targetPath.trim(),
      username: values.username.trim(),
      hostFingerprint,
      ...(authMode === 'password' ? { password: credential } : { privateKey: credential }),
      maxRetries: values.maxRetries ?? 3,
    };
  });
  return {
    name: values.name.trim(),
    fileId: values.fileId,
    targets,
    selection: resolution.selection,
    scopeVersion: resolution.scopeVersion,
    ...(scheduledAt ? { scheduledAt } : {}),
  };
}
