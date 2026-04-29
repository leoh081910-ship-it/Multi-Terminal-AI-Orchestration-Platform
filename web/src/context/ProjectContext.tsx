import React, { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { projectsApi } from '../api/projectsApi';
import { ProjectContext, type ProjectContextValue } from './projectContextValue';

const storageKey = 'aiop:selected-project';

export const ProjectProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const initialProjectId = typeof window === 'undefined' ? '' : window.localStorage.getItem(storageKey) ?? '';
  const [preferredProjectId, setPreferredProjectId] = useState(initialProjectId);
  const { data: projects = [], isLoading } = useQuery({
    queryKey: ['projects'],
    queryFn: projectsApi.list,
    staleTime: 60_000,
  });

  const projectId = useMemo(() => {
    if (!projects.length) {
      return preferredProjectId;
    }
    const matched = projects.find((item) => item.project_id === preferredProjectId);
    if (matched) {
      return matched.project_id;
    }
    return (projects.find((item) => item.default) ?? projects[0]).project_id;
  }, [preferredProjectId, projects]);

  const setProjectId = (nextProjectId: string) => {
    setPreferredProjectId(nextProjectId);
    window.localStorage.setItem(storageKey, nextProjectId);
  };

  const value = useMemo<ProjectContextValue>(() => ({
    projectId,
    project: projects.find((item) => item.project_id === projectId) ?? null,
    projects,
    isLoading,
    setProjectId,
  }), [isLoading, projectId, projects]);

  return <ProjectContext.Provider value={value}>{children}</ProjectContext.Provider>;
};
