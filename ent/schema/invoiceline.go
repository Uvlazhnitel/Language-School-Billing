package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type InvoiceLine struct{ ent.Schema }

func (InvoiceLine) Fields() []ent.Field {
	return []ent.Field{
		field.Int("invoice_id"),
		field.Int("enrollment_id"),
		field.String("description"),
		field.Int("qty"),
		field.Float("unit_price"),
		field.Float("amount"),
	}
}

func (InvoiceLine) Edges() []ent.Edge {
	return []ent.Edge{
		// МНОГО строк -> ОДИН счёт (FK хранится в invoice_id у строки)
		edge.From("invoice", Invoice.Type).
			Ref("lines").
			Field("invoice_id").
			Required().
			Unique(),

		// МНОГО строк (в разные месяцы) -> ОДНО зачисление (FK enrollment_id)
		edge.From("enrollment", Enrollment.Type).
			Ref("invoice_lines").
			Field("enrollment_id").
			Required().
			Unique(),
	}
}

func (InvoiceLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("invoice_id"),
		index.Fields("enrollment_id"),
	}
}
