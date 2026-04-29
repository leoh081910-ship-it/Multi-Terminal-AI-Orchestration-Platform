import client from './client';
import type { CreateProjectInput, ProjectSummary } from '../types/project';

export const projectsApi = {
  list: async (): Promise<ProjectSummary[]> => {
    const response = await client.get<ProjectSummary[]>('/projects');
    return response.data;
  },
  create: async (payload: CreateProjectInput): Promise<ProjectSummary> => {
    const response = await client.post<ProjectSummary>('/projects', payload);
    return response.data;
  },
};
