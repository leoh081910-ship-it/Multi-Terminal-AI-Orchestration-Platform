// Package knowledge provides the shared knowledge space service.
package knowledge

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/contextentry"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/document"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/knowledgespace"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/message"
)

// Service provides knowledge space CRUD operations.
type Service struct {
	client *ent.Client
}

// NewService creates a new knowledge service.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// --- Knowledge Space ---

type CreateSpaceInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
}

type SpaceView struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Service) CreateSpace(ctx context.Context, orgID string, input CreateSpaceInput) (*SpaceView, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	id := uuid.New().String()
	pid := input.ProjectID
	if pid == "" {
		pid = "default"
	}
	space, err := s.client.KnowledgeSpace.Create().
		SetID(id).
		SetOrgID(orgID).
		SetProjectID(pid).
		SetName(input.Name).
		SetNillableDescription(nstr(input.Description)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create space: %w", err)
	}
	return spaceToView(space), nil
}

func (s *Service) ListSpaces(ctx context.Context, orgID string) ([]*SpaceView, error) {
	spaces, err := s.client.KnowledgeSpace.Query().
		Where(knowledgespace.OrgID(orgID)).
		Order(ent.Asc(knowledgespace.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list spaces: %w", err)
	}
	result := make([]*SpaceView, len(spaces))
	for i, sp := range spaces {
		result[i] = spaceToView(sp)
	}
	return result, nil
}

func (s *Service) GetSpace(ctx context.Context, id string) (*SpaceView, error) {
	space, err := s.client.KnowledgeSpace.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get space: %w", err)
	}
	return spaceToView(space), nil
}

func (s *Service) DeleteSpace(ctx context.Context, id string) error {
	return s.client.KnowledgeSpace.DeleteOneID(id).Exec(ctx)
}

// --- Document ---

type CreateDocInput struct {
	Title         string `json:"title"`
	Content       string `json:"content"`
	Type          string `json:"type,omitempty"`
	AuthorAgentID string `json:"author_agent_id,omitempty"`
}

type UpdateDocInput struct {
	Content *string `json:"content,omitempty"`
	Title   *string `json:"title,omitempty"`
}

type DocView struct {
	ID            string    `json:"id"`
	SpaceID       string    `json:"space_id"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	Type          string    `json:"type"`
	AuthorAgentID string    `json:"author_agent_id,omitempty"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (s *Service) CreateDocument(ctx context.Context, spaceID string, input CreateDocInput) (*DocView, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	docType := input.Type
	if docType == "" {
		docType = "note"
	}
	id := uuid.New().String()
	d, err := s.client.Document.Create().
		SetID(id).
		SetSpaceID(spaceID).
		SetTitle(input.Title).
		SetContent(input.Content).
		SetType(docType).
		SetNillableAuthorAgentID(nstr(input.AuthorAgentID)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}
	return docToView(d), nil
}

func (s *Service) ListDocuments(ctx context.Context, spaceID string, docType string) ([]*DocView, error) {
	q := s.client.Document.Query().Where(document.SpaceID(spaceID))
	if docType != "" {
		q = q.Where(document.Type(docType))
	}
	docs, err := q.Order(ent.Asc(document.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	result := make([]*DocView, len(docs))
	for i, d := range docs {
		result[i] = docToView(d)
	}
	return result, nil
}

func (s *Service) GetDocument(ctx context.Context, id string) (*DocView, error) {
	d, err := s.client.Document.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return docToView(d), nil
}

func (s *Service) UpdateDocument(ctx context.Context, id string, input UpdateDocInput) (*DocView, error) {
	u := s.client.Document.UpdateOneID(id)
	if input.Content != nil {
		u.SetContent(*input.Content)
		u.AddVersion(1)
	}
	if input.Title != nil {
		u.SetTitle(*input.Title)
	}
	u.SetUpdatedAt(time.Now().UTC())
	d, err := u.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update document: %w", err)
	}
	return docToView(d), nil
}

func (s *Service) DeleteDocument(ctx context.Context, id string) error {
	return s.client.Document.DeleteOneID(id).Exec(ctx)
}

// --- Context Entry ---

type SetContextInput struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedBy string `json:"updated_by,omitempty"`
}

type ContextView struct {
	ID        string    `json:"id"`
	SpaceID   string    `json:"space_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedBy string    `json:"updated_by,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Service) SetContext(ctx context.Context, spaceID string, input SetContextInput) (*ContextView, error) {
	if input.Key == "" {
		return nil, fmt.Errorf("key is required")
	}
	// Upsert: try to find existing entry
	existing, _ := s.client.ContextEntry.Query().
		Where(contextentry.SpaceID(spaceID), contextentry.Key(input.Key)).
		Only(ctx)

	if existing != nil {
		u := s.client.ContextEntry.UpdateOneID(existing.ID).
			SetValue(input.Value).
			SetUpdatedAt(time.Now().UTC())
		if input.UpdatedBy != "" {
			u.SetUpdatedBy(input.UpdatedBy)
		}
		updated, err := u.Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("update context: %w", err)
		}
		return contextToView(updated), nil
	}

	id := uuid.New().String()
	ce, err := s.client.ContextEntry.Create().
		SetID(id).
		SetSpaceID(spaceID).
		SetKey(input.Key).
		SetValue(input.Value).
		SetNillableUpdatedBy(nstr(input.UpdatedBy)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create context: %w", err)
	}
	return contextToView(ce), nil
}

func (s *Service) GetContext(ctx context.Context, spaceID string, key string) (*ContextView, error) {
	ce, err := s.client.ContextEntry.Query().
		Where(contextentry.SpaceID(spaceID), contextentry.Key(key)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("get context: %w", err)
	}
	return contextToView(ce), nil
}

func (s *Service) ListContext(ctx context.Context, spaceID string) ([]*ContextView, error) {
	entries, err := s.client.ContextEntry.Query().
		Where(contextentry.SpaceID(spaceID)).
		Order(ent.Asc(contextentry.FieldKey)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list context: %w", err)
	}
	result := make([]*ContextView, len(entries))
	for i, ce := range entries {
		result[i] = contextToView(ce)
	}
	return result, nil
}

func (s *Service) DeleteContext(ctx context.Context, spaceID, key string) error {
	_, err := s.client.ContextEntry.Delete().
		Where(contextentry.SpaceID(spaceID), contextentry.Key(key)).
		Exec(ctx)
	return err
}

// --- Message ---

type SendMessageInput struct {
	FromAgentID string `json:"from_agent_id"`
	ToAgentID   string `json:"to_agent_id,omitempty"` // empty = broadcast
	Type        string `json:"type,omitempty"`
	Content     string `json:"content"`
}

type MessageView struct {
	ID          string    `json:"id"`
	SpaceID     string    `json:"space_id"`
	FromAgentID string    `json:"from_agent_id"`
	ToAgentID   string    `json:"to_agent_id,omitempty"`
	Type        string    `json:"type"`
	Content     string    `json:"content"`
	ReadAt      time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Service) SendMessage(ctx context.Context, spaceID string, input SendMessageInput) (*MessageView, error) {
	if input.FromAgentID == "" {
		return nil, fmt.Errorf("from_agent_id is required")
	}
	if input.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	msgType := input.Type
	if msgType == "" {
		msgType = "status_update"
	}
	id := uuid.New().String()
	m, err := s.client.Message.Create().
		SetID(id).
		SetSpaceID(spaceID).
		SetFromAgentID(input.FromAgentID).
		SetNillableToAgentID(nstr(input.ToAgentID)).
		SetType(msgType).
		SetContent(input.Content).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}
	return messageToView(m), nil
}

func (s *Service) ListMessages(ctx context.Context, spaceID string, limit int) ([]*MessageView, error) {
	if limit <= 0 {
		limit = 50
	}
	msgs, err := s.client.Message.Query().
		Where(message.SpaceID(spaceID)).
		Order(ent.Desc(message.FieldCreatedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	result := make([]*MessageView, len(msgs))
	for i, m := range msgs {
		result[i] = messageToView(m)
	}
	return result, nil
}

func (s *Service) GetAgentInbox(ctx context.Context, agentID string, limit int) ([]*MessageView, error) {
	if limit <= 0 {
		limit = 50
	}
	msgs, err := s.client.Message.Query().
		Where(message.ToAgentID(agentID)).
		Order(ent.Desc(message.FieldCreatedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("get inbox: %w", err)
	}
	result := make([]*MessageView, len(msgs))
	for i, m := range msgs {
		result[i] = messageToView(m)
	}
	return result, nil
}

func (s *Service) MarkRead(ctx context.Context, messageID string) error {
	return s.client.Message.UpdateOneID(messageID).
		SetReadAt(time.Now().UTC()).
		Exec(ctx)
}

// --- helpers ---

func nstr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func spaceToView(sp *ent.KnowledgeSpace) *SpaceView {
	return &SpaceView{
		ID:          sp.ID,
		OrgID:       sp.OrgID,
		ProjectID:   sp.ProjectID,
		Name:        sp.Name,
		Description: sp.Description,
		CreatedAt:   sp.CreatedAt,
	}
}

func docToView(d *ent.Document) *DocView {
	return &DocView{
		ID:            d.ID,
		SpaceID:       d.SpaceID,
		Title:         d.Title,
		Content:       d.Content,
		Type:          d.Type,
		AuthorAgentID: d.AuthorAgentID,
		Version:       d.Version,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

func contextToView(ce *ent.ContextEntry) *ContextView {
	return &ContextView{
		ID:        ce.ID,
		SpaceID:   ce.SpaceID,
		Key:       ce.Key,
		Value:     ce.Value,
		UpdatedBy: ce.UpdatedBy,
		UpdatedAt: ce.UpdatedAt,
	}
}

func messageToView(m *ent.Message) *MessageView {
	mv := &MessageView{
		ID:          m.ID,
		SpaceID:     m.SpaceID,
		FromAgentID: m.FromAgentID,
		ToAgentID:   m.ToAgentID,
		Type:        m.Type,
		Content:     m.Content,
		CreatedAt:   m.CreatedAt,
	}
	if !m.ReadAt.IsZero() {
		mv.ReadAt = m.ReadAt
	}
	return mv
}
