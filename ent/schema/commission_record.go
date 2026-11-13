package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// CommissionRecord holds the schema definition for the CommissionRecord entity.
type CommissionRecord struct {
	ent.Schema
}

func (CommissionRecord) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
	}
}

func (CommissionRecord) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("commission_rule_id", uuid.UUID{}),
		field.UUID("agent_id", uuid.UUID{}),
		field.UUID("payment_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.UUID("lease_id", uuid.UUID{}),
		field.Float("amount").
			SchemaType(map[string]string{
				dialect.Postgres: "numeric(12,2)",
			}),
		field.Enum("status").
			Values("PENDING", "PAID", "CANCELLED").
			Default("PENDING"),
		field.Time("earned_at").
			Default(time.Now),
		field.Time("paid_at").
			Optional().
			Nillable(),
	}
}

func (CommissionRecord) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("commission_rule", AgentCommissionRule.Type).
			Ref("commission_records").
			Field("commission_rule_id").
			Unique().
			Required(),
		edge.From("agent", Agent.Type).
			Ref("commission_records").
			Field("agent_id").
			Unique().
			Required(),
		edge.From("lease", Lease.Type).
			Ref("commission_records").
			Field("lease_id").
			Unique().
			Required(),
	}
}
