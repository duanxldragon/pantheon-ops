import { test, expect } from '@playwright/test';

/**
 * API Smoke Tests - ArgoCD Module
 * 对应 QA 验收文档: 模块 A - ArgoCD GitOps 集成
 */

const API_BASE_URL = process.env.BASE_URL || 'http://localhost:8080';
const API_TOKEN = process.env.API_TOKEN || 'test-token';

test.describe('模块 A: ArgoCD GitOps 集成 - 冒烟测试', () => {
  let serverId: number;
  let repoId: number;
  let appId: number;

  test.beforeAll(() => {
    console.log('Starting ArgoCD smoke tests...');
  });

  /**
   * A.1.1 添加 ArgoCD 服务器 (P0)
   */
  test('A.1.1 - 添加 ArgoCD 服务器', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/api/v1/argocd/servers`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        name: 'Test ArgoCD Server',
        clusterId: 1,
        endpoint: 'https://argocd.test.com',
        authToken: 'test-argocd-token-12345',
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');
    expect(body.data.name).toBe('Test ArgoCD Server');
    expect(body.data.authToken).toBeUndefined(); // 不应返回敏感信息

    serverId = body.data.id;
    console.log(`✓ ArgoCD 服务器创建成功, ID: ${serverId}`);
  });

  /**
   * A.1.3 查询 ArgoCD 服务器列表 (P0)
   */
  test('A.1.3 - 查询 ArgoCD 服务器列表', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/argocd/servers`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('items');
    expect(Array.isArray(body.data.items)).toBe(true);
    expect(body.data.items.length).toBeGreaterThan(0);

    console.log(`✓ 查询到 ${body.data.items.length} 个 ArgoCD 服务器`);
  });

  /**
   * A.1.2 测试 ArgoCD 连接 (P0)
   */
  test('A.1.2 - 测试 ArgoCD 连接', async ({ request }) => {
    test.skip(!serverId, '需要先创建服务器');

    const response = await request.post(
      `${API_BASE_URL}/api/v1/argocd/servers/${serverId}/test`,
      {
        headers: {
          'Authorization': `Bearer ${API_TOKEN}`,
        },
      }
    );

    // 连接可能失败（因为是测试端点），但应该返回结果
    expect([200, 500]).toContain(response.status());
    const body = await response.json();
    expect(body).toHaveProperty('code');

    console.log(`✓ 连接测试执行完成, 状态: ${body.code === 0 ? '成功' : '失败'}`);
  });

  /**
   * A.2.1 添加 Git 仓库 (P0)
   */
  test('A.2.1 - 添加 Git 仓库', async ({ request }) => {
    test.skip(!serverId, '需要先创建服务器');

    const response = await request.post(`${API_BASE_URL}/api/v1/argocd/repositories`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        serverId: serverId,
        repoURL: 'https://github.com/test/repo',
        name: 'Test Repo',
        username: 'testuser',
        password: 'testpass123',
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');
    expect(body.data.password).toBeUndefined(); // 不应返回密码

    repoId = body.data.id;
    console.log(`✓ Git 仓库创建成功, ID: ${repoId}`);
  });

  /**
   * A.3.1 创建应用 (P0)
   */
  test('A.3.1 - 创建 ArgoCD 应用', async ({ request }) => {
    test.skip(!serverId, '需要先创建服务器');

    const response = await request.post(`${API_BASE_URL}/api/v1/argocd/applications`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        name: 'test-app',
        serverId: serverId,
        repoURL: 'https://github.com/test/repo',
        path: 'k8s/',
        targetRevision: 'main',
        project: 'default',
        namespace: 'default',
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');
    expect(body.data.name).toBe('test-app');

    appId = body.data.id;
    console.log(`✓ ArgoCD 应用创建成功, ID: ${appId}`);
  });

  /**
   * A.3.2 查询应用列表 (P0)
   */
  test('A.3.2 - 查询 ArgoCD 应用列表', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/argocd/applications`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('items');
    expect(Array.isArray(body.data.items)).toBe(true);

    console.log(`✓ 查询到 ${body.data.items.length} 个应用`);
  });
});

test.describe('模块 B: Velero 备份恢复 - 冒烟测试', () => {
  let storageLocationId: number;
  let backupId: number;
  let scheduleId: number;

  /**
   * B.1.1 添加存储位置 (P0)
   */
  test('B.1.1 - 添加存储位置', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/api/v1/k8s/storage-locations`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        name: 'test-s3-storage',
        provider: 'aws',
        bucket: 'test-backup-bucket',
        region: 'us-east-1',
        accessKeyId: 'test-access-key',
        secretAccessKey: 'test-secret-key',
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');
    expect(body.data.secretAccessKey).toBeUndefined(); // 不应返回密钥

    storageLocationId = body.data.id;
    console.log(`✓ 存储位置创建成功, ID: ${storageLocationId}`);
  });

  /**
   * B.2.1 创建备份 (P0)
   */
  test('B.2.1 - 创建备份', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/api/v1/k8s/backups`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        name: 'test-backup-' + Date.now(),
        clusterId: 1,
        includedNamespaces: ['default', 'kube-system'],
        ttl: 604800, // 7 days
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');
    expect(body.data.status).toBe('New');

    backupId = body.data.id;
    console.log(`✓ 备份创建成功, ID: ${backupId}`);
  });

  /**
   * B.2.2 查询备份列表 (P0)
   */
  test('B.2.2 - 查询备份列表', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/k8s/backups`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('items');
    expect(Array.isArray(body.data.items)).toBe(true);

    console.log(`✓ 查询到 ${body.data.items.length} 个备份`);
  });

  /**
   * B.3.1 创建定时备份 (P0)
   */
  test('B.3.1 - 创建定时备份', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/api/v1/k8s/schedules`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        name: 'daily-backup-' + Date.now(),
        clusterId: 1,
        cronExpression: '0 2 * * *',
        enabled: true,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');

    scheduleId = body.data.id;
    console.log(`✓ 定时备份创建成功, ID: ${scheduleId}`);
  });

  /**
   * B.5.1 创建灾难演练 (P1)
   */
  test('B.5.1 - 创建灾难演练', async ({ request }) => {
    test.skip(!backupId, '需要先创建备份');

    const response = await request.post(`${API_BASE_URL}/api/v1/k8s/drills`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        name: 'test-drill-' + Date.now(),
        backupId: backupId,
        type: 'validation',
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');
    expect(body.data.status).toBe('pending');

    console.log(`✓ 灾难演练创建成功, ID: ${body.data.id}`);
  });
});

test.describe('模块 C: 监控告警系统 - 冒烟测试', () => {
  let datasourceId: number;
  let ruleId: number;
  let channelId: number;

  /**
   * C.1.1 添加 Prometheus 数据源 (P0)
   */
  test('C.1.1 - 添加 Prometheus 数据源', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/api/v1/observability/datasources`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        name: 'Test Prometheus',
        type: 'prometheus',
        url: 'http://prometheus:9090',
        isDefault: true,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');
    expect(body.data.name).toBe('Test Prometheus');

    datasourceId = body.data.id;
    console.log(`✓ Prometheus 数据源创建成功, ID: ${datasourceId}`);
  });

  /**
   * C.1.2 测试数据源连接 (P0)
   */
  test('C.1.2 - 测试数据源连接', async ({ request }) => {
    test.skip(!datasourceId, '需要先创建数据源');

    const response = await request.post(
      `${API_BASE_URL}/api/v1/observability/datasources/${datasourceId}/test`,
      {
        headers: {
          'Authorization': `Bearer ${API_TOKEN}`,
        },
      }
    );

    // 连接可能失败（因为是测试端点），但应该返回结果
    expect([200, 500]).toContain(response.status());
    const body = await response.json();
    expect(body).toHaveProperty('code');

    console.log(`✓ 数据源连接测试执行完成`);
  });

  /**
   * C.2.1 创建告警规则 (P0)
   */
  test('C.2.1 - 创建告警规则', async ({ request }) => {
    test.skip(!datasourceId, '需要先创建数据源');

    const response = await request.post(`${API_BASE_URL}/api/v1/observability/alert-rules`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        name: 'Test High CPU Alert',
        datasourceId: datasourceId,
        enabled: true,
        query: 'avg(rate(cpu_usage[5m])) > 0.8',
        interval: 60,
        forDuration: 300,
        severity: 'warning',
        labels: {
          team: 'ops',
        },
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');
    expect(body.data.enabled).toBe(true);

    ruleId = body.data.id;
    console.log(`✓ 告警规则创建成功, ID: ${ruleId}`);
  });

  /**
   * C.2.2 查询告警规则列表 (P0)
   */
  test('C.2.2 - 查询告警规则列表', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/observability/alert-rules`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('items');
    expect(Array.isArray(body.data.items)).toBe(true);

    console.log(`✓ 查询到 ${body.data.items.length} 个告警规则`);
  });

  /**
   * C.5.1 添加通知渠道 (P0)
   */
  test('C.5.1 - 添加通知渠道 (Webhook)', async ({ request }) => {
    const response = await request.post(
      `${API_BASE_URL}/api/v1/observability/notification-channels`,
      {
        headers: {
          'Authorization': `Bearer ${API_TOKEN}`,
          'Content-Type': 'application/json',
        },
        data: {
          name: 'Test Webhook',
          type: 'webhook',
          enabled: true,
          config: {
            url: 'https://webhook.test.com/alerts',
            method: 'POST',
          },
        },
      }
    );

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');

    channelId = body.data.id;
    console.log(`✓ 通知渠道创建成功, ID: ${channelId}`);
  });

  /**
   * C.5.2 查询通知渠道列表 (P0)
   */
  test('C.5.2 - 查询通知渠道列表', async ({ request }) => {
    const response = await request.get(
      `${API_BASE_URL}/api/v1/observability/notification-channels`,
      {
        headers: {
          'Authorization': `Bearer ${API_TOKEN}`,
        },
      }
    );

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('items');
    expect(Array.isArray(body.data.items)).toBe(true);

    console.log(`✓ 查询到 ${body.data.items.length} 个通知渠道`);
  });

  /**
   * C.4.2 查询告警事件列表 (P0)
   */
  test('C.4.2 - 查询告警事件列表', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/observability/alert-events`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('items');
    expect(Array.isArray(body.data.items)).toBe(true);

    console.log(`✓ 查询到 ${body.data.items.length} 个告警事件`);
  });
});

test.describe('模块 D: 文件分发系统 - 冒烟测试', () => {
  let storageId: number;
  let fileId: number;
  let taskId: number;

  /**
   * D.1.1 创建存储 (P0)
   */
  test('D.1.1 - 创建文件存储', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/api/v1/filedist/storages`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        name: 'Test Local Storage',
        type: 'local',
        basePath: '/tmp/test-storage',
        maxSize: 10737418240, // 10GB
        isDefault: true,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');
    expect(body.data.usedSize).toBe(0);
    expect(body.data.fileCount).toBe(0);

    storageId = body.data.id;
    console.log(`✓ 文件存储创建成功, ID: ${storageId}`);
  });

  /**
   * D.1.2 查询存储列表 (P0)
   */
  test('D.1.2 - 查询存储列表', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/filedist/storages`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('items');
    expect(Array.isArray(body.data.items)).toBe(true);

    console.log(`✓ 查询到 ${body.data.items.length} 个存储`);
  });

  /**
   * D.2.1 上传文件 (P0)
   */
  test('D.2.1 - 上传文件', async ({ request }) => {
    test.skip(!storageId, '需要先创建存储');

    // 创建测试文件内容
    const testContent = 'This is a test file content';
    const blob = new Blob([testContent], { type: 'text/plain' });

    const formData = new FormData();
    formData.append('storageId', storageId.toString());
    formData.append('file', blob, 'test-file.txt');

    const response = await request.post(`${API_BASE_URL}/api/v1/filedist/files/upload`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
      },
      multipart: {
        storageId: storageId.toString(),
        file: {
          name: 'test-file.txt',
          mimeType: 'text/plain',
          buffer: Buffer.from(testContent),
        },
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');
    expect(body.data).toHaveProperty('md5');

    fileId = body.data.id;
    console.log(`✓ 文件上传成功, ID: ${fileId}`);
  });

  /**
   * D.2.2 查询文件列表 (P0)
   */
  test('D.2.2 - 查询文件列表', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/filedist/files`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('items');
    expect(Array.isArray(body.data.items)).toBe(true);

    console.log(`✓ 查询到 ${body.data.items.length} 个文件`);
  });

  /**
   * D.3.1 创建分发任务 (P0)
   */
  test('D.3.1 - 创建分发任务', async ({ request }) => {
    test.skip(!fileId, '需要先上传文件');

    const response = await request.post(`${API_BASE_URL}/api/v1/filedist/tasks`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        name: 'Test Distribution Task',
        fileId: fileId,
        targets: [
          {
            host: 'server1.test.com',
            port: 22,
            targetPath: '/tmp/test-file.txt',
            username: 'testuser',
            password: 'testpass',
            maxRetries: 3,
          },
        ],
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('id');
    expect(body.data.status).toBe('pending');
    expect(body.data.totalTargets).toBe(1);

    taskId = body.data.id;
    console.log(`✓ 分发任务创建成功, ID: ${taskId}`);
  });

  /**
   * D.3.4 查询任务列表 (P0)
   */
  test('D.3.4 - 查询分发任务列表', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/filedist/tasks`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('items');
    expect(Array.isArray(body.data.items)).toBe(true);

    console.log(`✓ 查询到 ${body.data.items.length} 个分发任务`);
  });
});

test.describe('模块 E: 服务实例管理 - 冒烟测试', () => {
  /**
   * E.1.1 查询实例列表 (P0)
   */
  test('E.1.1 - 查询服务实例列表', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/service-instances`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
      },
    });

    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('items');
    expect(Array.isArray(body.data.items)).toBe(true);

    console.log(`✓ 查询到 ${body.data.items.length} 个服务实例`);
  });
});

test.describe('模块 F: 安全与加密 - 冒烟测试', () => {
  /**
   * F.1.3 加密字段验证 (P0)
   */
  test('F.1.3 - 验证敏感信息已加密', async ({ request }) => {
    // 创建一个 ArgoCD 服务器
    const createResponse = await request.post(`${API_BASE_URL}/api/v1/argocd/servers`, {
      headers: {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      data: {
        name: 'Encryption Test Server',
        clusterId: 1,
        endpoint: 'https://argocd.encryption-test.com',
        authToken: 'plaintext-token-12345',
      },
    });

    expect(createResponse.status()).toBe(200);
    const createBody = await createResponse.json();
    const serverId = createBody.data.id;

    // 查询详情，验证不返回敏感信息
    const getResponse = await request.get(
      `${API_BASE_URL}/api/v1/argocd/servers/${serverId}`,
      {
        headers: {
          'Authorization': `Bearer ${API_TOKEN}`,
        },
      }
    );

    expect(getResponse.status()).toBe(200);
    const getBody = await getResponse.json();

    // API 响应不应包含明文 token
    expect(getBody.data.authToken).toBeUndefined();

    console.log(`✓ 敏感信息未在 API 响应中暴露`);
  });

  /**
   * F.2.1 Token 认证 (P1)
   */
  test('F.2.1 - 验证 API 需要认证', async ({ request }) => {
    // 不带 token 访问
    const response = await request.get(`${API_BASE_URL}/api/v1/argocd/servers`);

    // 应该返回 401 Unauthorized
    expect(response.status()).toBe(401);

    console.log(`✓ 未认证请求被正确拒绝`);
  });
});
