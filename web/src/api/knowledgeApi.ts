import client from './client';

export interface KnowledgeSpace {
  id: string;
  org_id: string;
  project_id: string;
  name: string;
  description?: string;
  created_at: string;
}

export interface Document {
  id: string;
  space_id: string;
  title: string;
  content: string;
  type: string;
  author_agent_id?: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface ContextEntry {
  id: string;
  space_id: string;
  key: string;
  value: string;
  updated_by?: string;
  updated_at: string;
}

export interface Message {
  id: string;
  space_id: string;
  from_agent_id: string;
  to_agent_id?: string;
  type: string;
  content: string;
  read_at?: string;
  created_at: string;
}

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

function unwrap(resp: { data: ApiResponse<unknown> }) {
  if (!resp.data.success) throw new Error(resp.data.error || 'request failed');
  return resp.data.data;
}

export const knowledgeApi = {
  // Spaces
  listSpaces: async (orgId: string): Promise<KnowledgeSpace[]> =>
    client.get(`/orgs/${orgId}/spaces`).then(unwrap) as Promise<KnowledgeSpace[]>,

  createSpace: async (orgId: string, input: { name: string; description?: string; project_id?: string }): Promise<KnowledgeSpace> =>
    client.post(`/orgs/${orgId}/spaces`, input).then(unwrap) as Promise<KnowledgeSpace>,

  deleteSpace: async (orgId: string, spaceId: string): Promise<void> =>
    client.delete(`/orgs/${orgId}/spaces/${spaceId}`).then(unwrap) as Promise<void>,

  // Documents
  listDocuments: async (orgId: string, spaceId: string, type?: string): Promise<Document[]> => {
    const params = type ? `?type=${type}` : '';
    return client.get(`/orgs/${orgId}/spaces/${spaceId}/documents${params}`).then(unwrap) as Promise<Document[]>;
  },

  createDocument: async (orgId: string, spaceId: string, input: { title: string; content: string; type?: string; author_agent_id?: string }): Promise<Document> =>
    client.post(`/orgs/${orgId}/spaces/${spaceId}/documents`, input).then(unwrap) as Promise<Document>,

  updateDocument: async (orgId: string, spaceId: string, docId: string, input: { content?: string; title?: string }): Promise<Document> =>
    client.patch(`/orgs/${orgId}/spaces/${spaceId}/documents/${docId}`, input).then(unwrap) as Promise<Document>,

  // Context
  listContext: async (orgId: string, spaceId: string): Promise<ContextEntry[]> =>
    client.get(`/orgs/${orgId}/spaces/${spaceId}/context`).then(unwrap) as Promise<ContextEntry[]>,

  setContext: async (orgId: string, spaceId: string, input: { key: string; value: string; updated_by?: string }): Promise<ContextEntry> =>
    client.put(`/orgs/${orgId}/spaces/${spaceId}/context`, input).then(unwrap) as Promise<ContextEntry>,

  // Messages
  listMessages: async (orgId: string, spaceId: string, limit?: number): Promise<Message[]> => {
    const params = limit ? `?limit=${limit}` : '';
    return client.get(`/orgs/${orgId}/spaces/${spaceId}/messages${params}`).then(unwrap) as Promise<Message[]>;
  },

  sendMessage: async (orgId: string, spaceId: string, input: { from_agent_id: string; to_agent_id?: string; type?: string; content: string }): Promise<Message> =>
    client.post(`/orgs/${orgId}/spaces/${spaceId}/messages`, input).then(unwrap) as Promise<Message>,

  // Agent inbox
  getAgentInbox: async (orgId: string, agentId: string, limit?: number): Promise<Message[]> => {
    const params = limit ? `?limit=${limit}` : '';
    return client.get(`/orgs/${orgId}/agents/${agentId}/inbox${params}`).then(unwrap) as Promise<Message[]>;
  },
};
