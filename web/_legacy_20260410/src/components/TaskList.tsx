import { useMemo } from 'react'
import { useRetryTask, useTasks } from '../hooks/useTasks'
import { Task, TaskCard } from '../types'

export function TaskList() {
  const { data: tasks, isLoading, error } = useTasks()

  if (isLoading) {
    return (
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">任务列表</h2>
          <p className="card-description">加载中...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">任务列表</h2>
          <p className="card-description text-error">加载失败: {error.message}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">任务列表</h2>
          <p className="card-description">共 {tasks?.length || 0} 个任务</p>
        </div>
      </div>

      <div className="table-container">
        <table className="table">
          <thead className="table-header">
            <tr className="table-row">
              <th className="table-head">ID</th>
              <th className="table-head">状态</th>
              <th className="table-head">类型</th>
              <th className="table-head">Dispatch</th>
              <th className="table-head">Wave</th>
              <th className="table-head">Topo Rank</th>
              <th className="table-head">创建时间</th>
              <th className="table-head">操作</th>
            </tr>
          </thead>
          <tbody className="table-body">
            {tasks?.map((task: Task) => (
              <TaskRow key={task.id} task={task} />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function TaskRow({ task }: { task: Task }) {
  const card = useParsedTaskCard(task.card_json)

  return (
    <tr className="table-row">
      <td className="table-cell font-mono text-xs">{task.id}</td>
      <td className="table-cell">
        <StateBadge state={task.state} />
      </td>
      <td className="table-cell">{card?.type || '-'}</td>
      <td className="table-cell font-mono text-xs">{task.dispatch_ref}</td>
      <td className="table-cell">{task.wave}</td>
      <td className="table-cell">{task.topo_rank}</td>
      <td className="table-cell text-xs">{formatTimestamp(task.created_at)}</td>
      <td className="table-cell">
        <TaskActions task={task} />
      </td>
    </tr>
  )
}

function useParsedTaskCard(cardJSON: string): TaskCard | null {
  return useMemo(() => {
    if (!cardJSON) {
      return null
    }

    try {
      return JSON.parse(cardJSON) as TaskCard
    } catch {
      return null
    }
  }, [cardJSON])
}

function formatTimestamp(value: string): string {
  const time = Date.parse(value)
  if (Number.isNaN(time)) {
    return '-'
  }
  return new Date(time).toLocaleString('zh-CN')
}

function StateBadge({ state }: { state: string }) {
  const stateConfig: Record<string, { label: string; className: string }> = {
    queued: { label: '排队中', className: 'badge-queued' },
    routed: { label: '已路由', className: 'badge-queued' },
    workspace_prepared: { label: '工作区就绪', className: 'badge-queued' },
    running: { label: '执行中', className: 'badge-running' },
    patch_ready: { label: '补丁就绪', className: 'badge-running' },
    verified: { label: '已验证', className: 'badge-success' },
    merged: { label: '已合并', className: 'badge-success' },
    done: { label: '完成', className: 'badge-success' },
    retry_waiting: { label: '等待重试', className: 'badge-warning' },
    verify_failed: { label: '验证失败', className: 'badge-error' },
    apply_failed: { label: '应用失败', className: 'badge-error' },
    failed: { label: '失败', className: 'badge-error' },
  }

  const config = stateConfig[state] || { label: state, className: 'badge-queued' }

  return <span className={`badge ${config.className}`}>{config.label}</span>
}

function TaskActions({ task }: { task: Task }) {
  const { mutate: retry, isPending, error } = useRetryTask()
  const canRetry = ['running', 'verify_failed'].includes(task.state)

  if (!canRetry) {
    return null
  }

  return (
    <div className="space-y-2">
      <div className="flex gap-2">
        <button
          onClick={() => retry(task.id)}
          className="button button-primary button-sm"
          disabled={isPending}
        >
          {isPending ? '处理中...' : '重试'}
        </button>
      </div>
      {error && <p className="text-xs text-error">{error.message}</p>}
    </div>
  )
}
