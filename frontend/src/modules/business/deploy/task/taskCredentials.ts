import type { DeployCredentialRow } from '../api';

export function credentialsForTask(
  credentials: DeployCredentialRow[],
  businessScopeId: number | undefined,
  allowGlobal: boolean,
) {
  return credentials.filter((credential) =>
    credential.status === 'active' &&
    (credential.businessScopeId === 0 ? allowGlobal : credential.businessScopeId === businessScopeId),
  );
}
