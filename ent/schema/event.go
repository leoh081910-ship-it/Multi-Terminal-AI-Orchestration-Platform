package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// Event holds the schema definition for the Event entity.
type Event struct {
	ent.Schema
}

// Fields of the Event.
func (Event) Fields() []ent.Field {
	return []ent.Field{
		field.Text("event_id").
			Unique(),
		field.Text("project_id").
			Default("default"),
		field.Text("task_id"),
		field.Text("event_type"),
		field.Text("from_state").
			Optional(),
		field.Text("to_state").
			Optional(),
		field.Time("timestamp").
			Default(time.Now),
		field.Text("reason").
			Optional(),
		field.Int("attempt").
			Default(0),
		field.Text("transport").
			Optional(),
		field.Text("runner_id").
			Optional(),
		field.Text("details").
			Optional(),
	}
}

// Edges of the Event.
func (Event) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the Event.
func (Event) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("task_id"),
		index.Fields("timestamp"),
		index.Fields("project_id", "timestamp"),
	}
}
