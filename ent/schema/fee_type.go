package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// FeeType holds the schema definition for the FeeType entity.
type FeeType struct {
	ent.Schema
}

func (FeeType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
	}
}

func (FeeType) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.String("code").
			MaxLen(50).
			Unique().
			NotEmpty(),
		field.String("category").
			MaxLen(50).
			NotEmpty(),
		field.Text("description").
			Optional(),
		field.Bool("is_active").
			Default(true),
	}
}

func (FeeType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("fee_rules", PropertyFeeRule.Type),
	}
}
