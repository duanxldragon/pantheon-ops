import fs from 'node:fs';
import path from 'node:path';

export const TASK_MANIFEST_ROOT = '.harness/tasks';
export const TASK_MANIFEST_FILE = 'manifest.json';
export const TASK_ARCHIVE_ROOT = '.harness/archive';
const ARCHIVE_TASKS_DIRNAME = 'tasks';
const ARCHIVE_EVIDENCE_DIRNAME = 'evidence';
const EVIDENCE_ROOT = '.harness/evidence';

// Canonical lifecycle vocabulary for the optional `status` field on
// .harness/tasks/<task-id>/manifest.json. The field is metadata, not a
// required field, but when present it must use one of these values so packet
// state stays greppable across the repo. Before this check existed the
// vocabulary had drifted to `done` / `complete` / `complete-with-explicit-gates`
// (2026-09-23), which would have silently broken any future `status ===
// 'completed'` query.
export const TASK_MANIFEST_STATUSES = new Set([
  'planned',
  'in-progress',
  'completed',
  'abandoned',
]);

function isNonEmptyString(value) {
  return typeof value === 'string' && value.trim() !== '';
}

function assertNonEmptyString(value, label, errors) {
  if (!isNonEmptyString(value)) {
    errors.push(`${label} must be a non-empty string.`);
  }
}

function assertStringArray(value, label, errors) {
  if (!Array.isArray(value)) {
    errors.push(`${label} must be an array of strings.`);
    return;
  }
  value.forEach((entry, index) => {
    if (!isNonEmptyString(entry)) {
      errors.push(`${label}[${index}] must be a non-empty string.`);
    }
  });
}

function assertOptionalStringArray(value, label, errors) {
  if (value === undefined || value === null) {
    return;
  }
  assertStringArray(value, label, errors);
}

function assertObject(value, label, errors) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    errors.push(`${label} must be an object.`);
    return false;
  }
  return true;
}

export function normalizeRepoRelativePath(value) {
  return String(value ?? '')
    .trim()
    .replace(/^`+/, '')
    .replace(/`+$/, '')
    .replaceAll('\\', '/')
    .replace(/^\.\/+/, '')
    .replace(/^\/+/, '');
}

export function ensureTrailingSlash(value) {
  return value.endsWith('/') ? value : `${value}/`;
}

export function buildTaskManifestPath(taskId) {
  return `${TASK_MANIFEST_ROOT}/${taskId}/${TASK_MANIFEST_FILE}`;
}

export function listTaskManifestPaths(rootDir) {
  const manifestRoot = path.join(rootDir, TASK_MANIFEST_ROOT);
  if (!fs.existsSync(manifestRoot)) {
    return [];
  }

  return fs
    .readdirSync(manifestRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => buildTaskManifestPath(entry.name))
    .filter((manifestPath) => {
      const absolutePath = resolveRepoPath(rootDir, manifestPath);
      return absolutePath && fs.existsSync(absolutePath);
    })
    .sort((left, right) => left.localeCompare(right));
}

export function extractTaskIdFromManifestPath(value) {
  const match = normalizeRepoRelativePath(value).match(
    /^\.harness\/(?:archive\/\d{4}-\d{2}\/)?tasks\/(.+)\/manifest\.json$/i,
  );
  return match ? match[1] : null;
}

function listArchiveMonths(rootDir) {
  const archiveRoot = path.join(rootDir, TASK_ARCHIVE_ROOT);
  if (!fs.existsSync(archiveRoot)) {
    return [];
  }

  return fs
    .readdirSync(archiveRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .filter((month) => /^\d{4}-\d{2}$/.test(month))
    .sort((left, right) => right.localeCompare(left));
}

// Task archives keep manifests under `.harness/archive/<YYYY-MM>/tasks/...`.
// Docs and evidence reference the canonical `.harness/tasks/<task-id>` address,
// so once a task is archived the physical manifest moves while the reference
// stays valid. Resolution falls back to the newest archive month instead of
// forcing every archived doc to be rewritten each archiving round.
export function findArchivedTaskManifestPath(rootDir, taskId) {
  if (!taskId || typeof taskId !== 'string' || taskId.trim() === '') {
    return null;
  }

  for (const month of listArchiveMonths(rootDir)) {
    const candidate = path.join(
      rootDir,
      TASK_ARCHIVE_ROOT,
      month,
      ARCHIVE_TASKS_DIRNAME,
      taskId,
      TASK_MANIFEST_FILE,
    );
    if (fs.existsSync(candidate)) {
      return `${TASK_ARCHIVE_ROOT}/${month}/${ARCHIVE_TASKS_DIRNAME}/${taskId}/${TASK_MANIFEST_FILE}`;
    }
  }

  return null;
}

// Live trees that move wholesale into `.harness/archive/<YYYY-MM>/`, keyed by
// the live root a reference points at and the archive directory it lands in.
const ARCHIVABLE_TREES = [
  { liveRoot: `${TASK_MANIFEST_ROOT}/`, archiveDirname: ARCHIVE_TASKS_DIRNAME },
  { liveRoot: `${EVIDENCE_ROOT}/`, archiveDirname: ARCHIVE_EVIDENCE_DIRNAME },
];

export function findArchivedReferencePath(rootDir, relativePath) {
  const normalized = normalizeRepoRelativePath(relativePath);
  const tree = ARCHIVABLE_TREES.find((entry) => normalized.startsWith(entry.liveRoot));
  if (!tree) {
    return null;
  }

  const suffix = normalized.slice(tree.liveRoot.length);
  const separator = suffix.indexOf('/');
  const taskId = separator === -1 ? suffix : suffix.slice(0, separator);
  if (!taskId) {
    return null;
  }
  const remainder = separator === -1 ? '' : suffix.slice(separator + 1);

  for (const month of listArchiveMonths(rootDir)) {
    const candidate = `${TASK_ARCHIVE_ROOT}/${month}/${tree.archiveDirname}/${taskId}/${remainder}`;
    const absolutePath = resolveRepoPath(rootDir, candidate);
    if (absolutePath && fs.existsSync(absolutePath)) {
      return candidate;
    }
  }

  return null;
}

// Returns the spelling of a task/evidence reference that actually exists, so
// callers can report the resolved location instead of a false "missing" warning.
export function resolveArchiveAwarePath(rootDir, relativePath) {
  const direct = resolveRepoPath(rootDir, relativePath);
  if (direct && fs.existsSync(direct)) {
    return normalizeRepoRelativePath(relativePath);
  }
  return findArchivedReferencePath(rootDir, relativePath);
}

export function resolveRepoPath(rootDir, relativePath) {
  const normalized = normalizeRepoRelativePath(relativePath);
  if (
    !normalized ||
    normalized.includes('://') ||
    path.isAbsolute(normalized) ||
    normalized.startsWith('..')
  ) {
    return null;
  }
  return path.join(rootDir, normalized);
}

export function validateTaskManifest(payload, options = {}) {
  const errors = [];
  const manifestPath = normalizeRepoRelativePath(options.manifestPath ?? '');
  const manifestTaskId = extractTaskIdFromManifestPath(manifestPath);

  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    throw new Error('task manifest must be a JSON object');
  }

  assertNonEmptyString(payload.taskId, 'taskManifest.taskId', errors);
  assertNonEmptyString(payload.goal, 'taskManifest.goal', errors);
  assertNonEmptyString(payload.primaryLayer, 'taskManifest.primaryLayer', errors);

  if (manifestTaskId && payload.taskId !== manifestTaskId) {
    errors.push(
      `taskManifest.taskId must match manifest path task id "${manifestTaskId}".`,
    );
  }

  if (payload.scope === null || typeof payload.scope !== 'object' || Array.isArray(payload.scope)) {
    errors.push('taskManifest.scope must be an object.');
  } else {
    assertStringArray(payload.scope.in, 'taskManifest.scope.in', errors);
    assertStringArray(payload.scope.out, 'taskManifest.scope.out', errors);
  }

  assertOptionalStringArray(
    payload.dependencyLayers,
    'taskManifest.dependencyLayers',
    errors,
  );

  if ('implementationNotes' in payload) {
    assertStringArray(
      payload.implementationNotes,
      'taskManifest.implementationNotes',
      errors,
    );
  }

  if ('taskDoc' in payload && payload.taskDoc !== null && !isNonEmptyString(payload.taskDoc)) {
    errors.push('taskManifest.taskDoc must be a non-empty string when present.');
  }

  if ('title' in payload && payload.title !== null && !isNonEmptyString(payload.title)) {
    errors.push('taskManifest.title must be a non-empty string when present.');
  }

  assertOptionalStringArray(
    payload.contractAnchors,
    'taskManifest.contractAnchors',
    errors,
  );

  if ('harnessProfile' in payload && payload.harnessProfile !== null) {
    if (assertObject(payload.harnessProfile, 'taskManifest.harnessProfile', errors)) {
      if ('template' in payload.harnessProfile) {
        assertNonEmptyString(
          payload.harnessProfile.template,
          'taskManifest.harnessProfile.template',
          errors,
        );
      }
      if ('overlay' in payload.harnessProfile) {
        assertNonEmptyString(
          payload.harnessProfile.overlay,
          'taskManifest.harnessProfile.overlay',
          errors,
        );
      }
      if ('qualityProfile' in payload.harnessProfile) {
        assertNonEmptyString(
          payload.harnessProfile.qualityProfile,
          'taskManifest.harnessProfile.qualityProfile',
          errors,
        );
      }
      if ('portableFailureClass' in payload.harnessProfile) {
        assertNonEmptyString(
          payload.harnessProfile.portableFailureClass,
          'taskManifest.harnessProfile.portableFailureClass',
          errors,
        );
      }
      if ('ownerLayer' in payload.harnessProfile) {
        assertNonEmptyString(
          payload.harnessProfile.ownerLayer,
          'taskManifest.harnessProfile.ownerLayer',
          errors,
        );
      }
      if ('coverageDimensions' in payload.harnessProfile) {
        assertStringArray(
          payload.harnessProfile.coverageDimensions,
          'taskManifest.harnessProfile.coverageDimensions',
          errors,
        );
      }
    }
  }

  if ('expectedFiles' in payload && payload.expectedFiles !== null) {
    if (assertObject(payload.expectedFiles, 'taskManifest.expectedFiles', errors)) {
      for (const [key, label] of [
        ['create', 'taskManifest.expectedFiles.create'],
        ['modify', 'taskManifest.expectedFiles.modify'],
        ['doNotTouch', 'taskManifest.expectedFiles.doNotTouch'],
      ]) {
        if (key in payload.expectedFiles) {
          assertStringArray(payload.expectedFiles[key], label, errors);
        }
      }
    }
  }

  if (
    !payload.linkage ||
    typeof payload.linkage !== 'object' ||
    Array.isArray(payload.linkage)
  ) {
    errors.push('taskManifest.linkage must be an object.');
  } else {
    assertNonEmptyString(
      payload.linkage.evidenceDir,
      'taskManifest.linkage.evidenceDir',
      errors,
    );
    assertNonEmptyString(
      payload.linkage.reviewFile,
      'taskManifest.linkage.reviewFile',
      errors,
    );
    assertNonEmptyString(
      payload.linkage.changeRef,
      'taskManifest.linkage.changeRef',
      errors,
    );
    assertStringArray(
      payload.linkage.planRefs,
      'taskManifest.linkage.planRefs',
      errors,
    );
    if (
      'summaryFile' in payload.linkage &&
      payload.linkage.summaryFile !== null &&
      !isNonEmptyString(payload.linkage.summaryFile)
    ) {
      errors.push(
        'taskManifest.linkage.summaryFile must be a non-empty string when present.',
      );
    }
  }

  if ('structuralScope' in payload && payload.structuralScope !== null) {
    if (
      typeof payload.structuralScope !== 'object' ||
      Array.isArray(payload.structuralScope)
    ) {
      errors.push('taskManifest.structuralScope must be an object when present.');
    } else {
      for (const [key, label] of [
        ['affectedSubgraph', 'taskManifest.structuralScope.affectedSubgraph'],
        ['boundaryCrossings', 'taskManifest.structuralScope.boundaryCrossings'],
        ['riskNodes', 'taskManifest.structuralScope.riskNodes'],
        ['graphFocus', 'taskManifest.structuralScope.graphFocus'],
      ]) {
        if (key in payload.structuralScope) {
          assertStringArray(payload.structuralScope[key], label, errors);
        }
      }
    }
  }

  if ('methodReadiness' in payload && payload.methodReadiness !== null) {
    if (assertObject(payload.methodReadiness, 'taskManifest.methodReadiness', errors)) {
      for (const [key, label] of [
        ['consumerSpecificControls', 'taskManifest.methodReadiness.consumerSpecificControls'],
        ['requiredSensors', 'taskManifest.methodReadiness.requiredSensors'],
        ['requiredEvidence', 'taskManifest.methodReadiness.requiredEvidence'],
        ['deferredCodeIssues', 'taskManifest.methodReadiness.deferredCodeIssues'],
      ]) {
        if (key in payload.methodReadiness) {
          assertStringArray(payload.methodReadiness[key], label, errors);
        }
      }
      if ('ratchetDecision' in payload.methodReadiness) {
        assertNonEmptyString(
          payload.methodReadiness.ratchetDecision,
          'taskManifest.methodReadiness.ratchetDecision',
          errors,
        );
      }
    }
  }

  if ('deliveryGovernance' in payload && payload.deliveryGovernance !== null) {
    if (assertObject(payload.deliveryGovernance, 'taskManifest.deliveryGovernance', errors)) {
      for (const [key, label] of [
        ['designGate', 'taskManifest.deliveryGovernance.designGate'],
        ['developmentGate', 'taskManifest.deliveryGovernance.developmentGate'],
        ['qaAcceptanceGate', 'taskManifest.deliveryGovernance.qaAcceptanceGate'],
        ['githubGovernanceGate', 'taskManifest.deliveryGovernance.githubGovernanceGate'],
      ]) {
        if (key in payload.deliveryGovernance) {
          assertStringArray(payload.deliveryGovernance[key], label, errors);
        }
      }
    }
  }

  if ('executionRoles' in payload && payload.executionRoles !== null) {
    if (assertObject(payload.executionRoles, 'taskManifest.executionRoles', errors)) {
      if ('implementerPosture' in payload.executionRoles) {
        assertNonEmptyString(
          payload.executionRoles.implementerPosture,
          'taskManifest.executionRoles.implementerPosture',
          errors,
        );
      }
      if ('reviewerPosture' in payload.executionRoles) {
        assertStringArray(
          payload.executionRoles.reviewerPosture,
          'taskManifest.executionRoles.reviewerPosture',
          errors,
        );
      }
    }
  }

  if ('verificationPlan' in payload && payload.verificationPlan !== null) {
    if (assertObject(payload.verificationPlan, 'taskManifest.verificationPlan', errors)) {
      for (const [key, label] of [
        ['commands', 'taskManifest.verificationPlan.commands'],
        ['runtimeEvidence', 'taskManifest.verificationPlan.runtimeEvidence'],
      ]) {
        if (key in payload.verificationPlan) {
          assertStringArray(payload.verificationPlan[key], label, errors);
        }
      }
      if ('visualEvidence' in payload.verificationPlan && payload.verificationPlan.visualEvidence !== null) {
        if (
          assertObject(
            payload.verificationPlan.visualEvidence,
            'taskManifest.verificationPlan.visualEvidence',
            errors,
          )
        ) {
          for (const [key, label] of [
            ['viewports', 'taskManifest.verificationPlan.visualEvidence.viewports'],
            ['states', 'taskManifest.verificationPlan.visualEvidence.states'],
          ]) {
            if (!(key in payload.verificationPlan.visualEvidence)) {
              errors.push(`${label} is required.`);
              continue;
            }
            assertStringArray(payload.verificationPlan.visualEvidence[key], label, errors);
            if (
              Array.isArray(payload.verificationPlan.visualEvidence[key]) &&
              payload.verificationPlan.visualEvidence[key].length === 0
            ) {
              errors.push(`${label} must include at least one value.`);
            }
          }
          if ('routes' in payload.verificationPlan.visualEvidence) {
            assertStringArray(
              payload.verificationPlan.visualEvidence.routes,
              'taskManifest.verificationPlan.visualEvidence.routes',
              errors,
            );
          }
        }
      }
    }
  }

  if ('runtimeSensitive' in payload && typeof payload.runtimeSensitive !== 'boolean') {
    errors.push('taskManifest.runtimeSensitive must be a boolean when present.');
  }

  if ('status' in payload && payload.status !== null) {
    assertNonEmptyString(payload.status, 'taskManifest.status', errors);
    if (
      isNonEmptyString(payload.status) &&
      !TASK_MANIFEST_STATUSES.has(payload.status.trim())
    ) {
      errors.push(
        `taskManifest.status must be one of: ${Array.from(TASK_MANIFEST_STATUSES).join(', ')} (got "${payload.status}").`,
      );
    }
  }

  for (const [key, label] of [
    ['evidenceRequired', 'taskManifest.evidenceRequired'],
    ['humanGates', 'taskManifest.humanGates'],
    ['completionChecklist', 'taskManifest.completionChecklist'],
  ]) {
    if (key in payload) {
      assertStringArray(payload[key], label, errors);
    }
  }

  if (errors.length > 0) {
    throw new Error(errors.join('\n'));
  }

  return payload;
}

export function readTaskManifest(rootDir, reference) {
  const normalizedReference = normalizeRepoRelativePath(reference);
  const referenceTaskId = extractTaskIdFromManifestPath(normalizedReference);
  const manifestPath = referenceTaskId
    ? normalizedReference
    : buildTaskManifestPath(normalizedReference);
  const absolutePath = resolveRepoPath(rootDir, manifestPath);

  const resolvedPath =
    absolutePath && fs.existsSync(absolutePath)
      ? manifestPath
      : findArchivedTaskManifestPath(rootDir, referenceTaskId ?? normalizedReference) ?? manifestPath;
  const resolvedAbsolute = resolveRepoPath(rootDir, resolvedPath);

  if (!resolvedAbsolute || !fs.existsSync(resolvedAbsolute)) {
    throw new Error(`task manifest does not exist: ${manifestPath}`);
  }

  let payload;
  try {
    payload = JSON.parse(fs.readFileSync(resolvedAbsolute, 'utf8'));
  } catch (error) {
    throw new Error(`task manifest is not valid JSON: ${resolvedPath}: ${error.message}`);
  }

  return {
    path: resolvedPath,
    absolutePath: resolvedAbsolute,
    payload: validateTaskManifest(payload, { manifestPath: resolvedPath }),
  };
}

export function deriveTaskContextFromManifest(manifest) {
  const goal = isNonEmptyString(manifest.goal) ? manifest.goal.trim() : '';
  const scopeIn = Array.isArray(manifest.scope?.in)
    ? manifest.scope.in.map((entry) => String(entry).trim()).filter(Boolean)
    : [];
  const scopeOut = Array.isArray(manifest.scope?.out)
    ? manifest.scope.out.map((entry) => String(entry).trim()).filter(Boolean)
    : [];
  const implementationNotes = Array.isArray(manifest.implementationNotes)
    ? manifest.implementationNotes.map((entry) => String(entry).trim()).filter(Boolean)
    : [];
  const changeSummary = [...new Set([goal, ...scopeIn].filter(Boolean))].slice(0, 6);
  const ownershipSummary = implementationNotes[0] ?? '';
  const outOfScopeSummary =
    scopeOut.length === 0
      ? ''
      : scopeOut.length === 1
        ? scopeOut[0]
        : `Out of scope for this change: ${scopeOut.join('; ')}`;

  return {
    changeSummary,
    ownershipSummary,
    outOfScopeSummary,
  };
}
