package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Service holds the schema definition for third-party or internal integration services.
type Service struct {
	ent.Schema
}

func (Service) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "services"},
	}
}

func (Service) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (Service) Fields() []ent.Field {
	return []ent.Field{
		field.String("identifier").
			Unique().
			NotEmpty(),
		field.String("name").
			NotEmpty(),
		field.String("provider").
			Optional().
			Default(""),
		field.Bool("disabled").
			Default(false),
	}
}

func (Service) Edges() []ent.Edge {
	return nil
}
