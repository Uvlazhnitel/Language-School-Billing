package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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
		// ВЛАДЕЛЕЦ FK: FK хранится в таблице enrollment -> edge.From + Field + Required
		edge.From("student", Student.Type).
			Ref("enrollments").
			Field("student_id").
			Required().
			Unique(), // у одного enrollment ровно 1 student

		edge.From("course", Course.Type).
			Ref("enrollments").
			Field("course_id").
			Required().
			Unique(), // у одного enrollment ровно 1 course

		// ОБРАТНЫЕ стороны (владельцы FK — другие сущности):
		edge.From("invoice_lines", InvoiceLine.Type).Ref("enrollment"),
		edge.From("price_overrides", PriceOverride.Type).Ref("enrollment"),
	}
}

func (Enrollment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("student_id", "course_id").Unique(),
	}

}
