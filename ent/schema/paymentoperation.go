package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PaymentOperation survives payment deletion so retrying an old request cannot recreate it.
type PaymentOperation struct{ ent.Schema }

func (PaymentOperation) Fields() []ent.Field {
	return []ent.Field{
		field.Int("actor_id"),
		field.String("operation_id"),
		field.String("request_hash"),
		field.String("result_json").Default(""),
		field.Time("created_at").Default(time.Now),
	}
}

func (PaymentOperation) Indexes() []ent.Index {
	return []ent.Index{index.Fields("actor_id", "operation_id").Unique()}
}
