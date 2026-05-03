package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Permission holds the schema definition for the Permission entity.
type Permission struct {
	ent.Schema
}

// Fields of the Permission.
func (Permission) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable().
			Comment("Permission ID (UUID)"),

		field.String("name").
			Unique().
			Comment("Permission name, e.g., 'tasks:read'"),

		field.String("resource").
			Comment("Resource type, e.g., 'tasks', 'projects', 'users'"),

		field.String("action").
			Comment("Action on resource, e.g., 'read', 'write', 'delete'"),

		field.String("description").
			Optional().
			Comment("Human-readable description"),

		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Creation time"),
	}
}

// Edges of the Permission.
func (Permission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("roles", Role.Type).
			Ref("permissions"),
	}
}

// Indexes of the Permission.
func (Permission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("resource", "action").Unique(),
		index.Fields("name"),
	}
}
