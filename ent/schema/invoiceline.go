package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
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
		edge.From("invoice", Invoice.Type).Ref("lines").Unique().Field("invoice_id"),
		edge.From("enrollment", Enrollment.Type).Ref("invoice_lines").Unique().Field("enrollment_id"),
	}
}
