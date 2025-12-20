package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Course struct{ ent.Schema }

func (Course) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.Enum("type").Values("group", "individual"),
		field.Float("lesson_price").Default(0),
		field.Float("subscription_price").Default(0),
		field.Bool("is_active").Default(true),
	}
}

func (Course) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("enrollments", Enrollment.Type),
	}
}
