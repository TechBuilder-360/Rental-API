package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Property holds the schema definition for the Property entity (represents the land itself).
type Property struct {
	ent.Schema
}

func (Property) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (Property) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty(),
		field.Text("address").
			NotEmpty(),
		field.String("street").
			Optional(),
		field.String("lga").
			Optional(),
		field.String("country").
			Optional(),
		field.String("geolocation").
			Optional(),
		field.Float("land_size").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "numeric(12,2)",
			}),
		field.Enum("land_size_unit").
			Values("sqm", "acres", "hectares").
			Optional().
			Nillable(),
		field.Text("description").
			Optional(),
	}
}

func (Property) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("structures", Structure.Type),
		edge.To("managers", PropertyManager.Type),
		edge.To("fee_rules", PropertyFeeRule.Type),
		edge.To("commission_rules", AgentCommissionRule.Type),
		edge.To("leases", Lease.Type),
	}
}
