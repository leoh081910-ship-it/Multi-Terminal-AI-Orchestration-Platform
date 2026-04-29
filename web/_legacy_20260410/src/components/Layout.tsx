import type { ReactNode } from 'react'

// Layout component implementation
export function Layout({ children, activeTab, onTabChange }: {
  children: ReactNode
  activeTab: 'tasks'
  onTabChange: (tab: 'tasks') => void
}) {
  return (
    <div className="app-container">
      <header className="app-header">
        <div className="app-header-content">
          <div className="app-logo">
            <svg className="w-8 h-8 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"
              />
            </svg>
            <span className="text-xl font-bold">多终端 AI 编排平台</span>
          </div>
          <nav className="app-nav">
            <div className="tab-list">
              <button
                onClick={() => onTabChange('tasks')}
                className="tab-trigger"
                data-state={activeTab === 'tasks' ? 'active' : 'inactive'}
              >
                任务列表
              </button>
            </div>
          </nav>
        </div>
      </header>
      <main className="app-main">{children}</main>
    </div>
  )
}
