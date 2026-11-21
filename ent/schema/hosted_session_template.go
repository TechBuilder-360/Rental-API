package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// HostedSessionTemplate holds the schema definition for the HostedSessionTemplate entity.
type HostedSessionTemplate struct {
	ent.Schema
}

func (HostedSessionTemplate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (HostedSessionTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			Comment("Human-readable template name (e.g. Standard Tenant Verification, Enhanced KYC)"),
		field.String("provider").
			NotEmpty().
			Comment("Provider identifier (e.g. didit)"),
		field.String("template_id").
			NotEmpty().
			Comment("External provider workflow or template identifier (e.g. Didit workflow ID)"),
		field.String("description").
			Optional().
			Comment("Description of what checks this template performs"),
		field.Bool("is_default").
			Default(false).
			Comment("Whether this is the default template for the provider"),
		field.Bool("active").
			Default(true).
			Comment("Whether this template is currently active"),
		field.JSON("metadata", map[string]any{}).
			Optional().
			Comment("Additional configuration parameters for this template"),
	}
}

func (HostedSessionTemplate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("hosted_sessions", HostedSession.Type),
	}
}

func (HostedSessionTemplate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("provider", "template_id").Unique(),
		index.Fields("provider", "is_default"),
	}
}
