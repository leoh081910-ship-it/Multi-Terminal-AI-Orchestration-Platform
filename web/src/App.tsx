import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Layout from './components/Layout';
import { ProjectProvider } from './context/ProjectContext';
import OrchestratorHome from './pages/OrchestratorHome';
import SchedulerBoard from './pages/SchedulerBoard';
import TimelinePage from './pages/TimelinePage';
import GoalPage from './pages/GoalPage';
import AgentWorkbenchPage from './pages/AgentWorkbenchPage';
import SwimLanePage from './pages/SwimLanePage';
import OrganizationPage from './pages/OrganizationPage';
import KnowledgeSpacePage from './pages/KnowledgeSpacePage';
import TaskDetailPage from './pages/TaskDetailPage';
import WaveManagementPage from './pages/WaveManagementPage';
import EventLogPage from './pages/EventLogPage';
import TriageDashboardPage from './pages/TriageDashboardPage';
import './styles/global.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ProjectProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route index element={<OrchestratorHome />} />
              <Route path="board" element={<SchedulerBoard />} />
              <Route path="waves" element={<WaveManagementPage />} />
              <Route path="events" element={<EventLogPage />} />
              <Route path="triage" element={<TriageDashboardPage />} />
              <Route path="timeline" element={<TimelinePage />} />
              <Route path="goals" element={<GoalPage />} />
              <Route path="agents" element={<AgentWorkbenchPage />} />
              <Route path="swimlane" element={<SwimLanePage />} />
              <Route path="org" element={<OrganizationPage />} />
              <Route path="knowledge" element={<KnowledgeSpacePage />} />
              <Route path="tasks/:taskId" element={<TaskDetailPage />} />
              <Route path="*" element={<OrchestratorHome />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ProjectProvider>
    </QueryClientProvider>
  );
}

export default App;
