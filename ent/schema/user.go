package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable().
			Comment("User ID (UUID)"),

		field.String("username").
			Unique().
			Comment("Username for login"),

		field.String("email").
			Unique().
			Comment("User email"),

		field.String("password_hash").
			Sensitive().
			Comment("Hashed password"),

		field.String("full_name").
			Optional().
			Comment("User full name"),

		field.Bool("active").
			Default(true).
			Comment("Whether user is active"),

		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("User creation time"),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("User last update time"),

		field.Time("last_login_at").
			Optional().
			Comment("Last login time"),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("roles", Role.Type),
		edge.To("tokens", APIToken.Type),
	}
}

// Indexes of the User.
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("username"),
		index.Fields("email"),
		index.Fields("active"),
	}
}
