package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type PriceOverride struct{ ent.Schema }

func (PriceOverride) Fields() []ent.Field {
	return []ent.Field{
		field.Int("enrollment_id"),
		field.Time("valid_from"),
		field.Time("valid_to").Nillable().Optional(),
		field.Float("lesson_price").Nillable().Optional(),
		field.Float("subscription_price").Nillable().Optional(),
	}
}

func (PriceOverride) Edges() []ent.Edge {
	return []ent.Edge{
		// Owner FK: each override belongs to one enrollment (M:1)
		edge.From("enrollment", Enrollment.Type).
			Ref("price_overrides").
			Field("enrollment_id").
			Required().
			Unique(),
	}
}
