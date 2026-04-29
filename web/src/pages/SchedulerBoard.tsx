import React, { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, CheckCircle2, CircleDashed, Clock, Filter, Import, Info, Play, Plus, Radar, RefreshCcw, RotateCcw, Tag, Terminal, User, X, GitBranch, Activity, Zap } from 'lucide-react';
import { schedulerApi } from '../api/schedulerApi';
import { useProject } from '../hooks/useProject';
import { Agent, DispatchMode, DispatchStatus, TaskStatus, TaskType } from '../types/scheduler';
import type { BulkImportTaskDraft, CreateScheduledTaskInput, ScheduledTask, UpdateScheduledTaskInput } from '../types/scheduler';

const agentOptions = [Agent.CLAUDE, Agent.GEMINI, Agent.CODEX];
const statusOptions = [TaskStatus.BACKLOG, TaskStatus.READY, TaskStatus.ASSIGNED, TaskStatus.IN_PROGRESS, TaskStatus.REVIEW, TaskStatus.TRIAGE, TaskStatus.REVIEW_PENDING, TaskStatus.VERIFIED, TaskStatus.BLOCKED, TaskStatus.DONE];
const typeOptions = [TaskType.INTEGRATION, TaskType.UI, TaskType.ALGORITHM, TaskType.COLLECT, TaskType.ANALYZE, TaskType.SELECT, TaskType.LIST, TaskType.DECISION_LOGIC, TaskType.RANKING_RULE];
const columns = [
  { title: '待办', status: TaskStatus.BACKLOG },
  { title: '就绪', status: TaskStatus.READY },
  { title: '已分配', status: TaskStatus.ASSIGNED },
  { title: '执行中', status: TaskStatus.IN_PROGRESS },
  { title: '分诊中', status: TaskStatus.TRIAGE },
  { title: '待审核', status: TaskStatus.REVIEW_PENDING },
  { title: '待评审', status: TaskStatus.REVIEW },
  { title: '已验证', status: TaskStatus.VERIFIED },
  { title: '已阻塞', status: TaskStatus.BLOCKED },
  { title: '已完成', status: TaskStatus.DONE },
];
const statusLabels: Record<string, string> = { backlog: '待办', ready: '就绪', assigned: '已分配', in_progress: '执行中', triage: '分诊中', review_pending: '待审核', review: '待评审', verified: '已验证', blocked: '已阻塞', done: '已完成' };
const typeLabels: Record<string, string> = { integration: '集成', ui: '前端', algorithm: '算法', collect: '采集', analyze: '分析', select: '选品', list: '列表', 'decision-logic': '决策逻辑', 'ranking-rule': '排序规则' };
const dispatchModeLabels: Record<string, string> = { auto: '自动调度', manual: '手动调度' };
const dispatchStatusLabels: Record<string, string> = { pending: '待派发', queued: '排队中', dispatched: '已派发', running: '运行中', failed: '失败', completed: '已完成', triage: '分诊中', review_pending: '待审核' };
const failureCodeLabels: Record<string, string> = { artifact_missing: '产物缺失', artifact_spec_mismatch: '产物路径错误', git_no_changes: '无变更', command_exit_nonzero: '命令失败', workspace_write_failed: '写入失败', dependency_failed: '依赖失败', non_retryable_failure: '不可重试' };
const coordinationStageLabels: Record<string, string> = { triage: '分诊', remediation: '补救中', remediation_created: '已创建补救', under_review: '审核中', review_pending: '等待审核', review_approved: '审核通过', rework_created: '已创建返工', rework: '返工中', stopped: '已止损' };
const systemTaskTypeLabels: Record<string, string> = { 'artifact-fix': '产物修复', 'noop-review': '无操作审核', 'debug-failure': '调试失败', 'code-review': '代码审核', 'rework': '返工', 'triage-failure': '分诊失败' };
const inputStyle: React.CSSProperties = { background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-color)', borderRadius: 8, color: 'white', padding: '0.75rem' };
const bulkPanelStyle: React.CSSProperties = { marginTop: '1rem', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 12, padding: '1rem', background: 'rgba(255,255,255,0.02)' };
type ValidBulkDraft = BulkImportTaskDraft & { payload: CreateScheduledTaskInput };

const createDefaultForm = (): CreateScheduledTaskInput => ({ title: '', owner_agent: Agent.CLAUDE, status: TaskStatus.BACKLOG, type: TaskType.INTEGRATION, priority: 3, description: '', next_action: '', dispatch_mode: DispatchMode.AUTO, auto_dispatch_enabled: true });
const parseBulkTaskInput = (input: string, defaults: Pick<CreateScheduledTaskInput, 'owner_agent' | 'status' | 'type' | 'priority'>): BulkImportTaskDraft[] =>
  input.split(/\r?\n/).reduce<BulkImportTaskDraft[]>((drafts, line, index) => {
    const raw = line.trim();
    if (!raw) return drafts;
    const segments = raw.split('|').map((segment) => segment.trim());
    const title = segments[0];
    if (!title) {
      drafts.push({ line: index + 1, raw, error: '缺少任务标题' });
      return drafts;
    }
    drafts.push({
      line: index + 1,
      raw,
      payload: {
        title,
        description: segments[1] || undefined,
        next_action: segments[2] || undefined,
        owner_agent: defaults.owner_agent,
        status: defaults.status,
        type: defaults.type,
        priority: defaults.priority,
      },
    });
    return drafts;
  }, []);
const getPriorityColor = (priority: number) => { switch (priority) { case 1: return '#ff4d4f'; case 2: return '#ffa940'; case 3: return '#ffec3d'; default: return '#00f2ea'; } };
const getDispatchColor = (dispatchStatus: DispatchStatus) => { if (dispatchStatus === DispatchStatus.FAILED) return 'var(--accent-magenta)'; if (dispatchStatus === DispatchStatus.COMPLETED) return 'var(--accent-green)'; if (dispatchStatus === DispatchStatus.RUNNING) return 'var(--accent-cyan)'; if (dispatchStatus === DispatchStatus.QUEUED || dispatchStatus === DispatchStatus.DISPATCHED) return 'var(--accent-yellow)'; return 'var(--text-secondary)'; };
const getColumnIcon = (status: TaskStatus) => { if (status === TaskStatus.DONE) return <CheckCircle2 size={16} color="var(--accent-green)" />; if (status === TaskStatus.BLOCKED) return <AlertCircle size={16} color="var(--accent-magenta)" />; if (status === TaskStatus.VERIFIED) return <Radar size={16} color="var(--accent-yellow)" />; return <Clock size={16} color="var(--accent-cyan)" />; };
const canDispatch = (task: ScheduledTask) => task.dispatch_status !== DispatchStatus.RUNNING && task.dispatch_status !== DispatchStatus.DISPATCHED && task.dispatch_status !== DispatchStatus.QUEUED;
const canRetry = (task: ScheduledTask) => task.dispatch_status === DispatchStatus.FAILED || task.status === TaskStatus.BLOCKED;
const formatLabel = (map: Record<string, string>, value?: string) => (!value ? '-' : map[value] ?? value);

const SchedulerBoard: React.FC = () => {
  const queryClient = useQueryClient();
  const { projectId, project } = useProject();
  const [selectedTask, setSelectedTask] = useState<ScheduledTask | null>(null);
  const [agentFilter, setAgentFilter] = useState<'ALL' | Agent>('ALL');
  const [statusFilter, setStatusFilter] = useState<'ALL' | TaskStatus>('ALL');
  const [bulkImportOpen, setBulkImportOpen] = useState(false);
  const [bulkInput, setBulkInput] = useState('');
  const [bulkSummary, setBulkSummary] = useState<string | null>(null);
  const [bulkErrorLines, setBulkErrorLines] = useState<BulkImportTaskDraft[]>([]);
  const [form, setForm] = useState<CreateScheduledTaskInput>(createDefaultForm());
  const activeSelectedTask = selectedTask?.project_id === projectId || !selectedTask?.project_id ? selectedTask : null;
  const { data: tasks = [], isLoading } = useQuery({ queryKey: ['scheduled-tasks', projectId], queryFn: () => schedulerApi.getTasks(projectId), enabled: Boolean(projectId), refetchInterval: 4000 });
  const { data: stats } = useQuery({ queryKey: ['scheduler-stats', projectId], queryFn: () => schedulerApi.getStats(projectId), enabled: Boolean(projectId), refetchInterval: 4000 });
  const { data: systemHealth } = useQuery({ queryKey: ['system-health'], queryFn: () => schedulerApi.getSystemHealth(), refetchInterval: 5000 });
  const { data: systemWorkers } = useQuery({ queryKey: ['system-workers'], queryFn: () => schedulerApi.getSystemWorkers(), refetchInterval: 5000 });
  const { data: selectedExecution } = useQuery({ queryKey: ['task-execution', projectId, activeSelectedTask?.id], queryFn: () => schedulerApi.getTaskExecution(projectId, activeSelectedTask!.id), enabled: Boolean(projectId && activeSelectedTask?.id), refetchInterval: 2000 });
  const { data: selectedLineage } = useQuery({ queryKey: ['task-lineage', projectId, activeSelectedTask?.id], queryFn: () => schedulerApi.getTaskLineage(projectId, activeSelectedTask!.id), enabled: Boolean(projectId && activeSelectedTask?.id), refetchInterval: 4000 });
  const invalidateProjectQueries = () => { queryClient.invalidateQueries({ queryKey: ['scheduled-tasks', projectId] }); queryClient.invalidateQueries({ queryKey: ['scheduler-stats', projectId] }); };
  const createTaskMutation = useMutation({ mutationFn: (payload: CreateScheduledTaskInput) => schedulerApi.createTask(projectId, payload), onSuccess: () => { setForm(createDefaultForm()); invalidateProjectQueries(); } });
  const bulkImportMutation = useMutation({ mutationFn: (drafts: ValidBulkDraft[]) => schedulerApi.createTasksBulk(projectId, drafts), onSuccess: (result) => { const summaryParts: string[] = []; if (result.created.length) summaryParts.push(`已创建 ${result.created.length} 个任务`); if (result.failed.length) summaryParts.push(`${result.failed.length} 个任务创建失败`); setBulkSummary(summaryParts.length ? summaryParts.join(' ｜ ') : '没有创建任何任务'); setBulkErrorLines(result.failed); if (!result.failed.length) setBulkInput(''); invalidateProjectQueries(); } });
  const updateTaskMutation = useMutation({ mutationFn: ({ taskId, payload }: { taskId: string; payload: UpdateScheduledTaskInput }) => schedulerApi.updateTask(projectId, taskId, payload), onSuccess: (updatedTask) => { setSelectedTask(updatedTask); invalidateProjectQueries(); } });
  const dispatchTaskMutation = useMutation({ mutationFn: (taskId: string) => schedulerApi.dispatchTask(projectId, taskId), onSuccess: (updatedTask) => { setSelectedTask(updatedTask); invalidateProjectQueries(); queryClient.invalidateQueries({ queryKey: ['task-execution', projectId, updatedTask.id] }); } });
  const retryTaskMutation = useMutation({ mutationFn: (taskId: string) => schedulerApi.retryTask(projectId, taskId), onSuccess: (updatedTask) => { setSelectedTask(updatedTask); invalidateProjectQueries(); queryClient.invalidateQueries({ queryKey: ['task-execution', projectId, updatedTask.id] }); } });
  const recentUpdates = useMemo(() => stats?.recent_updates ?? [], [stats]);
  const runtimeHealth = useMemo(() => stats?.runtime_health ?? [], [stats]);
  const filteredTasks = useMemo(() => tasks.filter((task) => { const matchesAgent = agentFilter === 'ALL' || task.owner_agent === agentFilter; const matchesStatus = statusFilter === 'ALL' || task.status === statusFilter; return matchesAgent && matchesStatus; }), [tasks, agentFilter, statusFilter]);
  const bulkDrafts = useMemo(() => parseBulkTaskInput(bulkInput, { owner_agent: form.owner_agent, status: form.status ?? TaskStatus.BACKLOG, type: form.type ?? TaskType.INTEGRATION, priority: form.priority ?? 3 }), [bulkInput, form.owner_agent, form.priority, form.status, form.type]);
  const validBulkDrafts = useMemo(() => bulkDrafts.filter((draft): draft is ValidBulkDraft => Boolean(draft.payload)), [bulkDrafts]);
  const invalidBulkDrafts = useMemo(() => bulkDrafts.filter((draft) => !draft.payload), [bulkDrafts]);
  const updateSelectedTask = (payload: UpdateScheduledTaskInput) => { if (!activeSelectedTask) return; setSelectedTask({ ...activeSelectedTask, ...payload }); updateTaskMutation.mutate({ taskId: activeSelectedTask.id, payload }); };
  const handleBulkSubmit = () => { setBulkSummary(null); if (!validBulkDrafts.length) { setBulkErrorLines(invalidBulkDrafts.length ? invalidBulkDrafts : [{ line: 0, raw: '', error: '没有可导入的有效任务' }]); return; } bulkImportMutation.mutate(validBulkDrafts); };

  const renderTaskCard = (task: ScheduledTask) => (
    <div key={task.id} className="glass-card task-card" onClick={() => setSelectedTask(task)} style={{ padding: '0.8rem', marginBottom: '0.75rem', cursor: 'pointer', borderLeft: `4px solid ${getPriorityColor(task.priority)}`, transition: 'transform 0.2s' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '0.5rem' }}>
        <span className="mono" style={{ fontSize: '0.68rem', color: 'var(--text-secondary)' }}>{task.id}</span>
        <Tag size={12} color="var(--accent-cyan)" />
      </div>
      <h4 style={{ fontSize: '0.86rem', marginBottom: '0.5rem', lineHeight: 1.3 }}>{task.title}</h4>
      <p style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', minHeight: '2.5em' }}>{task.next_action || task.description || '暂无说明'}</p>
      {task.block_reason && task.status === TaskStatus.BLOCKED && <p style={{ fontSize: '0.7rem', color: 'var(--accent-magenta)', marginTop: '0.5rem' }}>{task.block_reason}</p>}
      {task.last_dispatch_error && <p style={{ fontSize: '0.7rem', color: 'var(--accent-magenta)', marginTop: '0.5rem' }}>{task.last_dispatch_error}</p>}
      {task.failure_code && <p style={{ fontSize: '0.7rem', color: 'var(--accent-yellow)', marginTop: '0.25rem' }}>失败分类: {formatLabel(failureCodeLabels, task.failure_code)}</p>}
      {task.coordination_stage && <p style={{ fontSize: '0.7rem', color: 'var(--accent-cyan)', marginTop: '0.25rem' }}>阶段: {formatLabel(coordinationStageLabels, task.coordination_stage)}</p>}
      {task.parent_task_id && <p style={{ fontSize: '0.68rem', color: 'var(--text-secondary)', marginTop: '0.25rem' }}>父任务: {task.parent_task_id}</p>}
      {task.auto_repair_count !== undefined && task.auto_repair_count > 0 && <p style={{ fontSize: '0.68rem', color: 'var(--text-secondary)', marginTop: '0.25rem' }}>补救次数: {task.auto_repair_count}/2</p>}
      {task.next_automatic_action && (task.status === TaskStatus.BLOCKED || task.status === TaskStatus.TRIAGE || task.status === TaskStatus.REVIEW_PENDING) && <p style={{ fontSize: '0.68rem', color: 'var(--accent-green)', marginTop: '0.25rem' }}>下一步: {task.next_automatic_action}</p>}
      {task.stop_loss_status && task.stop_loss_status !== 'budget OK' && <p style={{ fontSize: '0.68rem', color: task.stop_loss_status.includes('LIMIT_HIT') ? 'var(--accent-magenta)' : 'var(--text-secondary)', marginTop: '0.25rem' }}>止损: {task.stop_loss_status}</p>}
      {task.recoverable && task.status === TaskStatus.BLOCKED && <div style={{ background: 'rgba(0,242,234,0.1)', padding: '2px 6px', borderRadius: '4px', fontSize: '0.65rem', color: 'var(--accent-cyan)', marginTop: '0.25rem', display: 'inline-block' }}>可恢复</div>}
      {systemTaskTypeLabels[task.type] && <div style={{ background: 'rgba(255,236,61,0.1)', padding: '2px 6px', borderRadius: '4px', fontSize: '0.65rem', color: 'var(--accent-yellow)', marginTop: '0.35rem', display: 'inline-block' }}>{systemTaskTypeLabels[task.type]}</div>}
      <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginTop: '0.75rem', flexWrap: 'wrap' }}>
        <div style={{ background: 'rgba(255,255,255,0.05)', padding: '2px 6px', borderRadius: '4px', fontSize: '0.65rem', display: 'flex', alignItems: 'center', gap: '4px' }}><User size={10} />{task.owner_agent}</div>
        <div style={{ background: 'rgba(255,255,255,0.05)', padding: '2px 6px', borderRadius: '4px', fontSize: '0.65rem', color: getDispatchColor(task.dispatch_status) }}>{formatLabel(dispatchStatusLabels, task.dispatch_status)}</div>
        {task.execution_runtime && <div className="mono" style={{ fontSize: '0.65rem', opacity: 0.7 }}>{task.execution_runtime}</div>}
        {task.execution_session_id && <div className="mono" style={{ fontSize: '0.65rem', opacity: 0.7 }}>{task.execution_session_id.slice(0, 8)}</div>}
        {task.dispatch_status === DispatchStatus.RUNNING && task.last_heartbeat_at && (
          <div className="mono" style={{ fontSize: '0.65rem', color: task.stalled ? 'var(--accent-magenta)' : 'var(--accent-green)' }}>
            ♥ {new Date(task.last_heartbeat_at).toLocaleTimeString()}
            {task.stalled && <span style={{ marginLeft: '0.25rem' }}>⚠陈旧</span>}
          </div>
        )}
        {task.dispatch_status === DispatchStatus.RUNNING && !task.last_heartbeat_at && (
          <div className="mono" style={{ fontSize: '0.65rem', color: 'var(--accent-yellow)' }}>♥ 无心跳</div>
        )}
        <div className="mono" style={{ fontSize: '0.65rem', opacity: 0.6 }}>P{task.priority}</div>
        <div className="mono" style={{ fontSize: '0.65rem', opacity: 0.6 }}>{new Date(task.updated_at).toLocaleTimeString()}</div>
      </div>
      {task.dispatch_status !== DispatchStatus.PENDING && (
        <div style={{ marginTop: '0.75rem' }}>
          <button
            onClick={(event) => {
              event.stopPropagation();
              setSelectedTask(task);
            }}
            style={{ display: 'flex', alignItems: 'center', gap: '0.35rem', padding: '0.4rem 0.65rem', borderRadius: 8, border: '1px solid rgba(255,255,255,0.12)', background: 'rgba(255,255,255,0.04)', color: 'white', cursor: 'pointer', fontSize: '0.72rem' }}
          >
            <Terminal size={12} />
            查看执行
          </button>
        </div>
      )}
    </div>
  );

  return (
    <div className="scheduler-board-page">
      <div style={{ marginBottom: '1.5rem', display: 'grid', gridTemplateColumns: '1.25fr 0.75fr', gap: '1rem' }}>
        <div className="glass-card">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem', marginBottom: '1rem', flexWrap: 'wrap' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
              <Plus size={18} color="var(--accent-cyan)" />
              <div>
                <h2 className="neon-text" style={{ fontSize: '1.35rem' }}>调度看板</h2>
                <p style={{ color: 'var(--text-secondary)', fontSize: '0.78rem' }}>{project ? project.name : '加载项目中…'}</p>
              </div>
            </div>
            <button onClick={() => { setBulkImportOpen((open) => !open); setBulkSummary(null); setBulkErrorLines([]); }} style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', padding: '0.7rem 1rem', borderRadius: 8, border: '1px solid rgba(0, 242, 234, 0.35)', background: bulkImportOpen ? 'rgba(0, 242, 234, 0.12)' : 'transparent', color: 'var(--accent-cyan)', cursor: 'pointer', fontWeight: 700 }}>
              {bulkImportOpen ? <X size={16} /> : <Import size={16} />}
              {bulkImportOpen ? '关闭批量导入' : '批量导入'}
            </button>
          </div>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem', marginBottom: '1rem' }}>这里管理跨代理任务流。Claude、Gemini、Codex 会按分配状态和优先级推进执行。</p>
          <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr 1fr', gap: '0.75rem', marginBottom: '0.75rem' }}>
            <input value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} placeholder="任务标题" style={inputStyle} />
            <select value={form.owner_agent} onChange={(event) => setForm({ ...form, owner_agent: event.target.value as Agent })} style={{ ...inputStyle, background: '#11131a' }}>{agentOptions.map((agent) => <option key={agent} value={agent}>{agent}</option>)}</select>
            <select value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value as TaskStatus })} style={{ ...inputStyle, background: '#11131a' }}>{statusOptions.map((status) => <option key={status} value={status}>{formatLabel(statusLabels, status)}</option>)}</select>
            <select value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value as TaskType })} style={{ ...inputStyle, background: '#11131a' }}>{typeOptions.map((type) => <option key={type} value={type}>{formatLabel(typeLabels, type)}</option>)}</select>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr', gap: '0.75rem', marginBottom: '0.75rem' }}>
            <input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} placeholder="任务描述" style={inputStyle} />
            <input type="number" min={1} max={5} value={form.priority} onChange={(event) => setForm({ ...form, priority: Number(event.target.value || 3) })} placeholder="优先级" style={inputStyle} />
            <button onClick={() => createTaskMutation.mutate(form)} disabled={!projectId || !form.title.trim() || createTaskMutation.isPending} style={{ background: 'var(--accent-cyan)', color: '#071014', border: 'none', borderRadius: 8, fontWeight: 700, cursor: 'pointer' }}>{createTaskMutation.isPending ? '创建中…' : '创建任务'}</button>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr', gap: '0.75rem', marginBottom: '0.75rem' }}>
            <input value={form.next_action} onChange={(event) => setForm({ ...form, next_action: event.target.value })} placeholder="下一步行动" style={{ ...inputStyle, width: '100%' }} />
            <select value={form.dispatch_mode} onChange={(event) => setForm({ ...form, dispatch_mode: event.target.value as DispatchMode })} style={{ ...inputStyle, background: '#11131a' }}><option value={DispatchMode.AUTO}>{dispatchModeLabels[DispatchMode.AUTO]}</option><option value={DispatchMode.MANUAL}>{dispatchModeLabels[DispatchMode.MANUAL]}</option></select>
            <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', padding: '0.75rem', border: '1px solid var(--border-color)', borderRadius: 8 }}><input type="checkbox" checked={form.auto_dispatch_enabled ?? true} onChange={(event) => setForm({ ...form, auto_dispatch_enabled: event.target.checked })} />自动派发</label>
          </div>
          {bulkImportOpen && (
            <div style={bulkPanelStyle}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem', marginBottom: '0.75rem', flexWrap: 'wrap' }}>
                <div>
                  <h3 style={{ fontSize: '1rem', marginBottom: '0.25rem' }}>批量导入任务</h3>
                  <p style={{ color: 'var(--text-secondary)', fontSize: '0.78rem' }}>每行一个任务。格式：<span className="mono">标题 | 描述 | 下一步行动</span></p>
                </div>
                <div className="mono" style={{ fontSize: '0.75rem', opacity: 0.7 }}>默认值：{form.owner_agent} / {formatLabel(statusLabels, form.status)} / {formatLabel(typeLabels, form.type)} / P{form.priority}</div>
              </div>
              <textarea value={bulkInput} onChange={(event) => setBulkInput(event.target.value)} rows={8} placeholder={['补齐自动派发链路 | 让依赖满足后自动推进下一条任务 | 补充联调验证', '收敛看板筛选器 | 简化状态标签和间距 | 跑前端构建', '整理项目接入文档 | 统一接入步骤和约束 | 更新 README'].join('\n')} style={{ ...inputStyle, width: '100%', resize: 'vertical', marginBottom: '0.75rem' }} />
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem', flexWrap: 'wrap', marginBottom: '0.75rem' }}>
                <div style={{ color: 'var(--text-secondary)', fontSize: '0.78rem' }}>准备创建 <strong style={{ color: 'white' }}>{validBulkDrafts.length}</strong> 个任务{invalidBulkDrafts.length ? ` ｜ ${invalidBulkDrafts.length} 行需要处理` : ''}</div>
                <button onClick={handleBulkSubmit} disabled={!projectId || !validBulkDrafts.length || bulkImportMutation.isPending} style={{ background: 'var(--accent-yellow)', color: '#1d1400', border: 'none', borderRadius: 8, fontWeight: 700, cursor: 'pointer', padding: '0.7rem 1rem' }}>{bulkImportMutation.isPending ? '导入中…' : `创建 ${validBulkDrafts.length} 个任务`}</button>
              </div>
              {bulkSummary && <div style={{ marginBottom: '0.75rem', padding: '0.75rem', borderRadius: 8, background: 'rgba(255,255,255,0.04)', fontSize: '0.8rem' }}>{bulkSummary}</div>}
              {bulkErrorLines.length > 0 && <div style={{ padding: '0.75rem', borderRadius: 8, background: 'rgba(255, 0, 80, 0.08)', border: '1px solid rgba(255, 0, 80, 0.18)' }}><div style={{ fontSize: '0.8rem', fontWeight: 700, marginBottom: '0.4rem', color: 'var(--accent-magenta)' }}>需要处理的行</div><div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>{bulkErrorLines.map((draft, index) => <div key={`${draft.line}-${index}`} style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>第 {draft.line || '?'} 行：{draft.error || '导入失败'}{draft.raw ? ` ｜ ${draft.raw}` : ''}</div>)}</div></div>}
            </div>
          )}
        </div>
        <div className="glass-card">
          <h3 style={{ fontSize: '1rem', marginBottom: '0.9rem' }}>看板快照</h3>
          <div style={{ display: 'grid', gap: '0.85rem' }}>
            <div><span style={{ color: 'var(--text-secondary)', fontSize: '0.75rem' }}>总任务数</span><div className="mono" style={{ fontSize: '1.55rem', fontWeight: 700 }}>{stats?.total_tasks || 0}</div></div>
            <div><span style={{ color: 'var(--text-secondary)', fontSize: '0.75rem' }}>阻塞任务</span><div className="mono" style={{ fontSize: '1.55rem', fontWeight: 700 }}>{stats?.blocked_count || 0}</div></div>
            <div><span style={{ color: 'var(--text-secondary)', fontSize: '0.75rem' }}>待调度</span><div className="mono" style={{ fontSize: '1.55rem', fontWeight: 700 }}>{stats?.queued_tasks || 0}</div></div>
            <div><span style={{ color: 'var(--text-secondary)', fontSize: '0.75rem' }}>当前重点</span><div style={{ fontWeight: 600, lineHeight: 1.4 }}>{stats?.current_focus || '暂无'}</div></div>
          </div>
        </div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', gap: '1rem', marginBottom: '1rem' }}>
        <div className="glass-card">
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem', marginBottom: '0.75rem' }}><Filter size={16} color="var(--accent-cyan)" /><h3 style={{ fontSize: '0.95rem' }}>看板筛选</h3></div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr auto', gap: '0.75rem' }}>
            <select value={agentFilter} onChange={(event) => setAgentFilter(event.target.value as 'ALL' | Agent)} style={{ ...inputStyle, background: '#11131a' }}><option value="ALL">所有代理</option>{agentOptions.map((agent) => <option key={agent} value={agent}>{agent}</option>)}</select>
            <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value as 'ALL' | TaskStatus)} style={{ ...inputStyle, background: '#11131a' }}><option value="ALL">所有状态</option>{statusOptions.map((status) => <option key={status} value={status}>{formatLabel(statusLabels, status)}</option>)}</select>
            <button onClick={() => { setAgentFilter('ALL'); setStatusFilter('ALL'); invalidateProjectQueries(); }} style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.5rem', padding: '0 1rem', borderRadius: 8, border: '1px solid var(--border-color)', background: 'transparent', color: 'white', cursor: 'pointer' }}><RefreshCcw size={14} /> 刷新</button>
          </div>
        </div>
        <div className="glass-card">
          <h3 style={{ fontSize: '0.95rem', marginBottom: '0.75rem' }}>最近更新</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>{recentUpdates.length ? recentUpdates.map((task) => <div key={task.id} style={{ padding: '0.6rem', borderRadius: 8, background: 'rgba(255,255,255,0.03)' }}><div style={{ display: 'flex', justifyContent: 'space-between', gap: '0.75rem' }}><strong style={{ fontSize: '0.8rem' }}>{task.title}</strong><span className="mono" style={{ fontSize: '0.7rem', opacity: 0.6 }}>{new Date(task.updated_at).toLocaleTimeString()}</span></div><div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)' }}>{task.owner_agent} ｜ {formatLabel(statusLabels, task.status)} ｜ {formatLabel(dispatchStatusLabels, task.dispatch_status)}</div></div>) : <div style={{ color: 'var(--text-secondary)', fontSize: '0.8rem' }}>暂无更新</div>}</div>
        </div>
        <div className="glass-card">
          <h3 style={{ fontSize: '0.95rem', marginBottom: '0.75rem' }}>运行时健康</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>{runtimeHealth.length ? runtimeHealth.map((runtime) => <div key={runtime.owner_agent} style={{ padding: '0.6rem', borderRadius: 8, background: 'rgba(255,255,255,0.03)' }}><div style={{ display: 'flex', justifyContent: 'space-between', gap: '0.75rem' }}><strong style={{ fontSize: '0.8rem' }}>{runtime.owner_agent}</strong><span className="mono" style={{ fontSize: '0.7rem', color: runtime.status === 'available' ? 'var(--accent-green)' : 'var(--accent-magenta)' }}>{runtime.status === 'available' ? '可用' : '不可用'}</span></div><div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)' }}>{runtime.runtime} ｜ 会话 {runtime.active_sessions}</div></div>) : <div style={{ color: 'var(--text-secondary)', fontSize: '0.8rem' }}>暂无运行时数据</div>}</div>
        </div>
        <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.5rem' }}>
            <Activity size={16} color={systemHealth?.status === 'ok' ? 'var(--accent-green)' : 'var(--accent-magenta)'} />
            <h3 style={{ fontSize: '0.9rem' }}>后端状态</h3>
          </div>
          {systemHealth ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                <Zap size={12} color={systemHealth.status === 'ok' ? 'var(--accent-green)' : 'var(--accent-magenta)'} />
                <span className="mono" style={{ fontSize: '0.75rem', color: systemHealth.status === 'ok' ? 'var(--accent-green)' : 'var(--accent-magenta)' }}>{systemHealth.status === 'ok' ? '在线' : '离线'}</span>
                <span className="mono" style={{ fontSize: '0.7rem', opacity: 0.6 }}>v{systemHealth.version}</span>
              </div>
              <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>运行 {systemHealth.uptime}</div>
              {systemWorkers?.workers && (
                <div style={{ marginTop: '0.35rem', fontSize: '0.72rem' }}>
                  <div style={{ color: 'var(--text-secondary)', marginBottom: '0.25rem' }}>后台 workers：</div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                    {systemWorkers.workers.filter(w => w.running).map((w) => (
                      <div key={w.name} style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
                        <CheckCircle2 size={10} color="var(--accent-green)" />
                        <span style={{ color: 'var(--accent-green)' }}>{w.name}</span>
                      </div>
                    ))}
                    {systemWorkers.workers.filter(w => !w.running).map((w) => (
                      <div key={w.name} style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
                        <CircleDashed size={10} color="var(--text-secondary)" />
                        <span style={{ color: 'var(--text-secondary)' }}>{w.name}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div style={{ color: 'var(--accent-magenta)', fontSize: '0.8rem', display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
              <AlertCircle size={12} /> 后端离线
            </div>
          )}
        </div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '1rem', alignItems: 'start', paddingBottom: '1rem' }}>
        {columns.map((column) => {
          const columnTasks = filteredTasks.filter((task) => task.status === column.status);
          return (
            <div key={column.title} style={{ minWidth: 0, minHeight: '320px', background: 'rgba(255,255,255,0.02)', borderRadius: '12px', padding: '1rem', display: 'flex', flexDirection: 'column' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '1rem', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '0.5rem' }}>
                {getColumnIcon(column.status)}
                <h3 style={{ fontSize: '0.9rem', fontWeight: 700, letterSpacing: '1px' }}>{column.title}</h3>
                <span style={{ marginLeft: 'auto', fontSize: '0.75rem', opacity: 0.5 }}>{columnTasks.length}</span>
              </div>
              <div style={{ flex: 1 }}>
                {isLoading ? <p style={{ fontSize: '0.8rem', opacity: 0.5 }}>加载中…</p> : columnTasks.length ? columnTasks.map(renderTaskCard) : <div style={{ padding: '1rem 0.5rem', opacity: 0.45, fontSize: '0.8rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}><CircleDashed size={14} /> 暂无任务</div>}
              </div>
            </div>
          );
        })}
      </div>
      {activeSelectedTask && (
        <div style={{ position: 'fixed', top: 0, right: 0, width: '420px', height: '100vh', background: '#0a0a0f', borderLeft: '1px solid var(--border-color)', boxShadow: '-10px 0 30px rgba(0,0,0,0.5)', zIndex: 1000, padding: '2rem', display: 'flex', flexDirection: 'column', overflowY: 'auto' }}>
          <button onClick={() => setSelectedTask(null)} style={{ alignSelf: 'flex-start', background: 'none', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', marginBottom: '2rem' }}>关闭</button>
          <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: '1rem', flexWrap: 'wrap' }}>
            <span className="badge" style={{ fontSize: '0.7rem' }}>{formatLabel(typeLabels, activeSelectedTask.type)}</span>
            <span className="mono" style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>{activeSelectedTask.id}</span>
            <span className="mono" style={{ fontSize: '0.75rem', color: getDispatchColor(activeSelectedTask.dispatch_status) }}>{formatLabel(dispatchStatusLabels, activeSelectedTask.dispatch_status)}</span>
          </div>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1.5rem' }}>{activeSelectedTask.title}</h2>
          <div className="glass-card" style={{ padding: '1rem', marginBottom: '1rem' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
              <label style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>负责人</label>
              <select value={activeSelectedTask.owner_agent} onChange={(event) => updateSelectedTask({ owner_agent: event.target.value as Agent })} style={{ ...inputStyle, background: '#11131a' }}>{agentOptions.map((agent) => <option key={agent} value={agent}>{agent}</option>)}</select>
              <label style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>状态</label>
              <select value={activeSelectedTask.status} onChange={(event) => updateSelectedTask({ status: event.target.value as TaskStatus })} style={{ ...inputStyle, background: '#11131a' }}>{statusOptions.map((status) => <option key={status} value={status}>{formatLabel(statusLabels, status)}</option>)}</select>
              <label style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>优先级</label>
              <input type="number" min={1} max={5} value={activeSelectedTask.priority} onChange={(event) => updateSelectedTask({ priority: Number(event.target.value || activeSelectedTask.priority) })} style={inputStyle} />
            </div>
          </div>
          <div className="glass-card" style={{ padding: '1rem', marginBottom: '1rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem', marginBottom: '0.75rem' }}>
              <h4 style={{ fontSize: '0.9rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}><Terminal size={14} /> 调度执行</h4>
              <div style={{ display: 'flex', gap: '0.5rem' }}>
                <button onClick={() => dispatchTaskMutation.mutate(activeSelectedTask.id)} disabled={!projectId || !canDispatch(activeSelectedTask) || dispatchTaskMutation.isPending} style={{ background: 'var(--accent-cyan)', color: '#071014', border: 'none', borderRadius: 8, cursor: 'pointer', padding: '0.5rem 0.75rem', display: 'flex', alignItems: 'center', gap: '0.35rem' }}><Play size={12} /> 派发</button>
                <button onClick={() => retryTaskMutation.mutate(activeSelectedTask.id)} disabled={!projectId || !canRetry(activeSelectedTask) || retryTaskMutation.isPending} style={{ background: 'var(--accent-yellow)', color: '#1d1400', border: 'none', borderRadius: 8, cursor: 'pointer', padding: '0.5rem 0.75rem', display: 'flex', alignItems: 'center', gap: '0.35rem' }}><RotateCcw size={12} /> 重试</button>
              </div>
            </div>
            <div style={{ display: 'grid', gap: '0.55rem', fontSize: '0.78rem' }}>
              <div>模式：<span className="mono">{formatLabel(dispatchModeLabels, activeSelectedTask.dispatch_mode)}</span></div>
              <div>运行时：<span className="mono">{activeSelectedTask.execution_runtime || 'pending'}</span></div>
              <div>会话：<span className="mono">{activeSelectedTask.execution_session_id || 'n/a'}</span></div>
              <div>尝试次数：<span className="mono">{activeSelectedTask.dispatch_attempts}</span></div>
              <div>最后错误：<span style={{ color: 'var(--accent-magenta)' }}>{activeSelectedTask.last_dispatch_error || '无'}</span></div>
              {activeSelectedTask.dispatch_status === DispatchStatus.RUNNING && (
                <>
                  <div style={{ borderTop: '1px solid rgba(255,255,255,0.08)', marginTop: '0.5rem', paddingTop: '0.5rem' }}>
                    <div style={{ color: 'var(--accent-cyan)', marginBottom: '0.25rem' }}>执行心跳</div>
                    {activeSelectedTask.started_at && <div>启动时间：<span className="mono">{new Date(activeSelectedTask.started_at).toLocaleTimeString()}</span></div>}
                    {activeSelectedTask.last_heartbeat_at ? (
                      <div style={{ color: activeSelectedTask.stalled ? 'var(--accent-magenta)' : 'var(--accent-green)' }}>
                        最后心跳：<span className="mono">{new Date(activeSelectedTask.last_heartbeat_at).toLocaleTimeString()}</span>
                        {activeSelectedTask.stalled && <span style={{ marginLeft: '0.5rem', fontWeight: 700 }}>⚠ 数据陈旧</span>}
                      </div>
                    ) : <div style={{ color: 'var(--accent-yellow)' }}>最后心跳：<span>未记录</span></div>}
                    {activeSelectedTask.timeout_at && <div>超时时间：<span className="mono">{new Date(activeSelectedTask.timeout_at).toLocaleTimeString()}</span></div>}
                  </div>
                </>
              )}
            </div>
            {selectedExecution && (
              <div style={{ marginTop: '1rem', paddingTop: '1rem', borderTop: '1px solid rgba(255,255,255,0.08)' }}>
                <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>执行时间线</div>
                <div style={{ display: 'grid', gap: '0.45rem' }}>
                  {selectedExecution.events.map((event, index) => (
                    <div key={`${event.event}-${index}`} style={{ fontSize: '0.74rem', padding: '0.55rem', borderRadius: 8, background: 'rgba(255,255,255,0.03)' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', gap: '0.75rem' }}>
                        <strong>{event.event}</strong>
                        <span className="mono" style={{ opacity: 0.65 }}>{new Date(event.timestamp).toLocaleTimeString()}</span>
                      </div>
                      <div style={{ color: 'var(--text-secondary)', marginTop: '0.2rem' }}>{event.message}</div>
                    </div>
                  ))}
                </div>
                <div style={{ marginTop: '1rem' }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem', marginBottom: '0.45rem' }}>
                    <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>实时输出</div>
                    <div className="mono" style={{ fontSize: '0.68rem', opacity: 0.6 }}>2 秒刷新</div>
                  </div>
                  <div style={{ display: 'grid', gap: '0.45rem', fontSize: '0.74rem', marginBottom: '0.75rem' }}>
                    {selectedExecution.command && <div>命令：<span className="mono" style={{ opacity: 0.8 }}>{selectedExecution.command}</span></div>}
                    {selectedExecution.workspace && <div>工作区：<span className="mono" style={{ opacity: 0.8 }}>{selectedExecution.workspace}</span></div>}
                    {selectedExecution.artifacts && <div>产物目录：<span className="mono" style={{ opacity: 0.8 }}>{selectedExecution.artifacts}</span></div>}
                  </div>
                  <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-word', maxHeight: '320px', overflow: 'auto', padding: '0.85rem', borderRadius: 10, background: 'rgba(0,0,0,0.32)', border: '1px solid rgba(255,255,255,0.08)', color: '#d7f7f4', fontSize: '0.72rem', lineHeight: 1.45 }}>
                    {selectedExecution.output_tail || '当前还没有执行输出。'}
                  </pre>
                </div>
              </div>
            )}
          </div>
          <div style={{ marginBottom: '1rem' }}><h4 style={{ fontSize: '0.9rem', marginBottom: '0.5rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}><Info size={14} /> 描述</h4><textarea value={activeSelectedTask.description || ''} onChange={(event) => setSelectedTask({ ...activeSelectedTask, description: event.target.value })} onBlur={(event) => updateSelectedTask({ description: event.target.value })} rows={4} style={{ ...inputStyle, width: '100%' }} /></div>
          {(activeSelectedTask.failure_code || activeSelectedTask.coordination_stage || activeSelectedTask.next_automatic_action || activeSelectedTask.stop_loss_status) && (
            <div className="glass-card" style={{ padding: '1rem', marginBottom: '1rem' }}>
              <h4 style={{ fontSize: '0.9rem', marginBottom: '0.5rem' }}>任务诊断</h4>
              <div style={{ display: 'grid', gap: '0.45rem', fontSize: '0.78rem' }}>
                {activeSelectedTask.failure_code && <div>失败分类: <span style={{ color: 'var(--accent-yellow)' }}>{formatLabel(failureCodeLabels, activeSelectedTask.failure_code)}</span></div>}
                {activeSelectedTask.coordination_stage && <div>协调阶段: <span style={{ color: 'var(--accent-cyan)' }}>{formatLabel(coordinationStageLabels, activeSelectedTask.coordination_stage)}</span></div>}
                {activeSelectedTask.next_automatic_action && <div>下一步自动动作: <span style={{ color: 'var(--accent-green)' }}>{activeSelectedTask.next_automatic_action}</span></div>}
                {activeSelectedTask.stop_loss_status && <div>止损状态: <span style={{ color: activeSelectedTask.stop_loss_status.includes('LIMIT_HIT') ? 'var(--accent-magenta)' : 'var(--text-secondary)' }}>{activeSelectedTask.stop_loss_status}</span></div>}
                {activeSelectedTask.recoverable !== undefined && <div>可恢复: <span style={{ color: activeSelectedTask.recoverable ? 'var(--accent-green)' : 'var(--text-secondary)' }}>{activeSelectedTask.recoverable ? '是' : '否'}</span></div>}
                {activeSelectedTask.parent_task_id && <div>父任务: <span className="mono">{activeSelectedTask.parent_task_id}</span></div>}
                {activeSelectedTask.root_task_id && <div>根任务: <span className="mono">{activeSelectedTask.root_task_id}</span></div>}
                {activeSelectedTask.review_decision && <div>审核结果: <span style={{ color: activeSelectedTask.review_decision === 'approved' ? 'var(--accent-green)' : 'var(--accent-magenta)' }}>{activeSelectedTask.review_decision}</span></div>}
                {activeSelectedTask.auto_repair_count !== undefined && <div>补救次数: <span className="mono">{activeSelectedTask.auto_repair_count}/2</span></div>}
              </div>
            </div>
          )}
          {selectedLineage && (selectedLineage.ancestors.length > 0 || selectedLineage.descendants.length > 0) && (
            <div className="glass-card" style={{ padding: '1rem', marginBottom: '1rem' }}>
              <h4 style={{ fontSize: '0.9rem', marginBottom: '0.5rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}><GitBranch size={14} /> 任务谱系</h4>
              {selectedLineage.ancestors.length > 0 && (
                <div style={{ marginBottom: '0.75rem' }}>
                  <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginBottom: '0.35rem' }}>祖先任务链</div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                    {selectedLineage.ancestors.map((ancestor) => (
                      <div key={ancestor.id} style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.75rem', padding: '0.35rem 0.5rem', background: 'rgba(255,255,255,0.03)', borderRadius: '4px' }}>
                        <span className="mono" style={{ color: 'var(--accent-cyan)', fontSize: '0.7rem' }}>{ancestor.id}</span>
                        <span style={{ opacity: 0.8 }}>{ancestor.title}</span>
                        <span className="mono" style={{ fontSize: '0.65rem', opacity: 0.6 }}>{formatLabel(statusLabels, ancestor.status)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
              {selectedLineage.descendants.length > 0 && (
                <div>
                  <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginBottom: '0.35rem' }}>后代任务链</div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                    {selectedLineage.descendants.map((descendant) => (
                      <div key={descendant.id} style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.75rem', padding: '0.35rem 0.5rem', background: 'rgba(255,255,255,0.03)', borderRadius: '4px' }}>
                        <span className="mono" style={{ color: 'var(--accent-yellow)', fontSize: '0.7rem' }}>{descendant.id}</span>
                        <span style={{ opacity: 0.8 }}>{descendant.title}</span>
                        <span className="mono" style={{ fontSize: '0.65rem', opacity: 0.6 }}>{formatLabel(statusLabels, descendant.status)}</span>
                        {descendant.type === 'code-review' && <span style={{ fontSize: '0.6rem', color: 'var(--accent-cyan)' }}>(审核)</span>}
                        {descendant.type === 'rework' && <span style={{ fontSize: '0.6rem', color: 'var(--accent-magenta)' }}>(返工)</span>}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
          <div style={{ marginBottom: '1rem' }}><h4 style={{ fontSize: '0.9rem', marginBottom: '0.5rem' }}>当前重点</h4><textarea value={activeSelectedTask.current_focus || ''} onChange={(event) => setSelectedTask({ ...activeSelectedTask, current_focus: event.target.value })} onBlur={(event) => updateSelectedTask({ current_focus: event.target.value })} rows={2} style={{ ...inputStyle, width: '100%' }} /></div>
          <div style={{ marginBottom: '1rem' }}><h4 style={{ fontSize: '0.9rem', marginBottom: '0.5rem' }}>阻塞原因</h4><textarea value={activeSelectedTask.block_reason || ''} onChange={(event) => setSelectedTask({ ...activeSelectedTask, block_reason: event.target.value })} onBlur={(event) => updateSelectedTask({ block_reason: event.target.value })} rows={3} style={{ ...inputStyle, width: '100%' }} /></div>
          <div style={{ marginBottom: '1rem' }}><h4 style={{ fontSize: '0.9rem', marginBottom: '0.5rem' }}>下一步</h4><textarea value={activeSelectedTask.next_action || ''} onChange={(event) => setSelectedTask({ ...activeSelectedTask, next_action: event.target.value })} onBlur={(event) => updateSelectedTask({ next_action: event.target.value })} rows={3} style={{ ...inputStyle, width: '100%' }} /></div>
          <div><h4 style={{ fontSize: '0.9rem', marginBottom: '0.5rem' }}>结果摘要</h4><textarea value={activeSelectedTask.result_summary || ''} onChange={(event) => setSelectedTask({ ...activeSelectedTask, result_summary: event.target.value })} onBlur={(event) => updateSelectedTask({ result_summary: event.target.value })} rows={3} style={{ ...inputStyle, width: '100%' }} /></div>
        </div>
      )}
    </div>
  );
};

export default SchedulerBoard;
