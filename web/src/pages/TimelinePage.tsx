import { useQuery } from '@tanstack/react-query';
import { useProject } from '../hooks/useProject';
import { schedulerApi } from '../api/schedulerApi';
import GanttChart from '../components/GanttChart';
import { useWebSocket } from '../hooks/useWebSocket';

export default function TimelinePage() {
  const { projectId } = useProject();
  useWebSocket(projectId);

  const { data: tasks = [], isLoading } = useQuery({
    queryKey: ['tasks', projectId],
    queryFn: () => schedulerApi.getTasks(projectId),
    refetchInterval: 10000,
  });

  return (
    <div style={{ padding: 24 }}>
      <h2 style={{ color: '#00f2ea', fontSize: 20, fontWeight: 700, marginBottom: 16 }}>
        Timeline / Gantt View
      </h2>
      {isLoading ? (
        <div style={{ color: '#888', textAlign: 'center', padding: 60 }}>Loading tasks...</div>
      ) : (
        <GanttChart tasks={tasks} />
      )}
    </div>
  );
}
