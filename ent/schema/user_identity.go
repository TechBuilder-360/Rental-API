package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// UserIdentity holds the schema definition for the UserIdentity entity.
type UserIdentity struct {
	ent.Schema
}

func (UserIdentity) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (UserIdentity) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.UUID{}).
			Comment("User this identity belongs to"),
		field.UUID("hosted_session_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("The hosted session that verified this identity"),
		field.String("provider").
			NotEmpty().
			Comment("Provider that verified the identity (e.g. didit)"),

		// Extracted Personal Details
		field.String("first_name").
			Optional().
			Nillable().
			Comment("Verified user first name"),
		field.String("last_name").
			Optional().
			Nillable().
			Comment("Verified user last name"),
		field.Time("dob").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}).
			Comment("Verified date of birth"),
		field.String("gender").
			Optional().
			Nillable().
			Comment("Verified gender"),
		field.String("nationality").
			Optional().
			Nillable().
			Comment("Nationality code or name"),

		// Extracted Document Details
		field.String("document_type").
			Optional().
			Nillable().
			Comment("Type of identity document (e.g. passport, id_card, driving_license)"),
		field.String("document_number").
			Optional().
			Nillable().
			Comment("Identity document number"),
		field.String("document_country").
			Optional().
			Nillable().
			Comment("Document issuing country (ISO 2/3 letter code)"),
		field.Time("document_expiration").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}).
			Comment("Document expiration date"),
		field.String("address").
			Optional().
			Nillable().
			Comment("Extracted or verified residential address"),

		// Verification status & raw provider payload
		field.Bool("is_verified").
			Default(false).
			Comment("Whether this identity is actively verified"),
		field.Time("verified_at").
			Optional().
			Nillable().
			Comment("Timestamp when this identity was officially verified"),
		field.JSON("raw_data", map[string]any{}).
			Optional().
			Comment("Raw identity payload returned by the provider"),
	}
}

func (UserIdentity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("identities").
			Field("user_id").
			Unique().
			Required(),
		edge.From("session", HostedSession.Type).
			Ref("identities").
			Field("hosted_session_id").
			Unique(),
	}
}

func (UserIdentity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("document_type", "document_number"),
	}
}
