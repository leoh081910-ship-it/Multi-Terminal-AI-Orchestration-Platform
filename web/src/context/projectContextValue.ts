import { createContext } from 'react';
import type { ProjectSummary } from '../types/project';

export interface ProjectContextValue {
  projectId: string;
  project: ProjectSummary | null;
  projects: ProjectSummary[];
  isLoading: boolean;
  setProjectId: (projectId: string) => void;
}

export const ProjectContext = createContext<ProjectContextValue | null>(null);
