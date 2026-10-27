import { apiRequest } from '../../../api/request';
import type { LoginResp } from '../login/api';

export interface MFAVerifyPayload {
  challengeId: string;
  code: string;
  tenantId?: number;
}

export function verifyMFA(data: MFAVerifyPayload) {
  return apiRequest<LoginResp>({
    url: '/auth/mfa/verify',
    method: 'post',
    data,
    skipErrorMessage: true,
  });
}
