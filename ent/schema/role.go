package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// Role holds the schema definition for the Role entity.
type Role struct {
	ent.Schema
}

// Fields of the Role.
func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			Unique().
			Immutable(),
		field.Text("org_id"),
		field.Text("team_id"),
		field.Text("name"),
		field.Text("capabilities").
			Default("[]"),
		field.Text("permissions").
			Default("[]"),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges of the Role.
func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("users", User.Type).
			Ref("roles"),
	}
}

// Indexes of the Role.
func (Role) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("org_id"),
		index.Fields("team_id"),
	}
}
