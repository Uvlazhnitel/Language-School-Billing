// ent/schema/settings.go
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Settings struct{ ent.Schema }

func (Settings) Fields() []ent.Field {
	return []ent.Field{
		field.Int("singleton_id").Unique(),
		field.String("org_name").Default(""),
		field.String("address").Default(""),
		field.String("invoice_prefix").Default("LS"),
		field.Int("next_seq").Default(1),
		field.Int("invoice_day_of_month").Default(1),
		field.Bool("auto_issue").Default(false),
		field.String("currency").Default("EUR"),
		field.String("locale").Default("ru-RU"),
	}
}
