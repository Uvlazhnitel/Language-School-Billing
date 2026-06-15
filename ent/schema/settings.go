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
		field.String("currency").Default("EUR"),
		field.String("locale").Default("en-US"),
		field.String("invoice_email_subject_template").Default(""),
		field.String("invoice_email_body_template").Default(""),
		field.String("invoice_reply_to").Default(""),
		field.Bool("money_cents_migrated").Default(false),
	}
}
