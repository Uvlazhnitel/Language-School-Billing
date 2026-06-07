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
		field.Float("qty"),
		field.Float("legacy_unit_price").StorageKey("unit_price").Default(0),
		field.Float("legacy_amount").StorageKey("amount").Default(0),
		field.Int64("unit_price_cents").Default(0),
		field.Int64("amount_cents").Default(0),
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
