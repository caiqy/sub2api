import { apiClient } from '../client'

export interface ModelTraceSettings {
  enabled: boolean
  model: string
  rounds: number
  interval_minutes: number
}

export interface ModelTraceTask {
  id: number
  account_id: number
  source: 'manual' | 'auto'
  status: 'queued' | 'running' | 'completed' | 'failed'
  model: string
  target_model: string
  rounds: number
  completed_rounds: number
  result: '' | 'normal' | 'degraded'
  winner: string
  probabilities: Record<string, number>
  version: string
  created_at: string
  started_at?: string | null
  finished_at?: string | null
  duration_ms: number
  error: string
}

export type ModelTraceLatest = Pick<ModelTraceTask,
  'result' | 'model' | 'target_model' | 'winner' | 'probabilities' | 'finished_at' | 'version'> & { task_id: number }

export interface ModelTraceHistory {
  items: ModelTraceTask[]
  total: number
  page: number
  page_size: number
  pages: number
  active: ModelTraceTask | null
}

export async function getModels(accountId?: number): Promise<{ models: string[]; version: string }> {
  const { data } = await apiClient.get('/admin/modeltrace/models', {
    params: accountId === undefined ? undefined : { account_id: accountId }
  })
  return data
}

export async function getSettings(): Promise<ModelTraceSettings> {
  const { data } = await apiClient.get('/admin/modeltrace/settings')
  return data
}

export async function updateSettings(settings: ModelTraceSettings): Promise<ModelTraceSettings> {
  const { data } = await apiClient.put('/admin/modeltrace/settings', settings)
  return data
}

export async function start(accountId: number, request: { model: string; rounds: number }): Promise<ModelTraceTask> {
  const { data } = await apiClient.post(`/admin/accounts/${accountId}/modeltrace`, request)
  return data
}

export async function list(accountId: number, page = 1, pageSize = 20): Promise<ModelTraceHistory> {
  const { data } = await apiClient.get(`/admin/accounts/${accountId}/modeltrace`, {
    params: { page, page_size: pageSize }
  })
  return data
}

export default { getModels, getSettings, updateSettings, start, list }
