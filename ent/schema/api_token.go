package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// APIToken holds the schema definition for the APIToken entity.
type APIToken struct {
	ent.Schema
}

// Fields of the APIToken.
func (APIToken) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable().
			Comment("Token ID (UUID)"),

		field.String("token").
			Unique().
			Sensitive().
			Comment("Token value (hashed)"),

		field.String("name").
			Comment("Token name for identification"),

		field.String("description").
			Optional().
			Comment("Token description"),

		field.String("user_id").
			Comment("User ID who owns this token"),

		field.JSON("scopes", []string{}).
			Optional().
			Comment("Token scopes/permissions"),

		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Token creation time"),

		field.Time("expires_at").
			Optional().
			Comment("Token expiration time"),

		field.Time("last_used_at").
			Optional().
			Comment("Last time token was used"),

		field.Bool("revoked").
			Default(false).
			Comment("Whether token is revoked"),

		field.Time("revoked_at").
			Optional().
			Comment("Token revocation time"),

		field.String("revoked_reason").
			Optional().
			Comment("Reason for revocation"),
	}
}

// Indexes of the APIToken.
func (APIToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token"),
		index.Fields("user_id"),
		index.Fields("revoked"),
		index.Fields("expires_at"),
	}
}
