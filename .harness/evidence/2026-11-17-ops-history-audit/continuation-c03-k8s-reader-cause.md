# C03 round 2: preserve K8s provider failures

Owner files: `backend/modules/business/k8s/capability.go` and
`backend/modules/business/k8s/capability_error_test.go`. File hashes are recorded
in `continuation-c03-k8s-reader-cause.hashes.json`; these are working-tree files,
not a commit or a complete reconstructed product snapshot.

## Change and evidence

Deployment, StatefulSet, and DaemonSet target reads now share one error branch.
Only `apierrors.IsNotFound(err)` becomes the existing exact
`business.service.k8s_target_not_found` key. All other errors return unchanged,
preserving Kubernetes status classification, wrapped transport causes, and
context cancellation. Successful target metadata reads are unchanged.

The parent ran both commands with CGO=1, readonly dependencies, race enabled,
and a disposable real MySQL fixture required by the Service integration tests.

- `continuation-c03-k8s-reader-cause-red.log` and its receipt: exit 1, package
  0.862s. All three top-level tests failed. Real provider 404 controls passed;
  forbidden/server/transport errors and in-flight cancellation lost their
  classification or cause. Service reclassified these faults as target-not-found.
- `continuation-c03-k8s-reader-cause-fixed.log` and its receipt: exit 0, package
  1.973s, 2026-11-17 14:50:50–14:51:12 UTC. All three top-level tests passed.
- Provider matrix: real client-go REST clients against `httptest` responses for
  404/403/500 and a custom transport with an `errors.Is` sentinel, for all three
  workload kinds. In-flight cancellation uses a transport-entry gate and no
  sleeps. Failed reads do not proceed into ownership metadata.
- Service trace: the real provider reader is injected into a real Service
  manager/MySQL transaction and the actual instance-edit HTTP handler. 404
  remains canonical 404; forbidden/server/transport faults preserve their
  original classification/cause and return a 500 business envelope. All owner
  facts remain unchanged after direct and HTTP refusals.
- `gofmt` and scoped `git diff --check`: exit 0. This worker started no tests,
  database processes, background helpers, commits, archives, or goal changes.

## Scope and remaining gates

The test cluster adapter returns a real client-go client directly through the
existing narrow `GetClientset` seam. Cluster factory endpoint/SSRF validation,
live credentials, JWT sessions, and live Kubernetes access are not exercised.
No production bypass, configuration change, or dependency was introduced.

The provider suppression gap previously recorded in the C03 owner round 1 note
is repaired by this candidate. Typed target lifetime protection and K8s
namespace binding/environment/action admissibility remain unproved. C04's
Kubernetes Service resource files, namespace files, and permission files were
not edited here. Final language diagnostics, build, full compatibility/race,
runtime verification, and independent re-review remain parent-owned gates.
