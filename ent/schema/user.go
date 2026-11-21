package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("first_name").
			NotEmpty(),
		field.String("last_name").
			NotEmpty(),
		field.Time("dob").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.String("phone_number").
			Optional().
			Nillable().
			Unique(),
		// email is stored normalized (trimmed, lower-cased) by the auth service.
		field.String("email").
			Unique().
			NotEmpty(),
		field.Bool("email_verified").
			Default(false),
		field.Bool("identity_verified").
			Default(false),
		field.String("password").
			Sensitive().
			NotEmpty(),
		field.Bool("active").
			Default(true),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("property_managers", PropertyManager.Type),
		edge.To("agent_profile", Agent.Type).
			Unique(),
		edge.To("leases", Lease.Type),
		edge.To("hosted_sessions", HostedSession.Type),
		edge.To("identities", UserIdentity.Type),
	}
}
