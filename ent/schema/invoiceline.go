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
		// MANY lines -> ONE invoice (FK in invoice_id of the line)
		edge.From("invoice", Invoice.Type).
			Ref("lines").
			Field("invoice_id").
			Required().
			Unique(), // IMPORTANT: a line belongs to one invoice

		// MANY lines -> ONE enrollment (FK in enrollment_id of the line)
		edge.From("enrollment", Enrollment.Type).
			Ref("invoice_lines").
			Field("enrollment_id").
			Required().
			Unique(), // IMPORTANT: a line belongs to one enrollment
	}
}

func (InvoiceLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("invoice_id"),
		index.Fields("enrollment_id"),
	}
}
