package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// Message holds the schema definition for the Message entity.
type Message struct {
	ent.Schema
}

// Fields of the Message.
func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			Unique().
			Immutable(),
		field.Text("space_id"),
		field.Text("from_agent_id"),
		field.Text("to_agent_id").
			Optional(),
		field.Text("type").
			Default("status_update"),
		field.Text("content"),
		field.Time("read_at").
			Optional(),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges of the Message.
func (Message) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the Message.
func (Message) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("space_id"),
		index.Fields("to_agent_id"),
		index.Fields("from_agent_id"),
	}
}
