package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Invoice struct{ ent.Schema }

func (Invoice) Fields() []ent.Field {
	return []ent.Field{
		field.Int("student_id"),
		field.Int("period_year"),
		field.Int("period_month"),
		field.Float("total_amount").Default(0),
		field.Enum("status").Values("draft", "issued", "paid", "canceled").Default("draft"),
		field.String("number").Nillable().Optional(), // заполнится на этапе "выставления"
	}
}

func (Invoice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("student", Student.Type).Ref("invoices").Unique().Field("student_id"),
		edge.To("lines", InvoiceLine.Type),
	}
}

func (Invoice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("student_id", "period_year", "period_month").Unique(), // один инвойс на ученика/месяц
	}
}
