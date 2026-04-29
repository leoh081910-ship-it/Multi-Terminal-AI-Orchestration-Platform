import React, { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { NavLink, Outlet } from 'react-router-dom';
import { FolderKanban, Kanban, Plus, Terminal } from 'lucide-react';
import { projectsApi } from '../api/projectsApi';
import { useProject } from '../hooks/useProject';
import type { CreateProjectInput } from '../types/project';

const inputStyle: React.CSSProperties = {
  width: '100%',
  background: 'rgba(255,255,255,0.04)',
  border: '1px solid var(--border-color)',
  borderRadius: 8,
  color: 'white',
  padding: '0.7rem 0.8rem',
};

const createDefaultForm = (): CreateProjectInput => ({
  project_id: '',
  name: '',
  repo_root: '',
});

const Layout: React.FC = () => {
  const queryClient = useQueryClient();
  const { project, projectId, projects, setProjectId, isLoading } = useProject();
  const [createOpen, setCreateOpen] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [form, setForm] = useState<CreateProjectInput>(createDefaultForm());
  const hasMultipleProjects = projects.length > 1;
  const navItems = [
    { to: '/', icon: <Terminal size={20} />, label: '控制中心' },
    { to: '/board', icon: <Kanban size={20} />, label: '调度看板' },
  ];

  const createProjectMutation = useMutation({
    mutationFn: projectsApi.create,
    onSuccess: async (createdProject) => {
      setCreateOpen(false);
      setCreateError(null);
      setForm(createDefaultForm());
      await queryClient.invalidateQueries({ queryKey: ['projects'] });
      setProjectId(createdProject.project_id);
    },
    onError: (error) => {
      const message = error instanceof Error ? error.message : '新增项目失败';
      setCreateError(message);
    },
  });

  const handleCreateProject = () => {
    setCreateError(null);
    if (!form.name.trim() || !form.repo_root.trim()) {
      setCreateError('项目名称和仓库路径不能为空');
      return;
    }
    createProjectMutation.mutate({
      project_id: form.project_id?.trim() || undefined,
      name: form.name.trim(),
      repo_root: form.repo_root.trim(),
    });
  };

  return (
    <div className="app-container">
      <aside className="sidebar">
        <div className="logo" style={{ marginBottom: '3rem' }}>
          <h1 style={{ fontSize: '1.5rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
            <span style={{ color: 'var(--accent-cyan)' }}>AIOP</span>
            <span style={{ fontWeight: 300 }}>平台</span>
          </h1>
          <p style={{ fontSize: '0.7rem', color: 'var(--text-secondary)', marginTop: '0.2rem', letterSpacing: '0.08em' }}>
            多项目编排控制台
          </p>
        </div>

        <div style={{ marginBottom: '1.5rem' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.5rem', marginBottom: '0.5rem', color: 'var(--accent-cyan)' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <FolderKanban size={14} />
              <span style={{ fontSize: '0.72rem', fontWeight: 700, letterSpacing: '0.08em' }}>当前项目</span>
            </div>
            <button
              onClick={() => setCreateOpen(true)}
              style={{ display: 'flex', alignItems: 'center', gap: '0.25rem', border: 'none', background: 'transparent', color: 'var(--accent-cyan)', cursor: 'pointer', padding: 0, fontSize: '0.72rem', fontWeight: 700 }}
            >
              <Plus size={14} /> 新增
            </button>
          </div>
          <select
            value={projectId}
            onChange={(event) => setProjectId(event.target.value)}
            disabled={isLoading || !hasMultipleProjects}
            style={inputStyle}
          >
            {projects.map((item) => (
              <option key={item.project_id} value={item.project_id}>
                {item.name}
              </option>
            ))}
          </select>
          <div style={{ fontSize: '0.68rem', color: 'var(--text-secondary)', marginTop: '0.45rem', lineHeight: 1.5 }}>
            {isLoading
              ? '加载项目元数据中…'
              : !projects.length
                ? '当前没有可用项目'
                : !hasMultipleProjects
                  ? '当前只注册了 1 个项目。可以直接点右上角“新增”。'
                  : project?.repo_root}
          </div>
        </div>

        <nav style={{ flex: 1 }}>
          <ul style={{ listStyle: 'none' }}>
            {navItems.map((item) => (
              <li key={item.to} style={{ marginBottom: '0.5rem' }}>
                <NavLink
                  to={item.to}
                  style={({ isActive }) => ({
                    display: 'flex',
                    alignItems: 'center',
                    gap: '0.75rem',
                    padding: '0.75rem 1rem',
                    borderRadius: '8px',
                    textDecoration: 'none',
                    color: isActive ? 'var(--accent-cyan)' : 'var(--text-secondary)',
                    background: isActive ? 'rgba(0, 242, 234, 0.08)' : 'transparent',
                    transition: 'all 0.2s ease',
                    fontWeight: isActive ? 600 : 400,
                  })}
                >
                  {item.icon}
                  <span>{item.label}</span>
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
      </aside>

      <main className="main-content">
        <header style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: '2rem' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
            <div style={{ textAlign: 'right' }}>
              <p style={{ fontSize: '0.8rem', fontWeight: 600 }}>{project?.name || '项目'}</p>
              <p style={{ fontSize: '0.65rem', color: 'var(--text-secondary)' }}>{projectId || 'default'}</p>
            </div>
            <div
              style={{
                width: 40,
                height: 40,
                borderRadius: '50%',
                background: 'linear-gradient(135deg, var(--accent-cyan), var(--accent-magenta))',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontWeight: 700,
                fontSize: '0.9rem',
              }}
            >
              AI
            </div>
          </div>
        </header>

        <Outlet />
      </main>

      {createOpen && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.55)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1200 }}>
          <div className="glass-card" style={{ width: 520, maxWidth: 'calc(100vw - 2rem)', padding: '1.5rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1rem' }}>
              <h3 style={{ fontSize: '1.1rem' }}>新增项目</h3>
              <button onClick={() => setCreateOpen(false)} style={{ border: 'none', background: 'transparent', color: 'var(--text-secondary)', cursor: 'pointer' }}>关闭</button>
            </div>
            <div style={{ display: 'grid', gap: '0.75rem' }}>
              <div>
                <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)', marginBottom: '0.35rem' }}>项目名称</div>
                <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder="比如：抖音选品二期" style={inputStyle} />
              </div>
              <div>
                <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)', marginBottom: '0.35rem' }}>项目 ID，可选</div>
                <input value={form.project_id || ''} onChange={(event) => setForm({ ...form, project_id: event.target.value })} placeholder="不填会自动生成" style={inputStyle} />
              </div>
              <div>
                <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)', marginBottom: '0.35rem' }}>仓库路径</div>
                <input value={form.repo_root} onChange={(event) => setForm({ ...form, repo_root: event.target.value })} placeholder="E:\\Projects\\your-repo" style={inputStyle} />
              </div>
              <div style={{ fontSize: '0.74rem', color: 'var(--text-secondary)', lineHeight: 1.6 }}>
                会自动创建该项目自己的 <span className="mono">.orchestrator/worktrees</span>、<span className="mono">workspaces</span>、<span className="mono">artifacts</span>，并继承当前默认项目的运行时配置。
              </div>
              {createError && <div style={{ color: 'var(--accent-magenta)', fontSize: '0.8rem' }}>{createError}</div>}
            </div>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '0.75rem', marginTop: '1rem' }}>
              <button onClick={() => setCreateOpen(false)} style={{ borderRadius: 8, border: '1px solid var(--border-color)', background: 'transparent', color: 'white', padding: '0.7rem 1rem', cursor: 'pointer' }}>取消</button>
              <button onClick={handleCreateProject} disabled={createProjectMutation.isPending} style={{ borderRadius: 8, border: 'none', background: 'var(--accent-cyan)', color: '#071014', padding: '0.7rem 1rem', cursor: 'pointer', fontWeight: 700 }}>
                {createProjectMutation.isPending ? '创建中…' : '创建项目'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Layout;
