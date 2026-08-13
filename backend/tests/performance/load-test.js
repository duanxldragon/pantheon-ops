import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// 自定义指标
const errorRate = new Rate('errors');
const loginDuration = new Trend('login_duration');

// 测试配置
export const options = {
  stages: [
    { duration: '1m', target: 10 },   // 热身：1分钟达到10个用户
    { duration: '3m', target: 50 },   // 负载：3分钟达到50个用户
    { duration: '2m', target: 100 },  // 压力：2分钟达到100个用户
    { duration: '2m', target: 50 },   // 降压：2分钟降到50个用户
    { duration: '1m', target: 0 },    // 冷却：1分钟降到0
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% 的请求应在 500ms 内完成
    http_req_failed: ['rate<0.05'],   // 错误率应低于 5%
    errors: ['rate<0.1'],              // 自定义错误率应低于 10%
  },
};

// 测试数据
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TEST_USER = {
  username: __ENV.TEST_USERNAME || 'test_user',
  password: __ENV.TEST_PASSWORD || '',
};

// 设置阶段
export function setup() {
  console.log('Starting performance test...');
  console.log('Base URL:', BASE_URL);

  // 可以在这里创建测试数据
  return { baseUrl: BASE_URL };
}

// 主测试逻辑
export default function loadScenario(data) {
  // 1. 测试健康检查
  const healthRes = http.get(`${data.baseUrl}/api/v1/health`);
  check(healthRes, {
    'health check status is 200': (r) => r.status === 200,
    'health check response time < 200ms': (r) => r.timings.duration < 200,
  }) || errorRate.add(1);

  sleep(1);

  // 2. 测试登录
  const loginStart = Date.now();
  const loginPayload = JSON.stringify(TEST_USER);
  const loginParams = {
    headers: { 'Content-Type': 'application/json' },
  };

  const loginRes = http.post(
    `${data.baseUrl}/api/v1/auth/login`,
    loginPayload,
    loginParams
  );

  const loginSuccess = check(loginRes, {
    'login status is 200 or 401': (r) => r.status === 200 || r.status === 401,
    'login response has data': (r) => r.json('data') !== undefined,
  });

  if (!loginSuccess) {
    errorRate.add(1);
  }

  loginDuration.add(Date.now() - loginStart);

  // 如果登录成功，提取 token
  let token = null;
  if (loginRes.status === 200) {
    const responseData = loginRes.json('data');
    token = responseData?.accessToken;
  }

  sleep(2);

  // 3. 测试需要认证的接口（如果有 token）
  if (token) {
    const authParams = {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
    };

    // 获取用户信息
    const meRes = http.get(`${data.baseUrl}/api/v1/auth/me`, authParams);
    check(meRes, {
      'get user info status is 200': (r) => r.status === 200,
      'get user info response time < 300ms': (r) => r.timings.duration < 300,
    }) || errorRate.add(1);

    sleep(1);

    // 获取菜单
    const menuRes = http.get(`${data.baseUrl}/api/v1/system/menu/tree`, authParams);
    check(menuRes, {
      'get menu status is 200': (r) => r.status === 200,
      'get menu response time < 500ms': (r) => r.timings.duration < 500,
    }) || errorRate.add(1);
  }

  sleep(2);

  // 4. 测试 Prometheus metrics 端点
  const metricsRes = http.get(`${data.baseUrl}/metrics`);
  check(metricsRes, {
    'metrics endpoint status is 200': (r) => r.status === 200,
    'metrics endpoint returns prometheus format': (r) =>
      r.body.includes('pantheon_http_requests_total'),
  }) || errorRate.add(1);

  sleep(1);
}

// 清理阶段
export function teardown(data) {
  console.log('Performance test completed');
}
