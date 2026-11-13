package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Unit holds the schema definition for the Unit entity (individual living or occupiable space within a structure).
type Unit struct {
	ent.Schema
}

func (Unit) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (Unit) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("structure_id", uuid.UUID{}),
		field.String("name").
			NotEmpty(),
		field.String("unit_type").
			NotEmpty(),
		field.Int("bedrooms").
			Default(0),
		field.Int("bathrooms").
			Default(0),
		field.Int("floor_number").
			Default(0),
		field.Float("size").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "numeric(12,2)",
			}),
		field.Text("description").
			Optional(),
		field.Enum("availability").
			Values("Vacant", "Occupied", "Not available").
			Default("Vacant"),
	}
}

func (Unit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("structure", Structure.Type).
			Ref("units").
			Field("structure_id").
			Unique().
			Required(),
		edge.To("pricing", UnitPricing.Type),
		edge.To("fee_rules", PropertyFeeRule.Type),
		edge.To("leases", Lease.Type),
	}
}
