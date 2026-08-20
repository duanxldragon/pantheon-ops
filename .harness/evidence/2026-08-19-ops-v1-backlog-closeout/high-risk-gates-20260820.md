# V1 High-Risk Gate Evidence

## Concurrent ownership

`PANTHEON_TEST_DSN=<local isolated MySQL>` with
`go test -count=1 ./modules/business/k8s/cluster` passed.
`TestClusterUpdateRejectsConcurrentStaleScopeMutation` moved a cluster from
department 10/scope 1 to department 20/scope 2 while a stale department-10
writer raced; the stale writer returned `k8s.cluster.not_found` and did not
overwrite the name or ownership fields.

## Snapshot duplicate/index rehearsal

The full local `pantheon_ops` snapshot was restored to a uniquely named schema.
`TestProductionSnapshotRollbackAndReapply` passed through `22 -> 16 -> 22`.
Before and after reapply, active duplicate candidates were zero for business
scope, CMDB host/label, Deploy package/template, and K8s cluster keys. Generated
columns matched their expressions and all six named unique indexes were unique
on the expected generated column. Core business row counts were unchanged; the
source schema was never migrated and the rehearsal schema/dump were deleted.

## Kubernetes mutation

Using `config/kubeconfig.yaml --insecure-skip-tls-verify=true`, namespace
`pantheon-v1-gate-20260820-1142` was created and deleted after verification.
ConfigMap resourceVersion changed `4369819 -> 4369820`; a stale replace returned
Conflict and the final marker remained `v2`. A Secret was created and rotated;
only metadata/key count was recorded, never its value.

## SSH mutation

Using the supplied isolated host `192.168.56.12`, a temporary `/tmp` marker was
written, hashed (`cc634792b4fa94519f4c7712c2a42aa982a06c894e549d61ed7402b6fb92f8d3`),
and deleted. The verified ED25519 host-key fingerprint was
`SHA256:NXRzIxTVlvx8umUI0PvPtJ+bx29C1R22qA2BQhSlFfA`; no password or secret was
written to this artifact.

## Harness inheritance

The clean overlay rebuild from the locked Base snapshot passes strict method
health and adoption with zero findings. Root strict checks remain externally
blocked because the locked release predates the current shell landing contract;
Base commit `303c6bdb` has 156 pre-existing unresolved SonarCloud issues and its
Release Gate is red. Generic Harness files were not copied into Ops.
