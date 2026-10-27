import { apiRequest } from '../../../api/request';

export interface LoginPayload {
  username: string;
  password: string;
  tenantId?: number;
}

export interface LoginTenantCandidate {
  tenantId: number;
  code: string;
  name: string;
  role: string;
}

export interface LoginResp {
  tenantSelectionRequired?: boolean;
  tenantCandidates?: LoginTenantCandidate[];
  tenantId?: number;
  mfaRequired?: boolean;
  challengeId?: string;
  setupRequired?: boolean;
  totpSecret?: string;
  totpProvisionUri?: string;
  expiresAt?: string;
}

export function login(data: LoginPayload) {
  return apiRequest<LoginResp>({
    url: '/auth/login',
    method: 'post',
    data,
    skipErrorMessage: true,
  });
}
