package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// ContextEntry holds the schema definition for the ContextEntry entity.
type ContextEntry struct {
	ent.Schema
}

// Fields of the ContextEntry.
func (ContextEntry) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			Unique().
			Immutable(),
		field.Text("space_id"),
		field.Text("key"),
		field.Text("value"),
		field.Text("updated_by").
			Optional(),
		field.Time("updated_at").
			Default(time.Now),
	}
}

// Edges of the ContextEntry.
func (ContextEntry) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the ContextEntry.
func (ContextEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("space_id"),
		index.Fields("space_id", "key").Unique(),
	}
}
