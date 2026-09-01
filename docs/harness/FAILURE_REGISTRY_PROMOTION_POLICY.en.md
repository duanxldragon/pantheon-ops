# Failure Registry Promotion Policy

Chinese version: [FAILURE_REGISTRY_PROMOTION_POLICY.md](./FAILURE_REGISTRY_PROMOTION_POLICY.md)

Type: Policy
Layer: platform
Status: Active

This policy defines when the failure registry should move from “recommended” to “required landing file”.

The failure registry is one landing file in the ratchet loop defined by `FAILURE_RATCHET_POLICY.en.md`, but it is not the only possible landing target.

## 1. Current Mode

`check-failure-registry.mjs` currently reports a missing default registry path as a warning so new repositories can adopt the registry gradually.

## 2. Promotion Threshold

Promote default registry presence to a required landing file only when all of the following are true:

1. At least 2 downstream consumer repositories have seeded `docs/harness/failure-registry.md`
2. Those 2 consumer repositories retain the file and keep it structurally valid for 2 consecutive release cycles
3. Method maintainers confirm during the latest release review that:
   - the registry is no longer just a pilot asset
   - missing the registry materially weakens the ratchet loop

## 3. Promotion Actions

Any promotion must also:

- update `check-failure-registry.mjs` so default absence becomes a finding instead of a warning
- update `check-method-health.mjs` required landing files
- update `HARNESS_OPEN_TASKS.md`
- record the cutover in the release notes

## 4. Rollback Condition

If promotion reveals that consumer bootstrap still widely omits the registry and the new rule causes clear adoption friction, the rule may temporarily fall back to warning mode, but the rollback reason and next review date must be recorded.
