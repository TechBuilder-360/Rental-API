package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Agent holds the schema definition for the Agent entity.
type Agent struct {
	ent.Schema
}

func (Agent) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (Agent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.UUID{}).
			Unique(),
		field.Bool("is_active").
			Default(true),
	}
}

func (Agent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("agent_profile").
			Field("user_id").
			Unique().
			Required(),
		edge.To("commission_rules", AgentCommissionRule.Type),
		edge.To("commission_records", CommissionRecord.Type),
		edge.To("leases", Lease.Type),
	}
}
