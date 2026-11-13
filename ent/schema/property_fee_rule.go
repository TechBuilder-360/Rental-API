package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// PropertyFeeRule holds the schema definition for the PropertyFeeRule entity.
type PropertyFeeRule struct {
	ent.Schema
}

func (PropertyFeeRule) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
	}
}

func (PropertyFeeRule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("property_id", uuid.UUID{}),
		field.UUID("structure_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.UUID("unit_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.UUID("fee_type_id", uuid.UUID{}),
		field.Float("amount").
			SchemaType(map[string]string{
				dialect.Postgres: "numeric(12,2)",
			}),
		field.String("currency").
			MaxLen(3).
			Default("NGN"),
		field.Enum("calculation_method").
			Values("FIXED", "PERCENTAGE", "USAGE_BASED"),
		field.Enum("billing_frequency").
			Values("DAILY", "MONTHLY", "YEARLY", "ONE_TIME"),
		field.Enum("application_basis").
			Values("PER_UNIT", "PER_PROPERTY", "PER_USAGE"),
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

func (PropertyFeeRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("property", Property.Type).
			Ref("fee_rules").
			Field("property_id").
			Unique().
			Required(),
		edge.From("structure", Structure.Type).
			Ref("fee_rules").
			Field("structure_id").
			Unique(),
		edge.From("unit", Unit.Type).
			Ref("fee_rules").
			Field("unit_id").
			Unique(),
		edge.From("fee_type", FeeType.Type).
			Ref("fee_rules").
			Field("fee_type_id").
			Unique().
			Required(),
	}
}
