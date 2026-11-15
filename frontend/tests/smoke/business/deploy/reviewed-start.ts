import { expect, type APIRequestContext } from '@playwright/test';
import { apiBaseUrl } from '../../helpers/auth';
import type { StartDeployTaskPayload } from '../../../../src/modules/business/deploy/api';

// Positive API fixtures follow the public preview/confirm contract.
export async function startReviewedTask(
  request: APIRequestContext,
  taskId: number,
  options: { headers?: Record<string, string>; data?: StartDeployTaskPayload } = {},
) {
  const data = options.data ?? {};
  const response = await request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/preview`, {
    headers: options.headers, data,
  });
  const preview = await response.json();
  expect(response.ok(), `Preview HTTP ${response.status()}`).toBeTruthy();
  expect(preview.code, preview.message).toBe(200);
  expect(preview.data.intentVersion).toEqual(expect.any(String));
  return request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/start`, {
    headers: options.headers,
    data: { ...data, intentVersion: preview.data.intentVersion },
  });
}
