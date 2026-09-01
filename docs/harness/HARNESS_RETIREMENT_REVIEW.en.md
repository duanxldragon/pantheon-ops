# Harness Retirement Review

Chinese version: [HARNESS_RETIREMENT_REVIEW.md](./HARNESS_RETIREMENT_REVIEW.md)

Type: Policy
Layer: platform
Status: Active

This policy defines when harness constraints should be downgraded, replaced, or removed instead of accumulating forever.

## 1. Goal

The harness exists to constrain the current probabilistic runtime, not to preserve every historical workaround permanently.

If an old rule no longer reduces failures and now mainly causes friction, it should enter retirement review.

## 2. Triggers

Run a retirement review when any of the following is true:

1. a major model, toolchain, or repo-shell upgrade happened
2. several consecutive tasks no longer needed an old workaround
3. the same rule repeatedly causes false positives, false blocking, or clear adoption friction
4. a manual reminder has been superseded by a better sensor or native capability

Minimum cadence:

- every method release review
- after every major model or tool upgrade

## 3. Common Retirement Candidates

Prioritize rules such as:

- cumbersome steps that existed only for older model context limits
- duplicate documentation rules now covered by a checker or newer contract
- constraints tightly bound to one tool implementation detail with no remaining shared value
- manual checklists that no longer help but still add process weight

## 4. Review Questions

For each candidate, answer at least:

1. what failure pattern was it originally protecting against?
2. does that failure pattern still materially exist?
3. is there now a lighter, more accurate, or more automated replacement?
4. what is the worst plausible regression risk if the rule is removed or downgraded?
5. if the rule stays, is the friction now larger than the value?

## 5. Allowed Outcomes

The review outcome must be one of:

- keep
- downgrade
- replace
- remove

Any outcome other than `keep` must also record rollback conditions.

## 6. Recording Requirements

Each retirement review should record at least:

- the rule or artifact under review
- the verdict: keep / downgrade / replace / remove
- the reasoning
- the linked change, review note, release note, or open task

Recommended update locations:

- `HARNESS_OPEN_TASKS.md`
- release notes
- the relevant policy, contract, or checker README

If a rule is removed but its failure pattern is still important, record what new control replaces it.
