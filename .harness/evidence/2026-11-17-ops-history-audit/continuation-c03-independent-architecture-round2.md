# C03 independent architecture round 2

Coordinator transcription of the architect report prepared at 2026-11-17T14:44:27Z. Backend checkpoint 14:40:44Z; UI checkpoint 14:37:17.3963866Z. Working-tree assessment, without final revision binding.

```json
{
  "taskId": "2026-11-16-ops-service-catalog-usability",
  "reviewRound": 2,
  "agentRole": "architect",
  "architecturalStatus": "WATCH",
  "backendManifestFilesMatching": 8,
  "uiManifestFilesMatching": 7,
  "ownerRaceTestsPassed": 17,
  "ownerRaceExitCode": 0,
  "runtimeRebindLockReview": "CLEAR within reviewed Service owner paths",
  "getOnlyPermissionBindingsReview": "CLEAR for declared routes",
  "realK8sFaultCausePreservation": "OPEN",
  "taskAcceptanceApproved": false,
  "archiveApproved": false,
  "gapWaiverGranted": false,
  "testsExecutedByReviewer": false,
  "renderedOutputInspectedByReviewer": false,
  "openFinding": "C03-ARCH-R2-01 concrete K8s reader masks non-not-found failures"
}
```

READ COMMITTED runtime edits lock Service and Application before the nonlocking incompatible-instance count. Rebind and create share the Service lock; there is no new Service→instance locking inversion. The controlled test covers rebind-first then runtime rejection; the reverse ordering is supported by source, not independently executed here.

Scoped detail GET returns the camelCase catalog response. Seven actual GET routes are bound to list/view without binding mutation or granting any role. Isolated Casbin tests are not live-role synchronization evidence. Cross-tab creation correctly uses form parent queries rather than another tab's filtered total.

Service exact-key not-found mapping preserves unexpected reader errors locally. However, the concrete K8s reader converts every Deployment/StatefulSet/DaemonSet Get error to k8s_target_not_found before Service can preserve the cause. Network, 403, timeout and cancellation can therefore appear as nonexistent targets. The fixture-reader fault test does not exercise this chain. Only apierrors.IsNotFound should produce the known key; real adapter regressions are required.

The architect inspected all fifteen source hashes and seventeen passing real-MySQL CGO race tests / exit 0 / 9.761s. No tests or rendered output were produced by this lane. Cross-owner target lifetime, NamespaceBinding/environment/action, live UI/role acceptance, complete race, rebuild and final source binding remain unwaived gaps. C03 remains open.
