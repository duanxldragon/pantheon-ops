# C01 continuation plan

Scope remains the original C01 packet. The parent ultragoal G003 is now in progress; G002 is retained unfinished with dependency steering, not completed or superseded.

1. Reproduce invalid association admission with the existing real-MySQL Deploy fixture. Check partial IDs and an unavailable owner for both create and update; require no task mutation. Preserve the red run.
2. Reuse Service-owned Reader projections and its transaction seam. Validate both IDs, active service/application/instance, business scope, and the actual target. Explicit independent tasks keep both IDs zero and do not invoke state callbacks. Host/group deployment cannot masquerade as a Kubernetes instance target.
3. Perform admission within the same transaction as task writes and the final start claim. Bind owner reads and state commands to that transaction; lock the same Service rows used by deletion. Run side-effect-free association preflight before start reservations; preserve the established task/host/lease -> CMDB -> Service lock order for authoritative admission. Any tentative task-host/lease writes roll back on failed admission; no committed partial state, secret persistence or remote execution is allowed. Address lock ordering and snapshot visibility explicitly.
4. Add owner reference checks for executing deployment tasks to service-instance and service deletion. Service owner keeps Service persistence; Deploy owner counts its own references. Preserve historical snapshots and retired/terminal-task history.
5. Verify matched success/failure callbacks, replay/out-of-order callbacks and concurrent deletion on real MySQL. Use the existing lifecycle version and deployment fencing rather than introducing a parallel state machine.
6. Update business errors/i18n/seed/smoke and canonical C01 documentation as needed. Run targeted CGO race, boundary checks, official-lock rebuild, source binding and required full race after implementation; obtain two independent review lanes. Keep C01/C02/C05 open until their original acceptance criteria are met.

Preflight evidence: Deploy currently persists ServiceID/ServiceInstanceID without an owner reader in create/update; start's begin-state command occurs after the task-claim transaction; Service DeleteInstance currently deletes after a visible read without a Deploy reference guard. These are source findings. The concurrency impact is not yet dynamically reproduced.

No platform/system source, foundation lock, real business data, publishing or production changes are in scope.
