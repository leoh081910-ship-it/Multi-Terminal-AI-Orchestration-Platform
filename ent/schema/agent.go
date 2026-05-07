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
		// runner_type replaces hardcoded agent type parsing
		field.Text("runner_type").
			Default("cli"),
		// runner_config stores JSON config specific to each runner type
		// e.g., for CLI: {"base_path": "...", "main_repo": "..."}
		// for HTTP: {"endpoint": "...", "api_key_ref": "..."}
		field.Text("runner_config").
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
		// index on runner_type enables fast lookup by runtime type
		index.Fields("runner_type"),
		// Unique constraint: no two agents with the same name in the same org
		index.Fields("org_id", "name").Unique(),
	}
}
