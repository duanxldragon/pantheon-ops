# Review: T14 phase 1

Independent CODE/ARCH review: **not performed**. Recorded gap, not an approval.

## What a reviewer should check

1. `000043` up script: the `@old` discovery block (unique, single-column, by column set) and that dropping it cannot hit a foreign key.
2. That the new key names equal the names in the model tags (`uk_argocd_server_name` etc.), so AutoMigrate and the migration converge instead of creating two indexes.
3. That keys without `deleted_at` is the intended choice (it preserves the old reuse semantics).
4. Whether phase 2 should cover core and extension modules together (finding 2).
