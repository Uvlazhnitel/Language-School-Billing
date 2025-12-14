package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Payment struct{ ent.Schema }

func (Payment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("student_id"),
		field.Int("invoice_id").Optional().Nillable(),
		field.Time("paid_at").Default(time.Now),
		field.Float("amount"),
		field.Enum("method").Values("cash", "bank"),
		field.String("note").Default(""),
		field.Time("created_at").Default(time.Now),
	}
}

func (Payment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("student", Student.Type).Ref("payments").Unique().Field("student_id"),
		edge.From("invoice", Invoice.Type).Ref("payments").Unique().Field("invoice_id"),
	}
}

func (Payment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("student_id", "paid_at"),
		index.Fields("invoice_id"),
	}
}
