package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AttendanceMonth struct{ ent.Schema }

func (AttendanceMonth) Fields() []ent.Field {
	return []ent.Field{
		field.Int("student_id"),
		field.Int("course_id"),
		field.Int("year"),
		field.Int("month"),
		field.Int("lessons_count").Default(0),
		field.Bool("locked").Default(false),
		field.Enum("source").Values("monthly", "daily_agg").Default("monthly"),
	}
}

func (AttendanceMonth) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("student_id", "course_id", "year", "month").Unique(),
	}
}
