import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Layout from './components/Layout';
import { ProjectProvider } from './context/ProjectContext';
import OrchestratorHome from './pages/OrchestratorHome';
import SchedulerBoard from './pages/SchedulerBoard';
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
              <Route path="*" element={<OrchestratorHome />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ProjectProvider>
    </QueryClientProvider>
  );
}

export default App;
