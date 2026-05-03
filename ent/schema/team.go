package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// Team holds the schema definition for the Team entity.
type Team struct {
	ent.Schema
}

// Fields of the Team.
func (Team) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			Unique().
			Immutable(),
		field.Text("org_id"),
		field.Text("dept_id"),
		field.Text("name"),
		field.Text("description").
			Optional(),
		field.Text("lead_agent_id").
			Optional(),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges of the Team.
func (Team) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the Team.
func (Team) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("org_id"),
		index.Fields("dept_id"),
		index.Fields("lead_agent_id"),
	}
}
