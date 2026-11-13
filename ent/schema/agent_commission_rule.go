package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// AgentCommissionRule holds the schema definition for the AgentCommissionRule entity.
type AgentCommissionRule struct {
	ent.Schema
}

func (AgentCommissionRule) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
	}
}

func (AgentCommissionRule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("property_id", uuid.UUID{}),
		field.UUID("agent_id", uuid.UUID{}),
		field.Enum("calculation_method").
			Values("FIXED", "PERCENTAGE"),
		field.Float("amount").
			SchemaType(map[string]string{
				dialect.Postgres: "numeric(12,2)",
			}),
		field.Enum("commission_trigger").
			Values("PER_PAYMENT", "ONE_TIME"),
		field.Enum("percentage_basis").
			Values("PAYMENT_AMOUNT", "TOTAL_RENT").
			Optional().
			Nillable(),
		field.String("currency").
			MaxLen(3).
			Default("NGN"),
		field.Time("effective_from").
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Time("effective_to").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Bool("is_active").
			Default(true),
		field.Time("created_at").
			Immutable().
			Default(time.Now),
	}
}

func (AgentCommissionRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("property", Property.Type).
			Ref("commission_rules").
			Field("property_id").
			Unique().
			Required(),
		edge.From("agent", Agent.Type).
			Ref("commission_rules").
			Field("agent_id").
			Unique().
			Required(),
		edge.To("commission_records", CommissionRecord.Type),
	}
}
