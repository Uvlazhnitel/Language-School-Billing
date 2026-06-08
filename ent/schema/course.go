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
		field.String("teacher_name").Default(""),
		field.Int("teacher_id").Optional().Nillable(),
		field.Enum("type").Values("group", "individual"),
		field.Float("legacy_lesson_price").StorageKey("lesson_price").Default(0),
		field.Float("legacy_subscription_price").StorageKey("subscription_price").Default(0),
		field.Int64("lesson_price_cents").Default(0),
		field.Int64("subscription_price_cents").Default(0),
		field.Bool("is_active").Default(true),
	}
}

func (Course) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("teacher", Teacher.Type).
			Ref("courses").
			Field("teacher_id").
			Unique(),
		edge.To("enrollments", Enrollment.Type),
		edge.To("month_stats", CourseMonthStat.Type),
	}
}
