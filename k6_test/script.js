import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  // 定义压测场景
  stages: [
    // 阶段1: 1分钟内，虚拟用户数(VUs)从0缓慢增加到200
    { duration: '1m', target: 200 }, 
    // 阶段2: 保持200个虚拟用户，持续压测3分钟
    { duration: '3m', target: 200 },
    // 阶段3: 30秒内，虚拟用户数从200降至0
    { duration: '30s', target: 0 }, 
  ],
  // 定义通过/失败的阈值
  thresholds: {
    // 要求95%的请求延迟必须在500ms以内
    'http_req_duration': ['p(95)<500'], 
    // 要求99%的请求延迟必须在1500ms(1.5s)以内
    'http_req_duration': ['p(99)<1500'],
  },
};

export default function () {
  // 定义每个虚拟用户要执行的操作
  const res = http.get('http://127.0.0.1:8080/filemeta');
  
  // 检查响应状态码是否为200
  check(res, { 'status was 200': (r) => r.status == 200 });

  // 每个请求后随机等待一小段时间，模仿真实用户行为
} 