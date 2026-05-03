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
		field.Text("org_id").
			Optional(),
		field.Text("team_id").
			Optional(),
		field.Text("name"),
		field.Text("description").
			Optional(),
		field.Text("capabilities").
			Default("[]"),
		field.Text("legacy_permissions").
			Default("[]").
			StorageKey("permissions"),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges of the Role.
func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("users", User.Type).
			Ref("roles"),
		edge.To("permissions", Permission.Type),
	}
}

// Indexes of the Role.
func (Role) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("org_id"),
		index.Fields("team_id"),
		index.Fields("name"),
	}
}
