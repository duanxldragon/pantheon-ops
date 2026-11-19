# C03 Service owner: independent review round 1 fixes

Candidate: local working tree, no commit. All 8 owner source/test files matched
`continuation-c03-owner-review-fixes.hashes.json` when this note was written.
The earlier cached reconstruction checkpoint is a different candidate and is
not evidence for these review fixes.

## Changes

- `handler.go` registers `GET /services/:id`. `GetServiceDetail` preserves the
  request context and DataScope and returns the existing camelCase
  `ServiceResponse`, including parent business scope and instance count.
  Missing or out-of-scope owner rows return a 404 business envelope; unexpected
  failures retain their cause and use a generic failure response.
- `module.go` registers the Service-owned read policy provider before module
  initialization. `permission_routes.go` binds the existing `list` and `view`
  keys to exactly seven existing GET routes. No role is automatically assigned
  a key, and no mutation route is included.
- `UpdateServiceWithContext` runs in a READ COMMITTED owner transaction. It
  acquires the Service and Application locks before the nonlocking incompatible
  instance count and runtime update. It does not lock instances after Service.
  The legacy method preserves the manager's database context; HTTP passes the
  request context explicitly.
- `validateK8sTarget` translates only the documented exact
  `business.service.k8s_target_not_found` key into `target_not_found`. All other
  owner errors propagate unchanged, including wrapped infrastructure causes.

## Fresh verification

Commands were run by the parent task, not this worker. Both runs required a
disposable real MySQL DSN and CGO=1 and used Go race with readonly dependencies.

- `continuation-c03-review-round1-red.log` and its receipt: exit 1, Service
  1.463s. All four new top-level tests failed: absent detail route, missing read
  policy binding, runtime update accepting a now-incompatible K8s rebind, and
  the known K8s missing-target key remaining unclassified. The unexpected
  infrastructure-cause subtest passed before the production fix.
- `continuation-c03-review-round1-fixed.log` and its receipt: exit 0, Service
  9.761s, 2026-11-17 14:40:47–14:41:01 UTC. The four review regressions and all
  thirteen prior `TestServiceInstanceUpdate` regressions passed.
- Runtime/rebind interleaving is controlled by a blocked typed K8s owner read
  and an operation-context-tagged GORM lock/write callback. It uses no sleeps.
  The rebind commits first, and the waiting incompatible runtime edit refuses.
- The read-policy tests use the released Casbin `keyMatch2` matcher against
  actual registered routes, then exercise scoped detail GET and denied PUT
  through the real Service handler. This is not a live JWT/session test.
- `gofmt` and scoped `git diff --check`: exit 0. No debug or temporary source
  files, database processes, commits, archives, or goal changes were created.

## Remaining evidence and boundary gaps

- Language diagnostics, build, complete Service/Deploy compatibility, complete
  backend race, live role-policy reconciliation, and independent re-review
  remain parent-owned gates for this final candidate.
- Typed CMDB/K8s readers do not hold target lifetime protection through the
  Service transaction. Concurrent target retire/delete remains unproved.
- K8s namespace binding, environment, and action admissibility remain a K8s
  owner/C04 gap. The old K8s reader's suppression of upstream errors is not
  repaired by this exact documented-key mapping.
- Architecture status remains WATCH. These seventeen focused passes do not
  close C03 or substitute for final runtime/visual acceptance.

## Round 2 follow-up

The provider suppression gap above records the round 1 checkpoint. The later
`continuation-c03-k8s-reader-cause.md` and its file hashes/command receipts record
the bounded provider correction: only real Kubernetes not-found is normalized;
403/500/transport/context failures preserve their original cause. Its three
focused race tests passed at 2026-11-17 14:51:12 UTC. Target lifetime protection
and K8s binding/environment/action admissibility remain explicit gaps.
