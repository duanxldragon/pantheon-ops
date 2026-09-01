# Failure Ratchet Policy

Chinese version: [FAILURE_RATCHET_POLICY.md](./FAILURE_RATCHET_POLICY.md)

Type: Policy
Layer: platform
Status: Active

This policy defines how a repeated failure should move from a one-off problem into a repository-owned method asset.

## 1. Goal

The harness should not merely record that a problem happened. It should define where the control surface must move when the same failure pattern happens again.

The ratchet loop exists to:

- keep repeated failures out of chat-only memory
- promote experience from manual reminder into reusable rules
- eventually let deterministic sensors or gates catch high-frequency failures

## 2. What Counts As A Repeated Failure

Treat the following as repeated failures by default:

1. the same root cause or control gap appears again across different tasks, PRs, or sessions
2. review repeatedly reports the same issue class, but the repository still lacks a guide, template, checker, or smoke path for it
3. the same gap recurs in both `pantheon-base` and a business repository
4. the same class of issue reappears after handoff because context was lost

Do not force the label for:

- one-off typos
- clearly isolated business exceptions with no reusable value
- checker false positives that are not real failure patterns

## 3. Ratchet Levels

### 3.1 Level 0: Single Failure

On the first occurrence, record the failure pattern in at least one of:

- the closeout note
- a review finding
- `knownGaps` in verification evidence

The goal is traceability, not punishment.

### 3.2 Level 1: Second Occurrence

On the second occurrence, promote the issue into at least one feedforward control:

- `AGENTS.md`
- a task-packet template
- an acceptance checklist
- a contract, design, or review guide
- a repo-local how-to or playbook

If nothing is promoted, the ratchet did not happen.

### 3.3 Level 2: Third Occurrence Or Cross-Repo Recurrence

On the third occurrence, or when the issue already recurs across repositories or modules, prefer promoting it into a feedback control:

- test
- static check
- harness checker
- smoke case
- browser or runtime evidence gate
- failure-registry entry

The `failure registry` is one default landing file, but not the only one. If the problem can be automated directly, prefer landing it as a sensor.

### 3.4 Level 3: Severe Or Persistent Failure

Any of the following may skip levels:

- security, permission, audit, schema, or data-damage risk
- high-impact work that already crosses a human gate
- issues already affecting release quality or inheritance stability

These may be promoted directly into a blocking gate, required landing file, or required release-review item.

## 4. Recording Requirements

Each ratchet promotion should record at least:

- what the failure pattern is
- which artifact or sensor it was promoted into
- why that control level was chosen
- what the next verification window is

Recommended recording locations:

- `docs/harness/failure-registry.md`
- `HARNESS_OPEN_TASKS.md`
- release notes
- the relevant task packet, review, or evidence

## 5. Relation To The Failure Registry

The `failure registry` records which issues are important enough to remain durable failure patterns.

It does not replace:

- guide updates
- template updates
- automated closure through checkers or smoke

Use this rule of thumb:

- do not add one-off explanations to the registry
- if repeated human reminder is needed but automation is not ready, prefer the registry
- if automation already exists, keep only a short registry summary and source linkage

## 6. Rollback And Exceptions

If a promotion later turns out to be based on a false positive, bad attribution, or a temporary tool defect, it may be downgraded.

Any downgrade must also record:

- why the original judgment no longer holds
- which artifact was downgraded or removed
- when the next review should happen
