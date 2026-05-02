package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// KnowledgeSpace holds the schema definition for the KnowledgeSpace entity.
type KnowledgeSpace struct {
	ent.Schema
}

// Fields of the KnowledgeSpace.
func (KnowledgeSpace) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			Unique().
			Immutable(),
		field.Text("org_id"),
		field.Text("project_id").
			Default("default"),
		field.Text("name"),
		field.Text("description").
			Optional(),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges of the KnowledgeSpace.
func (KnowledgeSpace) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the KnowledgeSpace.
func (KnowledgeSpace) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("org_id"),
		index.Fields("project_id"),
	}
}
