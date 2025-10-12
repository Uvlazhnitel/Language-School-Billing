package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Enrollment struct{ ent.Schema }

func (Enrollment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("student_id"),
		field.Int("course_id"),
		field.Enum("billing_mode").Values("subscription", "per_lesson"),
		field.Time("start_date"),
		field.Time("end_date").Nillable().Optional(),
		field.Float("discount_pct").Default(0),
		field.String("note").Default(""),
	}
}

func (Enrollment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("student", Student.Type).Ref("enrollments").Unique().Field("student_id"),
		edge.From("course", Course.Type).Ref("enrollments").Unique().Field("course_id"),
	}
}
