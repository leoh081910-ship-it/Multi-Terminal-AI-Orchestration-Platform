package server

func (s *Server) getProjectRegistry() *compatProjectRegistry {
	s.projectsMu.RLock()
	defer s.projectsMu.RUnlock()
	return s.projects
}

func (s *Server) setProjectRegistry(registry *compatProjectRegistry) {
	s.projectsMu.Lock()
	defer s.projectsMu.Unlock()
	s.projects = registry
}

func (s *Server) SetProjectConfigStore(store *ProjectConfigStore) {
	s.projectConfigStore = store
}

func (s *Server) SetProjectQueueManager(manager *ProjectQueueManager) {
	s.projectQueueManager = manager
}
