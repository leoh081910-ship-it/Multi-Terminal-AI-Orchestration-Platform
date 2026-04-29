package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// Wave holds the schema definition for the Wave entity.
type Wave struct {
	ent.Schema
}

// Fields of the Wave.
func (Wave) Fields() []ent.Field {
	return []ent.Field{
		field.Text("project_id").
			Default("default"),
		field.Text("dispatch_ref"),
		field.Int("wave"),
		field.Time("sealed_at").
			Optional(),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges of the Wave.
func (Wave) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the Wave.
func (Wave) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("project_id", "dispatch_ref", "wave").Unique(),
	}
}
