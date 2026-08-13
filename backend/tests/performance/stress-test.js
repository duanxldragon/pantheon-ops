import http from 'k6/http';
import { check, sleep } from 'k6';

// 压力测试配置 - 逐步增加负载直到系统崩溃
export const options = {
  stages: [
    { duration: '2m', target: 100 },   // 快速达到 100 用户
    { duration: '5m', target: 200 },   // 增加到 200 用户
    { duration: '5m', target: 300 },   // 增加到 300 用户
    { duration: '5m', target: 400 },   // 增加到 400 用户
    { duration: '5m', target: 500 },   // 增加到 500 用户
    { duration: '10m', target: 500 },  // 保持 500 用户 10 分钟
    { duration: '3m', target: 0 },     // 逐步降到 0
  ],
  thresholds: {
    http_req_duration: ['p(99)<1000'],  // 99% 的请求应在 1s 内完成
    http_req_failed: ['rate<0.1'],      // 错误率应低于 10%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function stressScenario() {
  // 混合负载测试
  const testCases = [
    // 健康检查
    () => http.get(`${BASE_URL}/api/v1/health`),

    // 登录请求
    () => http.post(
      `${BASE_URL}/api/v1/auth/login`,
      JSON.stringify({ username: 'test', password: 'test' }),
      { headers: { 'Content-Type': 'application/json' } }
    ),

    // Metrics 请求
    () => http.get(`${BASE_URL}/metrics`),
  ];

  // 按 VU/迭代序号轮转选择测试用例：分布均匀且可复现（无需伪随机数）
  const selectedTest = testCases[(__VU + __ITER) % testCases.length];
  const res = selectedTest();

  check(res, {
    'status is not 500': (r) => r.status !== 500,
    'response time < 2s': (r) => r.timings.duration < 2000,
  });

  // 1-3 秒确定性思考时间，按迭代序号错开，避免所有 VU 同步冲击
  sleep((__ITER % 3) + 1);
}
