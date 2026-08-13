import http from 'k6/http';
import { check } from 'k6';

// 峰值测试配置 - 短时间内产生大量请求
export const options = {
  stages: [
    { duration: '10s', target: 500 },  // 10秒内达到 500 用户
    { duration: '1m', target: 500 },   // 保持 500 用户 1 分钟
    { duration: '10s', target: 1000 }, // 10秒内达到 1000 用户
    { duration: '1m', target: 1000 },  // 保持 1000 用户 1 分钟
    { duration: '10s', target: 0 },    // 快速降到 0
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed: ['rate<0.2'],  // 允许 20% 错误率（峰值测试）
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function spikeScenario() {
  const res = http.get(`${BASE_URL}/api/v1/health`);

  check(res, {
    'status is 200 or 503': (r) => r.status === 200 || r.status === 503,
  });
}
