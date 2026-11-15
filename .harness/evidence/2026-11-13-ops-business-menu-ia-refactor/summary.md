# Business menu IA execution evidence

## Coordination continuation — 2026-11-14 (freshly verified)

Claude coordinator re-verified the two previously-open local gaps on the active worktree and closed both:

- **Active adoption is now complete**: official `pantheon-base-v0.18.8` is active (`foundation-release.lock.json` commit `34428814a9efc997fd93d4434406a53df6460323`). r7 succeeded by normal file operations; earlier r5/r6 failures and rollback archives remain preserved. This supersedes the prior "blocked at v0.18.6" state.
- **Empty-schema double boot now PASSES**: against native MySQL/Redis, two consecutive boots (`autoMigrate=false`, formal SQL migrations to a fresh empty schema) both reached 248 `system_menu` rows and became ready (50.0s / 38.5s). The menu **content set is byte-identical** across boots (sorted set SHA-256 `88ac5d3af764487e4826e64d942ef42de170229164277b049118998965d2d8ef`). Assertions green: `bizscope` M+3C (`/business/bizscope/scopes|host-bindings|environments`) and `service-catalog` M+3C (`applications|services|instances`) present; retired paths `/business/business-scope[/:id]` and `/business/service` absent; `business/service/ServiceCatalog` component key absent; 0 orphan `system_role_menu` rows. Advisory (pre-existing, unrelated to IA): 19 `business.cmdb` F-permission rows are assigned new PK ids each boot; their content is unchanged, only id-order differs. Evidence: `double-boot.json`, retained logs at `.local/ops-acceptance/ia-double-boot-logs/20261114_*`.

Residual for this menu-IA packet is only human/visual: complete populated mobile/theme/state matrix and final functional/visual maintainer acceptance. The broader P0 real-server, production (D4) and release-renaming gates remain open and are tracked by the umbrella packet.

## Earlier continuation (superseded by r7 active install)

**Active installation remains blocked at v0.18.6.** Fresh official v0.18.8 candidate `.tmp/ops-v0188-install-r6` passes standard asset/assembler/dependency/overlay/drift checks. Independent source identity836backend/165frontend matches the prior race/unit/browser-validated r5. After60 attempted writes the exact tenant test file denied writing again; all165 planned preimages and the Git index were restored. The user authorized elevation, but this tool session has approval_policy=never; no elevated calls or ACL changes were used. The initial external-Go-cache failure and successful explicit task-cache retry are retained. Earlier r5 evidence below remains historical;87 redundant staging evidence copies were pruned only after matching canonical originals, which remain preserved. Base53-release naming normalization is remote-verified, with55 tag refs and all assets/states unchanged. Base publisher policy edits are local and uncommitted. Six P0 parents and all stated final acceptance gaps remain open.

[r6 installation readout](../2026-11-14-ops-latest-foundation-and-plans/verification/r6-final-installation-readout.json).

## Earlier r5 checkpoint

Status: **in-progress; implemented and scoped candidate validation passed; complete acceptance remains open**.

BizScope and Service each expose M+3C; BusinessEnvironment has paired migration000038, CRUD, parent lifecycle locking and permission-aware host bindings. ServiceCatalog was split into applications/services/instances with create and pagination. Application Delivery replaces the displayed ArgoCD name; custom roles require reauthorization after retired menu cleanup.

Final official v0.18.8 candidate frontend lint/prebuild/type-check/build and414 units pass. Actual isolated24-test browser run passes13 BizScope/7Service/4CMDB with unchanged final specs, zero retry/failure/skip/flaky. Native runner did not pass; authenticated local IPC/real Chromium CDP adaptation is explicit. Standard full backend race retry passes70 packages/3583 tests, with63 cached packages and7 fresh packages;6 external tests remain skipped. Prior disk-full failures and original targeted race/overlay receipts remain in [parent execution evidence](../2026-11-14-ops-latest-foundation-and-plans/summary.md).

Fifteen rendered images are in `visual/`. Six desktop/mobile lists and the environment create form are readable at the checked sizes; mobile is mainly empty samples and Application/Instance right columns need table scrolling. Application Delivery screenshots contain permission/loading failures and repeated toasts, so complete page function/visual approval is withheld. The complete mobile/data/theme matrix and real empty-database double boot are still open. Final independent review is [review.md](./review.md), and [commands.json](./commands.json) records passed, failed and unexecuted gaps without replacing history.

Active installation is blocked by Windows directory writes and was fully rolled back; root lock/report are v0.18.6. Candidate validation does not establish active adoption. Human final acceptance, complete parent P0 criteria and production signatures remain open. No shared override, waiver or original-fixture replay.
