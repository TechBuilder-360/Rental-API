package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PropertyManager holds the schema definition for the PropertyManager entity.
type PropertyManager struct {
	ent.Schema
}

func (PropertyManager) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (PropertyManager) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.UUID{}),
		field.UUID("property_id", uuid.UUID{}),
		field.Enum("role").
			Values("Owner", "Manager", "Agent"),
	}
}

func (PropertyManager) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("property_managers").
			Field("user_id").
			Unique().
			Required(),
		edge.From("property", Property.Type).
			Ref("managers").
			Field("property_id").
			Unique().
			Required(),
	}
}

func (PropertyManager) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "property_id", "role").Unique(),
	}
}
