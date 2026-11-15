```json
{
  "subject": "Base-owner-lifecycle-v1",
  "lane": "ARCH",
  "independent": true,
  "readOnly": true,
  "decision": "APPROVE",
  "architecturalStatus": "CLEAR",
  "decisionScope": "exact11 Base source/test/doc 的独立架构审批；不批准 Ops consumer implementation、candidate、runtime acceptance 或正式发行。",
  "sourceHashes": {
    "backend/pkg/contracts/businessauth/businessauth.go": "08152f22c3c3a67cc3fd665c0e347b3932e9ae2814c35c640e2d3276eb302412",
    "backend/pkg/contracts/business_tenant_groups.go": "c6afc7ea681a731cc9c70b9aef634d598b7d117bbce64a83ae00db1acb176b87",
    "backend/pkg/contracts/business_owner_lifecycle_test.go": "adecf0f315498b99784a93e4c9db0118d52bb61b36be4def4b8d73a783e61bde",
    "backend/modules/auth/login/business_authorization.go": "4c06cca9aca7f87c98908e7899ceb511eb72084707059c0727e76de4cb020874",
    "backend/modules/auth/login/business_owner_lifecycle_test.go": "a31dfff026325d3180c0c94ec940e1cc142ac3657692b818898b9ecfdaa69e23",
    "backend/modules/system/iam/tenant/tenant_service.go": "73900316ef39b6e54491101fe28a02acfe9c6d30ad0bce71d637dabab562a8db",
    "backend/modules/system/iam/tenant/tenant_owner_lifecycle_test.go": "983551c2e08e161ec4c30d05fba26873cebe1866a133c4d6a7460b3d66651de5",
    "docs/contracts/SYSTEM_AUTH_CONTRACT.md": "a4b9e528d5710396d2647d3015e985e0e9d49ae42c31d8e8b43bb01d7afa2d1f",
    "docs/contracts/SYSTEM_AUTH_CONTRACT.en.md": "e0760aa6b0c9ba3c08d01d877f3fac5137dd5e2f8c1b34da056c87b25cef8c83",
    "docs/contracts/TENANT_CONTRACT_V1.md": "48f35d5dbb56dc4393dcba37a06782ad92fe80646b63ffc4c8aed3293daf29c4",
    "docs/contracts/TENANT_CONTRACT_V1.en.md": "585383d91bd5a05770571dcdaadab857c22ec5ed919dfcc8ecc2e6cfabd6674b"
  },
  "sourceHashManifest": ".local/o4-owner-lifecycle-20261111/source-hashes.json",
  "sourceHashManifestSHA256": "3928967623d3426cbc11ad01a8164d98898b7fc993f6ee20f1b8909c8a6dcda0",
  "handoffSHA256": "c0718f2549b325ff17c791fe20b5f2698a6830c4a44380136262e67568699e9c",
  "verificationSHA256": "5ab7ae148a590adbe0cca34edfdda701a04726054e5366e0081824ba052b8c17",
  "approvedPlanSHA256": "9aaa0e53b43c040a4d04949693d3ef6ab4b086336fe953c3e04105400e4caaf7",
  "findings": [],
  "support": [
    {
      "claim": "旧 Reader 未扩签名；存在性端口独立于访问授权，false 仅在 nil error 时有权威含义。",
      "reference": "backend/pkg/contracts/businessauth/businessauth.go:47"
    },
    {
      "claim": "同一私有 adapter 只读取 authoritative actor/tenant metadata；nil、取消、actor0、schema/query failure 返回固定错误和空投影。",
      "reference": "backend/modules/auth/login/business_authorization.go:32"
    },
    {
      "claim": "getter 同步复用既有注册 provider，typed nil 拒绝、preseal 可用、旧 Reader seal 兼容，没有新增 registry。",
      "reference": "backend/pkg/contracts/business_tenant_groups.go:72"
    },
    {
      "claim": "UpdateTenant 当前状态校验与全部字段写入共享行锁事务，DeleteTenant 采用同一 tenant row 锁；消除 stale activation revival 和部分字段提交。",
      "references": [
        "backend/modules/system/iam/tenant/tenant_service.go:474",
        "backend/modules/system/iam/tenant/tenant_service.go:612"
      ]
    }
  ],
  "actualChecks": {
    "sourceHashReadback": { "count": 11, "mismatches": 0 },
    "performed": [
      "读取 workspace/Base AGENTS、code-review skill、父 task 和 approved plan 相关约束。",
      "打开 source/test/doc 变更及关键跨写路径；检查接口、注册、锁顺序、no-op、rollback 和 postcommit revocation。",
      "读取作者 RED/GREEN、focused race、最终四包 regression、native evidence、handoff 和 verification。",
      "CodeGraph CLI status/context/callers；index 有 pending changes，关键结论使用另行打开的当前文件。",
      "git diff --check exit0。"
    ],
    "runtimeOrTestsRerunByReviewer": false,
    "formatRedeliveryOnly": true
  },
  "authorChecksInspected": {
    "red": "实际 MySQL 中1和6 delayed activation writers 复活 deleted row；晚字段失败留下早字段提交。",
    "green": "相同场景通过，实际 InnoDB NOWAIT3572 锁探针。",
    "focusedRace": "四个受影响包 count3 PASS。",
    "targetedFullRegression": "最终四包 CGO race PASS。",
    "vetBuild": "作者记录最终 source vet/build 均 exit0。",
    "nativeRuntime": "作者记录 preseal capability、disabled global actor、tenant no-revival、retained metadata 和 synthetic ciphertext。",
    "reference": ".local/o4-owner-lifecycle-20261111/handoff.md:17"
  },
  "preservedLimits": [
    {
      "claim": "Subordinate lifecycle windows 未修复：AddMember 和 ProvisionTenant 的既有检查/写入窗口仍存在；这些已检查路径不写 tenants.status，不构成本次 owner ID 复活路径。",
      "reference": ".local/o4-owner-lifecycle-20261111/handoff.md:28"
    },
    {
      "claim": "Plan consistency 未修复：锁前 PlanStore cache validation 不提供与并发 plan-disable 的跨实体原子保证；保留既有模型。提前查询避免单连接事务内再次取连接。",
      "references": [
        "backend/modules/system/iam/tenant/tenant_service.go:465",
        "backend/modules/system/iam/tenant/tenant_plan_store.go:321"
      ]
    },
    {
      "claim": "有限检查的 User update 不写 deleted_at，DeleteUser 使用删除操作；不扩大 actor 实现，也不宣称 repository-wide actor 生命周期审计完成。",
      "references": [
        "backend/modules/system/iam/user/user_service.go:460",
        "backend/modules/system/iam/user/user_service.go:729"
      ]
    }
  ],
  "consumerDesign": {
    "implemented": false,
    "decision": "COMMENT",
    "architecturalStatus": "WATCH",
    "affectsBaseSourceVerdict": false,
    "sha256": "5fe4ae940b3aef16c5b14764a3552728ba8e43e8012736a904e74153b957f1c2",
    "watch": [
      {
        "id": "CONSUMER-DESIGN-POOL",
        "severity": "MEDIUM",
        "concern": "锁内 public reader 使用独立 runtime.db connection；pool=1 或事务占满连接时，deadline 保证结束但不能自行证明 cleanup liveness。",
        "acceptance": "新 consumer attempt 必须验证窄池/事务占满竞争下有界结束、释放锁与连接、游标推进，并在压力解除后的后续轮次访问和清理较后 orphan；若支持 pool=1，明确解决读取容量或经审查的公共 seam。error unknown 始终保留。",
        "reference": "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r1.md:51"
      },
      {
        "id": "CONSUMER-DESIGN-RETENTION",
        "severity": "LOW",
        "concern": "unknown owner 暂缓整个 row 的 expiry/source cleanup 是合理保留；持续故障会延长 retention，保留 guards 也有累积成本。",
        "acceptance": "记录保留成本；用有限 cycleMax、live/unknown 游标推进、table rotation、wrap retry 证明后续可达；固定安全 aggregate failure，不泄露 owner/SQL/credential，不 silent purge。",
        "reference": "pantheon-ops/.local/ops-acceptance/o4-backend-cf-current/lifecycle-consumer-design-20261111-r1.md:48"
      }
    ]
  },
  "completion": {
    "baseScopedArchApproved": true,
    "consumerImplementationApproved": false,
    "runtimeAcceptanceApproved": false,
    "fullRepositoryGatesVerified": false,
    "newCandidateApproved": false,
    "pairedAssemblyApproved": false,
    "officialReleaseConsumptionApproved": false,
    "P0Accepted": false,
    "finalHumanAcceptanceComplete": false
  },
  "stopCondition": "Base exact11 ARCH APPROVE/CLEAR 保持；consumer DESIGN WATCH 单独传递。full gates、新 candidate、consumer CODE/ARCH/runtime、P0 与最终验收仍由 Root 收口。"
}
```
