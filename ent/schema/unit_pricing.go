package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// UnitPricing holds the schema definition for the UnitPricing entity.
type UnitPricing struct {
	ent.Schema
}

func (UnitPricing) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "unit_pricing"},
	}
}

func (UnitPricing) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (UnitPricing) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("unit_id", uuid.UUID{}),
		field.Float("amount").
			SchemaType(map[string]string{
				dialect.Postgres: "numeric(12,2)",
			}),
		field.String("currency").
			MaxLen(3).
			Default("NGN"),
		field.Enum("billing_frequency").
			Values("DAILY", "WEEKLY", "MONTHLY", "YEARLY"),
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
	}
}

func (UnitPricing) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("unit", Unit.Type).
			Ref("pricing").
			Field("unit_id").
			Unique().
			Required(),
	}
}
