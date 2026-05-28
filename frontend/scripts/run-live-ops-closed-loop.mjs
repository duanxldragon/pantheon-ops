const apiOrigin = process.env.PANTHEON_API_ORIGIN || 'http://127.0.0.1:8080';
const apiBaseUrl = `${apiOrigin}/api/v1`;

const adminUsername = process.env.PANTHEON_SMOKE_ADMIN_USERNAME || 'admin';
const adminPassword = process.env.PANTHEON_SMOKE_ADMIN_PASSWORD || '123456';

const hostIP = requiredEnv('PANTHEON_LIVE_HOST_IP');
const sshUser = requiredEnv('PANTHEON_LIVE_SSH_USER');
const sshPassword = requiredEnv('PANTHEON_LIVE_SSH_PASSWORD');
const hostFingerprint = requiredEnv('PANTHEON_LIVE_HOST_FINGERPRINT');

const componentKey = (process.env.PANTHEON_LIVE_COMPONENT || 'nginx').trim().toLowerCase();
const componentCatalog = {
  nginx: {
    packageName: 'nginx',
    packageVersion: '1.30.2',
    templateCode: 'nginx_systemd',
    templateName: 'nginx_lifecycle',
    parameters: {
      installRoot: '/data/nginx',
      serviceName: 'nginx',
    },
  },
  redis: {
    packageName: 'redis',
    packageVersion: '7.2.5',
    templateCode: 'redis_systemd',
    templateName: 'redis_lifecycle',
    parameters: {
      installRoot: '/data/redis',
      dataRoot: '/data/redis/data',
      serviceName: 'redis',
      port: '6379',
      requirePassword: 'Redis_123',
    },
  },
  minio: {
    packageName: 'minio',
    packageVersion: '2025-05-24',
    templateCode: 'minio_systemd',
    templateName: 'minio_lifecycle',
    parameters: {
      installRoot: '/data/minio',
      dataRoot: '/data/minio/data',
      serviceName: 'minio',
      apiPort: '9000',
      consolePort: '9001',
      rootUser: 'minioadmin',
      rootPassword: 'Minio_123',
    },
  },
  mysql: {
    packageName: 'mysql',
    packageVersion: '8.0.39',
    templateCode: 'mysql_systemd',
    templateName: 'mysql_lifecycle',
    requiresSource: true,
    parameters: {
      installRoot: '/data/mysql',
      dataRoot: '/data/mysql/data',
      serviceName: 'mysqld',
      port: '3306',
      rootPassword: 'Mysql_123',
    },
  },
  harbor: {
    packageName: 'harbor',
    packageVersion: '2.11.1',
    templateCode: 'harbor_offline',
    templateName: 'harbor_lifecycle',
    requiresSource: true,
    parameters: {
      installRoot: '/data/harbor',
      dataRoot: '/data/harbor/data',
      hostname: 'harbor.local',
      httpPort: '8088',
      adminPassword: 'Harbor_123',
    },
  },
};

const component = componentCatalog[componentKey];
if (!component) {
  throw new Error(`unsupported component: ${componentKey}`);
}

const bizScopeCode = process.env.PANTHEON_LIVE_BIZSCOPE_CODE || 'his';
const bizScopeName = process.env.PANTHEON_LIVE_BIZSCOPE_NAME || 'his';
const bizScopeEnvironment = process.env.PANTHEON_LIVE_BIZSCOPE_ENV || 'dev';
const bizScopeStatus = process.env.PANTHEON_LIVE_BIZSCOPE_STATUS || 'active';

const parentGroupName = process.env.PANTHEON_LIVE_PARENT_GROUP || '测试环境';
const childGroupName = process.env.PANTHEON_LIVE_CHILD_GROUP || '中间件';
const hostLabelEnv = process.env.PANTHEON_LIVE_LABEL_ENV || 'test';
const hostLabelRole = process.env.PANTHEON_LIVE_LABEL_ROLE || component.packageName;
const hostLabelBiz = process.env.PANTHEON_LIVE_LABEL_BIZ || bizScopeCode;

const hostName = process.env.PANTHEON_LIVE_HOSTNAME || `${bizScopeCode}-dev-${component.packageName}-01`;
const hostOS = process.env.PANTHEON_LIVE_HOST_OS || 'linux';
const packageName = process.env.PANTHEON_LIVE_PACKAGE_NAME || component.packageName;
const packageVersion = process.env.PANTHEON_LIVE_PACKAGE_VERSION || component.packageVersion;
const templateName = process.env.PANTHEON_LIVE_TEMPLATE_NAME || component.templateName;
const templateVersion = process.env.PANTHEON_LIVE_TEMPLATE_VERSION || 'v1';
const sourceUrl = (process.env.PANTHEON_LIVE_SOURCE_URL || '').trim();
const sourceObjectKey = (process.env.PANTHEON_LIVE_SOURCE_OBJECT_KEY || '').trim();
const sourceFileName = (process.env.PANTHEON_LIVE_SOURCE_FILE_NAME || '').trim();

const runtimeParameters = buildRuntimeParameters(component.parameters);

if (component.requiresSource && !sourceUrl && !sourceObjectKey) {
  throw new Error(`component ${componentKey} requires source package env: PANTHEON_LIVE_SOURCE_URL or PANTHEON_LIVE_SOURCE_OBJECT_KEY`);
}

const runToken = new Date().toISOString().replace(/[-:.TZ]/g, '').slice(0, 14);
const taskName = process.env.PANTHEON_LIVE_TASK_NAME || `${component.packageName}-live-${runToken}`;

function requiredEnv(key) {
  const value = (process.env[key] || '').trim();
  if (!value) {
    throw new Error(`missing required env: ${key}`);
  }
  return value;
}

function buildRuntimeParameters(defaults) {
  const result = { ...defaults };
  for (const key of Object.keys(defaults)) {
    const envKey = `PANTHEON_LIVE_PARAM_${key.replace(/([a-z])([A-Z])/g, '$1_$2').toUpperCase()}`;
    const value = process.env[envKey];
    if (typeof value === 'string' && value.trim()) {
      result[key] = value.trim();
    }
  }
  return result;
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function authHeaders(accessToken, csrfToken = 'pantheon-live-csrf') {
  return {
    Authorization: `Bearer ${accessToken}`,
    'X-CSRF-Token': csrfToken,
    Cookie: `pantheon_csrf_token=${csrfToken}`,
    'Content-Type': 'application/json',
  };
}

async function requestJson(path, options = {}) {
  const response = await fetch(`${apiBaseUrl}${path}`, options);
  const text = await response.text();
  let payload = null;
  try {
    payload = text ? JSON.parse(text) : null;
  } catch {
    throw new Error(`invalid json from ${path}: ${text}`);
  }
  if (!response.ok) {
    throw new Error(`${path} http ${response.status}: ${text}`);
  }
  if (payload?.code !== 200) {
    throw new Error(`${path} business error ${payload?.code}: ${payload?.message || text}`);
  }
  return payload.data;
}

async function login() {
  const data = await requestJson('/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: adminUsername, password: adminPassword }),
  });
  return data.accessToken || data.token;
}

async function findBizScope(headers) {
  const list = await requestJson(
    `/business/bizscope/list?code=${encodeURIComponent(bizScopeCode)}&page=1&pageSize=50`,
    { headers },
  );
  return (list.items || []).find((item) => item.code === bizScopeCode) || null;
}

async function ensureBizScope(headers) {
  const existing = await findBizScope(headers);
  if (existing) {
    return requestJson(`/business/bizscope/${existing.id}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify({
        name: bizScopeName,
        environment: bizScopeEnvironment,
        status: bizScopeStatus,
      }),
    });
  }
  return requestJson('/business/bizscope', {
    method: 'POST',
    headers,
    body: JSON.stringify({
      code: bizScopeCode,
      name: bizScopeName,
      environment: bizScopeEnvironment,
      status: bizScopeStatus,
    }),
  });
}

function flattenGroups(groups, bucket = []) {
  for (const item of groups || []) {
    bucket.push(item);
    if (Array.isArray(item.children) && item.children.length > 0) {
      flattenGroups(item.children, bucket);
    }
  }
  return bucket;
}

async function ensureGroup(headers, name, parentId, conditions) {
  const groups = await requestJson('/business/cmdb/groups', { headers });
  const flat = flattenGroups(groups);
  const existing = flat.find((item) => item.name === name && Number(item.parentId || 0) === Number(parentId || 0));
  if (existing) {
    return requestJson(`/business/cmdb/groups/${existing.id}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify({ conditions, description: `${name} 自动校验分组` }),
    });
  }
  return requestJson('/business/cmdb/groups', {
    method: 'POST',
    headers,
    body: JSON.stringify({
      name,
      parentId,
      description: `${name} 自动校验分组`,
      conditions,
    }),
  });
}

async function findHost(headers) {
  const list = await requestJson(
    `/business/cmdb/hosts?keyword=${encodeURIComponent(hostIP)}&page=1&pageSize=50`,
    { headers },
  );
  return (list.items || []).find((item) => item.ip === hostIP) || null;
}

async function ensureHost(headers) {
  const payload = {
    hostname: hostName,
    ip: hostIP,
    sshPort: 22,
    os: hostOS,
    labels: [
      { key: 'env', val: hostLabelEnv },
      { key: 'role', val: hostLabelRole },
      { key: 'biz', val: hostLabelBiz },
    ],
    owner: sshUser,
    remark: `live closed loop ${component.packageName} ${runToken}`,
  };
  const existing = await findHost(headers);
  if (existing) {
    return requestJson(`/business/cmdb/hosts/${existing.id}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(payload),
    });
  }
  return requestJson('/business/cmdb/hosts', {
    method: 'POST',
    headers,
    body: JSON.stringify(payload),
  });
}

async function collectHost(headers, hostId) {
  return requestJson(`/business/cmdb/hosts/${hostId}/collect`, {
    method: 'POST',
    headers,
    body: JSON.stringify({
      sshUser,
      sshPassword,
      hostFingerprint,
      authMode: 'password',
    }),
  });
}

async function bindHostToScope(headers, hostId, scopeId) {
  return requestJson(`/business/cmdb/hosts/${hostId}`, {
    method: 'PUT',
    headers,
    body: JSON.stringify({ businessScopeId: scopeId }),
  });
}

async function getGroupMembers(headers, groupId) {
  return requestJson(`/business/cmdb/groups/${groupId}/members`, { headers });
}

async function findPackage(headers) {
  const list = await requestJson(
    `/business/deploy/packages?keyword=${encodeURIComponent(packageName)}&page=1&pageSize=50`,
    { headers },
  );
  return (list.items || []).find((item) => item.name === packageName && item.version === packageVersion) || null;
}

async function ensurePackage(headers) {
  const payload = {
    name: packageName,
    version: packageVersion,
    executionMode: 'fixed',
    templateCode: component.templateCode,
    templateConfig: { scenario: component.templateCode },
    status: 'enabled',
    ...(sourceObjectKey ? { sourceObjectKey } : {}),
    ...(sourceFileName ? { sourceFileName } : {}),
    ...(sourceUrl ? { sourceUrl } : {}),
  };
  const existing = await findPackage(headers);
  if (existing) {
    return requestJson(`/business/deploy/packages/${existing.id}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(payload),
    });
  }
  return requestJson('/business/deploy/packages', {
    method: 'POST',
    headers,
    body: JSON.stringify(payload),
  });
}

async function findTemplate(headers) {
  const list = await requestJson(
    `/business/deploy/templates?keyword=${encodeURIComponent(templateName)}&page=1&pageSize=50`,
    { headers },
  );
  return (list.items || []).find((item) => item.name === templateName && item.version === templateVersion) || null;
}

async function ensureTemplate(headers, pkgId) {
  const payload = {
    name: templateName,
    version: templateVersion,
    category: 'middleware',
    executionMode: 'fixed',
    defaultAction: 'install',
    packageId: pkgId,
    parameterSchema: runtimeParameters,
    status: 'enabled',
    steps: [
      {
        stepCode: `${component.packageName}_runtime`,
        stepName: `${component.packageName} runtime`,
        stepType: 'package',
        action: 'install',
        packageId: pkgId,
        templateParams: runtimeParameters,
        sort: 1,
      },
    ],
  };
  const existing = await findTemplate(headers);
  if (existing) {
    return requestJson(`/business/deploy/templates/${existing.id}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(payload),
    });
  }
  return requestJson('/business/deploy/templates', {
    method: 'POST',
    headers,
    body: JSON.stringify(payload),
  });
}

async function createTask(headers, templateId, scopeId, hostId, action, nameSuffix) {
  return requestJson('/business/deploy/tasks', {
    method: 'POST',
    headers,
    body: JSON.stringify({
      name: `${taskName}-${nameSuffix}`,
      templateId,
      businessScopeId: scopeId,
      action,
      targetType: 'host',
      targetIds: [hostId],
      executorType: 'ssh',
      templateParams: runtimeParameters,
      remark: `live closed loop ${component.packageName} ${runToken}`,
    }),
  });
}

async function startTask(headers, taskId) {
  return requestJson(`/business/deploy/tasks/${taskId}/start`, {
    method: 'POST',
    headers,
    body: JSON.stringify({
      sshUser,
      sshPassword,
      hostFingerprint,
      authMode: 'password',
    }),
  });
}

async function getTask(headers, taskId) {
  return requestJson(`/business/deploy/tasks/${taskId}`, { headers });
}

async function getHost(headers, hostId) {
  return requestJson(`/business/cmdb/hosts/${hostId}`, { headers });
}

async function waitTask(headers, taskId) {
  const pollLimit = Number(process.env.PANTHEON_LIVE_TASK_POLL_LIMIT || 120);
  const pollIntervalMs = Number(process.env.PANTHEON_LIVE_TASK_POLL_INTERVAL_MS || 10000);
  for (let attempt = 1; attempt <= pollLimit; attempt += 1) {
    const detail = await getTask(headers, taskId);
    const hostRow = detail.hosts?.[0];
    console.log(`[poll ${attempt}] task=${detail.status} host=${hostRow?.status || 'n/a'} traceSteps=${hostRow?.traceSteps?.length || 0}`);
    if (detail.status === 'success' || detail.status === 'failed' || detail.status === 'canceled') {
      return detail;
    }
    await sleep(pollIntervalMs);
  }
  throw new Error(`task ${taskId} did not finish within polling window`);
}

function findInstalledComponent(hostDetail) {
  return (hostDetail.installedComponents || []).find((item) => item.name === packageName);
}

function countOtherInstalledComponents(hostDetail) {
  return (hostDetail.installedComponents || []).filter((item) => item.name !== packageName).length;
}

async function main() {
  const accessToken = await login();
  const headers = authHeaders(accessToken);

  console.log(`[1/10] ensure bizscope ${bizScopeCode}`);
  const bizScope = await ensureBizScope(headers);
  console.log(JSON.stringify({ id: bizScope.id, code: bizScope.code, name: bizScope.name, environment: bizScope.environment, status: bizScope.status }, null, 2));

  console.log(`[2/10] ensure cmdb groups ${parentGroupName} / ${childGroupName}`);
  const parentGroup = await ensureGroup(headers, parentGroupName, 0, {
    operator: 'AND',
    rules: [{ key: 'env', op: 'eq', val: hostLabelEnv }],
  });
  const childGroup = await ensureGroup(headers, childGroupName, parentGroup.id, {
    operator: 'AND',
    rules: [{ key: 'role', op: 'eq', val: hostLabelRole }],
  });
  console.log(JSON.stringify({ parentGroupId: parentGroup.id, childGroupId: childGroup.id }, null, 2));

  console.log(`[3/10] ensure host ${hostIP}`);
  const host = await ensureHost(headers);
  console.log(JSON.stringify({ id: host.id, hostname: host.hostname, ip: host.ip, status: host.status }, null, 2));

  console.log('[4/10] collect host facts via ssh');
  const collected = await collectHost(headers, host.id);
  console.log(JSON.stringify({
    id: collected.id,
    os: collected.os,
    osVersion: collected.osVersion,
    cpuCores: collected.cpuCores,
    memoryGb: collected.memoryGb,
    diskGb: collected.diskGb,
  }, null, 2));

  console.log('[5/10] bind host to bizscope');
  const bound = await bindHostToScope(headers, host.id, bizScope.id);
  console.log(JSON.stringify({
    id: bound.id,
    businessScopeId: bound.businessScopeId,
    businessScopeName: bound.businessScopeName,
    status: bound.status,
  }, null, 2));
  if (bound.status !== 'assigned' && bound.status !== 'online') {
    throw new Error(`expected host status assigned or online after binding, got ${bound.status}`);
  }

  console.log('[6/10] verify group membership');
  const members = await getGroupMembers(headers, childGroup.id);
  const matched = (members.members || []).some((item) => item.id === host.id);
  if (!matched) {
    throw new Error(`host ${host.id} not matched into child group ${childGroupName}`);
  }
  console.log(JSON.stringify({
    groupId: members.groupId,
    groupName: members.groupName,
    memberCount: (members.members || []).length,
  }, null, 2));

  console.log(`[7/10] ensure deploy package ${packageName}@${packageVersion}`);
  const pkg = await ensurePackage(headers);
  console.log(JSON.stringify({
    id: pkg.id,
    name: pkg.name,
    version: pkg.version,
    executionMode: pkg.executionMode,
    templateCode: pkg.templateCode,
    sourceUrl: pkg.sourceUrl || '',
  }, null, 2));

  console.log(`[8/10] ensure deploy template ${templateName}@${templateVersion}`);
  const template = await ensureTemplate(headers, pkg.id);
  console.log(JSON.stringify({
    id: template.id,
    name: template.name,
    version: template.version,
    stepCount: template.stepCount,
    parameterSchema: template.parameterSchema,
  }, null, 2));

  const currentHost = await getHost(headers, host.id);
  const currentComponent = findInstalledComponent(currentHost);
  let uninstallDetail = null;
  if (currentComponent) {
    console.log('[9/10] create uninstall task');
    const uninstallTask = await createTask(headers, template.id, bizScope.id, host.id, 'uninstall', 'uninstall');
    console.log(JSON.stringify({ id: uninstallTask.id, status: uninstallTask.status, name: uninstallTask.name }, null, 2));
    await startTask(headers, uninstallTask.id);
    uninstallDetail = await waitTask(headers, uninstallTask.id);
    const hostAfterUninstall = await getHost(headers, host.id);
    const expectedStatusAfterUninstall = countOtherInstalledComponents(hostAfterUninstall) > 0 ? 'online' : 'assigned';
    if (uninstallDetail.status !== 'success') {
      throw new Error(`uninstall task failed with status ${uninstallDetail.status}`);
    }
    if (hostAfterUninstall.status !== expectedStatusAfterUninstall) {
      throw new Error(`expected host status ${expectedStatusAfterUninstall} after uninstall, got ${hostAfterUninstall.status}`);
    }
    if (findInstalledComponent(hostAfterUninstall)) {
      throw new Error(`expected component ${packageName} removed after uninstall`);
    }
  } else {
    console.log('[9/10] skip uninstall, component not installed on host');
  }

  const deployAction = currentComponent ? 'reinstall' : 'install';
  console.log(`[10/10] create ${deployAction} task`);
  const reinstallTask = await createTask(headers, template.id, bizScope.id, host.id, deployAction, deployAction);
  console.log(JSON.stringify({ id: reinstallTask.id, status: reinstallTask.status, name: reinstallTask.name }, null, 2));
  await startTask(headers, reinstallTask.id);
  const finalTask = await waitTask(headers, reinstallTask.id);
  const finalHost = await getHost(headers, host.id);
  const installedComponent = findInstalledComponent(finalHost);

  const summary = {
    component: {
      key: componentKey,
      templateCode: component.templateCode,
      packageName,
      packageVersion,
      runtimeParameters,
    },
    bizScope: {
      id: bizScope.id,
      code: bizScope.code,
      name: bizScope.name,
      environment: bizScope.environment,
    },
    groups: {
      parent: { id: parentGroup.id, name: parentGroup.name },
      child: { id: childGroup.id, name: childGroup.name },
    },
    host: {
      id: finalHost.id,
      hostname: finalHost.hostname,
      ip: finalHost.ip,
      status: finalHost.status,
      matchedGroups: finalHost.matchedGroups || [],
      installedComponents: finalHost.installedComponents || [],
    },
    task: {
      uninstall: {
        skipped: !uninstallDetail,
        id: uninstallDetail?.id || 0,
        name: uninstallDetail?.name || '',
        status: uninstallDetail?.status || 'skipped',
        hostStatus: uninstallDetail?.hosts?.[0]?.status || '',
        stdout: uninstallDetail?.hosts?.[0]?.stdout || '',
        stderr: uninstallDetail?.hosts?.[0]?.stderr || '',
        errorMessage: uninstallDetail?.hosts?.[0]?.errorMessage || '',
        traceSteps: uninstallDetail?.hosts?.[0]?.traceSteps || [],
      },
      reinstall: {
        id: finalTask.id,
        name: finalTask.name,
        status: finalTask.status,
        hostStatus: finalTask.hosts?.[0]?.status || '',
        stdout: finalTask.hosts?.[0]?.stdout || '',
        stderr: finalTask.hosts?.[0]?.stderr || '',
        errorMessage: finalTask.hosts?.[0]?.errorMessage || '',
        traceSteps: finalTask.hosts?.[0]?.traceSteps || [],
      },
    },
  };

  console.log('--- LIVE CLOSED LOOP SUMMARY ---');
  console.log(JSON.stringify(summary, null, 2));

  if (finalTask.status !== 'success') {
    throw new Error(`deployment task failed with status ${finalTask.status}`);
  }
  if (finalHost.status !== 'online') {
    throw new Error(`expected host status online after success, got ${finalHost.status}`);
  }
  if (!installedComponent || installedComponent.version !== packageVersion) {
    throw new Error(`expected installed component ${packageName}@${packageVersion} on host`);
  }
}

main().catch((error) => {
  console.error('LIVE_CLOSED_LOOP_FAILED');
  console.error(error?.stack || String(error));
  process.exit(1);
});
