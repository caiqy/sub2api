import type { ConversationPartType } from './types'

export const TOOL_LABELS: Record<string, string> = {
  bash: '执行命令',
  read: '查看',
  write: '写入',
  edit: '编辑',
  multiedit: '批量编辑',
  apply_patch: '应用补丁',
  list: '列出',
  glob: '文件查找',
  grep: '文本查找',
  webfetch: '抓取网页',
  websearch: '搜索网页',
  task: '任务',
  question: '提问',
  todoread: '读取任务列表',
  todowrite: '更新任务列表',
  skill: '加载技能',
}

type ToolDisplayOptions = {
  tool: string
  callId?: string
  input?: unknown
  title?: string
  output?: unknown
  label?: string
  labelResolver?: (tool: string) => string
}

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

const stringValue = (value: unknown): string | undefined => {
  return typeof value === 'string' && value.trim() ? value : undefined
}

export const getToolLabel = (tool: string): string => TOOL_LABELS[tool] ?? tool

export const getToolDisplayName = ({ tool, input, title, output, label: explicitLabel, labelResolver }: ToolDisplayOptions): string => {
  const label = stringValue(explicitLabel) ?? labelResolver?.(tool) ?? getToolLabel(tool)
  const inputRecord = isRecord(input) ? input : undefined
  const detail = stringValue(title) ?? detailForTodoOutput(tool, output) ?? detailForTool(tool, inputRecord)

  return detail ? `${label}：${detail}` : label
}

const detailForTodoOutput = (tool: string, output: unknown): string | undefined => {
  if (tool !== 'todowrite' && tool !== 'todoread') return undefined

  try {
    const items: unknown = typeof output === 'string' ? JSON.parse(output) : output
    if (!Array.isArray(items)) return undefined

    const completed = items.filter((item) => isRecord(item) && item.status === 'completed').length
    return completed > 0 ? `已完成 ${completed}/${items.length}` : `共 ${items.length} 项`
  } catch {
    return undefined
  }
}

const extractFilename = (filePath: string): string => {
  const parts = filePath.replace(/\\/g, '/').split('/')
  return parts[parts.length - 1] || filePath
}

const detailForTool = (tool: string, input: Record<string, unknown> | undefined): string | undefined => {
  if (!input) return undefined

  if (tool === 'bash') return stringValue(input.description)
  if (tool === 'read' || tool === 'write' || tool === 'edit' || tool === 'multiedit') {
    const path = stringValue(input.filePath) ?? stringValue(input.path)
    return path ? extractFilename(path) : undefined
  }
  if (tool === 'glob') return stringValue(input.pattern)
  if (tool === 'grep') {
    const pattern = stringValue(input.pattern)
    const include = stringValue(input.include)
    if (pattern && include) return `${pattern} (${include})`
    return pattern
  }
  if (tool === 'webfetch') return stringValue(input.url)
  if (tool === 'websearch') return stringValue(input.query)
  if (tool === 'task' || tool === 'question') return stringValue(input.description) ?? stringValue(input.prompt)
  if (tool === 'skill') return stringValue(input.name)

  return undefined
}

export const partPriority = (type: ConversationPartType): number => {
  if (type === 'reasoning') return 0
  if (type === 'text') return 1
  if (type === 'tool') return 2
  return 3
}
