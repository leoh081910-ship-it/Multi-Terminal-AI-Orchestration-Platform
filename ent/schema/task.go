package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// Task holds the schema definition for the Task entity.
type Task struct {
	ent.Schema
}

// Fields of the Task.
func (Task) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			Unique().
			Immutable(),
		field.Text("project_id").
			Default("default"),
		field.Text("dispatch_ref"),
		field.Text("state").
			Default("pending"),
		field.Int("retry_count").
			Default(0),
		field.Int("loop_iteration_count").
			Default(0),
		field.Text("transport"),
		field.Int("wave"),
		field.Int("topo_rank").
			Default(0),
		field.Text("workspace_path").
			Optional(),
		field.Text("artifact_path").
			Optional(),
		field.Text("last_error_reason").
			Optional(),
		field.Time("created_at").
			Default(time.Now),
		field.Time("updated_at").
			Default(time.Now),
		field.Time("terminal_at").
			Optional(),
		field.Text("card_json"),
	}
}

// Edges of the Task.
func (Task) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the Task.
func (Task) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("dispatch_ref"),
		index.Fields("state"),
		index.Fields("wave"),
		index.Fields("project_id", "state"),
	}
}
