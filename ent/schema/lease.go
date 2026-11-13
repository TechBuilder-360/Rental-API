package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Lease holds the schema definition for the Lease entity (unifying unit occupancy and lease duration).
type Lease struct {
	ent.Schema
}

func (Lease) Mixin() []ent.Mixin {
	return []ent.Mixin{
		UUIDMixin{},
		TimeMixin{},
	}
}

func (Lease) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("unit_id", uuid.UUID{}),
		field.UUID("property_id", uuid.UUID{}),
		field.UUID("user_id", uuid.UUID{}),
		field.UUID("agent_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.Time("entry_date").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Time("start_date").
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Time("end_date").
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Time("termination_date").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Bool("active").
			Default(true),
	}
}

func (Lease) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("unit", Unit.Type).
			Ref("leases").
			Field("unit_id").
			Unique().
			Required(),
		edge.From("property", Property.Type).
			Ref("leases").
			Field("property_id").
			Unique().
			Required(),
		edge.From("tenant", User.Type).
			Ref("leases").
			Field("user_id").
			Unique().
			Required(),
		edge.From("agent", Agent.Type).
			Ref("leases").
			Field("agent_id").
			Unique(),
		edge.To("commission_records", CommissionRecord.Type),
	}
}
