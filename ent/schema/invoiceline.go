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
    // ВЛАДЕЛЕЦ FK: строка принадлежит одному счету (M:1) — обязываем и уникализируем ссылку в рамках ребра
    edge.From("invoice", Invoice.Type).
      Ref("lines").
      Field("invoice_id").
      Required().
      Unique(),

    // ВЛАДЕЛЕЦ FK: строка относится к одному зачислению (M:1)
    edge.From("enrollment", Enrollment.Type).
      Ref("invoice_lines").
      Field("enrollment_id").
      Required().
      Unique(),
  }
}
