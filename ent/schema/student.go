package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Student struct{ ent.Schema }

func (Student) Fields() []ent.Field {
	return []ent.Field{
		field.String("full_name"),
		field.String("phone").Default(""),
		field.String("email").Default(""),
		field.String("note").Default(""),
		field.Bool("is_active").Default(true),
	}
}

func (Student) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("enrollments", Enrollment.Type),
		edge.To("invoices", Invoice.Type),
	}
}
