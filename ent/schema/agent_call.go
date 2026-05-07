package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentCall records each agent execution for observability and debugging.
type AgentCall struct {
	ent.Schema
}

// Fields of the AgentCall.
func (AgentCall) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			Unique().
			Immutable(),
		field.Text("task_id"),
		field.Text("agent_id"),
		field.Text("runner_type"),
		field.Text("task_type"),
		field.Text("trace_id").
			Optional(),
		field.Text("status"), // "success", "failure", "timeout"
		field.Int("exit_code").
			Default(0),
		field.Text("error_message").
			Optional(),
		field.Text("output_summary").
			Optional(),
		field.Int64("duration_ms").
			Default(0),
		field.Time("started_at"),
		field.Time("finished_at"),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges of the AgentCall.
func (AgentCall) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the AgentCall.
func (AgentCall) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id"),
		index.Fields("agent_id"),
		index.Fields("trace_id"),
		index.Fields("status"),
		index.Fields("started_at"),
	}
}
