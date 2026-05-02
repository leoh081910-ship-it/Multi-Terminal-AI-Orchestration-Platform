package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// Document holds the schema definition for the Document entity.
type Document struct {
	ent.Schema
}

// Fields of the Document.
func (Document) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			Unique().
			Immutable(),
		field.Text("space_id"),
		field.Text("title"),
		field.Text("content"),
		field.Text("type").
			Default("note"),
		field.Text("author_agent_id").
			Optional(),
		field.Int("version").
			Default(1),
		field.Time("created_at").
			Default(time.Now),
		field.Time("updated_at").
			Default(time.Now),
	}
}

// Edges of the Document.
func (Document) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the Document.
func (Document) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("space_id"),
		index.Fields("author_agent_id"),
		index.Fields("type"),
	}
}
