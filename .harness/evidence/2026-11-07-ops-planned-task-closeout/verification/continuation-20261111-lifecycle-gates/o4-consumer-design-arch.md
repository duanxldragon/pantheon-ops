```json
{
  "subject": "O4-owner-lifecycle-consumer-design-r2",
  "lane": "ARCH",
  "independent": true,
  "decision": "APPROVE",
  "architecturalStatus": "CLEAR",
  "decisionScope": "Exact r2 design only. Implementation may proceed after Root releases a reviewed Base candidate; this is not consumer-source, assembly, runtime, release or P0 approval.",
  "designPath": ".local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md",
  "designSHA256": "e9fb90cd3e00b449744ad69c2ae3844e72aca9dc5b5a31e2ddb80fd527f44732",
  "receiptPath": ".local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2-receipt.json",
  "receiptSHA256": "d3010c7ed712d5ff58733bbda624861ae8f37db933229fd2ca9c7ce6eecfface",
  "sourceBinding": {
    "sourceFileCount": 49,
    "initialMatches": 49,
    "finalMatches": 49,
    "mismatches": [],
    "sourceStable": true,
    "meaning": "Hash equality with the receipt, not a full review of all49 source bodies."
  },
  "findings": [],
  "rootCause": {
    "claim": "A transaction already holds one connection while the authoritative public reader obtains another through runtime.db. A finite pool1 therefore cannot support the locked fresh read; larger configured limits do not reserve spare capacity.",
    "references": [
      "pantheon-base/backend/modules/auth/login/business_authorization.go:37",
      "pantheon-ops/backend/modules/business/observability/query_workspace_service.go:77",
      "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:23"
    ]
  },
  "support": [
    {
      "claim": "The design resolves the pool1 support ambiguity through fixed startup rejection before business bootstrap/seed/routes. It does not alter the global pool, copy lifecycle SQL or add a private-system dependency.",
      "references": [
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:25",
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:34",
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:58"
      ]
    },
    {
      "claim": "Capacity observation is explicitly not a reservation. A competing acquisition must produce bounded unknown, untouched metadata, rollback and connection/lock release; liveness is required after pressure ends.",
      "references": [
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:38",
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:40",
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:68"
      ]
    },
    {
      "claim": "The finite ceiling, visited-ID advancement and table rotation address low-ID starvation without falsely claiming progress when enumeration cannot obtain a connection.",
      "references": [
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:44",
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:46"
      ]
    },
    {
      "claim": "The locked fresh read remains mandatory. Owner/source/metadata identity changes are skipped instead of acquiring additional locks out of order. This matches the existing owner-then-ascending-source transaction protocol.",
      "references": [
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:17",
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:48",
        "pantheon-ops/backend/modules/business/observability/query_workspace_service.go:74"
      ]
    },
    {
      "claim": "Existence remains distinct from authorization. Tenant0 still requires a real actor; disabled actors, suspended/archived tenants and membership/session/grant revocation do not authorize purge. Errors and partial zero facts are unknown.",
      "references": [
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:13",
        "pantheon-base/backend/pkg/contracts/businessauth/businessauth.go:52",
        "pantheon-base/backend/modules/auth/login/business_authorization.go:32"
      ]
    },
    {
      "claim": "The prescribed error branches reuse actual fixed public errors and avoid SQL/provider text leakage.",
      "references": [
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:36",
        "pantheon-base/backend/pkg/contracts/business_tenant_groups.go:72",
        "pantheon-ops/backend/modules/business/observability/query_workspace_maintenance.go:17"
      ]
    }
  ],
  "strongestCounterargument": {
    "claim": "A pool2 may pass the spare-capacity check and still lose its remaining slot before the locked read. Repeated contention can delay cleanup even though startup succeeds. Removing the second read would reduce this dependency, but the boolean contract does not establish universal terminal absence for both owner types.",
    "assessment": "r2 addresses the counterexample explicitly with conservative rollback, conditional liveness and an actual post-observation acquisition barrier test. This is adequate for design approval; no liveness result exists yet.",
    "references": [
      "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:19",
      "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:38",
      "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:68"
    ]
  },
  "preservedTradeoffs": [
    {
      "claim": "Unknown retains the entire row, including expired history/source orphans. Persistent failure may extend retention beyond30days; aggregate retained storage is not absolutely capped by write quotas, and budget guards accumulate.",
      "reference": "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:52",
      "assessment": "Explicit conservative design cost, not an unresolved design blocker or a runtime acceptance."
    },
    {
      "claim": "Finite pool1 is unsupported at startup. MaxOpen0 and >=2 provide configured capability only, with no guarantee under permanent server failure or adversarial occupancy.",
      "reference": "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:27"
    }
  ],
  "requiredImplementationEvidence": [
    {
      "case": "Exact reviewed paired composition",
      "requirement": "Verify the registered authoritative public reader and Ops transaction share the intended actual pool in the new reviewed Base candidate/startup fixture. No overlays, fake-success provider or sealed-tree edits.",
      "reference": "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:64"
    },
    {
      "case": "Pool1 RED and startup GREEN",
      "requirement": "Actual locked public acquisition timeout retains row and releases locks/connection. Put the new opsinit guard before initialization effects, including RegisterQueryPolicies/development bootstrap; assert exact fixed errors and zero business seed/route/maintenance effects, plus direct-module protection.",
      "references": [
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:66",
        "pantheon-ops/backend/modules/business/opsinit/opsinit.go:26",
        "pantheon-ops/backend/modules/business/observability/observability.go:95"
      ]
    },
    {
      "case": "Healthy minimum and unbounded configuration",
      "requirement": "Real pool2 completes both public reads and deletes a reliable later orphan; guards and legacy ciphertext/data digests remain. Test MaxOpen0 separately without claiming physical capacity.",
      "reference": "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:67"
    },
    {
      "case": "Post-observation slot抢占",
      "requirement": "Use the real reader with a controlled second-slot acquisition barrier. Prove safe timeout/rollback, unchanged row, and release through actual SQL stats plus a row/lock probe; timeout alone is insufficient.",
      "reference": "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:68"
    },
    {
      "case": "Full occupancy and recovery",
      "requirement": "All slots held: bounded error, no deletion, cursor unchanged. One slot released: enumeration advances visited unavailable records. All pressure released: bounded finite passes delete a higher-ID orphan and wrap to retry earlier unknown rows.",
      "reference": "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:69"
    },
    {
      "case": "Runtime2→1→2",
      "requirement": "Only the fixture pool is changed. At1 retain rows and advance only successfully enumerated visits, without doomed locked reads or product pool setters. At2 demonstrate both-table rotation, later orphan deletion and earlier-row wrap retry.",
      "reference": "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:70"
    },
    {
      "case": "Fairness, isolation and fresh facts",
      "requirement": "Over100 low-ID live/unknown rows, continuous high-ID inserts, sparse/removed IDs, exhausted first-table deadline, overlap guard, owner semantic matrix, changed locked identity/source, initial absent→locked live/unknown, expiry renewal and concurrent writer/source purge lock ordering.",
      "references": [
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:46",
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:71",
        "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:72"
      ]
    },
    {
      "case": "Current implementation gates",
      "requirement": "New source handoff includes opsinit guard changes; paired CGO race, vet/build, startup checks and hash readback. Follow with independent consumer CODE/ARCH, new full assembly/race and real public product acceptance.",
      "reference": "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r2.md:73"
    }
  ],
  "actualValidation": {
    "designAndReceiptHashes": "PASS",
    "initialAndFinalSourceReadback": "49/49 PASS",
    "performed": [
      "Read workspace/Base/Ops AGENTS and installed architect role surface.",
      "Read r2 design/receipt and preceding independent Base ARCH report.",
      "Opened current public reader, public lifecycle interface/getter, maintenance, owner/source transaction and Ops initialization code.",
      "Inspected current Base server auth-before-business composition call sites."
    ],
    "runtimeTestsRerun": false,
    "filesWritten": false,
    "remoteOperations": false,
    "limitation": "New reviewed paired candidate and actual pressure-release-liveness fixture do not exist in this design review; they remain implementation evidence gates."
  },
  "consumerImplementationApproved": false,
  "OpsRuntimeApproved": false,
  "pairedAssemblyApproved": false,
  "officialReleaseConsumptionApproved": false,
  "P0": false,
  "stopCondition": "Exact r2 design ARCH APPROVE/CLEAR. Root may release implementation only with its reviewed Base candidate; all implementation/runtime/full-source/final acceptance gates remain open."
}
```
