package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Invoice struct{ ent.Schema }

func (Invoice) Mixin() []ent.Mixin {
	return []ent.Mixin{
		VersionMixin{},
	}
}

func (Invoice) Fields() []ent.Field {
	return []ent.Field{
		field.Int("student_id"),
		field.Int("period_year"),
		field.Int("period_month"),
		field.Float("legacy_total_amount").StorageKey("total_amount").Default(0),
		field.Int64("total_amount_cents").Default(0),
		field.Enum("status").Values("draft", "issued", "paid", "canceled").Default("draft"),
		field.String("number").Nillable().Optional(),
		field.String("pdf_filename").Nillable().Optional(),
		field.Time("pdf_generated_at").Optional().Nillable(),
		field.Int("pdf_revision").Optional().Nillable(),
		field.Time("created_at").Optional().Nillable().Default(time.Now),
		field.Time("updated_at").Optional().Nillable().Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Invoice) Edges() []ent.Edge {
	return []ent.Edge{
		// OWNER FK: invoice.student_id -> edge.From + Field + Required
		edge.From("student", Student.Type).
			Ref("invoices").
			Field("student_id").
			Required().
			Unique(), // one invoice has exactly one student

		// Inverse side to InvoiceLine.invoice (FK in the lines)
		edge.To("lines", InvoiceLine.Type),
		edge.To("payments", Payment.Type),
	}
}

func (Invoice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("student_id", "period_year", "period_month").Unique(),
	}
}
