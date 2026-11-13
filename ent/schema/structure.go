package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Structure holds the schema definition for the Structure entity (physical structure built on a property).
type Structure struct {
	ent.Schema
}

func (Structure) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (Structure) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("property_id", uuid.UUID{}),
		field.String("name").
			NotEmpty(),
		field.String("structure_type").
			NotEmpty(),
		field.Int("floors").
			Default(1),
		field.Text("description").
			Optional(),
	}
}

func (Structure) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("property", Property.Type).
			Ref("structures").
			Field("property_id").
			Unique().
			Required(),
		edge.To("units", Unit.Type),
		edge.To("fee_rules", PropertyFeeRule.Type),
	}
}
