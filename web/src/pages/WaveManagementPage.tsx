import React from 'react';
import { KanbanSquare, Layers3 } from 'lucide-react';

const cardStyle: React.CSSProperties = {
  display: 'grid',
  gap: '0.75rem',
  padding: '1rem',
  background: 'rgba(255,255,255,0.03)',
  border: '1px solid rgba(255,255,255,0.06)',
  borderRadius: 12,
};

const WaveManagementPage: React.FC = () => {
  return (
    <div style={{ display: 'grid', gap: '1.5rem' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', gap: '1rem' }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--accent-cyan)', marginBottom: '0.5rem' }}>
            <Layers3 size={16} />
            <span style={{ fontSize: '0.75rem', fontWeight: 700, letterSpacing: '2px' }}>WAVE CONTROL SURFACE</span>
          </div>
          <h2 className="neon-text" style={{ fontSize: '2.2rem' }}>Wave 管理</h2>
          <p style={{ color: 'var(--text-secondary)' }}>正式 Wave 控制入口已保留，下一步接入项目化 Wave 查询与 seal 流程。</p>
        </div>
      </div>

      <div className="glass-card" style={{ display: 'grid', gap: '1rem' }}>
        <div style={cardStyle}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
            <KanbanSquare size={18} color="var(--accent-yellow)" />
            <strong>当前状态</strong>
          </div>
          <p style={{ color: 'var(--text-secondary)', lineHeight: 1.7 }}>
            该页面已成为正式一级入口，避免继续把 Wave 能力埋在旧 dispatch 路由或临时入口里。
          </p>
          <p style={{ color: 'var(--text-secondary)', lineHeight: 1.7 }}>
            后续实现会补上 Wave 列表、open / sealed 状态、任务计数、seal 操作，以及跳转到 board / task detail 的黄金路径。
          </p>
        </div>
      </div>
    </div>
  );
};

export default WaveManagementPage;
