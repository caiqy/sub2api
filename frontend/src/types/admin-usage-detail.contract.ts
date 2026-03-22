import type { AdminUsageDetail } from './index'

// 后端 DTO 约定：headers/body 都是字符串（或 null）
const adminUsageDetailContractSample: AdminUsageDetail = {
  usage_log_id: 1,
  request_headers: '{"content-type":"application/json"}',
  request_body: '{"prompt":"hello"}',
  upstream_request_headers: '{"x-upstream":"gateway"}',
  upstream_request_body: '{"model":"gpt-4.1"}',
  response_headers: '{"x-request-id":"req_123"}',
  response_body: '{"id":"resp_123"}',
  created_at: '2026-03-20T00:00:00Z'
}

void adminUsageDetailContractSample
