package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type CourseMonthStat struct{ ent.Schema }

func (CourseMonthStat) Fields() []ent.Field {
	return []ent.Field{
		field.Int("course_id"),
		field.Int("year"),
		field.Int("month"),
		field.Float("subscription_lessons_held").Default(0),
	}
}

func (CourseMonthStat) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("course", Course.Type).
			Ref("month_stats").
			Field("course_id").
			Required().
			Unique(),
	}
}

func (CourseMonthStat) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("course_id", "year", "month").Unique(),
	}
}
