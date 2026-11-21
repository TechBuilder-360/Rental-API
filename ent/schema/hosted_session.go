package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// HostedSession holds the schema definition for the HostedSession entity.
// It stores the hosted session URL, status, and provider lifecycle data.
type HostedSession struct {
	ent.Schema
}

func (HostedSession) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (HostedSession) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.UUID{}).
			Comment("User undergoing identity verification"),
		field.UUID("template_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("Hosted session template used for this session"),

		// Provider & Session URL details
		field.String("provider").
			NotEmpty().
			Comment("Provider identifier, e.g. didit"),
		field.String("session_id").
			NotEmpty().
			Unique().
			Comment("External session identifier from the provider"),
		field.String("session_token").
			Optional().
			Comment("Session token for SDK/hosted session access"),
		field.String("verification_url").
			NotEmpty().
			Comment("Hosted verification URL for user redirection"),
		field.String("vendor_data").
			Optional().
			Comment("Custom vendor tracking reference passed to provider"),

		// Lifecycle State & Outcome
		field.String("status").
			Default("Created").
			Comment("Verification status: Created, InProgress, Approved, Declined, Review, Expired, Abandoned"),
		field.String("decision").
			Optional().
			Nillable().
			Comment("Outcome decision: Approved, Declined, Inconclusive"),
		field.String("reason").
			Optional().
			Nillable().
			Comment("Decline or review explanation"),
		field.Float("risk_score").
			Optional().
			Nillable().
			Comment("Assessed risk score from provider"),

		// Session Timestamps
		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("Expiration timestamp of the hosted session"),
		field.Time("verified_at").
			Optional().
			Nillable().
			Comment("Timestamp when session was successfully approved"),
		field.Time("declined_at").
			Optional().
			Nillable().
			Comment("Timestamp when session was declined"),
	}
}

func (HostedSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("hosted_sessions").
			Field("user_id").
			Unique().
			Required(),
		edge.From("template", HostedSessionTemplate.Type).
			Ref("hosted_sessions").
			Field("template_id").
			Unique(),
		edge.To("identities", UserIdentity.Type),
	}
}

func (HostedSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id").Unique(),
		index.Fields("user_id", "status"),
		index.Fields("provider", "status"),
	}
}
