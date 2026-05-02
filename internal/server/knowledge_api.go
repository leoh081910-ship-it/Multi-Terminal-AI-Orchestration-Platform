package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/knowledge"
)

// registerKnowledgeRoutes registers knowledge space API routes.
func (s *Server) registerKnowledgeRoutes(r chi.Router) {
	r.Route("/orgs/{orgID}/spaces", func(r chi.Router) {
		r.Get("/", s.handleListSpaces)
		r.Post("/", s.handleCreateSpace)
		r.Route("/{spaceID}", func(r chi.Router) {
			r.Get("/", s.handleGetSpace)
			r.Delete("/", s.handleDeleteSpace)

			// Documents
			r.Route("/documents", func(r chi.Router) {
				r.Get("/", s.handleListDocuments)
				r.Post("/", s.handleCreateDocument)
				r.Route("/{docID}", func(r chi.Router) {
					r.Get("/", s.handleGetDocument)
					r.Patch("/", s.handleUpdateDocument)
					r.Delete("/", s.handleDeleteDocument)
				})
			})

			// Context (key-value store)
			r.Route("/context", func(r chi.Router) {
				r.Get("/", s.handleListContext)
				r.Put("/", s.handleSetContext)
				r.Delete("/{key}", s.handleDeleteContext)
			})

			// Messages
			r.Route("/messages", func(r chi.Router) {
				r.Get("/", s.handleListMessages)
				r.Post("/", s.handleSendMessage)
			})
		})
	})

	// Agent inbox
	r.Route("/orgs/{orgID}/agents/{agentID}/inbox", func(r chi.Router) {
		r.Get("/", s.handleGetAgentInbox)
	})
}

// --- Space handlers ---

func (s *Server) handleListSpaces(w http.ResponseWriter, r *http.Request) {
	spaces, err := s.knowledgeSvc.ListSpaces(r.Context(), chi.URLParam(r, "orgID"))
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: spaces})
}

func (s *Server) handleCreateSpace(w http.ResponseWriter, r *http.Request) {
	var input knowledge.CreateSpaceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	space, err := s.knowledgeSvc.CreateSpace(r.Context(), chi.URLParam(r, "orgID"), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: space})
}

func (s *Server) handleGetSpace(w http.ResponseWriter, r *http.Request) {
	space, err := s.knowledgeSvc.GetSpace(r.Context(), chi.URLParam(r, "spaceID"))
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: space})
}

func (s *Server) handleDeleteSpace(w http.ResponseWriter, r *http.Request) {
	if err := s.knowledgeSvc.DeleteSpace(r.Context(), chi.URLParam(r, "spaceID")); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

// --- Document handlers ---

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	docType := r.URL.Query().Get("type")
	docs, err := s.knowledgeSvc.ListDocuments(r.Context(), chi.URLParam(r, "spaceID"), docType)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: docs})
}

func (s *Server) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	var input knowledge.CreateDocInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	doc, err := s.knowledgeSvc.CreateDocument(r.Context(), chi.URLParam(r, "spaceID"), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: doc})
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	doc, err := s.knowledgeSvc.GetDocument(r.Context(), chi.URLParam(r, "docID"))
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: doc})
}

func (s *Server) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	var input knowledge.UpdateDocInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	doc, err := s.knowledgeSvc.UpdateDocument(r.Context(), chi.URLParam(r, "docID"), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: doc})
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	if err := s.knowledgeSvc.DeleteDocument(r.Context(), chi.URLParam(r, "docID")); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

// --- Context handlers ---

func (s *Server) handleListContext(w http.ResponseWriter, r *http.Request) {
	entries, err := s.knowledgeSvc.ListContext(r.Context(), chi.URLParam(r, "spaceID"))
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: entries})
}

func (s *Server) handleSetContext(w http.ResponseWriter, r *http.Request) {
	var input knowledge.SetContextInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	entry, err := s.knowledgeSvc.SetContext(r.Context(), chi.URLParam(r, "spaceID"), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: entry})
}

func (s *Server) handleDeleteContext(w http.ResponseWriter, r *http.Request) {
	if err := s.knowledgeSvc.DeleteContext(r.Context(), chi.URLParam(r, "spaceID"), chi.URLParam(r, "key")); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

// --- Message handlers ---

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	msgs, err := s.knowledgeSvc.ListMessages(r.Context(), chi.URLParam(r, "spaceID"), limit)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: msgs})
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var input knowledge.SendMessageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	msg, err := s.knowledgeSvc.SendMessage(r.Context(), chi.URLParam(r, "spaceID"), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: msg})
}

// --- Agent inbox ---

func (s *Server) handleGetAgentInbox(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	msgs, err := s.knowledgeSvc.GetAgentInbox(r.Context(), chi.URLParam(r, "agentID"), limit)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: msgs})
}
