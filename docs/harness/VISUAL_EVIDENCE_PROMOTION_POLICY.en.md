# Visual Evidence Promotion Policy

Chinese version: [VISUAL_EVIDENCE_PROMOTION_POLICY.md](./VISUAL_EVIDENCE_PROMOTION_POLICY.md)

Type: Policy
Layer: platform
Status: Active

This policy defines when `check-visual-evidence.mjs --strict` should move from an observational gate to a blocking gate.

## 1. Current Mode

The default mode is observational:

- `--strict` still exits with failure when warnings exist
- CI may continue to run it with `continue-on-error: true` while data is being gathered

## 2. Promotion Threshold

Promote the CI visual-evidence job to a blocking gate only when all of the following are true:

1. The latest 3 consecutive UI-affecting non-trivial tasks all include:
   - a viewport plan
   - a state plan
   - browser evidence or screenshot evidence, or an explicit visual gap
2. All 3 tasks produce zero warnings under `check-visual-evidence.mjs --strict`
3. No rule rollback was required in that same window because of checker false positives

## 3. Rollback Condition

If, after promotion, 2 consecutive tasks produce false positives caused by incomplete checker or policy definitions, then:

- revert to observational mode temporarily
- fix the checker or policy
- restart the promotion window

## 4. Recording Requirements

Any promotion or rollback must update:

- `HARNESS_OPEN_TASKS.md`
- the relevant failure-registry or review notes
- CI workflow comments
