package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Enrollment struct{ ent.Schema }

func (Enrollment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		VersionMixin{},
	}
}

func (Enrollment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("student_id"),
		field.Int("course_id"),
		field.Enum("billing_mode").Values("subscription", "per_lesson"),
		field.Bool("charge_materials").Default(true),
		field.Float("discount_pct").Default(0),
		field.Float("subscription_discount_pct").Default(20),
		field.String("note").Default(""),
		field.Time("created_at").Optional().Nillable().Default(time.Now),
		field.Time("updated_at").Optional().Nillable().Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Enrollment) Edges() []ent.Edge {
	return []ent.Edge{
		// OWNER FK: The FK is stored in the enrollment table -> edge.From + Field + Required
		edge.From("student", Student.Type).
			Ref("enrollments").
			Field("student_id").
			Required().
			Unique(), // one enrollment has exactly 1 student

		edge.From("course", Course.Type).
			Ref("enrollments").
			Field("course_id").
			Required().
			Unique(), // one enrollment has exactly 1 course

		// Inverse sides to the FK owners from other tables:
		edge.To("invoice_lines", InvoiceLine.Type), // inverse to InvoiceLine.enrollment
	}
}

func (Enrollment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("student_id", "course_id").Unique(), // unique pair student+course
	}
}
