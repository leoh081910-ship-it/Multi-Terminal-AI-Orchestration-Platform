package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// Agent holds the schema definition for the Agent entity.
type Agent struct {
	ent.Schema
}

// Fields of the Agent.
func (Agent) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			Unique().
			Immutable(),
		field.Text("org_id"),
		field.Text("name"),
		field.Text("type"),
		field.Text("role_id").
			Optional(),
		field.Text("status").
			Default("idle"),
		field.Text("specialties").
			Default("[]"),
		field.Text("config").
			Default("{}"),
		field.Time("last_heartbeat_at").
			Optional(),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges of the Agent.
func (Agent) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the Agent.
func (Agent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("org_id"),
		index.Fields("role_id"),
		index.Fields("status"),
		index.Fields("type"),
		index.Fields("name"),
	}
}
