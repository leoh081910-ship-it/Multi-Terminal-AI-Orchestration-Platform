import React from 'react';
import { Activity, Radio } from 'lucide-react';

const panelStyle: React.CSSProperties = {
  display: 'grid',
  gap: '0.75rem',
  padding: '1rem',
  background: 'rgba(255,255,255,0.03)',
  border: '1px solid rgba(255,255,255,0.06)',
  borderRadius: 12,
};

const EventLogPage: React.FC = () => {
  return (
    <div style={{ display: 'grid', gap: '1.5rem' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', gap: '1rem' }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--accent-cyan)', marginBottom: '0.5rem' }}>
            <Activity size={16} />
            <span style={{ fontSize: '0.75rem', fontWeight: 700, letterSpacing: '2px' }}>EVENT BROWSER</span>
          </div>
          <h2 className="neon-text" style={{ fontSize: '2.2rem' }}>事件日志</h2>
          <p style={{ color: 'var(--text-secondary)' }}>正式事件浏览入口已保留，下一步会补历史查询与实时追加的统一体验。</p>
        </div>
      </div>

      <div className="glass-card" style={{ display: 'grid', gap: '1rem' }}>
        <div style={panelStyle}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
            <Radio size={18} color="var(--accent-green)" />
            <strong>当前状态</strong>
          </div>
          <p style={{ color: 'var(--text-secondary)', lineHeight: 1.7 }}>
            该页面已经成为正式核心入口，避免把事件查看继续退回到单纯的 websocket 实时流面板。
          </p>
          <p style={{ color: 'var(--text-secondary)', lineHeight: 1.7 }}>
            后续实现会补 task / dispatch_ref 过滤、完整事件链历史读取、实时追加与任务详情跳转。
          </p>
        </div>
      </div>
    </div>
  );
};

export default EventLogPage;
