// Copyright 2025 The fawa Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
    // 要求99%的请求延迟必须在1500ms(1.5s)以内
    'http_req_duration': ['p(95)<500', 'p(99)<1500'],
  },
};

export default function () {
  // 定义每个虚拟用户要执行的操作
  const res = http.get('http://127.0.0.1:8080/filemeta');
  
  // 检查响应状态码是否为200
  check(res, { 'status was 200': (r) => r.status == 200 });
  
} 