# Argo CD reviewed-actions v1 provider extension

This business-owned patch extends official Argo CD v3.2.0, commit
`66b2f302d91a42cc151808da0eec0846bbe1062c`. It must be independently reviewed
before Ops adopts or deploys it. It is a provider extension, with a separate
maintenance and upgrade cost; it is not unmodified-provider acceptance.

`reviewed-actions-v1.patch` contains the server core, native protobuf/gateway
registration, generated API output, consolidated Swagger additions and focused
tests. No Ops Go dependency, Argo controller, Kubernetes credential substitution,
Base override or stock Sync/Rollback replacement is introduced.

## Contract

All three methods use Argo's existing ApplicationService and authentication
interceptors, token claims, application namespace/project scope and RBAC.

| Method                        | HTTP route                                                      | Request                                                             |
| ----------------------------- | --------------------------------------------------------------- | ------------------------------------------------------------------- |
| `ReviewedActionsCapabilities` | `GET /api/v1/applications/{name}/reviewed-actions/capabilities` | Existing `ApplicationSyncWindowsQuery`: name, appNamespace, project |
| `ReviewedSync`                | `POST /api/v1/applications/{name}/reviewed-actions/sync`        | Existing `ApplicationSyncRequest` plus `reviewed`                   |
| `ReviewedRollback`            | `POST /api/v1/applications/{name}/reviewed-actions/rollback`    | Existing `ApplicationRollbackRequest` plus `reviewed`               |

Capability response is exactly:

```json
{
  "protocol": "pantheon-reviewed-actions/v1",
  "revisionScope": "single-git-sha1",
  "markerName": "pantheon-action-id",
  "conditionalUpdate": true,
  "nativePolicy": true,
  "correlatedRollback": true
}
```

Discovery requires native application read access. It does not grant sync access
or guarantee that later native policy checks will pass. Native authentication
configuration and token validation remain native. Reviewed endpoints additionally
require the native session principal; anonymous default-role access and
disable-auth mode cannot submit reviewed actions or advertise capabilities.
Current-token validity and revocation still require real authenticated-provider
evidence, beyond unit fixtures containing claims.

Example sync body (include the native proto's required `name` in the body):

```json
{
  "name": "reviewed-app",
  "appNamespace": "argocd",
  "project": "default",
  "revision": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "prune": false,
  "reviewed": {
    "expectedUID": "the-reviewed-application-uid",
    "expectedResourceVersion": "12345",
    "resolvedRevision": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "actionMarker": "one-unique-action-marker"
  }
}
```

UID and resourceVersion must be nonempty bounded printable ASCII (maximum 128
characters). The resolved revision is a full lowercase 40-character Git SHA-1.
The marker is 1–128 ASCII letters, digits, underscore or hyphen. Sync accepts no
caller `infos`; the server inserts exactly one `pantheon-action-id` operation
info. These fields are mandatory on reviewed routes even though the reusable
native request's added `reviewed` protobuf field is optional.

Rollback uses native `id` (required, nonnegative; zero is valid) instead of a
client-supplied source or branch. It selects that exact history entry from the
same version-bound Application, requires its historical revision to match
`resolvedRevision`, and uses its immutable historical source and native prune,
sync options, apply strategy and initiator. The same action marker is added.

V1 accepts one Git source. Multi-source Applications/operations, Helm chart
repository versions, local manifests and supplied infos fail closed. A Git
repository containing Helm manifests remains a Git source. Native automatic
sync restrictions, Sync windows, sync/override RBAC, retry/options/prune and
operation construction are retained where supported. **Native Rollback does
not enforce Sync windows**; reviewed rollback preserves its sync RBAC and
automatic-sync restriction and adds no new window policy. Controller-time
source/destination/signature enforcement remains native and needs live proof.

## Acceptance and recovery

The reviewed variants share the actual native Sync/Rollback methods. They read
the Application through the native authorized client, match UID/resourceVersion,
and build the operation from that same Application. Pending, deleting and invalid
operations are rejected before mutation. Resource-tree health projection is not
written back as a side effect of reviewed submission.

One request-context-bound Kubernetes Application `Update` writes the operation
and clears the old operationState using that exact UID/resourceVersion. Conflict
or disappearance returns gRPC Aborted / HTTP 409 without a reread or retry. The
Kubernetes update contract enforces UID and resourceVersion at its storage
boundary; server-side storage internals are not a client reload/retry.

The returned Application is the actual update response. A controller consuming
the operation immediately afterwards does not cause a name-only confirmation
read or a manufactured terminal result. Request cancellation or a lost reply
can leave acceptance uncertain: Ops must retain Unknown and recover the same
reserved action by UID, resolved revision and marker without redispatch. There
is no durable receipt capability if later operation evidence is overwritten.
The condition binds operation acceptance, not every managed resource until
terminal; native controller execution/retry behavior is retained.

Stock Argo CD has none of these distinct RPC/HTTP routes. Missing, malformed or
incompatible capabilities must block reviewed dispatch. Never fall back to native
Sync/Rollback, direct CRD writes, JSON PATCH or additive fields on a stock native
route. Existing native Sync/Rollback wrappers retain their supported behavior;
they do not enforce `reviewed` fields.

## Reproduce and maintain

In a separate clean checkout of the pinned official commit:

```sh
test "$(git rev-parse HEAD)" = 66b2f302d91a42cc151808da0eec0846bbe1062c
git apply --check /path/to/reviewed-actions-v1.patch
git apply /path/to/reviewed-actions-v1.patch
go test -race -mod=readonly ./server/application ./pkg/apiclient/application \
  -run '^(TestReviewed|TestSyncAndTerminate$|TestSyncHelm$|TestSyncGit$|TestSync_WithRefresh$|TestRollbackApp)' -count=1
go vet -mod=readonly ./server/application ./pkg/apiclient/application
go build -mod=readonly ./server/application ./pkg/apiclient/application
```

Windows race uses `D:\msys64\mingw64\bin` at the front of PATH with
`CGO_ENABLED=1`. The package build above is not a patched-server binary/image
build or deployment result.

Regeneration uses official upstream pins: protoc 29.3, gogo/gogofast 1.3.2 and
grpc-gateway/grpc-gateway+swagger 1.16.0. The scoped local generator is
`.local/ops-acceptance/resume-argo-v6-provider-generate.py`; it reproduces the
unmodified service's original generated outputs before generating the changed
service. It merges generated reviewed routes and four definitions into the
existing consolidated Swagger using the upstream collector transformations.
It avoids CRD/client/controller regeneration and unrelated output changes.

On each Argo upgrade, re-review the native methods, history semantics,
authentication registration, project/controller enforcement, API generator pins
and complete changed-path manifest. Rebase and regenerate against the new
official version; a successful old-version patch application alone is insufficient.

Exact source/test/patch hashes, command exits, skip counts and remaining gates
are recorded in current P0 `resume-argo-v6-provider-handoff.md`. Patched-server
binary/image, deployment, independent provider review, Ops adapter adoption,
current-session/IAM/audit, actual controller terminal plus Service restoration,
historical rollback, lost-reply recovery and full O3/D1/P0 acceptance remain
root-owned open gates.
