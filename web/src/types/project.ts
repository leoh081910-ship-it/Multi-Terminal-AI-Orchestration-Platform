export interface ProjectSummary {
  project_id: string;
  name: string;
  repo_root: string;
  default: boolean;
}

export interface CreateProjectInput {
  project_id?: string;
  name: string;
  repo_root: string;
}
