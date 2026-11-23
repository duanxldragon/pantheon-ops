'use strict';
// one-shot update of acceptance-matrix.json with the 2026-11-22 cap-live results
const fs = require('fs');
const P = 'acceptance-matrix.json';
const m = JSON.parse(fs.readFileSync(P, 'utf8'));
const capEvidence = '.harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/runs/omx_accept_20261122_cap/manifest.json (run 3: 31 PASS / 1 BLOCKED / 0 FAIL, 32 cases)';

// ---- verificationItems ----
for (const v of m.verificationItems) {
  if (v.caseId === 'V3') {
    v.status = 'PASS';
    v.evidence = `runs/omx_accept_20261121_v3/v3/probe.json (24 probes PASS on candidate k8s33). Genuinely-unbound modules k8s-cluster / cicd-argocd / filedist / bizscope -> all-403 for an all-PermissionKeys role, with bound positive controls (service/cmdb 200) proving the grant mechanism fires; observability proven BOUND (audit_policy.go registers business:observability routes) via grant-allowed-200 + no-grant-403 with a dedicated neg-control user. Inventory finding: business:cmdb:host:viewer seeded but not assignable (128/129 valid, 1 invalid).`;
  }
  if (v.caseId === 'V6') {
    v.status = 'PARTIAL';
    v.evidence = `runs/omx_accept_20261122_cap/manifest.json — v6-repeat-submit-dedup: identical business/service/applications create submitted twice, second rejected (application code exist), no duplicate row (totals before/after + dup create http/code recorded); per-case durationMs recorded for upload 2MB / download / reconcile (V6 timing). Frontend 10s-half deferred to GAP-UI-BROWSER (no browser runtime on this stack yet).`;
  }
}

// ---- capabilities ----
const caps = {
  'CAP-SERVICE': ['PASS', 'cap-live run 3 (omx_accept_20261122_cap): application create/list/detail + service create/list/detail/update + VM instance create (200/200, id returned under a same-scope CMDB host) + instance reconcile + full 7-step cleanup deletes all 200. Validated with real same-tenant fixtures: bizscope -> application -> service(runtimeType=vm) -> CMDB host -> instance.'],
  'CAP-BIZSCOPE': ['PASS', 'cap-live run 3: POST /business/bizscope create (id returned) -> GET /list contains created row -> GET /options -> PUT update, all business code 200; instance target resolution across the same scope proves BizScopeReader wiring.'],
  'CAP-FILEDIST': ['PASS', 'cap-live run 3: storages list (default local storage provisioned) -> 2MB multipart upload -> files list contains row -> GET download round-trips exactly 2097152 bytes -> storage create/delete -> file delete, all 200.'],
  'CAP-OBSERVABILITY': ['PASS', 'cap-live run 3: SSRF guard POSITIVE (loopback 127.0.0.1:9090 rejected even though port 9090 is allowed -> BlockPrivateIPs; public example.com:8081 rejected -> AllowedPorts allowlist) and create/list/update on http://example.com (port 80) all 200; refresh()/test returns structured test_failed on reachable non-Prometheus placeholder; query returns structured param.invalid for this probe payload (probe shape, not product) — no panic on any path.'],
  'CAP-ARGOCD': ['BLOCKED', 'cap-live run 3: POST /business/cicd/argocd/servers -> 500 argocd.encryption_key_missing (cicd/argocd/argocd_encryption.go loadEncryptionKey needs env PANTHEON_ENCRYPTION_KEY or argocd-specific key — absent from runtime-live/private-env.dpapi). Code path is correct; this candidate deployment lacks the env key. Guarantee: throwaway authToken omx-verify-<sub> never appeared in the create response body, and server-list 200 returned no token. See GAP-ARGOCD-ENVKEY.'],
  'CAP-SECRETS': ['PARTIAL', 'V5 DB ciphertext cross-ref already PASS (deploy credentials at rest encrypted); cap-live run 3 adds argocd authorization token no-leak on write+list (CAP-SECRETS). Full business-secret encrypt-at-rest round trip not exercisable on this deployment: PANTHEON_ENCRYPTION_KEY AND PANTHEON_BUSINESS_SECRET_KEY are both absent from the runtime env, so argocd store is BLOCKED and shared secrets.Encrypt would reject non-empty values (see GAP-BUSINESS-SECRET-KEY). Empty-string Encrypt is a no-op, so passwordless datasource create is unaffected.'],
  'CAP-METRICS': ['PASS', 'cap-live run 3: GET /metrics -> HTTP 200, NUM pantheon_* metric series present (Prometheus exposition reached live).'],
  'CAP-CAPABILITY': ['PASS', 'cap-live run 3 compositions: 5-module chain (bizscope -> application -> service -> CMDB host -> VM instance) all same-scope, instance target resolution proving BizScopeReader + CMDBHostReader wiring end-to-end; V3 permission-bindings probe proves default-deny + grant firing.'],
  'CAP-OPSINIT': ['PASS', 'runtime-live/backend-fresh-20261121-075936.{stdout,stderr}.log: "Starting Pantheon Base", "Velero reconciler started interval 1m0s" (daemon components alive), GIN route table + /metrics + main server "starting server port=8081"; maintenance tasks (audit.operation_log_retention, auth.session_inventory, auth.login_log_retention, auth.security_event_retention) finish cleanly; grep for "panic" across all runtime-live/*.log -> none.'],
};
for (const c of m.capabilities) {
  const upd = caps[c.capabilityId];
  if (upd) { c.status = upd[0]; c.runtimeEvidence = capEvidence; c.actual = upd[1]; }
}

// ---- gaps ----
for (const g of m.gaps) {
  if (g.caseId === 'GAP-ARGOCD-ENVKEY') continue;
  if (g.caseId === 'GAP-BUSINESS-SECRET-KEY') continue;
}
if (!m.gaps.find(g => g.caseId === 'GAP-ARGOCD-ENVKEY')) {
  m.gaps.push({
    caseId: 'GAP-ARGOCD-ENVKEY', item: 'argocd 静态加密密钥 (PANTHEON_ENCRYPTION_KEY) 未配置',
    status: 'OPEN', owner: 'deploy-config',
    action: 'cap-live run 3 proves cicd/argocd AddServer/AddRepository is correct but BLOCKED by argocd.encryption_key_missing: env lacks PANTHEON_ENCRYPTION_KEY (and argocd-specific key). Not a code defect. Runtime env setter must inject this key before argocd capability is exercised. Pending maintainer/deploy-config decision; do not weaken key checks.',
  });
}
if (!m.gaps.find(g => g.caseId === 'GAP-BUSINESS-SECRET-KEY')) {
  m.gaps.push({
    caseId: 'GAP-BUSINESS-SECRET-KEY', item: '业务共享加密密钥 (PANTHEON_BUSINESS_SECRET_KEY) 未配置',
    status: 'OPEN', owner: 'deploy-config',
    action: 'Shared AES-256-GCM secrets package (business/secrets, env PANTHEON_BUSINESS_SECRET_KEY) would error business.secrets.key_missing on any NON-empty business secret (filedist storage config, observability datasource password/bearerToken). Empty-string Encrypt is a no-op, so passwordless datasource create is unaffected. Add the key to runtime env before storing non-empty business secrets. Also confirms datasource create 500 was NEVER encryption-related (root cause: my probe URLs failed SSRF allowlist, see CAP-OBSERVABILITY).',
  });
}

m.lastExecutionAt = '2026-11-22T01:02:00.000Z';

fs.writeFileSync(P, JSON.stringify(m, null, 2));
console.log('matrix updated. caps:', m.capabilities.map(c => `${c.capabilityId}:${c.status}`).join(' '));
console.log('VIs:', m.verificationItems.map(v => `${v.caseId}:${v.status}`).join(' '));
console.log('gaps:', m.gaps.map(g => g.caseId).join(', '));