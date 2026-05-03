package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// Department holds the schema definition for the Department entity.
type Department struct {
	ent.Schema
}

// Fields of the Department.
func (Department) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			Unique().
			Immutable(),
		field.Text("org_id"),
		field.Text("name"),
		field.Text("description").
			Optional(),
		field.Text("lead_agent_id").
			Optional(),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges of the Department.
func (Department) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the Department.
func (Department) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("org_id"),
		index.Fields("lead_agent_id"),
	}
}
